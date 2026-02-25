package game

import (
	"context"
	"math"
	"sync"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/rs/zerolog/log"
	"github.com/zrurf/quiver/server/game/internal"
	"github.com/zrurf/quiver/server/game/internal/dao"
	"github.com/zrurf/quiver/server/game/internal/model"
	"github.com/zrurf/quiver/server/game/internal/proto/game_proto"
)

const (
	maxPlayerSpeed float32 = 250
)

type Room struct {
	id            uint64
	cfg           *internal.Config
	playerDAO     *dao.PlayerDAO
	enc           *internal.Encryptor
	comp          *internal.Compressor
	avgRating     float64
	players       map[int64]*model.Player
	playersMu     sync.RWMutex
	projectiles   []*Projectile
	projectilesMu sync.RWMutex
	events        []*GameEvent
	eventsMu      sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
	onDestroy     func(roomID uint64)
	lastActivity  time.Time
	sendFunc      func(roomID uint64, targetUID int64, data []byte)
	nextProjID    uint64
	roomKey       []byte // 房间对称密钥
}

func NewRoom(id uint64, cfg *internal.Config, playerDAO *dao.PlayerDAO,
	enc *internal.Encryptor, comp *internal.Compressor, onDestroy func(uint64),
	sendFunc func(uint64, int64, []byte),
	initRating ...float64) *Room {
	rating := 1500.0
	if len(initRating) > 0 {
		rating = initRating[0]
	}
	r := &Room{
		id:           id,
		cfg:          cfg,
		playerDAO:    playerDAO,
		enc:          enc,
		comp:         comp,
		avgRating:    rating,
		players:      make(map[int64]*model.Player),
		projectiles:  make([]*Projectile, 0),
		events:       make([]*GameEvent, 0),
		stopCh:       make(chan struct{}),
		onDestroy:    onDestroy,
		lastActivity: time.Now(),
		sendFunc:     sendFunc,
		nextProjID:   1,
		roomKey:      enc.GetRoomKey(),
	}
	r.wg.Add(1)
	go r.gameLoop()
	return r
}

func (r *Room) gameLoop() {
	ticker := time.NewTicker(30 * time.Millisecond)
	ratingTicker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	defer ratingTicker.Stop()
	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.update()
			r.broadcastState()
			if time.Since(r.lastActivity) > r.cfg.Server.IdleRoomTimeout {
				log.Info().Uint64("room", r.id).Msg("room idle timeout, stopping")
				r.Stop()
				return
			}
		case <-ratingTicker.C:
			r.updateAvgRating()
		}
	}
}

func (r *Room) update() {
	now := time.Now().UnixMilli()
	dt := int64(30)

	// 更新玩家
	r.playersMu.RLock()
	players := make([]*model.Player, 0, len(r.players))
	for _, p := range r.players {
		players = append(players, p)
	}
	r.playersMu.RUnlock()

	for _, p := range players {
		x, y, _, _ := p.GetPosition()
		if x < 0 || x > 1000 || y < 0 || y > 1000 {
			p.SetPosition(clamp(x, 0, 1000), clamp(y, 0, 1000), 0, 0)
		}
	}

	// 更新抛射物
	r.projectilesMu.Lock()
	active := r.projectiles[:0]
	for _, proj := range r.projectiles {
		proj.Update(dt)
		if now-proj.CreatedAt > proj.LifeTime {
			continue
		}
		active = append(active, proj)
	}
	r.projectiles = active
	r.projectilesMu.Unlock()

	r.lastActivity = time.Now()
}

func (r *Room) broadcastState() {
	r.playersMu.RLock()
	defer r.playersMu.RUnlock()

	builder := flatbuffers.NewBuilder(2048)

	// 构建玩家状态列表
	playerStates := make([]flatbuffers.UOffsetT, 0, len(r.players))
	for _, p := range r.players {
		x, y, _, _ := p.GetPosition()
		health := p.GetHealth()
		buffs := p.GetBuffs() // 获取副本
		buffsOff := builder.CreateByteVector(buffs)
		game_proto.PlayerStateStart(builder)
		game_proto.PlayerStateAddUid(builder, uint64(p.UID))
		game_proto.PlayerStateAddPosX(builder, x)
		game_proto.PlayerStateAddPosY(builder, y)
		game_proto.PlayerStateAddHealth(builder, uint32(health))
		game_proto.PlayerStateAddBuffs(builder, buffsOff)
		playerStates = append(playerStates, game_proto.PlayerStateEnd(builder))
	}
	game_proto.GameStateUpdateStartPlayersVector(builder, len(playerStates))
	for _, off := range playerStates {
		builder.PrependUOffsetT(off)
	}
	playersVec := builder.EndVector(len(playerStates))

	// 抛射物
	r.projectilesMu.RLock()
	projStates := make([]flatbuffers.UOffsetT, len(r.projectiles))
	for i, proj := range r.projectiles {
		projStates[i] = proj.ToProto(builder)
	}
	r.projectilesMu.RUnlock()
	game_proto.GameStateUpdateStartProjectilesVector(builder, len(projStates))
	for _, off := range projStates {
		builder.PrependUOffsetT(off)
	}
	projVec := builder.EndVector(len(projStates))

	// 事件
	r.eventsMu.RLock()
	eventStates := make([]flatbuffers.UOffsetT, len(r.events))
	for i, ev := range r.events {
		eventStates[i] = ev.ToProto(builder)
	}
	r.eventsMu.RUnlock()
	game_proto.GameStateUpdateStartEventsVector(builder, len(eventStates))
	for _, off := range eventStates {
		builder.PrependUOffsetT(off)
	}
	eventVec := builder.EndVector(len(eventStates))

	game_proto.GameStateUpdateStart(builder)
	game_proto.GameStateUpdateAddPlayers(builder, playersVec)
	game_proto.GameStateUpdateAddProjectiles(builder, projVec)
	game_proto.GameStateUpdateAddEvents(builder, eventVec)
	game_proto.GameStateUpdateAddTimestamp(builder, uint64(time.Now().UnixMilli()))
	updateOff := game_proto.GameStateUpdateEnd(builder)

	game_proto.GamePacketStart(builder)
	game_proto.GamePacketAddBodyType(builder, game_proto.GameMessageGameStateUpdate)
	game_proto.GamePacketAddBody(builder, updateOff)
	packetOff := game_proto.GamePacketEnd(builder)

	builder.Finish(packetOff)
	data := builder.FinishedBytes()

	// 使用房间密钥加密
	if r.enc != nil {
		encData, err := internal.Encrypt(data, r.roomKey) // 使用固定密钥
		if err != nil {
			log.Error().Err(err).Msg("encrypt failed in broadcast")
			return
		}
		data = encData
	}
	if r.comp != nil {
		data = r.comp.Compress(data)
	}

	// 广播给所有玩家
	r.sendFunc(r.id, 0, data)

	// 清空事件（已广播）
	r.eventsMu.Lock()
	r.events = r.events[:0]
	r.eventsMu.Unlock()
}

