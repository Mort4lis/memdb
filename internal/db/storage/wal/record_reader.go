package wal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
)

type SegmentDirectory interface {
	List() iter.Seq2[[]byte, error]
}

type RecordReader struct {
	sd SegmentDirectory
}

func NewRecordReader(sd SegmentDirectory) *RecordReader {
	return &RecordReader{sd: sd}
}

func (rr *RecordReader) All() iter.Seq2[Record, error] {
	return func(yield func(Record, error) bool) {
		for data, err := range rr.sd.List() {
			if err != nil {
				yield(Record{}, err)
				return
			}

			dec := json.NewDecoder(bytes.NewBuffer(data))
			for {
				var r Record
				decErr := dec.Decode(&r)
				if errors.Is(decErr, io.EOF) {
					break
				}
				if decErr != nil {
					decErr = fmt.Errorf("decode record: %w", decErr)
					yield(Record{}, decErr)
					return
				}

				if !yield(r, nil) {
					return
				}
			}
		}
	}
}
