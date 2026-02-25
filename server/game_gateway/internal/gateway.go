package internal

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog/log"
	"github.com/xtaci/kcp-go/v5"
	"github.com/zrurf/quiver/server/game_gateway/internal/dao"
	"github.com/zrurf/quiver/server/game_gateway/internal/proto/net_proto" // 由 net.fbs.txt 生成
	"golang.org/x/time/rate"
)

// 会话状态
const (
	SessionStateUnauthed = iota // 未认证
	SessionStateAuthed          // 已认证但未加入房间
	SessionStateInRoom          // 已加入房间
)

// ClientSession 代表一个客户端连接
type ClientSession struct {
	conn           *kcp.UDPSession
	sessionID      uint64 // 网关生成的会话ID
	uid            int64  // 用户ID（认证后有效）
	state          int    // 会话状态
	roomID         uint64 // 当前所在房间ID（0表示未加入）
	gameServerAddr string // 当前房间对应的游戏服务器地址
	remoteAddr     string // 客户端地址（用于限流日志）
	lastHeartbeat  time.Time
	mu             sync.RWMutex
}

// Gateway 网关主结构
type Gateway struct {
	config        *Config
	roomDao       *dao.RoomRepository
	sessionDao    *dao.SessionRepository
	natsDao       *dao.NatsClient
	clients       map[uint64]*ClientSession // sessionID -> session
	clientsByUID  map[int64]*ClientSession  // uid -> session
	rooms         map[uint64]*RoomSession   // roomID -> room 信息（缓存）
	rateLimiter   *IPRateLimiter
	zstdDecoder   *zstd.Decoder
	zstdEncoder   *zstd.Encoder
	nextSessionID uint64                           // 原子递增生成sessionID
	gameConns     map[string]*GameServerConnection // 游戏服务器连接池（地址->连接）
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// RoomSession 房间信息（网关侧缓存）
type RoomSession struct {
	RoomID      uint64
	GameServer  string // 游戏服务器地址 "ip:port"
	PlayerCount int32
	AvgRating   float64
	ExpireAt    time.Time // 空闲超时时间
}

// GameServerConnection 到游戏服务器的连接
type GameServerConnection struct {
	conn net.Conn
	mu   sync.Mutex
}

// NewGateway 创建网关实例
func NewGateway(cfg *Config, sessionDao *dao.SessionRepository, roomDao *dao.RoomRepository, nataDao *dao.NatsClient) *Gateway {
	ctx, cancel := context.WithCancel(context.Background())
	return &Gateway{
		config:        cfg,
		roomDao:       roomDao,
		sessionDao:    sessionDao,
		natsDao:       nataDao,
		clients:       make(map[uint64]*ClientSession),
		clientsByUID:  make(map[int64]*ClientSession),
		rooms:         make(map[uint64]*RoomSession),
		rateLimiter:   NewIPRateLimiter(rate.Limit(cfg.Server.RateLimit), cfg.Server.RateLimit),
		gameConns:     make(map[string]*GameServerConnection),
		nextSessionID: 1,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start 启动网关
func (g *Gateway) Start() {
	addr := fmt.Sprintf(":%d", g.config.Server.KCPPort)
	listener, err := kcp.ListenWithOptions(addr, nil, 0, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen KCP")
	}
	defer listener.Close()

	log.Info().Msgf("KCP gateway listening on %s", addr)

	// 启动空闲房间清理协程
	go g.cleanIdleRooms()

	for {
		conn, err := listener.AcceptKCP()
		if err != nil {
			log.Error().Err(err).Msg("accept KCP connection error")
			continue
		}
		// 设置KCP参数（低延迟模式）
		conn.SetWriteDelay(false)
		conn.SetNoDelay(1, 20, 2, 1)

		// 获取客户端IP
		remoteAddr := conn.RemoteAddr().String()
		ip, _, _ := net.SplitHostPort(remoteAddr)

		// 限流
		if !g.rateLimiter.Allow(ip) {
			log.Warn().Str("ip", ip).Msg("rate limit exceeded, closing connection")
			conn.Close()
			continue
		}

		// 处理新连接
		go g.handleClient(conn, ip)
	}
}

// handleClient 处理单个客户端连接
func (g *Gateway) handleClient(conn *kcp.UDPSession, ip string) {
	sessionID := g.generateSessionID()
	client := &ClientSession{
		conn:          conn,
		sessionID:     sessionID,
		state:         SessionStateUnauthed,
		remoteAddr:    ip,
		lastHeartbeat: time.Now(),
	}

	g.mu.Lock()
	g.clients[sessionID] = client
	g.mu.Unlock()

	log.Info().Uint64("session", sessionID).Str("ip", ip).Msg("new client connected")

	// 启动读循环
	defer func() {
		g.mu.Lock()
		delete(g.clients, sessionID)
		if client.uid != 0 {
			delete(g.clientsByUID, client.uid)
		}
		g.mu.Unlock()
		conn.Close()
		log.Info().Uint64("session", sessionID).Msg("client disconnected")
	}()

	// 读循环：处理消息
	buf := make([]byte, 65536) // 最大消息长度
	for {
		// 设置心跳超时检查
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		if n < 4 {
			continue // 无效包
		}

		// 处理消息（长度头 + FlatBuffers数据）
		if err := g.processMessage(client, buf[:n]); err != nil {
			log.Error().Err(err).Uint64("session", sessionID).Msg("process message error")
			break
		}

		// 更新最后活动时间
		client.mu.Lock()
		client.lastHeartbeat = time.Now()
		client.mu.Unlock()
	}
}

// processMessage 解析并处理一个完整的消息包
// 包格式：4字节长度（小端）+ FlatBuffers数据
func (g *Gateway) processMessage(client *ClientSession, data []byte) error {
	if !g.rateLimiter.Allow(client.remoteAddr) {
		return errors.New("rate limit exceeded")
	}
	if len(data) < 4 {
		return errors.New("packet too short")
	}
	// 读取长度
	msgLen := binary.LittleEndian.Uint32(data[:4])
	if int(msgLen)+4 != len(data) {
		return errors.New("packet length mismatch")
	}
	fbData := data[4 : 4+msgLen]

	// 解析FlatBuffers Message
	msg := net_proto.GetRootAsMessage(fbData, 0)

	// 获取头部
	header := msg.Header(nil)
	if header == nil {
		return errors.New("missing packet header")
	}

	// 获取 union 体的 table
	var tab flatbuffers.Table
	if !msg.Body(&tab) {
		return errors.New("empty message body")
	}

	// 根据消息类型分发
	switch msg.BodyType() {
	case net_proto.AnyMessageAuthRequest:
		req := net_proto.AuthRequest{}
		req.Init(tab.Bytes, tab.Pos)
		return g.handleAuth(client, header, &req)

	case net_proto.AnyMessageJoinRoom:
		req := net_proto.JoinRoom{}
		req.Init(tab.Bytes, tab.Pos)
		return g.handleJoinRoom(client, header, &req)

	case net_proto.AnyMessageGameData:
		req := net_proto.GameData{}
		req.Init(tab.Bytes, tab.Pos)
		return g.handleGameData(client, header, &req)

	case net_proto.AnyMessageHeartbeat:
		req := net_proto.Heartbeat{}
		req.Init(tab.Bytes, tab.Pos)
		return g.handleHeartbeat(client, header, &req)

	default:
		log.Warn().Uint64("session", client.sessionID).Uint16("type", uint16(msg.BodyType())).Msg("unknown message type")
		return nil
	}
}

// handleAuth 处理认证请求
func (g *Gateway) handleAuth(client *ClientSession, header *net_proto.PacketHeader, req *net_proto.AuthRequest) error {
	token := req.Token()
	if token == nil {
		return g.sendAuthResponse(client, false, "missing token")
	}

	// 从内存数据库验证token并获取uid
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	uid, err := g.sessionDao.GetUidByAccessToken(ctx, string(token))
	if err != nil {
		log.Error().Err(err).Str("token", string(token)).Msg("token verification failed")
		return g.sendAuthResponse(client, false, "invalid token")
	}

	// 更新会话
	client.mu.Lock()
	client.uid = uid
	client.state = SessionStateAuthed
	client.mu.Unlock()

	g.mu.Lock()
	g.clientsByUID[uid] = client
	g.mu.Unlock()

	log.Info().Int64("uid", uid).Uint64("session", client.sessionID).Msg("user authenticated")

	// 发送成功响应
	return g.sendAuthResponse(client, true, "")
}

// sendAuthResponse 发送认证响应
func (g *Gateway) sendAuthResponse(client *ClientSession, success bool, errMsg string) error {
	builder := flatbuffers.NewBuilder(256)

	// 构建 AuthResponse
	net_proto.AuthResponseStart(builder)
	net_proto.AuthResponseAddSuccess(builder, success)
	if !success {
		errOff := builder.CreateString(errMsg)
		net_proto.AuthResponseAddErrorMessage(builder, errOff)
	}
	respOff := net_proto.AuthResponseEnd(builder)

	// 构建并发送消息
	return g.buildAndSendMessage(client, net_proto.AnyMessageAuthResponse, respOff, builder)
}

// handleJoinRoom 处理加入房间请求
func (g *Gateway) handleJoinRoom(client *ClientSession, header *net_proto.PacketHeader, req *net_proto.JoinRoom) error {
	// 检查认证状态
	client.mu.RLock()
	uid := client.uid
	state := client.state
	client.mu.RUnlock()

	if state < SessionStateAuthed {
		return g.sendJoinRoomResponse(client, false, 0, "", "not authenticated")
	}

	// 获取玩家rating（用于匹配）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rating, rd, err := g.roomDao.GetPlayerRating(ctx, uid)
	if err != nil {
		log.Error().Err(err).Int64("uid", uid).Msg("get player rating failed")
		return g.sendJoinRoomResponse(client, false, 0, "", "internal error")
	}

	// 根据模式和rating匹配房间
	mode := req.Mode()
	var targetRoomID uint64
	var gameServerAddr string

	switch mode {
	case net_proto.GameModeQuickMatch:
		// 根据rating寻找合适的房间
		targetRoomID, gameServerAddr = g.findRoomByRating(rating, rd)
		if targetRoomID == 0 {
			// 没有合适房间，创建新房间
			targetRoomID, gameServerAddr = g.createRoomOnGameServer(rating)
		}
	case net_proto.GameModeSpecificRoom:
		// 指定房间
		targetRoomID = req.TargetRoomId()
		gameServerAddr = g.getGameServerForRoom(targetRoomID)
		if gameServerAddr == "" {
			return g.sendJoinRoomResponse(client, false, 0, "", "room not found")
		}
	default:
		return g.sendJoinRoomResponse(client, false, 0, "", "invalid mode")
	}

	// 更新客户端状态
	client.mu.Lock()
	client.roomID = targetRoomID
	client.gameServerAddr = gameServerAddr
	client.state = SessionStateInRoom
	client.mu.Unlock()

	// 更新房间缓存人数
	g.mu.Lock()
	if room, ok := g.rooms[targetRoomID]; ok {
		room.PlayerCount++
	}
	g.mu.Unlock()

	log.Info().Int64("uid", uid).Uint64("room", targetRoomID).Str("gs", gameServerAddr).Msg("joined room")

	// 发送响应
	return g.sendJoinRoomResponse(client, true, targetRoomID, gameServerAddr, "")
}

// findRoomByRating 根据评分寻找合适的房间
func (g *Gateway) findRoomByRating(rating, rd float64) (uint64, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rooms, err := g.roomDao.GetActiveRooms(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get active rooms from imdb")
		return 0, ""
	}
	var bestRoom uint64
	var bestAddr string
	var bestScore float64 = -1
	for _, room := range rooms {
		if room.PlayerCount >= g.config.Play.MaxPlayersPerRoom {
			continue
		}
		// 根据评分差和 RD 计算匹配度
		diff := abs(room.AvgRating - rating)
		// 考虑 RD 因素
		if diff < 200 {
			score := diff
			if bestRoom == 0 || score < bestScore {
				bestScore = score
				bestRoom = room.ID
				bestAddr = room.Addr
			}
		}
	}
	return bestRoom, bestAddr
}

// createRoomOnGameServer 在某个游戏服务器上创建新房间
// 返回新房间ID和游戏服务器地址
func (g *Gateway) createRoomOnGameServer(initRating float64) (uint64, string) {
	servers := g.config.GameServers
	if len(servers) == 0 {
		log.Error().Msg("no game servers available")
		return 0, ""
	}
	gsAddr := servers[0]
	roomID := uint64(time.Now().UnixNano())

	// 保存到 Garnet
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := g.roomDao.SaveRoom(ctx, roomID, gsAddr, initRating, 0)
	if err != nil {
		log.Error().Err(err).Msg("save room to imdb failed")
	}

	// 发送 NATS 通知
	if g.natsDao != nil {
		if err := g.natsDao.PublishRoomCreated(roomID, gsAddr, initRating); err != nil {
			log.Error().Err(err).Msg("publish room.created failed")
		}
	}

	// 本地缓存（可选）
	g.mu.Lock()
	g.rooms[roomID] = &RoomSession{
		RoomID:      roomID,
		GameServer:  gsAddr,
		PlayerCount: 0,
		AvgRating:   initRating,
		ExpireAt:    time.Now().Add(g.config.Server.IdleRoomTimeout),
	}
	g.mu.Unlock()

	return roomID, gsAddr
}

// getGameServerForRoom 获取房间对应的游戏服务器地址
func (g *Gateway) getGameServerForRoom(roomID uint64) string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if room, ok := g.rooms[roomID]; ok {
		return room.GameServer
	}
	return ""
}

// sendJoinRoomResponse 发送加入房间响应
func (g *Gateway) sendJoinRoomResponse(client *ClientSession, success bool, roomID uint64, addr, errMsg string) error {
	builder := flatbuffers.NewBuilder(256)

	// 构建 JoinRoomResponse
	net_proto.JoinRoomResponseStart(builder)
	net_proto.JoinRoomResponseAddSuccess(builder, success)
	if success {
		net_proto.JoinRoomResponseAddRoomId(builder, roomID)
		addrOff := builder.CreateString(addr)
		net_proto.JoinRoomResponseAddGameServerAddr(builder, addrOff)
	} else {
		net_proto.JoinRoomResponseAddErrorCode(builder, -1)
	}
	respOff := net_proto.JoinRoomResponseEnd(builder)
	return g.buildAndSendMessage(client, net_proto.AnyMessageJoinRoomResponse, respOff, builder)
}

// handleGameData 处理游戏数据（透传到对应的游戏服务器）
func (g *Gateway) handleGameData(client *ClientSession, header *net_proto.PacketHeader, req *net_proto.GameData) error {
	client.mu.RLock()
	roomID := client.roomID
	gsAddr := client.gameServerAddr
	uid := client.uid
	state := client.state
	client.mu.RUnlock()

	if state != SessionStateInRoom {
		return errors.New("not in a room")
	}
	if roomID == 0 || gsAddr == "" {
		return errors.New("invalid room state")
	}

	// 获取游戏数据字节
	dataBytes := req.DataBytes()
	if len(dataBytes) == 0 {
		return nil
	}

	// 转发到对应的游戏服务器
	go g.forwardGameData(gsAddr, roomID, uid, dataBytes)

	return nil
}

// forwardGameData 将客户端游戏数据转发给游戏服务器
// 转发格式：8字节房间ID + 8字节UID + 数据
func (g *Gateway) forwardGameData(gsAddr string, roomID uint64, uid int64, data []byte) {
	// 获取或创建到游戏服务器的连接
	conn, err := g.getGameServerConn(gsAddr)
	if err != nil {
		log.Error().Err(err).Str("gs", gsAddr).Msg("failed to connect game server")
		return
	}

	// 构造转发包
	buf := make([]byte, 16+len(data))
	binary.BigEndian.PutUint64(buf[0:8], roomID)
	binary.BigEndian.PutUint64(buf[8:16], uint64(uid))
	copy(buf[16:], data)

	// 发送数据
	conn.mu.Lock()
	defer conn.mu.Unlock()
	_, err = conn.conn.Write(buf)
	if err != nil {
		log.Error().Err(err).Str("gs", gsAddr).Msg("forward data error")
		// 从池中移除连接
		g.removeGameServerConn(gsAddr)
	}
}

// getGameServerConn 获取到游戏服务器的TCP连接
func (g *Gateway) getGameServerConn(addr string) (*GameServerConnection, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if conn, ok := g.gameConns[addr]; ok {
		return conn, nil
	}

	c, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	gsConn := &GameServerConnection{conn: c}
	g.gameConns[addr] = gsConn

	// 启动下行数据接收协程
	go g.handleGameServerDownlink(addr, c)

	return gsConn, nil
}

func (g *Gateway) removeGameServerConn(addr string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if conn, ok := g.gameConns[addr]; ok {
		conn.conn.Close()
		delete(g.gameConns, addr)
	}
}

// handleGameServerDownlink 处理从游戏服务器发来的下行数据
func (g *Gateway) handleGameServerDownlink(gsAddr string, conn net.Conn) {
	defer func() {
		g.removeGameServerConn(gsAddr)
		conn.Close()
	}()

	buf := make([]byte, 4096)
	for {
		// 读取4字节长度头
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			log.Error().Err(err).Str("gs", gsAddr).Msg("read downlink length error")
			return
		}
		msgLen := binary.LittleEndian.Uint32(buf[:4])
		if msgLen > 65536 {
			log.Error().Uint32("len", msgLen).Msg("downlink packet too large")
			return
		}
		// 确保缓冲区足够
		if cap(buf) < int(msgLen) {
			buf = make([]byte, msgLen)
		} else {
			buf = buf[:msgLen]
		}
		if _, err := io.ReadFull(conn, buf); err != nil {
			log.Error().Err(err).Msg("read downlink body error")
			return
		}

		// 解析内部头部：8字节房间ID + 8字节目标UID + 数据
		if msgLen < 16 {
			log.Error().Msg("downlink packet too short")
			continue
		}
		roomID := binary.BigEndian.Uint64(buf[:8])
		targetUID := binary.BigEndian.Uint64(buf[8:16])
		payload := buf[16:msgLen]

		// 查找目标客户端并发送
		g.mu.RLock()
		if targetUID == 0 {
			// 广播：收集房间内所有 client 的引用，然后逐个发送
			var targets []*ClientSession
			g.mu.RLock()
			for _, client := range g.clients {
				client.mu.RLock()
				if client.roomID == roomID && client.state == SessionStateInRoom {
					targets = append(targets, client)
				}
				client.mu.RUnlock()
			}
			g.mu.RUnlock()
			for _, client := range targets {
				g.sendGameDataToClient(client, payload)
			}
		} else {
			// 单播
			g.mu.RLock()
			client, ok := g.clientsByUID[int64(targetUID)]
			g.mu.RUnlock()
			if ok {
				client.mu.RLock()
				if client.roomID == roomID && client.state == SessionStateInRoom {
					g.sendGameDataToClient(client, payload)
				}
				client.mu.RUnlock()
			}
		}
		g.mu.RUnlock()
	}
}

// sendGameDataToClient 向客户端发送 GameData 消息
func (g *Gateway) sendGameDataToClient(client *ClientSession, data []byte) {
	builder := flatbuffers.NewBuilder(len(data) + 64)
	dataOff := builder.CreateByteVector(data)
	net_proto.GameDataStart(builder)
	net_proto.GameDataAddData(builder, dataOff)
	bodyOff := net_proto.GameDataEnd(builder)

	g.buildAndSendMessage(client, net_proto.AnyMessageGameData, bodyOff, builder)
}

// handleHeartbeat 处理心跳
func (g *Gateway) handleHeartbeat(client *ClientSession, header *net_proto.PacketHeader, req *net_proto.Heartbeat) error {
	// 更新心跳时间
	builder := flatbuffers.NewBuilder(32)
	net_proto.HeartbeatStart(builder)
	net_proto.HeartbeatAddPing(builder, req.Ping())
	respOff := net_proto.HeartbeatEnd(builder)

	return g.buildAndSendMessage(client, net_proto.AnyMessageHeartbeat, respOff, builder)
}

// buildAndSendMessage 构建并发送消息
func (g *Gateway) buildAndSendMessage(client *ClientSession, msgType net_proto.AnyMessage, bodyOff flatbuffers.UOffsetT, builder *flatbuffers.Builder) error {
	// 构建 PacketHeader
	headerOff := net_proto.CreatePacketHeader(
		builder,
		0x4B435057, // magic
		1,          // version
		0,          // flags
		client.sessionID,
		client.roomID,
		uint16(msgType),
		0,                         // reserved
		uint64(time.Now().Unix()), // timestamp
	)

	// 构建 Message table
	net_proto.MessageStart(builder)
	net_proto.MessageAddHeader(builder, headerOff)
	net_proto.MessageAddBodyType(builder, msgType)
	net_proto.MessageAddBody(builder, bodyOff)
	msgOff := net_proto.MessageEnd(builder)

	builder.Finish(msgOff)

	// 获取 bytes
	data := builder.FinishedBytes()

	// 添加长度头并发送
	packet := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(packet[:4], uint32(len(data)))
	copy(packet[4:], data)

	_, err := client.conn.Write(packet)
	return err
}

// generateSessionID 生成唯一会话ID
func (g *Gateway) generateSessionID() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := g.nextSessionID
	g.nextSessionID++
	return id
}

// cleanIdleRooms 定期清理空闲房间
func (g *Gateway) cleanIdleRooms() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			g.mu.Lock()
			now := time.Now()
			for id, room := range g.rooms {
				if room.PlayerCount <= 0 && now.After(room.ExpireAt) {
					delete(g.rooms, id)
					log.Info().Uint64("room", id).Msg("room cleaned due to idle timeout")

					// 发布房间销毁通知
					if g.natsDao != nil {
						if err := g.natsDao.PublishRoomDestroyed(id); err != nil {
							log.Error().Err(err).Uint64("room", id).Msg("failed to publish room.destroyed")
						}
					}
				}
			}
			g.mu.Unlock()
		}
	}
}

// Stop 停止网关
func (g *Gateway) Stop() {
	g.cancel()
	// 关闭所有客户端连接
	g.mu.Lock()
	for _, client := range g.clients {
		client.conn.Close()
	}
	for _, gsConn := range g.gameConns {
		gsConn.conn.Close()
	}
	g.mu.Unlock()
}

// abs 浮点数绝对值
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
