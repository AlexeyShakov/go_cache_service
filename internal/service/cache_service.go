package service

import (
	"context"
	"github.com/yourname/go_cache_service/internal/cache"
	"github.com/yourname/go_cache_service/internal/domain"
)

type CacheService struct {
	repo  domain.Repository
	cache *cache.InMemoryCache
}

func (r *CacheService) GetByKey(ctx context.Context, key domain.Key) (domain.Value, error) {
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

func (r *CacheService) getByKeyFromDB(ctx context.Context, key domain.Key) (domain.Value, error) {
	res, err := r.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r *CacheService) getByKeyFromCache(key domain.Key) (domain.Value, bool) {
	return r.cache.GetByKey(key)
}

func (r *CacheService) Refresh(ctx context.Context) error {
	res, err := r.repo.GetAll(ctx)
	if err != nil {
		return err
	}
	r.cache.ReplaceAll(res)
	return nil
}

func NewCacheService(repo domain.Repository, cache *cache.InMemoryCache) *CacheService {
	return &CacheService{repo: repo, cache: cache}
}
