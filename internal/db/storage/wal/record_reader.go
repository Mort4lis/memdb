package wal

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"

	"google.golang.org/protobuf/encoding/protodelim"
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

func (rr *RecordReader) All() iter.Seq2[*Record, error] {
	return func(yield func(*Record, error) bool) {
		for data, err := range rr.sd.List() {
			if err != nil {
				yield(nil, err)
				return
			}

			buf := bytes.NewBuffer(data)
			for {
				var r Record
				decErr := protodelim.UnmarshalFrom(buf, &r)
				if errors.Is(decErr, io.EOF) {
					break
				}

				if decErr != nil {
					decErr = fmt.Errorf("decode record: %w", decErr)
					yield(nil, decErr)
					return
				}

				if !yield(&r, nil) {
					return
				}
			}
		}
	}
}
