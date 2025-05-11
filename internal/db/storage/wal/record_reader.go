package wal

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"

	"google.golang.org/protobuf/encoding/protodelim"
)

//go:generate mockery --inpackage --testonly --case underscore --name SegmentDirectory
type SegmentDirectory interface {
	List() iter.Seq2[[]byte, error]
}

type recordReader struct {
	sd SegmentDirectory
}

func newRecordReader(sd SegmentDirectory) *recordReader {
	return &recordReader{sd: sd}
}

func (rr *recordReader) All() iter.Seq2[*Record, error] {
	return func(yield func(*Record, error) bool) {
		for data, err := range rr.sd.List() {
			if err != nil {
				yield(nil, err)
				return
			}

			rs, decErr := DecodeRecords(data)
			if decErr != nil {
				yield(nil, decErr)
				return
			}

			for _, r := range rs {
				if !yield(r, nil) {
					return
				}
			}
		}
	}
}

func DecodeRecords(b []byte) ([]*Record, error) {
	var rs []*Record
	buf := bytes.NewBuffer(b)

	for {
		var r Record
		err := protodelim.UnmarshalFrom(buf, &r)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode record: %w", err)
		}

		rs = append(rs, &r)
	}

	return rs, nil
}
