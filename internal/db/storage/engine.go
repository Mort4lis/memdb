package storage

import (
	"context"
	"sync"

	dberrors "github.com/Mort4lis/memdb/internal/db/errors"
)

type Engine struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewEngine() *Engine {
	return &Engine{
		data: make(map[string]string),
	}
}

func (e *Engine) Set(_ context.Context, key, value string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.data[key] = value
	return nil
}

func (e *Engine) Get(_ context.Context, key string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	value, ok := e.data[key]
	if !ok {
		return "", dberrors.ErrNotFound
	}
	return value, nil
}

func (e *Engine) Del(_ context.Context, key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.data, key)
	return nil
}
