package service

import (
	"context"
	"github.com/yourname/go_cache_service/internal/cache"
	"github.com/yourname/go_cache_service/internal/domain"
	"log/slog"
	"sync"
)

type repository interface {
	GetByKey(ctx context.Context, key domain.Key) (domain.Value, error)
	GetByKeys(ctx context.Context, keys []domain.Key) ([]string, error)
}

// CacheService реализует read-through кэш поверх in-memory слоя и репозитория.
// На промахе выполняет загрузку из репозитория и обновляет кэш.
// Поддерживает fail-fast дедупликацию параллельных запросов на один ключ.
// Безопасен для конкурентного использования.
type CacheService struct {
	repo           repository
	cache          *cache.InMemoryCache
	updateBatchLen int
	inFlight       map[domain.Key]struct{}
	mu             sync.Mutex
	logger         *slog.Logger
}

// GetByKey возвращает значение по ключу, используя in-memory кэш как быстрый путь.
// На промахе читает из репозитория и сохраняет результат в кэш.
// Если загрузка ключа из репозитория уже выполняется, возвращает domain.ErrRepeatedRequest.
// Безопасен для конкурентного доступа
func (r *CacheService) GetByKey(ctx context.Context, key domain.Key) (domain.Value, error) {
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

func (r *CacheService) getByKeyFromDB(ctx context.Context, key domain.Key) (domain.Value, error) {
	res, err := r.repo.GetByKey(ctx, key)
	if err != nil {
		return "", err
	}
	return res, nil
}

func (r *CacheService) getByKeyFromCache(key domain.Key) (domain.Value, bool) {
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
			return err
		}
		r.cache.ReplaceKeys(keysBatch, newVals)
	}
	r.logger.Info("Количество ключей обновлено", "value", len(keys))
	return nil
}

// NewCacheService создаёт CacheService.
// updateBatchLen задаёт размер батча при Refresh; если значение 0, используется дефолт.
func NewCacheService(repo repository, cache *cache.InMemoryCache, updateBatchLen int, logger *slog.Logger) *CacheService {
	if updateBatchLen == 0 {
		updateBatchLen = 100
	}
	inFlight := make(map[domain.Key]struct{})
	return &CacheService{
		repo:           repo,
		cache:          cache,
		updateBatchLen: updateBatchLen,
		inFlight:       inFlight,
		mu:             sync.Mutex{},
		logger:         logger,
	}
}
