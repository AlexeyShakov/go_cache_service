// Package cache содержит реализацию in-memory кэша,
// используемого как быстрый слой поверх внешнего хранилища.
package cache

import (
	"sync"
)

// InMemoryCache реализует потокобезопасный кэш на основе map и RWMutex.
// Предназначен для быстрого чтения и записи данных в памяти.
type InMemoryCache struct {
	mu    sync.RWMutex
	cache map[string]string
}

// GetByKey возвращает значение по ключу и признак его наличия.
// Использует read-lock(несколько горутин могу одновременно читать map) и безопасен для конкурентного доступа.
func (r *InMemoryCache) GetByKey(key string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.cache[key]
	return val, ok
}

// AddKey добавляет или обновляет значение по ключу.
// Безопасен для конкурентного использования.
func (r *InMemoryCache) AddKey(key string, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[key] = value
}

// GetAllKeys возвращает срез всех ключей кэша
// Возвращаемый срез является копией и может безопасно изменяться вызывающим кодом.
// Безопасен для конкурентного использования
func (r *InMemoryCache) GetAllKeys() []string {
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

// ReplaceKeys обновляет или удаляет значения для переданных ключей.
// На вход принмается массив ключей с массивом значений для замены существующих значений соответствующих ключей
// Если для ключа новое значение пустое, ключ удаляется.
// Операция выполняется под write-lock, следовательно потокобезопасна
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

// NewInMemoryCache создаёт и инициализирует пустой in-memory кэш.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{mu: sync.RWMutex{}, cache: make(map[string]string)}
}
