package main

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinePuller_IsReady(t *testing.T) {
	s := newMapStorage()
	lp := &linePuller{
		Mutex:              sync.Mutex{},
		linesProviderAddr:  "",
		sportNames:         []string{"soccer", "football"},
		storage:            s,
		isLineProviderDown: false,
		wg:                 nil,
	}

	require.Equal(t, notReady, lp.isReady())

	s.Upload("soccer", 0)
	require.Equal(t, notReady, lp.isReady())

	s.Upload("soccer", 0)
	require.Equal(t, notReady, lp.isReady())

	s.Upload("football", 0)
	require.Equal(t, ready, lp.isReady())
}
