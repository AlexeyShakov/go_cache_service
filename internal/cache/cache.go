package cache

import (
	"sync"
)

type InMemoryCache struct {
	mu    sync.RWMutex
	cache map[string]string
}

func (r *InMemoryCache) GetByKey(key string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.cache[key]
	return val, ok
}

func (r *InMemoryCache) ReplaceAll(snapshot map[string]string) {
	// TODO нужно доработать логику замены, надо делать батчами
	r.mu.Lock()
	defer r.mu.Unlock()
	copyMap := make(map[string]string, len(snapshot))
	for k, v := range snapshot {
		copyMap[k] = v
	}
	r.cache = copyMap
}

func (r *InMemoryCache) AddKey(key string, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[key] = value
}

func (r *InMemoryCache) GetAllKeys() []string {
	// Should I return pointer? Probably no, because keys might change at runtime. Though a pointer is better.
	// it's okay to lose some keys
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, len(r.cache))
	i := 0
	for k := range r.cache {
		keys[i] = k
		i++
	}
	return keys
}

func (r *InMemoryCache) ReplaceKeys(keys []string, vals []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, k := range keys {
		if vals[i] == "" {
			delete(r.cache, k)
			continue
		}
		r.cache[k] = vals[i]
	}
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{cache: make(map[string]string)}
}
