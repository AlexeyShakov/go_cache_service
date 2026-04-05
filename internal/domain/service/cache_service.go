package service

import (
	"context"
	"log/slog"
	"sync"

	"github.com/yourname/go_cache_service/internal/cache"
	"github.com/yourname/go_cache_service/internal/domain"
	"github.com/yourname/go_cache_service/internal/infrastructure/logx"
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
func (c *CacheService) GetByKey(ctx context.Context, key domain.Key) (domain.Value, error) {
	// Сначала смотрим в кэше
	if res, ok := c.getByKeyFromCache(key); ok {
		logger := logx.WithContext(ctx, c.logger)
		logger.Debug(
			"Получили значение через кэш",
			"key", key,
			"value", res,
		)
		return res, nil
	}
	// Если в кэше нет, то проверим, вдруг по этому ключу мы уже идем в БД
	c.mu.Lock()
	if _, exists := c.inFlight[key]; exists {
		c.mu.Unlock()
		return "", domain.ErrRepeatedRequest
	}
	c.inFlight[key] = struct{}{}
	c.mu.Unlock()

	// В самом конце удалим ключ из inFlight, т.к. ключ уже должен быть в кэше
	// Если ключа в кэше не будет, то мы должны следующей горутине дать шанс на получение значения,
	// вдруг его уже добавили м БД
	defer func() {
		c.mu.Lock()
		delete(c.inFlight, key)
		c.mu.Unlock()
	}()

	// Получаем значение по ключу из БД и кладем в кэш
	res, err := c.getByKeyFromDB(ctx, key)
	if err != nil {
		return "", err
	}

	c.cache.AddKey(key, res)
	return res, nil
}

func (c *CacheService) getByKeyFromDB(ctx context.Context, key domain.Key) (domain.Value, error) {
	res, err := c.repo.GetByKey(ctx, key)
	logger := logx.WithContext(ctx, c.logger)
	if err != nil {
		logger.Debug("Ошибка при получении ключа", "val", key)
		return "", err
	}
	logger.Debug(
		"Получили значение через БД",
		"key", key,
		"value", res,
	)
	return res, nil
}

func (c *CacheService) getByKeyFromCache(key domain.Key) (domain.Value, bool) {

	return c.cache.GetByKey(key)
}

// Refresh обновляет значения для текущих ключей кэша, читая их из репозитория батчами.
// Использует ctx для отмены и дедлайнов.
// Ошибки нормализуются в доменные категории (timeout/cancelled/unavailable/internal).
func (c *CacheService) Refresh(ctx context.Context) error {
	keys := c.cache.GetAllKeys()
	for i := 0; i < len(keys); i += c.updateBatchLen {
		end := min(i+c.updateBatchLen, len(keys))
		keysBatch := keys[i:end]

		newVals, err := c.repo.GetByKeys(ctx, keysBatch)
		if err != nil {
			return err
		}
		c.cache.ReplaceKeys(keysBatch, newVals)
	}
	logger := logx.WithContext(ctx, c.logger)
	logger.Info("Количество ключей обновлено", "value", len(keys))
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
