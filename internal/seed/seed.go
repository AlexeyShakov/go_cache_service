package seed

import (
	"context"
	"errors"
	"fmt"
	redislib "github.com/redis/go-redis/v9"
	"time"
)

// Run записывает подготовленный dataset в Redis Hash.
// data — набор пар key/value, который будет сохранён в хеше hashKey.
func Run(data map[string]string, client *redislib.Client, ctx context.Context, hashKey string) error {
	if client == nil {
		return errors.New("redis client is nil")
	}
	if hashKey == "" {
		return errors.New("hashKey must be not empty")
	}
	if len(data) == 0 {
		return errors.New("dataset is empty")
	}
	// Ограничиваем время выполнения сидирования.
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return write(timeoutCtx, hashKey, data, client)
}

// write вспомогательная функция для записи данных в Redis
func write(ctx context.Context, hashKey string, data map[string]string, client *redislib.Client) error {
	// Пишем данные батчем через pipeline.
	_, err := client.Pipelined(ctx, func(p redislib.Pipeliner) error {
		for k, v := range data {
			p.HSet(ctx, hashKey, k, v)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("seed redis hash %q: %w", hashKey, err)
	}
	return nil
}
