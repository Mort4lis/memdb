package wal

import (
	"bytes"
	"fmt"
	"io"

	"github.com/Mort4lis/memdb/internal/db/storage/model"
)

const defaultBufferSize = 4096

//go:generate mockery --inpackage --testonly --case underscore --name SegmentWriter
type SegmentWriter interface {
	io.Closer
	Write([]byte) error
}

type recordWriter struct {
	w   SegmentWriter
	buf *bytes.Buffer
}

func newRecordWriter(w SegmentWriter) *recordWriter {
	return &recordWriter{
		w:   w,
		buf: bytes.NewBuffer(make([]byte, 0, defaultBufferSize)),
	}
}

func (rw *recordWriter) Write(rs []*model.Record) error {
	rw.buf.Reset()
	if err := model.EncodeRecords(rs, rw.buf); err != nil {
		return fmt.Errorf("encode records: %w", err)
	}
	if err := rw.w.Write(rw.buf.Bytes()); err != nil {
		return fmt.Errorf("write records to segment: %w", err)
	}
	return nil
}

func (rw *recordWriter) Close() error {
	if err := rw.w.Close(); err != nil {
		return fmt.Errorf("close segment writer: %w", err)
	}
	return nil
}
