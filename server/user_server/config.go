package main

type Config struct {
	Server struct {
		Listen string `mapstructure:"listen"`
	} `mapstructure:"server"`
	Logger struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"logger"`
	Database struct {
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
	Opaque struct {
		OPRFSeedFile        string `mapstructure:"oprf-seed-file"`
		ServerPublicKeyFile string `mapstructure:"server-public-key-file"`
		ServerSecretKeyFile string `mapstructure:"server-secret-key-file"`
	} `mapstructure:"opaque"`
}
