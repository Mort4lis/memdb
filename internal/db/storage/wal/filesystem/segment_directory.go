package filesystem

import (
	"fmt"
	"iter"
	"os"
	"sort"

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

func (sd *SegmentDirectory) List() iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		paths, err := fsutils.ListDir(sd.dirPath)
		if err != nil {
			err = fmt.Errorf("list all segments: %w", err)
			yield(nil, err)
			return
		}
		sort.Strings(paths)

		for _, path := range paths {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				readErr = fmt.Errorf("read segment %s: %w", path, readErr)
				yield(nil, readErr)
				return
			}
			if !yield(data, nil) {
				return
			}
		}
	}
}
