package testutils

import (
	"context"

	"github.com/yourname/go_cache_service/internal/domain"
)

// InMemoryRepo - fake репозиторий для тестов
type InMemoryRepo struct {
	storage map[string]string
}

func (i *InMemoryRepo) GetByKey(ctx context.Context, key domain.Key) (domain.Value, error) {
	if val, ok := i.storage[key]; ok {
		return val, nil
	}
	return "", domain.ErrNotFound
}

func (i *InMemoryRepo) GetByKeys(ctx context.Context, keys []domain.Key) ([]string, error) {
	out := make([]string, len(keys))
	for idx, key := range keys {
		val, ok := i.storage[key]
		if ok {
			out[idx] = val
			continue
		}
		out[idx] = ""
	}
	return out, nil
}

func (i *InMemoryRepo) AddData(data map[string]string) {
	for k, v := range data {
		i.storage[k] = v
	}
}

func NewRepo() *InMemoryRepo {
	storage := map[string]string{}
	return &InMemoryRepo{storage}
}
