package wal

import (
	"bytes"
	"iter"

	"github.com/Mort4lis/memdb/internal/db/storage/model"
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

func (rr *recordReader) All() iter.Seq2[*model.Record, error] {
	return func(yield func(*model.Record, error) bool) {
		for data, err := range rr.sd.List() {
			if err != nil {
				yield(nil, err)
				return
			}

			buf := bytes.NewBuffer(data)
			rs, decErr := model.DecodeRecords(buf)
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
