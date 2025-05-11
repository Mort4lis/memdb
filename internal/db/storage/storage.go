package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/Mort4lis/memdb/internal/db/compute"
	"github.com/Mort4lis/memdb/internal/pkg/concurrency"
)

var (
	ErrSlaveMutable = errors.New("mutable transaction on slave")
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

type Replica interface {
	IsSlave() bool
	Shutdown(ctx context.Context) error
}

type SlaveReplica interface {
	StartHandle(fn func(cid compute.CommandID, args []string) error)
}

type Option func(s *Storage)

func WithEngine(e Engine) Option {
	return func(s *Storage) {
		s.engine = e
	}
}

func WithWAL(w WAL) Option {
	return func(s *Storage) {
		s.wal = w
	}
}

func WithReplica(r Replica) Option {
	return func(s *Storage) {
		s.replica = r
	}
}

type Storage struct {
	engine  Engine
	wal     WAL
	replica Replica
}

func NewStorage(opts ...Option) (*Storage, error) {
	store := &Storage{}
	for _, opt := range opts {
		opt(store)
	}

	if store.wal != nil {
		if err := store.wal.Restore(store.restoreCallback); err != nil {
			return nil, fmt.Errorf("restore WAL: %w", err)
		}
	}
	if store.replica != nil {
		if slave, ok := store.replica.(SlaveReplica); ok {
			slave.StartHandle(store.restoreCallback)
		}
	}
	return store, nil
}

func (s *Storage) restoreCallback(cid compute.CommandID, args []string) error {
	ctx := context.Background()
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
}

func (s *Storage) Set(ctx context.Context, key, value string) error {
	if s.replica != nil && s.replica.IsSlave() {
		return ErrSlaveMutable
	}
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
	if s.replica != nil && s.replica.IsSlave() {
		return ErrSlaveMutable
	}
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
	var errs []error
	if s.replica != nil {
		if err := s.replica.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown replica: %w", err))
		}
	}
	if s.wal != nil {
		if err := s.wal.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown WAL: %w", err))
		}
	}
	return errors.Join(errs...)
}
