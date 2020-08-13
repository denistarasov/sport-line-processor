package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMapStorage_Simple(t *testing.T) {
	s := newMapStorage()
	require.Equal(t, 0, s.Count())

	_, exists := s.Get("football")
	require.False(t, exists)

	s.Upload("football", 0.1)
	line, exists := s.Get("football")
	require.True(t, exists)
	require.Equal(t, 0.1, line)
	require.Equal(t, 1, s.Count())
}

func TestMapStorage_Update(t *testing.T) {
	s := newMapStorage()
	s.Upload("football", 0.1)
	require.Equal(t, 1, s.Count())

	s.Upload("football", 0.2)
	require.Equal(t, 1, s.Count())
	line, _ := s.Get("football")
	require.Equal(t, 0.2, line)
}

func TestMapStorage_Count(t *testing.T) {
	s := newMapStorage()

	s.Upload("football", 0.1)
	require.Equal(t, 1, s.Count())

	s.Upload("baseball", 0.1)
	require.Equal(t, 2, s.Count())

	s.Upload("soccer", 0.1)
	require.Equal(t, 3, s.Count())
}

func TestMapStorage_GetKeys(t *testing.T) {
	s := newMapStorage()

	expected := make(map[string]struct{})
	require.Equal(t, expected, s.GetKeys())

	key := "football"
	expected[key] = struct{}{}
	s.Upload(key, 0.1)
	require.Equal(t, expected, s.GetKeys())

	key = "baseball"
	expected[key] = struct{}{}
	s.Upload(key, 0.1)
	require.Equal(t, expected, s.GetKeys())

	key = "soccer"
	expected[key] = struct{}{}
	s.Upload(key, 0.1)
	require.Equal(t, expected, s.GetKeys())
}

func TestDBStorage_Simple(t *testing.T) {
	s := newDBStorage()
	require.Equal(t, 0, s.Count())

	_, exists := s.Get("football")
	require.False(t, exists)

	s.Upload("football", 0.1)
	line, exists := s.Get("football")
	require.True(t, exists)
	require.Equal(t, 0.1, line)
	require.Equal(t, 1, s.Count())
}

func TestDBStorage_Update(t *testing.T) {
	s := newDBStorage()
	s.Upload("football", 0.1)
	require.Equal(t, 1, s.Count())

	s.Upload("football", 0.2)
	require.Equal(t, 1, s.Count())
	line, _ := s.Get("football")
	require.Equal(t, 0.2, line)
}

func TestDBStorage_Count(t *testing.T) {
	s := newDBStorage()

	s.Upload("football", 0.1)
	require.Equal(t, 1, s.Count())

	s.Upload("baseball", 0.1)
	require.Equal(t, 2, s.Count())

	s.Upload("soccer", 0.1)
	require.Equal(t, 3, s.Count())
}

func TestDBStorage_GetKeys(t *testing.T) {
	s := newDBStorage()

	expected := make(map[string]struct{})
	require.Equal(t, expected, s.GetKeys())

	key := "football"
	expected[key] = struct{}{}
	s.Upload(key, 0.1)
	require.Equal(t, expected, s.GetKeys())

	key = "baseball"
	expected[key] = struct{}{}
	s.Upload(key, 0.1)
	require.Equal(t, expected, s.GetKeys())

	key = "soccer"
	expected[key] = struct{}{}
	s.Upload(key, 0.1)
	require.Equal(t, expected, s.GetKeys())
}
