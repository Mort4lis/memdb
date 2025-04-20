package wal

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Mort4lis/memdb/internal/pkg/concurrency"
)

var ErrClosed = fmt.Errorf("wal closed")

type WAL struct {
	flushBatchSize     int
	flushBatchInterval time.Duration
	rr                 *RecordReader
	rw                 *RecordWriter

	lsn    atomic.Int64
	inCh   chan Record
	doneCh chan struct{}
	cancel func()
}

func NewWAL(dir SegmentDirectory, w SegmentWriter, flushBatchSize int, flushBatchInterval time.Duration) (*WAL, error) {
	wal := &WAL{
		flushBatchSize:     flushBatchSize,
		flushBatchInterval: flushBatchInterval,
		rr:                 NewRecordReader(dir),
		rw:                 NewRecordWriter(w),
		inCh:               make(chan Record),
		doneCh:             make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	go wal.handleIncomingEntries(ctx)

	wal.cancel = cancel
	return wal, nil
}

func (wal *WAL) Restore(fn func(r Record) error) error {
	var lsn int64
	for r, err := range wal.rr.All() {
		if err != nil {
			return fmt.Errorf("read record: %w", err)
		}
		if err = fn(r); err != nil {
			return fmt.Errorf("restore record: %w", err)
		}
		lsn = r.LSN
	}

	wal.lsn.Store(lsn)
	return nil
}

func (wal *WAL) handleIncomingEntries(ctx context.Context) {
	ticker := time.NewTicker(wal.flushBatchInterval)
	defer ticker.Stop()

	batch := make([]Record, 0, wal.flushBatchSize)

	for {
		select {
		case <-ctx.Done():
			wal.flushBatch(&batch)
			close(wal.doneCh)
			return
		case record := <-wal.inCh:
			batch = append(batch, record)
			if len(batch) == wal.flushBatchSize {
				wal.flushBatch(&batch)
				ticker.Reset(wal.flushBatchInterval)
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

	err := wal.rw.Write(batch)
	if err != nil {
		err = fmt.Errorf("write record batch: %w", err)
	}

	for i := range batch {
		batch[i].promise.Set(err)
	}
	*batchPtr = batch[:0]
}

func (wal *WAL) Append(r Record) concurrency.FutureError {
	r.LSN = wal.lsn.Add(1)
	r.promise = concurrency.NewPromise[error]()

	select {
	case wal.inCh <- r:
	case <-wal.doneCh:
		r.promise.Set(ErrClosed)
	}

	return r.promise.Future()
}

func (wal *WAL) Shutdown(ctx context.Context) (err error) {
	if wal.cancel == nil {
		return nil
	}
	defer func() {
		closeErr := wal.rw.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close record writer: %w", closeErr)
		}
	}()

	wal.cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wal.doneCh:
		return nil
	}
}
