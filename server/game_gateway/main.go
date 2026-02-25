package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/zrurf/quiver/server/game_gateway/internal"
	"github.com/zrurf/quiver/server/game_gateway/internal/dao"
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

	natsClient, err := dao.NewNATSClient(config.Mq.Addr)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to NATS")
	}
	defer natsClient.Close()

	// 初始化数据访问层
	roomDao := dao.NewRoomRepository(dbPool, imdb)
	sessionDao := dao.NewSessionRepository(imdb)

	go internal.StartKCPGateway(config, sessionDao, roomDao, natsClient)

	internal.StartHealthCheck(config.Server.Listen)

	select {}
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
	pflag.String("server.listen", ":80", "Server listen address (e.g., :80 or 127.0.0.1:8080)")
	pflag.Int("server.internal-listen", 8080, "Server internal listen port")
	pflag.Int("server.kcp-port", 8081, "Server KCP port")
	pflag.Duration("server.idle-room-timeout", 5*60, "Idle room timeout (seconds)")
	pflag.Int("server.rate-limit", 100, "Rate limit (requests per second)")

	// Logger
	pflag.String("logger.level", "info", "Log level (debug, info, warn, error, fatal)")

	// User Server
	pflag.String("user-server.url", "http://localhost:8082", "User server URL")

	// Game Servers
	pflag.StringSlice("game-servers", []string{}, "List of game server addresses (e.g., game_server_1:18650)")

	// Database
	pflag.String("database.host", "localhost", "Database host")
	pflag.String("database.port", "5432", "Database port")
	pflag.String("database.user", "postgres", "Database username")
	pflag.String("database.password", "", "Database password")
	pflag.String("database.name", "app", "Database name")

	// Imdb
	pflag.String("imdb.addr", "localhost:6379", "IMDB/Redis address")

	// Mq
	pflag.String("mq.addr", "nats://localhost:4222", "Message queue address")
	pflag.String("mq.subject", "default", "Message queue subject/topic")

	// Play
	pflag.Int("play.max-players-per-room", 50, "Maximum number of players per room")
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
		Addr: config.Imdb.Addr,
	})
	_, err := imdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to imdb")
	}
	return imdb, nil
}

func initDB(config *internal.Config) (*pgxpool.Pool, error) {
	dbPool, err := pgxpool.New(context.Background(), fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s",
		config.Database.Host, config.Database.Port, config.Database.User,
		config.Database.Password, config.Database.Name,
	))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to user db")
	}
	return dbPool, nil
}
