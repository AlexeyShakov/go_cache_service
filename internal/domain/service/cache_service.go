package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/yourname/go_cache_service/internal/cache"
	"github.com/yourname/go_cache_service/internal/domain"
	"github.com/yourname/go_cache_service/internal/infrastructure/redis"
	"io"
	"net"
)

type repository interface {
	GetAll(ctx context.Context) (map[Key]Value, error)
	GetByKey(ctx context.Context, key Key) (Value, error)
}

type CacheService struct {
	//TODO нужно добавить логику, что если в кэше нет значения, то мы идем в БД, а потом докладываем в кэш
	repo  repository
	cache *cache.InMemoryCache
}

func (r *CacheService) GetByKey(ctx context.Context, key Key) (Value, error) {
	res, ok := r.getByKeyFromCache(key)
	if !ok {
		res, err := r.getByKeyFromDB(ctx, key)
		if err != nil {
			return nil, err
		}
		return res, nil
	}
	return res, nil
}

func (r *CacheService) getByKeyFromDB(ctx context.Context, key Key) (Value, error) {
	res, err := r.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r *CacheService) getByKeyFromCache(key Key) (Value, bool) {
	return r.cache.GetByKey(key)
}

func (r *CacheService) Refresh(ctx context.Context) error {
	res, err := r.repo.GetAll(ctx)
	if err != nil {
		return categorizeRefreshError(ctx, err)
	}
	r.cache.ReplaceAll(res)
	return nil
}

func categorizeRefreshError(ctx context.Context, err error) error {
	// Контекст (таймаут/отмена) — это управление жизненным циклом, а не "ошибка Redis".
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", domain.ErrTimeout, err)
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("%w: %v", domain.ErrCancelled, err)
	}

	// Транспортные ошибки (соединение/чтение) — обычно временные, значит Unavailable.
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}
	rErr := redis.IsRedisSideError(err) // todo можно ли использовать в сервисе логику из инфры?
	if rErr == nil {
		return fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}
	return rErr
}

func NewCacheService(repo repository, cache *cache.InMemoryCache) *CacheService {
	return &CacheService{repo: repo, cache: cache}
}
