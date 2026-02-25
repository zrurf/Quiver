package internal

import (
	"time"
)

type Config struct {
	Server struct {
		ListenAddr        string        `mapstructure:"listen-addr"`       // TCP 监听地址，如 ":18650"
		IdleRoomTimeout   time.Duration `mapstructure:"idle-room-timeout"` // 房间空闲超时
		MaxPlayersPerRoom int           `mapstructure:"max-players-per-room"`
	} `mapstructure:"server"`

	Logger struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"logger"`

	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"name"`
	} `mapstructure:"database"`

	IMDB struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"imdb"`

	Mq struct {
		Addr    string `mapstructure:"addr"`
		Subject string `mapstructure:"subject"`
	} `mapstructure:"mq"`

	Encryption struct {
		Enabled         bool   `mapstructure:"enabled"`           // 是否启用加密
		MasterKeyBase64 string `mapstructure:"master-key-base64"` // 主密钥（Base64 编码，32 字节）
	} `mapstructure:"encryption"`

	Compression struct {
		Enabled bool `mapstructure:"enabled"` // 是否启用压缩
		Level   int  `mapstructure:"level"`   // zstd 压缩级别
	} `mapstructure:"compression"`
}
