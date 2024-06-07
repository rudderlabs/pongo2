package pongo2

import (
	"sync"
)

// Generic map wrapper with locking support
type SyncMap[K comparable, V any] struct {
	m    map[K]V
	lock sync.RWMutex
}

func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		m:    make(map[K]V),
		lock: sync.RWMutex{},
	}
}

func (sm *SyncMap[K, V]) Get(key K) (V, bool) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	val, ok := sm.m[key]
	return val, ok
}

func (sm *SyncMap[K, V]) Set(key K, value V) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.m[key] = value
}

func (sm *SyncMap[K, V]) Entries() map[K]V {
	result := make(map[K]V)
	for k, v := range sm.m {
		result[k] = v
	}
	return result
}

func (sm *SyncMap[K, V]) Length() int {
	return len(sm.m)
}