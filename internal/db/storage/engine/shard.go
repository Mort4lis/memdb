package engine

import (
	"sync"
)

type shard struct {
	mu   sync.RWMutex
	data map[string]string
}

func newShard() *shard {
	return &shard{data: make(map[string]string)}
}

func (s *shard) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *shard) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[key]
	return value, ok
}

func (s *shard) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}
