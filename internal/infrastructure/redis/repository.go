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

func (r *RedisRepository) GetAll(ctx context.Context) (map[service.Key]service.Value, error) {
	// TODO а что если в БД слишком много данных?
	res, err := r.Client.HGetAll(ctx, r.HashKey).Result()
	if err != nil {
		return nil, err
	}
	out := make(map[service.Key]service.Value, len(res))
	for k, v := range res {
		out[service.Key(k)] = []byte(v)
	}
	return out, nil
}

func (r *RedisRepository) GetByKey(ctx context.Context, key service.Key) (service.Value, error) {
	res, err := r.Client.HGet(ctx, r.HashKey, string(key)).Result()
	if err != nil {
		if errors.Is(err, redislib.Nil) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return []byte(res), nil
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
