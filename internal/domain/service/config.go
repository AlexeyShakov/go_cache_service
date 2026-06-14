package service

import (
	"errors"
	"time"
)

type RefreshConfig struct {
	Timeout      time.Duration
	Interval     time.Duration
	JitterMaxVal int
}

// NewRefreshConfig валидирует переданные параметры и инициализирует доменный конфиг для рефрешера
func NewRefreshConfig(timeout, interval time.Duration, jitterMaxVal int) (RefreshConfig, error) {
	if timeout <= 0 {
		return RefreshConfig{}, errors.New("timeout must be > 0")
	}
	if interval <= 0 {
		return RefreshConfig{}, errors.New("interval must be > 0")
	}
	if jitterMaxVal < 0 {
		return RefreshConfig{}, errors.New("jitterMaxVal must be >= 0")
	}
	return RefreshConfig{
		Timeout:      timeout,
		Interval:     interval,
		JitterMaxVal: jitterMaxVal,
	}, nil
}
