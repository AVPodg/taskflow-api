package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	HTTP     HTTPConfig
	Database DatabaseConfig
	Log      LogConfig
	JWT      JWTConfig
}

type HTTPConfig struct {
	Port    string        `env:"HTTP_PORT" envDefault:"8080"`
	Timeout time.Duration `env:"HTTP_TIMEOUT" envDefault:"2s"`
}

type DatabaseConfig struct {
	URL      string `env:"DB_URL,required"`
	MaxConns int32  `env:"DB_MAX_CONNS" envDefault:"10"`
	MinConns int32  `env:"DB_MIN_CONNS" envDefault:"2"`
}

type LogConfig struct {
	Level string `env:"LOG_LEVEL" envDefault:"info"`
}

type JWTConfig struct {
	Secret string        `env:"JWT_SECRET,required"`
	TTL    time.Duration `env:"JWT_TTL" envDefault:"2h"`
}

func Load() (Config, error) {
	_ = godotenv.Load()

	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("error read config: %w", err)
	}

	if cfg.HTTP.Timeout <= 0 {
		return Config{}, fmt.Errorf("HTTP_TIMEOUT must be greated than zero")
	}

	if cfg.Database.MaxConns <= 0 {
		return Config{}, fmt.Errorf("DB_MAX_CONNS must be greater than zero")
	}

	if cfg.Database.MinConns < 0 {
		return Config{}, fmt.Errorf("DB_MIN_CONNS must be zero or greater")
	}

	if cfg.Database.MinConns > cfg.Database.MaxConns {
		return Config{}, fmt.Errorf("DB_MIN_CONNS must not exceed DB_MAX_CONNS")
	}

	switch cfg.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("invalid LOG_LEVEL: %s", cfg.Log.Level)
	}

	if len(cfg.JWT.Secret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	if cfg.JWT.TTL <= 0 {
		return Config{}, fmt.Errorf("JWT_TTL must be greater than zero")
	}

	return cfg, nil
}
