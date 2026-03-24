package redis

import (
	"context"
	redislib "github.com/redis/go-redis/v9"
)

// NewClient создаёт и инициализирует клиента Redis на основе конфигурации.
// Проверяет доступность сервера через Ping.
// Контекст управляет таймаутом подключения.
func NewClient(ctx context.Context, cfg Config) (*redislib.Client, error) {

	// Формируем опции клиента.
	// Username и Password добавляются только если заданы,
	// чтобы не отправлять лишний AUTH-запрос.
	opt := &redislib.Options{
		Addr: cfg.Address,
		DB:   cfg.DB,
	}

	if cfg.Password != "" {
		opt.Password = cfg.Password
	}
	if cfg.User != "" {
		opt.Username = cfg.User
	}

	db := redislib.NewClient(opt)

	if err := db.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return db, nil
}
