package store

import "sync"

type Store[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func NewStore[K comparable, V any]() *Store[K, V] {
	return &Store[K, V]{data: make(map[K]V)}
}

func (s *Store[K, V]) Set(k K, v V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = v
}

func (s *Store[K, V]) Get(k K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[k]
	return val, ok
}

func (s *Store[K, V]) Delete(k K) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[k]; ok {
		delete(s.data, k)
		return true
	}
	return false
}

func (s *Store[K, V]) Snapshot() map[K]V {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copy := make(map[K]V)
	for k, v := range s.data {
		copy[k] = v
	}
	return copy
}

func (s *Store[K, V]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}
