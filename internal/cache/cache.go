package cache

import (
	"github.com/yourname/go_cache_service/internal/domain/service"
	"sync"
)

type InMemoryCache struct {
	mu    sync.RWMutex
	cache map[service.Key]service.Value
}

func (r *InMemoryCache) GetByKey(key service.Key) (service.Value, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.cache[key]
	return val, ok
}

func (r *InMemoryCache) ReplaceAll(snapshot map[service.Key]service.Value) {
	// todo нужно доработать логику замены, надо делать батчами
	r.mu.Lock()
	defer r.mu.Unlock()
	copyMap := make(map[service.Key]service.Value, len(snapshot))
	for k, v := range snapshot {
		copyMap[k] = v
	}
	r.cache = copyMap
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{cache: make(map[service.Key]service.Value)}
}
