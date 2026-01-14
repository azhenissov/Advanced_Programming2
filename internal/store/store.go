package store

import "sync"

type Store[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func New[K comparable, V any]() *Store[K, V] {
	return &Store[K, V]{
		data: make(map[K]V),
	}
}

func (s *Store[K, V]) Set(k K, v V) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = v
}

func (s *Store[K, V]) Get(k K) (V, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[k]
	return v, ok
}

func (s *Store[K, V]) Delete(k K) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[k]; !ok {
		return false
	}
	delete(s.data, k)
	return true
}

func (s *Store[K, V]) Snapshot() map[K]V {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[K]V, len(s.data))
	for k, v := range s.data {
		snapshot[k] = v
	}
	return snapshot
}

func (s *Store[K, V]) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

func (s *Store[K, V]) Keys() []K {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]K, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}
