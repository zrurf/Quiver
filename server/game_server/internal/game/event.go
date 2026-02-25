package game

import (
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/zrurf/quiver/server/game/internal/proto/game_proto"
)

type GameEvent struct {
	Type uint8
	Data []byte
}

func (e *GameEvent) ToProto(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	dataOff := builder.CreateByteVector(e.Data)
	game_proto.GameEventStart(builder)
	game_proto.GameEventAddEventType(builder, e.Type)
	game_proto.GameEventAddData(builder, dataOff)
	return game_proto.GameEventEnd(builder)
}
