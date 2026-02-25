package game

import (
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/zrurf/quiver/server/game/internal/proto/game_proto"
)

type Projectile struct {
	ID        uint64
	OwnerUID  int64
	PosX      float32
	PosY      float32
	VelX      float32
	VelY      float32
	Type      uint8
	Damage    int
	LifeTime  int64 // 毫秒
	CreatedAt int64
}

// Update 根据时间差更新位置
func (p *Projectile) Update(dtMs int64) {
	dt := float32(dtMs) / 1000.0 // 转换为秒
	p.PosX += p.VelX * dt
	p.PosY += p.VelY * dt
}

// ToProto 转换为 FlatBuffers 的 ProjectileState
func (p *Projectile) ToProto(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	game_proto.ProjectileStateStart(builder)
	game_proto.ProjectileStateAddId(builder, p.ID)
	game_proto.ProjectileStateAddOwnerUid(builder, uint64(p.OwnerUID))
	game_proto.ProjectileStateAddPosX(builder, p.PosX)
	game_proto.ProjectileStateAddPosY(builder, p.PosY)
	game_proto.ProjectileStateAddVelX(builder, p.VelX)
	game_proto.ProjectileStateAddVelY(builder, p.VelY)
	game_proto.ProjectileStateAddProjType(builder, p.Type)
	return game_proto.ProjectileStateEnd(builder)
}
