package wal

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Mort4lis/memdb/internal/db/config"
	"github.com/Mort4lis/memdb/internal/db/storage/wal/filesystem"
	"github.com/Mort4lis/memdb/internal/pkg/concurrency"
)

type WAL struct {
	conf    config.WAL
	lsn     atomic.Int64
	inCh    chan Record
	doneCh  chan struct{}
	segCtrl *filesystem.SegmentController
	cancel  func()
}

func NewWAL(conf config.WAL) (*WAL, error) {
	segCtrl, err := filesystem.NewSegmentController(conf.DataDir, conf.MaxSegmentSize)
	if err != nil {
		return nil, fmt.Errorf("create segment controller: %w", err)
	}

	wal := &WAL{
		conf:    conf,
		inCh:    make(chan Record),
		doneCh:  make(chan struct{}),
		segCtrl: segCtrl,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go wal.handleIncomingEntries(ctx)

	wal.cancel = cancel
	return wal, nil
}

func (wal *WAL) Walk(fn func(r Record) error) error {
	var buf bytes.Buffer
	err := wal.segCtrl.WalkLines(func(line []byte) error {
		var r Record
		buf.Write(line)
		if err := r.Decode(&buf); err != nil {
			return fmt.Errorf("decode record: %w", err)
		}

		buf.Reset()
		return fn(r)
	})
	if err != nil {
		return err //nolint:wrapcheck // ignore
	}
	return nil
}

func (wal *WAL) handleIncomingEntries(ctx context.Context) {
	ticker := time.NewTicker(wal.conf.FlushBatchInterval)
	defer ticker.Stop()

	batch := make([]Record, 0, wal.conf.FlushBatchSize)

	for {
		select {
		case <-ctx.Done():
			wal.flushBatch(&batch)
			close(wal.doneCh)
			return
		case record := <-wal.inCh:
			batch = append(batch, record)
			if len(batch) == wal.conf.FlushBatchSize {
				wal.flushBatch(&batch)
				ticker.Reset(wal.conf.FlushBatchInterval)
			}
		case <-ticker.C:
			wal.flushBatch(&batch)
		}
	}
}

func (wal *WAL) flushBatch(batchPtr *[]Record) {
	batch := *batchPtr
	if len(batch) == 0 {
		return
	}

	var buf bytes.Buffer
	var err error

	defer func() {
		for _, record := range batch {
			record.promise.Set(err)
		}
		*batchPtr = batch[:0]
	}()

	for _, record := range batch {
		if err = record.Encode(&buf); err != nil {
			return
		}
		if err = wal.segCtrl.Write(buf.Bytes()); err != nil {
			return
		}
	}

	err = wal.segCtrl.Flush()
}

func (wal *WAL) Append(r Record) concurrency.FutureError {
	r.LSN = wal.lsn.Add(1)
	r.promise = concurrency.NewPromise[error]()
	wal.inCh <- r
	return r.promise.Future()
}

func (wal *WAL) Shutdown(ctx context.Context) error {
	if wal.cancel == nil {
		return nil
	}

	wal.cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wal.doneCh:
		return nil
	}
}
