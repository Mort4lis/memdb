package wal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const defaultBufferSize = 4096

type SegmentWriter interface {
	io.Closer
	Write([]byte) error
}

type RecordWriter struct {
	w   SegmentWriter
	buf *bytes.Buffer
}

func NewRecordWriter(w SegmentWriter) *RecordWriter {
	return &RecordWriter{
		w:   w,
		buf: bytes.NewBuffer(make([]byte, 0, defaultBufferSize)),
	}
}

func (rw *RecordWriter) Write(rs []Record) error {
	rw.buf.Reset()
	enc := json.NewEncoder(rw.buf)

	for i := range rs {
		if err := enc.Encode(rs[i]); err != nil {
			return fmt.Errorf("encode record[%d]: %w", i, err)
		}
	}

	if err := rw.w.Write(rw.buf.Bytes()); err != nil {
		return fmt.Errorf("write records to segment: %w", err)
	}
	return nil
}

func (rw *RecordWriter) Close() error {
	if err := rw.w.Close(); err != nil {
		return fmt.Errorf("close segment writer: %w", err)
	}
	return nil
}
