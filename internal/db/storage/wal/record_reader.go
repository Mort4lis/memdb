package wal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
)

type segmentDirectory interface {
	List() (iter.Seq2[[]byte, error], error)
}

type RecordReader struct {
	sd segmentDirectory
}

func NewRecordReader(sd segmentDirectory) *RecordReader {
	return &RecordReader{sd: sd}
}

func (rr *RecordReader) All() (iter.Seq2[Record, error], error) {
	seq, err := rr.sd.List()
	if err != nil {
		return nil, err
	}

	return func(yield func(Record, error) bool) {
		var data []byte
		for data, err = range seq {
			if err != nil {
				yield(Record{}, err)
				return
			}

			dec := json.NewDecoder(bytes.NewBuffer(data))
			for {
				var r Record
				err = dec.Decode(&r)
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					err = fmt.Errorf("decode record: %w", err)
					yield(r, err)
					return
				}

				if !yield(r, nil) {
					return
				}
			}
		}
	}, nil
}
