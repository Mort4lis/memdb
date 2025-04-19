package wal

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const defaultBufferSize = 4096

type segmentWriter interface {
	Write([]byte) error
}

type RecordWriter struct {
	w   segmentWriter
	buf *bytes.Buffer
}

func NewRecordWriter(w segmentWriter) *RecordWriter {
	buf := bytes.Buffer{}
	buf.Grow(defaultBufferSize)
	return &RecordWriter{
		w:   w,
		buf: &buf,
	}
}

func (rw *RecordWriter) Write(rs []Record) error {
	rw.buf.Reset()
	enc := json.NewEncoder(rw.buf)

	for i, r := range rs {
		if err := enc.Encode(r); err != nil {
			return fmt.Errorf("encode record[%d]: %w", i, err)
		}
	}

	if err := rw.w.Write(rw.buf.Bytes()); err != nil {
		return fmt.Errorf("write records to segment: %w", err)
	}
	return nil
}