func (r *Room) HandleClientData(uid int64, payload []byte) {
	r.lastActivity = time.Now()
	var err error
	if r.enc != nil {
		payload, err = internal.Decrypt(payload, r.roomKey) // 使用固定密钥
		if err != nil {
			log.Error().Err(err).Int64("uid", uid).Msg("decrypt failed")
			return
		}
	}
	if r.comp != nil {
		var err error
		payload, err = r.comp.Decompress(payload)
		if err != nil {
			log.Error().Err(err).Int64("uid", uid).Msg("decompress failed")
			return
		}
	}

	packet := game_proto.GetRootAsGamePacket(payload, 0)
	var tab flatbuffers.Table
	if !packet.Body(&tab) {
		log.Error().Msg("empty game packet body")
		return
	}

	// 获取或创建玩家
	r.playersMu.Lock()
	p, exists := r.players[uid]
	if !exists {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var err error
		p, err = r.playerDAO.Load(ctx, uid)
		if err != nil {
			log.Error().Err(err).Int64("uid", uid).Msg("failed to load player")
			r.playersMu.Unlock()
			return
		}
		r.players[uid] = p
		log.Info().Int64("uid", uid).Uint64("room", r.id).Msg("player joined")
	}
	r.playersMu.Unlock()

	switch packet.BodyType() {
	case game_proto.GameMessagePlayerMove:
		msg := game_proto.PlayerMove{}
		msg.Init(tab.Bytes, tab.Pos)
		r.handlePlayerMove(uid, &msg)
	case game_proto.GameMessagePlayerShoot:
		msg := game_proto.PlayerShoot{}
		msg.Init(tab.Bytes, tab.Pos)
		r.handlePlayerShoot(uid, &msg)
	default:
		log.Warn().Uint64("room", r.id).Int64("uid", uid).Uint8("type", uint8(packet.BodyType())).Msg("unknown message")
	}
}

func (r *Room) handlePlayerMove(uid int64, msg *game_proto.PlayerMove) {
	r.playersMu.RLock()
	p, ok := r.players[uid]
	r.playersMu.RUnlock()
	if !ok {
		return
	}
	vx, vy := msg.VelX(), msg.VelY()
	speed := float32(math.Sqrt(float64(vx*vx + vy*vy)))
	if speed > maxPlayerSpeed {
		scale := maxPlayerSpeed / speed
		vx *= scale
		vy *= scale
	}
	p.SetPosition(msg.PosX(), msg.PosY(), vx, vy)
}

func (r *Room) handlePlayerShoot(uid int64, msg *game_proto.PlayerShoot) {
	// 获取玩家位置
	r.playersMu.RLock()
	p, ok := r.players[uid]
	if !ok {
		r.playersMu.RUnlock()
		return
	}
	px, py, _, _ := p.GetPosition()
	r.playersMu.RUnlock()

	r.projectilesMu.Lock()
	defer r.projectilesMu.Unlock()
	proj := &Projectile{
		ID:        r.nextProjID,
		OwnerUID:  uid,
		PosX:      px, // 从玩家位置发射
		PosY:      py,
		VelX:      (msg.AimX() - px) * 2,
		VelY:      (msg.AimY() - py) * 2,
		Type:      msg.WeaponType(),
		Damage:    10,
		LifeTime:  5000,
		CreatedAt: time.Now().UnixMilli(),
	}
	r.projectiles = append(r.projectiles, proj)
	r.nextProjID++
}

func (r *Room) Stop() {
	close(r.stopCh)
	r.wg.Wait()
	ctx := context.Background()
	r.playersMu.RLock()
	for _, p := range r.players {
		if err := r.playerDAO.Save(ctx, p); err != nil {
			log.Error().Err(err).Int64("uid", p.UID).Msg("failed to save player")
		}
	}
	r.playersMu.RUnlock()
	if r.onDestroy != nil {
		r.onDestroy(r.id)
	}
	log.Info().Uint64("room", r.id).Msg("room stopped")
}

func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (r *Room) updateAvgRating() {
	r.playersMu.RLock()
	defer r.playersMu.RUnlock()
	if len(r.players) == 0 {
		return
	}
	var sum float64
	for _, p := range r.players {
		sum += p.Rating
	}
	avg := sum / float64(len(r.players))
	r.avgRating = avg

	// 写入 Garnet
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	r.playerDAO.UpdateRoomRating(ctx, r.id, avg, len(r.players))
}
