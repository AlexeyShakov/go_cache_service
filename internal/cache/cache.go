package cache

import (
	"github.com/yourname/go_cache_service/internal/domain"
	"sync"
)

type InMemoryCache struct {
	mu    sync.RWMutex
	cache map[domain.Key]domain.Value
}

func (r *InMemoryCache) GetByKey(key domain.Key) (domain.Value, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.cache[key]
	return val, ok
}

func (r *InMemoryCache) ReplaceAll(snapshot map[domain.Key]domain.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyMap := make(map[domain.Key]domain.Value, len(snapshot))
	for k, v := range snapshot {
		copyMap[k] = v
	}
	r.cache = copyMap
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{cache: make(map[domain.Key]domain.Value)}
}
