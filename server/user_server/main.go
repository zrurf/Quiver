package main

import (
	"context"
	"crypto"
	"fmt"
	"os"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/bytemare/ksf"
	"github.com/bytemare/opaque"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/zrurf/quiver/server/user/internal"
	"github.com/zrurf/quiver/server/user/internal/dao"
	"github.com/zrurf/quiver/server/user/internal/services"
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

	// 初始化数据访问层
	userDao := dao.NewUserRepository(dbPool)
	sessionDao := dao.NewSessionRepository(imdb)

	// 读取OPAQUE密钥
	oprfSeed, err := os.ReadFile(config.Opaque.OPRFSeedFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Fatal to read opaque seed.")
		panic(2)
	}
	pubKey, err := os.ReadFile(config.Opaque.ServerPublicKeyFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Fatal to read opaque public key.")
		panic(2)
	}
	priKey, err := os.ReadFile(config.Opaque.ServerSecretKeyFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Fatal to read opaque secret key.")
		panic(2)
	}

	// 初始化微服务
	opaqueSvc, err := services.NewOpaqueService(&services.OpaqueConfig{
		Config: &opaque.Configuration{
			OPRF:    opaque.RistrettoSha512,
			KDF:     crypto.SHA512,
			MAC:     crypto.SHA512,
			Hash:    crypto.SHA512,
			KSF:     ksf.Argon2id,
			AKE:     opaque.RistrettoSha512,
			Context: nil,
		},
		OprfSeed:        oprfSeed,
		ServerPublicKey: pubKey,
		ServerSecretKey: priKey,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Fatal to init opaque service.")
		panic(2)
	}

	// 初始化fiber
	var app = fiber.New(fiber.Config{
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
	})

	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	app.Use(cors.New())

	internal.ConfigRoute(app, &internal.RouteDependencies{
		AuthSvc: services.NewAuthService(userDao, sessionDao, imdb, opaqueSvc),
	})

	app.Listen(config.Server.Listen)
}

func initConfig() (*Config, error) {
	var cfg Config
	initFlag()
	var v = viper.New()

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
	pflag.String("server.listen", ":80", "Server listen address (e.g., :80 or 127.0.0.1:8080)")

	// Logger
	pflag.String("logger.level", "info", "Log level (debug, info, warn, error, fatal)")

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

	// Opaque
	pflag.String("opaque.oprf-seed-file", "./oprf_seed.bin", "OPRF seed file path")
	pflag.String("opaque.server-public-key-file", "./server_public.key", "Server public key file path")
	pflag.String("opaque.server-secret-key-file", "./server_secret.key", "Server secret key file path")
	pflag.Parse()
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

func initDB(config *Config) (*pgxpool.Pool, error) {
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

func initIMDB(config *Config) (*redis.Client, error) {
	imdb := redis.NewClient(&redis.Options{
		Addr: config.Imdb.Addr,
	})
	_, err := imdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to imdb")
	}
	return imdb, nil
}
