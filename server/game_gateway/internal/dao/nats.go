package dao

import (
	"fmt"
	"strconv"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type NatsClient struct {
	conn *nats.Conn
}

func NewNATSClient(addr string) (*NatsClient, error) {
	nc, err := nats.Connect(addr)
	if err != nil {
		return nil, err
	}
	log.Info().Str("addr", addr).Msg("connected to NATS")
	return &NatsClient{conn: nc}, nil
}

func (c *NatsClient) PublishRoomCreated(roomID uint64, gameServerAddr string, initRating float64) error {
	data := []byte(fmt.Sprintf("%d:%s:%f", roomID, gameServerAddr, initRating))
	return c.conn.Publish("room.created", data)
}

func (c *NatsClient) PublishRoomDestroyed(roomID uint64) error {
	data := []byte(strconv.FormatUint(roomID, 10))
	return c.conn.Publish("room.destroyed", data)
}

func (c *NatsClient) Close() {
	c.conn.Close()
}
