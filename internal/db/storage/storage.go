package storage

import (
	"context"
	"fmt"

	"github.com/Mort4lis/memdb/internal/db/compute"
	"github.com/Mort4lis/memdb/internal/db/config"
	"github.com/Mort4lis/memdb/internal/db/storage/engine"
	"github.com/Mort4lis/memdb/internal/pkg/concurrency"
)

const (
	InMemoryEngine = "in_memory"
)

type Engine interface {
	Set(ctx context.Context, key, value string) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
}

type WAL interface {
	Append(cid compute.CommandID, args []string) concurrency.FutureError
	Restore(fn func(cid compute.CommandID, args []string) error) error
	Shutdown(ctx context.Context) error
}

type Option func(s *Storage)

func WithWAL(w WAL) Option {
	return func(s *Storage) {
		s.wal = w
	}
}

type Storage struct {
	engine Engine
	wal    WAL
}

func NewStorage(engineConf config.Engine, opts ...Option) (*Storage, error) {
	store := &Storage{}
	switch engineConf.Type {
	case InMemoryEngine:
		store.engine = engine.NewEngine()
	default:
		return nil, fmt.Errorf("unsupported engine type: %s", engineConf.Type)
	}

	for _, opt := range opts {
		opt(store)
	}

	if store.wal != nil {
		if err := store.restoreFromWAL(); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *Storage) restoreFromWAL() error {
	ctx := context.Background()
	err := s.wal.Restore(func(cid compute.CommandID, args []string) error {
		switch cid {
		case compute.SetCommandID:
			if err := s.engine.Set(ctx, args[0], args[1]); err != nil {
				return fmt.Errorf("set value in engine: %w", err)
			}
		case compute.DelCommandID:
			if err := s.engine.Del(ctx, args[0]); err != nil {
				return fmt.Errorf("delete value from engine: %w", err)
			}
		default:
			return fmt.Errorf("unsupported command: %s", cid)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("restore WAL: %w", err)
	}
	return nil
}

func (s *Storage) Set(ctx context.Context, key, value string) error {
	if s.wal != nil {
		future := s.wal.Append(compute.SetCommandID, []string{key, value})
		if err := future.Get(); err != nil {
			return fmt.Errorf("add record to WAL: %w", err)
		}
	}

	if err := s.engine.Set(ctx, key, value); err != nil {
		return fmt.Errorf("set value in engine: %w", err)
	}
	return nil
}

func (s *Storage) Get(ctx context.Context, key string) (string, error) {
	value, err := s.engine.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("get value from engine: %w", err)
	}
	return value, nil
}

func (s *Storage) Del(ctx context.Context, key string) error {
	if s.wal != nil {
		future := s.wal.Append(compute.DelCommandID, []string{key})
		if err := future.Get(); err != nil {
			return fmt.Errorf("add record to WAL: %w", err)
		}
	}

	if err := s.engine.Del(ctx, key); err != nil {
		return fmt.Errorf("delete value from engine: %w", err)
	}
	return nil
}

func (s *Storage) Shutdown(ctx context.Context) error {
	if s.wal != nil {
		if err := s.wal.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown WAL: %w", err)
		}
	}
	return nil
}
