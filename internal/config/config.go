package config

import (
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
)

const (
	EnvLocal       = "local"
	EnvDevelopment = "dev"
	EnvProduction  = "prod"
)

type Config struct {
	Env      string   `yaml:"env" env-required:"true"`
	Server   Server   `yaml:"http" env-required:"true"`
	TonProof TonProof `yaml:"ton_proof" env-required:"true"`
	Postgres Postgres `yaml:"postgres" env-required:"true"`
}

type Server struct {
	Address     string        `yaml:"address" env-required:"true"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type TonProof struct {
	PayloadSignatureKey    string        `yaml:"payload_signature_key" env-required:"true"`
	PayloadLifetimeSeconds time.Duration `yaml:"payload_lifetime_seconds" env-default:"300s"`
	ProofLifetimeSeconds   time.Duration `yaml:"proof_lifetime_seconds" env-default:"300s"`
}

type Postgres struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Name     string `yaml:"name"`
	Password string `yaml:"password"`
	ModeSSL  string `yaml:"sslmode"`
}

// MustLoad loads config to a new Config instance and return it
func MustLoad() *Config {
	_ = godotenv.Load()

	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		slog.Error("missed CONFIG_PATH parameter")
		os.Exit(1)
	}

	var err error
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		slog.Error("config file does not exist", slog.String("path", configPath))
		os.Exit(1)
	}

	var config Config

	if err = cleanenv.ReadConfig(configPath, &config); err != nil {
		slog.Error("cannot read config", sl.Err(err))
		os.Exit(1)
	}

	return &config
}

func Empty() *Config {
	return &Config{}
}
