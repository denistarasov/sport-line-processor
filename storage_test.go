package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStorage_Simple(t *testing.T) {
	s := NewStorage()
	require.Equal(t, 0, s.Count())

	_, exists := s.Get("football")
	require.False(t, exists)

	s.Upload("football", 0.1)
	line, exists := s.Get("football")
	require.True(t, exists)
	require.Equal(t, 0.1, line)
	require.Equal(t, 1, s.Count())
}

func TestStorage_Update(t *testing.T) {
	s := NewStorage()
	s.Upload("football", 0.1)
	require.Equal(t, 1, s.Count())

	s.Upload("football", 0.2)
	require.Equal(t, 1, s.Count())
	line, _ := s.Get("football")
	require.Equal(t, 0.2, line)
}

func TestStorage_Count(t *testing.T) {
	s := NewStorage()

	s.Upload("football", 0.1)
	require.Equal(t, 1, s.Count())

	s.Upload("baseball", 0.1)
	require.Equal(t, 2, s.Count())

	s.Upload("soccer", 0.1)
	require.Equal(t, 3, s.Count())
}
