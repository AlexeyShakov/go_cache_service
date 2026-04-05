package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	cache2 "github.com/yourname/go_cache_service/internal/cache"
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

func newRepo() *InMemoryRepo {
	storage := map[string]string{}
	return &InMemoryRepo{storage}
}

// TestGetByKeyInCache тестируем получение ключа, который лежит в кэше
func TestGetByKeyInCache(t *testing.T) {
	repo := newRepo()
	cache := cache2.NewInMemoryCache()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := NewCacheService(repo, cache, 0, logger)

	// Подготавливаем данные
	key, cacheVal := "key", "cacheVal"
	cache.AddKey(key, cacheVal)

	dbVal := "dbVal"
	repo.AddData(map[string]string{key: dbVal})

	ctx := context.Background()
	val, _ := srv.GetByKey(ctx, key)
	if val != cacheVal {
		t.Fatalf("wrong value=%s for key=%s. Expected value=%s", val, key, cacheVal)
	}
}

// TestGetByKeyInCache тестируем получение ключа, который лежит в БД
func TestGetByKeyInDB(t *testing.T) {
	repo := newRepo()
	cache := cache2.NewInMemoryCache()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := NewCacheService(repo, cache, 0, logger)

	// Подготавливаем данные
	key, dbVal := "key", "cacheVal"
	repo.AddData(map[string]string{key: dbVal})

	ctx := context.Background()
	val, _ := srv.GetByKey(ctx, key)
	if val != dbVal {
		t.Fatalf("wrong value=%s for key=%s. Expected value=%s", val, key, dbVal)
	}
}

// TestGetByKeyAbsent тестируем получение ключа, которого нет ни в кэше, ни в БД
func TestGetByKeyAbsent(t *testing.T) {
	repo := newRepo()
	cache := cache2.NewInMemoryCache()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := NewCacheService(repo, cache, 0, logger)

	// Подготавливаем данные
	key, cacheVal := "key", "cacheVal"
	cache.AddKey(key, cacheVal)

	dbVal := "dbVal"
	repo.AddData(map[string]string{key: dbVal})

	ctx := context.Background()
	anotherKey := "someKey"
	_, err := srv.GetByKey(ctx, anotherKey)
	if err == nil || !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("There must be no value for the key=%s", anotherKey)
	}
}

// TestRefresh тестируем мы благополучно обновили ключи в кэше из БД
func TestRefresh(t *testing.T) {
	repo := newRepo()
	cache := cache2.NewInMemoryCache()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := NewCacheService(repo, cache, 0, logger)

	// Подготавливаем данные
	key, cacheVal := "key", "cacheVal"
	cache.AddKey(key, cacheVal)

	dbVal := "dbVal"
	repo.AddData(map[string]string{key: dbVal})

	ctx := context.Background()
	// Так как мы используем fake репозиторий, то не ожидаем ожидаем ошибок
	srv.Refresh(ctx)
	val, err := srv.GetByKey(ctx, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != dbVal {
		t.Fatalf("got unexpected value=%s for key=%s. expected value=%s", val, key, dbVal)
	}
}
