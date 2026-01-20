package redis

import (
	"context"
	"errors"
	redislib "github.com/go-redis/redis/v8"
	"github.com/yourname/go_cache_service/internal/domain"
)

type RedisRepository struct {
	Client  *redislib.Client
	HashKey string
}

func (r *RedisRepository) GetAll(ctx context.Context) (map[domain.Key]domain.Value, error) {
	// TODO а что если в БД слишком много данных?
	res, err := r.Client.HGetAll(ctx, r.HashKey).Result()
	if err != nil {
		return nil, err
	}
	out := make(map[domain.Key]domain.Value, len(res))
	for k, v := range res {
		out[domain.Key(k)] = domain.Value([]byte(v))
	}
	return out, nil
}

func (r *RedisRepository) GetByKey(ctx context.Context, key domain.Key) (domain.Value, error) {
	res, err := r.Client.HGet(ctx, r.HashKey, string(key)).Result()
	if err != nil {
		if errors.Is(err, redislib.Nil) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return domain.Value([]byte(res)), nil
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
