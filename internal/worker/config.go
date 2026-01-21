package worker

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Timeout  time.Duration
	Interval time.Duration
}

func LoadRefresherConfig() (Config, error) {
	timeout, err := time.ParseDuration(os.Getenv("REFRESH_TIMEOUT"))
	if err != nil {
		return Config{}, fmt.Errorf("REFRESH_TIMEOUT: %w", err)
	}
	interval, err := time.ParseDuration(os.Getenv("REFRESH_INTERVAL"))
	if err != nil {
		return Config{}, fmt.Errorf("REFRESH_INTERVAL: %w", err)
	}
	var cfg = Config{
		Timeout:  timeout,
		Interval: interval,
	}
	return cfg, nil
}
