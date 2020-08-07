package main

import "sync"

type Storage struct {
	m sync.RWMutex
	s map[string]float64
}

func NewStorage() *Storage {
	return &Storage{
		m: sync.RWMutex{},
		s: make(map[string]float64),
	}
}

func (s *Storage) Upload(key string, value float64) {
	s.m.Lock()
	defer s.m.Unlock()
	s.s[key] = value
}

func (s *Storage) Get(key string) (float64, bool) {
	s.m.RLock()
	defer s.m.RUnlock()
	value, exists := s.s[key]
	return value, exists
}

func (s *Storage) GetKeys() map[string]struct{} {
	s.m.RLock()
	defer s.m.RUnlock()
	keys := make(map[string]struct{}, len(s.s))
	for key := range s.s {
		keys[key] = struct{}{}
	}
	return keys
}

func (s *Storage) Count() int {
	s.m.RLock()
	defer s.m.RUnlock()
	return len(s.s)
}
