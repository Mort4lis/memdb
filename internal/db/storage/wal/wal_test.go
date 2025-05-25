package wal

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"

	"github.com/Mort4lis/memdb/internal/db/compute"
	"github.com/Mort4lis/memdb/internal/db/storage/model"
	"github.com/Mort4lis/memdb/internal/pkg/concurrency"
)

func waitFuture(t *testing.T, fut concurrency.FutureError) error {
	t.Helper()

	ch := make(chan error, 1)
	go func() { ch <- fut.Get() }()

	select {
	case err := <-ch:
		return err
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for record WAL future")
		return nil
	}
}

func encodeRecords(t *testing.T, rs ...*model.Record) []byte {
	t.Helper()

	var buf bytes.Buffer
	for _, record := range rs {
		_, err := protodelim.MarshalTo(&buf, record)
		require.NoError(t, err)
	}
	return buf.Bytes()
}

func TestNewWAL_InitializesAndShutdownsCleanly(t *testing.T) {
	dir := NewMockSegmentDirectory(t)
	w := NewMockSegmentWriter(t)

	w.On("Close").Return(nil)

	wal, err := NewWAL(dir, w, 3, 50*time.Millisecond)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	assert.NoError(t, wal.Shutdown(ctx))
}

func TestWAL_Append_BatchFlush(t *testing.T) {
	dir := NewMockSegmentDirectory(t)
	w := NewMockSegmentWriter(t)

	wal, err := NewWAL(dir, w, 2, 3*time.Second)
	require.NoError(t, err)
	defer wal.Shutdown(context.Background()) //nolint:errcheck // ignore

	rs := []*model.Record{
		{Lsn: 1, CommandId: int64(compute.SetCommandID), Args: []string{"key", "val"}},
		{Lsn: 2, CommandId: int64(compute.DelCommandID), Args: []string{"key"}},
	}

	w.On("Write", encodeRecords(t, rs...)).Return(nil)
	w.On("Close").Return(nil)

	futures := make([]concurrency.FutureError, len(rs))
	for i, r := range rs {
		future := wal.Append(compute.CommandID(r.CommandId), r.Args)
		futures[i] = future
	}

	for _, future := range futures {
		assert.NoError(t, waitFuture(t, future))
	}
}

func TestWAL_Append_FlushesOnTimer(t *testing.T) {
	dir := NewMockSegmentDirectory(t)
	w := NewMockSegmentWriter(t)

	wal, err := NewWAL(dir, w, 1000, 50*time.Millisecond)
	require.NoError(t, err)
	defer wal.Shutdown(context.Background()) //nolint:errcheck // ignore

	r := &model.Record{Lsn: 1, CommandId: int64(compute.SetCommandID), Args: []string{"key", "val"}}

	w.On("Write", encodeRecords(t, r)).Return(nil)
	w.On("Close").Return(nil)

	future := wal.Append(compute.CommandID(r.CommandId), r.Args)
	assert.NoError(t, waitFuture(t, future))
}

func TestWAL_Append_PropagatesErrorToFutures(t *testing.T) {
	dir := NewMockSegmentDirectory(t)
	w := NewMockSegmentWriter(t)

	wal, err := NewWAL(dir, w, 1, 10*time.Millisecond)
	require.NoError(t, err)
	defer wal.Shutdown(context.Background()) //nolint:errcheck // ignore

	r := &model.Record{Lsn: 1, CommandId: int64(compute.SetCommandID), Args: []string{"key", "val"}}

	unexpectedErr := errors.New("unexpected error")
	w.On("Write", encodeRecords(t, r)).Return(unexpectedErr)
	w.On("Close").Return(nil)

	future := wal.Append(compute.CommandID(r.CommandId), r.Args)
	gotErr := waitFuture(t, future)
	assert.ErrorIs(t, gotErr, unexpectedErr)
}

func TestWAL_Append_ClosedError(t *testing.T) {
	dir := NewMockSegmentDirectory(t)
	w := NewMockSegmentWriter(t)

	wal, err := NewWAL(dir, w, 1, 10*time.Millisecond)
	require.NoError(t, err)
	w.On("Close").Return(nil)

	require.NoError(t, wal.Shutdown(context.Background()))

	future := wal.Append(compute.SetCommandID, []string{"key", "val"})
	gotErr := waitFuture(t, future)
	assert.ErrorIs(t, gotErr, ErrClosed)
}

func TestWAL_Restore_Success(t *testing.T) {
	dir := NewMockSegmentDirectory(t)
	w := NewMockSegmentWriter(t)

	rs := []*model.Record{
		{Lsn: 1, CommandId: int64(compute.SetCommandID), Args: []string{"key", "val"}},
		{Lsn: 2, CommandId: int64(compute.DelCommandID), Args: []string{"key"}},
		{Lsn: 3, CommandId: int64(compute.SetCommandID), Args: []string{"key2", "val2"}},
		{Lsn: 4, CommandId: int64(compute.SetCommandID), Args: []string{"key3", "val3"}},
	}

	wal, err := NewWAL(dir, w, 1, 10*time.Millisecond)
	require.NoError(t, err)
	defer wal.Shutdown(context.Background()) //nolint:errcheck // ignore

	listFunc := iter.Seq2[[]byte, error](func(yield func([]byte, error) bool) {
		yield(encodeRecords(t, rs...), nil)
	})

	dir.On("List").Return(listFunc)
	w.On("Close").Return(nil)

	var lsn int64
	gotRecords := make([]*model.Record, 0, len(rs))

	err = wal.Restore(func(cid compute.CommandID, args []string) error {
		lsn++
		gotRecords = append(gotRecords, &model.Record{
			Lsn:       lsn,
			CommandId: int64(cid),
			Args:      args,
		})
		return nil
	})
	require.NoError(t, err)
	for i, want := range rs {
		assert.Truef(t, proto.Equal(want, gotRecords[i]), "Record %d does not match", i)
	}
}

func TestWAL_Restore_ErrorOnList(t *testing.T) {
	dir := NewMockSegmentDirectory(t)
	w := NewMockSegmentWriter(t)

	wal, err := NewWAL(dir, w, 1, 10*time.Millisecond)
	require.NoError(t, err)
	defer wal.Shutdown(context.Background()) //nolint:errcheck // ignore

	unexpectedErr := errors.New("unexpected error")
	listFunc := iter.Seq2[[]byte, error](func(yield func([]byte, error) bool) {
		yield(nil, unexpectedErr)
	})

	dir.On("List").Return(listFunc)
	w.On("Close").Return(nil)

	err = wal.Restore(func(compute.CommandID, []string) error {
		return nil
	})
	assert.ErrorIs(t, err, unexpectedErr)
}
