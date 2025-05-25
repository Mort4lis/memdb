package storage

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Mort4lis/memdb/internal/db/config"
	"github.com/Mort4lis/memdb/internal/db/storage/engine"
	"github.com/Mort4lis/memdb/internal/db/storage/replication"
	"github.com/Mort4lis/memdb/internal/db/storage/wal"
	"github.com/Mort4lis/memdb/internal/db/storage/wal/filesystem"
	"github.com/Mort4lis/memdb/internal/network"
)

const (
	defaultDialTimeout  = 10 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultReadTimeout  = 10 * time.Second
	defaultIdleTimeout  = 2 * time.Minute
)

type Builder struct {
	opts []Option
}

func (b *Builder) Build(logger *slog.Logger, conf *config.Config) (*Storage, error) {
	if err := b.buildEngine(conf.Engine); err != nil {
		return nil, fmt.Errorf("build engine: %w", err)
	}
	if err := b.buildWAL(conf.WAL); err != nil {
		return nil, fmt.Errorf("build WAL: %w", err)
	}
	if err := b.buildReplica(logger, conf.WAL, conf.Replication); err != nil {
		return nil, fmt.Errorf("build replica: %w", err)
	}

	store, err := NewStorage(b.opts...)
	if err != nil {
		return nil, fmt.Errorf("new storage: %w", err)
	}

	b.opts = nil
	return store, nil
}

func (b *Builder) buildEngine(conf config.Engine) error {
	var e Engine
	switch conf.Type {
	case engine.InMemoryType:
		e = engine.NewEngine(conf.PartitionsNumber)
	default:
		return fmt.Errorf("unsupported engine type: %s", conf.Type)
	}

	b.opts = append(b.opts, WithEngine(e))
	return nil
}

func (b *Builder) buildWAL(conf config.WAL) error {
	if !conf.Enabled {
		return nil
	}

	segmentDir, err := filesystem.NewSegmentDirectory(conf.DataDir)
	if err != nil {
		return fmt.Errorf("new segment directory: %w", err)
	}

	segment, err := filesystem.NewSegment(conf.DataDir, conf.MaxSegmentSize)
	if err != nil {
		return fmt.Errorf("new segment: %w", err)
	}

	w, err := wal.NewWAL(segmentDir, segment, conf.FlushBatchSize, conf.FlushBatchInterval)
	if err != nil {
		return fmt.Errorf("new WAL: %w", err)
	}

	b.opts = append(b.opts, WithWAL(w))
	return nil
}

func (b *Builder) buildReplica(logger *slog.Logger, walConf config.WAL, replicaConf config.Replication) error {
	if !replicaConf.Enabled {
		return nil
	}
	if !walConf.Enabled {
		return errors.New("replication is not supported without WAL")
	}

	sd, sdErr := filesystem.NewSegmentDirectory(walConf.DataDir)
	if sdErr != nil {
		return fmt.Errorf("new segment directory: %w", sdErr)
	}

	var (
		r   Replica
		err error
	)
	switch replicaConf.Type {
	case replication.MasterType:
		opts := []network.TCPServerOption{
			network.WithServerListen(replicaConf.MasterAddr),
			network.WithServerMaxMessageSize(walConf.MaxSegmentSize),
			network.WithServerWriteTimeout(defaultWriteTimeout),
			network.WithServerIdleTimeout(defaultIdleTimeout),
		}
		r, err = replication.NewMaster(logger, sd, opts...)
	case replication.SlaveType:
		opts := []network.TCPClientOption{
			network.WithClientDialTimeout(defaultDialTimeout),
			network.WithClientWriteTimeout(defaultWriteTimeout),
			network.WithClientReadTimeout(defaultReadTimeout),
			network.WithClientReadBufferSize(int(float64(walConf.MaxSegmentSize) * 2)), //nolint:mnd // ignore
		}
		r, err = replication.NewSlave(logger, sd, replicaConf.MasterAddr, replicaConf.SyncInterval, opts...)
	default:
		return fmt.Errorf("unsupported replica type: %s", replicaConf.Type)
	}
	if err != nil {
		return fmt.Errorf("new %s: %w", replicaConf.Type, err)
	}

	b.opts = append(b.opts, WithReplica(r))
	return nil
}
