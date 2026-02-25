package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/zrurf/quiver/server/game/internal"
	"github.com/zrurf/quiver/server/game/internal/server"
)

func main() {
	// 初始化配置
	var config, err = initConfig()

	if err != nil {
		log.Fatal().Err(err).Msg("Fatal to init config.")
		panic(1)
	}

	// 初始化logger
	initLogger(strings.ToLower(config.Logger.Level))

	log.Info().Any("config", config).Msg("Config body")

	// 初始化基础设施
	dbPool, err := initDB(config)
	if err != nil {
		panic(2)
	}
	defer dbPool.Close()

	imdb, err := initIMDB(config)
	if err != nil {
		panic(2)
	}
	defer imdb.Close()

	var enc *internal.Encryptor
	if config.Encryption.Enabled {
		enc, err = internal.NewEncryptor(config.Encryption.MasterKeyBase64)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to init encryptor")
		}
	}

	// 创建压缩器
	var comp *internal.Compressor
	if config.Compression.Enabled {
		comp = internal.NewCompressor(config.Compression.Level)
	}

	srv := server.NewServer(config, dbPool, imdb, enc, comp)
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	go srv.StartNATSListener()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down...")
	srv.Stop()
}

func initConfig() (*internal.Config, error) {
	var cfg internal.Config
	initFlag()
	var v = viper.New()

	pflag.Parse()

	configFile, _ := pflag.CommandLine.GetString("config")
	v.SetConfigFile(configFile)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Warn().Msg("Config file cannot be found. Using default values and environment variables instead.")
		} else {
			log.Error().Err(err).Msg("Fatal to read config file.")
		}
	}

	v.SetEnvPrefix("QUIVER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	pflag.CommandLine.VisitAll(func(f *pflag.Flag) {
		if !f.Changed && v.IsSet(f.Name) {
			// 用户没有通过命令行设置该参数，且配置文件中存在该值
			// 使用配置文件的值为pflag设置新的默认值
			pflag.Set(f.Name, v.GetString(f.Name))
		}
	})

	pflag.Parse()

	pflag.CommandLine.VisitAll(func(f *pflag.Flag) {
		if f.Name != "config" {
			_ = viper.BindPFlag(f.Name, f)
		}
	})

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func initFlag() {
	pflag.String("config", "./config.toml", "Config file path")

	// Server
	pflag.String("server.listen-addr", ":18650", "Server TCP listen address")
	pflag.Duration("server.idle-room-timeout", 5*60, "Idle room timeout (seconds)")
	pflag.Int("server.max-players-per-room", 50, "Maximum number of players per room")

	// Logger
	pflag.String("logger.level", "info", "Log level (debug, info, warn, error, fatal)")

	// Database
	pflag.String("database.host", "localhost", "Database host")
	pflag.Int("database.port", 5432, "Database port")
	pflag.String("database.user", "postgres", "Database username")
	pflag.String("database.password", "", "Database password")
	pflag.String("database.name", "game_db", "Database name")

	// IMDB
	pflag.String("imdb.addr", "localhost:6379", "IMDB/Redis address")

	// MQ
	pflag.String("mq.addr", "nats://localhost:4222", "Message queue address")
	pflag.String("mq.subject", "default", "Message queue subject/topic")

	// Encryption
	pflag.Bool("encryption.enabled", false, "Enable encryption")
	pflag.String("encryption.master-key-base64", "", "Base64 encoded 32-byte master key")

	// Compression
	pflag.Bool("compression.enabled", false, "Enable compression")
	pflag.Int("compression.level", 3, "Zstd compression level (1-19)")
}

func initLogger(level_str string) {
	var level = zerolog.InfoLevel
	switch level_str {
	case "panic":
		level = zerolog.PanicLevel
	case "trace":
		level = zerolog.TraceLevel
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	case "fatal":
		level = zerolog.FatalLevel
	}
	log.Info().Msg("Log level: " + level.String())
	zerolog.SetGlobalLevel(level)
}

func initIMDB(config *internal.Config) (*redis.Client, error) {
	imdb := redis.NewClient(&redis.Options{
		Addr: config.IMDB.Addr,
	})
	_, err := imdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to imdb")
	}
	return imdb, nil
}

func initDB(config *internal.Config) (*pgxpool.Pool, error) {
	dbPool, err := pgxpool.New(context.Background(), fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s",
		config.Database.Host, config.Database.Port, config.Database.User,
		config.Database.Password, config.Database.Name,
	))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to user db")
	}
	return dbPool, nil
}
