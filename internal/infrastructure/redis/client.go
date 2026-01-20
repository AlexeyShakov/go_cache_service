package redis

import (
	"context"
	redislib "github.com/go-redis/redis/v8"
)

func NewClient(ctx context.Context, cfg Config) (*redislib.Client, error) {
	db := redislib.NewClient(&redislib.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		Username: cfg.User,
		DB:       cfg.DB,
	})
	if err := db.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return db, nil
}
