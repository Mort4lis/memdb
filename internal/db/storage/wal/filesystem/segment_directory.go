package filesystem

import (
	"fmt"
	"iter"
	"os"

	"github.com/Mort4lis/memdb/internal/pkg/fsutils"
)

type SegmentDirectory struct {
	dirPath string
}

func NewSegmentDirectory(dirPath string) (*SegmentDirectory, error) {
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return nil, fmt.Errorf("create segment directory: %w", err)
	}
	return &SegmentDirectory{dirPath: dirPath}, nil
}

func (sd *SegmentDirectory) List() (iter.Seq2[[]byte, error], error) {
	paths, err := fsutils.ListDir(sd.dirPath)
	if err != nil {
		return nil, fmt.Errorf("list all segments: %w", err)
	}

	return func(yield func([]byte, error) bool) {
		for _, path := range paths {
			var data []byte
			data, err = os.ReadFile(path)
			if err != nil {
				err = fmt.Errorf("read segment: %w", err)
				yield(nil, err)
				return
			}
			if !yield(data, nil) {
				return
			}
		}
	}, nil
}
