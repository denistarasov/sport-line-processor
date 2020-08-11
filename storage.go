package main

import "sync"

type storage struct {
	m sync.RWMutex
	s map[string]float64
}

func newStorage() *storage {
	return &storage{
		m: sync.RWMutex{},
		s: make(map[string]float64),
	}
}

func (s *storage) Upload(key string, value float64) {
	s.m.Lock()
	defer s.m.Unlock()
	s.s[key] = value
}

func (s *storage) Get(key string) (float64, bool) {
	s.m.RLock()
	defer s.m.RUnlock()
	value, exists := s.s[key]
	return value, exists
}

func (s *storage) GetKeys() map[string]struct{} {
	s.m.RLock()
	defer s.m.RUnlock()
	keys := make(map[string]struct{}, len(s.s))
	for key := range s.s {
		keys[key] = struct{}{}
	}
	return keys
}

func (s *storage) Count() int {
	s.m.RLock()
	defer s.m.RUnlock()
	return len(s.s)
}
