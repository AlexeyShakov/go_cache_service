package redis

import (
	"context"
	"errors"
	redislib "github.com/redis/go-redis/v9"
	"github.com/yourname/go_cache_service/internal/domain"
	"github.com/yourname/go_cache_service/internal/domain/service"
)

// RedisRepository реализует репозиторий поверх Redis Hash.
// Значения хранятся в одном хеше по ключу HashKey.
// Использует ctx для отмены и дедлайнов.
type RedisRepository struct {
	Client  *redislib.Client
	HashKey string
}

// GetByKey читает значение по ключу из Redis.
// Если ключ отсутствует, возвращает domain.ErrNotFound.
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

// GetByKeys читает значения по набору ключей из Redis.
// Возвращает срез значений той же длины, что и keys;
// для отсутствующих ключей возвращается пустая строка.
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

// NewRedisRepository создаёт репозиторий Redis.
func NewRedisRepository(client *redislib.Client, hashKey string) (*RedisRepository, error) {
	if hashKey == "" {
		return nil, errors.New("hashKey must be not empty")
	}
	return &RedisRepository{
		Client:  client,
		HashKey: hashKey,
	}, nil
}
