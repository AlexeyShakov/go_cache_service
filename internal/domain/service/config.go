package service

import (
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

type Config struct {
	Timeout      time.Duration `env:"REFRESH_TIMEOUT" env-default:"200ms"`
	Interval     time.Duration `env:"REFRESH_INTERVAL" env-default:"5s"`
	JitterMaxVal int           `env:"JITTER_MAX_VALUE" env-default:"10"`
}

func LoadRefresherConfig() (Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
