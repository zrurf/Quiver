package server

import (
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/zrurf/quiver/server/game/internal"
	"github.com/zrurf/quiver/server/game/internal/dao"
	"github.com/zrurf/quiver/server/game/internal/game"
)

type Server struct {
	cfg         *internal.Config
	db          *pgxpool.Pool
	rdb         *redis.Client
	natsConn    *nats.Conn
	enc         *internal.Encryptor
	comp        *internal.Compressor
	playerDAO   *dao.PlayerDAO
	rooms       map[uint64]*game.Room
	roomsMu     sync.RWMutex
	listener    net.Listener
	stopCh      chan struct{}
	wg          sync.WaitGroup
	gatewayConn net.Conn
	connMu      sync.Mutex
}

func NewServer(cfg *internal.Config, db *pgxpool.Pool, rdb *redis.Client, enc *internal.Encryptor, comp *internal.Compressor) *Server {
	return &Server{
		cfg:       cfg,
		db:        db,
		rdb:       rdb,
		enc:       enc,
		comp:      comp,
		playerDAO: dao.NewPlayerDAO(db, rdb),
		rooms:     make(map[uint64]*game.Room),
		stopCh:    make(chan struct{}),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.cfg.Server.ListenAddr)
	if err != nil {
		return err
	}
	s.listener = ln
	log.Info().Msgf("game server listening on %s", s.cfg.Server.ListenAddr)

	s.wg.Add(1)
	go s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				log.Error().Err(err).Msg("accept error")
				continue
			}
		}
		// 假设只接受一个网关连接，如果有新连接则替换旧的
		s.connMu.Lock()
		if s.gatewayConn != nil {
			s.gatewayConn.Close()
		}
		s.gatewayConn = conn
		s.connMu.Unlock()
		log.Info().Str("remote", conn.RemoteAddr().String()).Msg("gateway connected")

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		conn.Close()
		s.connMu.Lock()
		if s.gatewayConn == conn {
			s.gatewayConn = nil
		}
		s.connMu.Unlock()
		s.wg.Done()
	}()

	buf := make([]byte, 4)
	for {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		if _, err := io.ReadFull(conn, buf); err != nil {
			if err != io.EOF {
				log.Error().Err(err).Msg("read length header error")
			}
			return
		}
		msgLen := binary.LittleEndian.Uint32(buf)
		if msgLen > 65536 {
			log.Error().Uint32("len", msgLen).Msg("packet too large")
			return
		}
		data := make([]byte, msgLen)
		if _, err := io.ReadFull(conn, data); err != nil {
			log.Error().Err(err).Msg("read packet body error")
			return
		}
		if len(data) < 16 {
			log.Error().Msg("packet too short")
			continue
		}
		roomID := binary.BigEndian.Uint64(data[:8])
		uid := int64(binary.BigEndian.Uint64(data[8:16]))
		payload := data[16:]

		r := s.getOrCreateRoom(roomID)
		if r == nil {
			log.Error().Uint64("room", roomID).Msg("failed to get/create room")
			continue
		}
		r.HandleClientData(uid, payload)
	}
}

func (s *Server) getOrCreateRoom(roomID uint64) *game.Room {
	s.roomsMu.RLock()
	r, ok := s.rooms[roomID]
	s.roomsMu.RUnlock()
	if ok {
		return r
	}
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	if r, ok = s.rooms[roomID]; ok {
		return r
	}
	r = game.NewRoom(roomID, s.cfg, s.playerDAO, s.enc, s.comp,
		func(roomID uint64) {
			s.roomsMu.Lock()
			delete(s.rooms, roomID)
			s.roomsMu.Unlock()
		},
		s.sendToGateway,
	)
	s.rooms[roomID] = r
	log.Info().Uint64("room", roomID).Msg("room created")
	return r
}

// sendToGateway 将数据发送给网关（添加长度头）
func (s *Server) sendToGateway(roomID uint64, targetUID int64, payload []byte) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	if s.gatewayConn == nil {
		log.Error().Msg("no gateway connection")
		return
	}
	// 组装内部头部：8字节房间ID + 8字节目标UID
	header := make([]byte, 16)
	binary.BigEndian.PutUint64(header[0:8], roomID)
	binary.BigEndian.PutUint64(header[8:16], uint64(targetUID))
	fullData := append(header, payload...)

	// 添加4字节长度头
	packet := make([]byte, 4+len(fullData))
	binary.LittleEndian.PutUint32(packet[0:4], uint32(len(fullData)))
	copy(packet[4:], fullData)

	_, err := s.gatewayConn.Write(packet)
	if err != nil {
		log.Error().Err(err).Msg("write to gateway failed")
	}
}

func (s *Server) StartNATSListener() {
	nc, err := nats.Connect(s.cfg.Mq.Addr)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to NATS")
	}
	s.natsConn = nc
	_, err = nc.Subscribe("room.created", func(msg *nats.Msg) {
		parts := strings.Split(string(msg.Data), ":")
		if len(parts) != 3 {
			return
		}
		roomID, _ := strconv.ParseUint(parts[0], 10, 64)
		initRating, _ := strconv.ParseFloat(parts[2], 64)
		// 如果房间未创建，则创建
		s.roomsMu.Lock()
		defer s.roomsMu.Unlock()
		if _, ok := s.rooms[roomID]; !ok {
			room := game.NewRoom(roomID, s.cfg, s.playerDAO, s.enc, s.comp,
				func(id uint64) { s.removeRoom(id) },
				s.sendToGateway,
				initRating)
			s.rooms[roomID] = room
			log.Info().Uint64("room", roomID).Msg("room pre-created via NATS")
		}
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to subscribe to room.created")
	}
	_, err = s.natsConn.Subscribe("room.destroyed", func(msg *nats.Msg) {
		roomID, err := strconv.ParseUint(string(msg.Data), 10, 64)
		if err != nil {
			log.Error().Err(err).Str("data", string(msg.Data)).Msg("invalid room.destroyed message")
			return
		}
		s.roomsMu.Lock()
		room, ok := s.rooms[roomID]
		if ok {
			room.Stop()
		}
		s.roomsMu.Unlock()
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to subscribe to room.destroyed")
	}
}

func (s *Server) removeRoom(roomID uint64) {
	s.roomsMu.Lock()
	s.rooms[roomID].Stop()
	delete(s.rooms, roomID)
	s.roomsMu.Unlock()
}

func (s *Server) Stop() {
	close(s.stopCh)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	s.roomsMu.Lock()
	for _, r := range s.rooms {
		r.Stop()
	}
	s.roomsMu.Unlock()
}
