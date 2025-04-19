package storage

import (
	"context"
	"fmt"

	"github.com/Mort4lis/memdb/internal/db/compute"
	"github.com/Mort4lis/memdb/internal/db/config"
	"github.com/Mort4lis/memdb/internal/db/storage/engine"
	"github.com/Mort4lis/memdb/internal/db/storage/wal"
	"github.com/Mort4lis/memdb/internal/db/storage/wal/filesystem"
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
	Append(r wal.Record) concurrency.FutureError
	Restore(fn func(r wal.Record) error) error
	Shutdown(ctx context.Context) error
}

type Storage struct {
	engine Engine
	wal    WAL
}

func NewStorage(engineConf config.Engine, walConf *config.WAL) (*Storage, error) {
	store := &Storage{}

	switch engineConf.Type {
	case InMemoryEngine:
		store.engine = engine.NewEngine()
	default:
		return nil, fmt.Errorf("unsupported engine type: %s", engineConf.Type)
	}

	if walConf != nil {
		var err error
		store.wal, err = initWAL(walConf)
		if err != nil {
			return nil, fmt.Errorf("init WAL: %w", err)
		}

		err = store.restore()
		if err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *Storage) restore() error {
	ctx := context.Background()
	err := s.wal.Restore(func(r wal.Record) error {
		switch r.CommandID {
		case compute.SetCommandID:
			if err := s.engine.Set(ctx, r.Args[0], r.Args[1]); err != nil {
				return fmt.Errorf("set value in engine: %w", err)
			}
		case compute.DelCommandID:
			if err := s.engine.Del(ctx, r.Args[0]); err != nil {
				return fmt.Errorf("delete value from engine: %w", err)
			}
		default:
			return fmt.Errorf("unsupported command: %s", r.CommandID)
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
		future := s.wal.Append(wal.Record{
			CommandID: compute.SetCommandID,
			Args:      []string{key, value},
		})

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
		future := s.wal.Append(wal.Record{
			CommandID: compute.DelCommandID,
			Args:      []string{key},
		})
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

func initWAL(conf *config.WAL) (*wal.WAL, error) {
	segmentDir, err := filesystem.NewSegmentDirectory(conf.DataDir)
	if err != nil {
		return nil, fmt.Errorf("new segment directory: %w", err)
	}

	segment, err := filesystem.NewSegment(conf.DataDir, conf.MaxSegmentSize)
	if err != nil {
		return nil, fmt.Errorf("new segment: %w", err)
	}

	res, err := wal.NewWAL(segmentDir, segment, conf.FlushBatchSize, conf.FlushBatchInterval)
	if err != nil {
		return nil, fmt.Errorf("new WAL: %w", err)
	}
	return res, nil
}
