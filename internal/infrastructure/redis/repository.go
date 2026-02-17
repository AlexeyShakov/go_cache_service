package redis

import (
	"context"
	"errors"
	redislib "github.com/redis/go-redis/v9"
	"github.com/yourname/go_cache_service/internal/domain"
	"github.com/yourname/go_cache_service/internal/domain/service"
)

type RedisRepository struct {
	Client  *redislib.Client
	HashKey string
}

func (r *RedisRepository) GetByKey(ctx context.Context, key service.Key) (service.Value, error) {
	res, err := r.Client.HGet(ctx, r.HashKey, key).Result()
	if err != nil {
		if errors.Is(err, redislib.Nil) {
			return "", domain.ErrNotFound
		}
		return "", err
	}
	return res, nil
}

func (r *RedisRepository) GetByKeys(ctx context.Context, keys []service.Key) ([]string, error) {
	res, err := r.Client.HMGet(ctx, r.HashKey, keys...).Result()
	if err != nil {
		return nil, err
	}
	out := make([]string, len(res))
	for i, v := range res {
		if v != nil {
			s, ok := v.(string)
			if ok {
				out[i] = s
			}
		}
	}
	return out, nil

}

func NewRedisRepository(client *redislib.Client, hashKey string) (*RedisRepository, error) {
	if hashKey == "" {
		return nil, errors.New("hashKey must be not empty")
	}
	return &RedisRepository{
		Client:  client,
		HashKey: hashKey,
	}, nil
}
