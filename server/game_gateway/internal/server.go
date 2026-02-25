package internal

import (
	"fmt"
	"net"
	"net/http"

	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog/log"
	"github.com/xtaci/kcp-go/v5"
	"github.com/zrurf/quiver/server/game_gateway/internal/dao"
)

func StartHealthCheck(addr string) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	if addr == "" {
		addr = ":8080"
	}
	log.Info().Msgf("Health check listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal().Err(err).Msg("Health check server failed")
	}
}

func StartKCPGateway(cfg *Config, sessionDao *dao.SessionRepository, roomDao *dao.RoomRepository, natsClient *dao.NatsClient) {
	addr := fmt.Sprintf(":%d", cfg.Server.KCPPort)
	listener, err := kcp.ListenWithOptions(addr, nil, 0, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen KCP")
	}
	defer listener.Close()

	// 初始化ZSTD压缩器
	decoder, _ := zstd.NewReader(nil)
	encoder, _ := zstd.NewWriter(nil)
	defer decoder.Close()
	defer encoder.Close()

	gateway := NewGateway(cfg, sessionDao, roomDao, natsClient)

	log.Info().Msgf("KCP gateway listening on %s", addr)

	for {
		conn, err := listener.AcceptKCP()
		if err != nil {
			log.Error().Err(err).Msg("accept KCP connection error")
			continue
		}
		// 设置KCP参数
		conn.SetWriteDelay(false)
		conn.SetNoDelay(1, 20, 2, 1)

		// 获取客户端IP
		remoteAddr := conn.RemoteAddr().String()
		ip, _, _ := net.SplitHostPort(remoteAddr)

		// 限流
		if !gateway.rateLimiter.Allow(ip) {
			log.Warn().Str("ip", ip).Msg("rate limit exceeded, closing connection")
			conn.Close()
			continue
		}

		// 处理新连接
		go gateway.handleClient(conn, ip)
	}
}
