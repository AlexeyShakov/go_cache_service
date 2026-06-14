package main

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/yourname/go_cache_service/internal/domain/service"
)

type RefresherEnvConfig struct {
	Timeout      time.Duration `env:"REFRESH_TIMEOUT" env-default:"200ms"`
	Interval     time.Duration `env:"REFRESH_INTERVAL" env-default:"5s"`
	JitterMaxVal int           `env:"JITTER_MAX_VALUE" env-default:"10"`
}

// LoadRefresherConfig парсит энв для получения нужных переменных и создает доменный конфиг для рефрешера
func LoadRefresherConfig() (service.RefreshConfig, error) {
	var cfg RefresherEnvConfig
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return service.RefreshConfig{}, err
	}
	serviceCfg, err := service.NewRefreshConfig(cfg.Timeout, cfg.Interval, cfg.JitterMaxVal)
	if err != nil {
		return service.RefreshConfig{}, err
	}
	return serviceCfg, nil
}
