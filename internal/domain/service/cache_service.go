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
	"sync"
)

type repository interface {
	GetByKey(ctx context.Context, key Key) (Value, error)
	GetByKeys(ctx context.Context, keys []Key) ([]string, error)
}

// CacheService реализует read-through кэш поверх in-memory слоя и репозитория.
// На промахе выполняет загрузку из репозитория и обновляет кэш.
// Поддерживает fail-fast дедупликацию параллельных запросов на один ключ.
// Безопасен для конкурентного использования.
type CacheService struct {
	repo           repository
	cache          *cache.InMemoryCache
	updateBatchLen int
	inFlight       map[Key]struct{}
	mu             sync.Mutex
}

// GetByKey возвращает значение по ключу, используя in-memory кэш как быстрый путь.
// На промахе читает из репозитория и сохраняет результат в кэш.
// Если загрузка ключа из репозитория уже выполняется, возвращает domain.ErrRepeatedRequest.
// Безопасен для конкурентного доступа
func (r *CacheService) GetByKey(ctx context.Context, key Key) (Value, error) {
	if res, ok := r.getByKeyFromCache(key); ok {
		return res, nil
	}

	r.mu.Lock()
	if _, exists := r.inFlight[key]; exists {
		r.mu.Unlock()
		return "", domain.ErrRepeatedRequest
	}
	r.inFlight[key] = struct{}{}
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.inFlight, key)
		r.mu.Unlock()
	}()

	res, err := r.getByKeyFromDB(ctx, key)
	if err != nil {
		return "", err
	}

	r.cache.AddKey(key, res)
	return res, nil
}

func (r *CacheService) getByKeyFromDB(ctx context.Context, key Key) (Value, error) {
	res, err := r.repo.GetByKey(ctx, key)
	if err != nil {
		return "", err
	}
	return res, nil
}

func (r *CacheService) getByKeyFromCache(key Key) (Value, bool) {
	return r.cache.GetByKey(key)
}

// Refresh обновляет значения для текущих ключей кэша, читая их из репозитория батчами.
// Использует ctx для отмены и дедлайнов.
// Ошибки нормализуются в доменные категории (timeout/cancelled/unavailable/internal).
func (r *CacheService) Refresh(ctx context.Context) error {
	keys := r.cache.GetAllKeys()
	for i := 0; i < len(keys); i += r.updateBatchLen {
		end := min(i+r.updateBatchLen, len(keys))
		keysBatch := keys[i:end]
		newVals, err := r.repo.GetByKeys(ctx, keysBatch)
		if err != nil {
			return categorizeRefreshError(ctx, err)
		}
		r.cache.ReplaceKeys(keysBatch, newVals)
	}
	return nil
}

// categorizeRefreshError преобразует ошибки обновления в доменные категории.
// Приоритет: ошибки контекста (timeout/cancelled) → транспортные ошибки → ошибки Redis → internal.
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

// NewCacheService создаёт CacheService.
// updateBatchLen задаёт размер батча при Refresh; если значение 0, используется дефолт.
func NewCacheService(repo repository, cache *cache.InMemoryCache, updateBatchLen int) *CacheService {
	if updateBatchLen == 0 {
		updateBatchLen = 100
	}
	inFlight := make(map[Key]struct{})
	return &CacheService{repo: repo, cache: cache, updateBatchLen: updateBatchLen, inFlight: inFlight, mu: sync.Mutex{}}
}
