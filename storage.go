package main

import "sync"

type storage interface {
	Upload(key string, value float64)
	Get(key string) (float64, bool)
	GetKeys() map[string]struct{}
	Count() int
}

type mapStorage struct {
	m sync.RWMutex
	s map[string]float64
}

func newMapStorage() *mapStorage {
	return &mapStorage{
		m: sync.RWMutex{},
		s: make(map[string]float64),
	}
}

func (s *mapStorage) Upload(key string, value float64) {
	s.m.Lock()
	defer s.m.Unlock()
	s.s[key] = value
}

func (s *mapStorage) Get(key string) (float64, bool) {
	s.m.RLock()
	defer s.m.RUnlock()
	value, exists := s.s[key]
	return value, exists
}

func (s *mapStorage) GetKeys() map[string]struct{} {
	s.m.RLock()
	defer s.m.RUnlock()
	keys := make(map[string]struct{}, len(s.s))
	for key := range s.s {
		keys[key] = struct{}{}
	}
	return keys
}

func (s *mapStorage) Count() int {
	s.m.RLock()
	defer s.m.RUnlock()
	return len(s.s)
}
