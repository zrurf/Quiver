package internal

import (
	"time"
)

type Config struct {
	Server struct {
		Listen          string        `mapstructure:"listen"`
		KCPPort         int           `mapstructure:"kcp-port"`
		InternalListen  int           `mapstructure:"internal-listen"`
		IdleRoomTimeout time.Duration `mapstructure:"idle-room-timeout"`
		RateLimit       int           `mapstructure:"rate-limit"`
	} `mapstructure:"server"`
	Logger struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"logger"`
	UserServer struct {
		URL string `mapstructure:"url"`
	} `mapstructure:"user-server"`
	GameServers []string `mapstructure:"game-servers"`
	Database    struct {
		Host     string `mapstructure:"host"`
		Port     string `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"name"`
	} `mapstructure:"database"`
	Imdb struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"imdb"`
	Mq struct {
		Addr    string `mapstructure:"addr"`
		Subject string `mapstructure:"subject"`
	} `mapstructure:"mq"`
	Play struct {
		MaxPlayersPerRoom int `mapstructure:"max-players-per-room"`
	} `mapstructure:"play"`
}
