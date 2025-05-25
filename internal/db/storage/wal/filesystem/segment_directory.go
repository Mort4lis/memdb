package filesystem

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
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
		paths, err := fsutils.ListDirPaths(sd.dirPath)
		if err != nil {
			err = fmt.Errorf("list all segment paths: %w", err)
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

func (sd *SegmentDirectory) ContentByName(name string) ([]byte, error) {
	path := filepath.Join(sd.dirPath, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read segment %s: %w", path, err)
	}
	return data, nil
}

func (sd *SegmentDirectory) NextRotatedSegmentName(from string) (string, error) {
	names, err := fsutils.ListDirNames(sd.dirPath)
	if err != nil {
		return "", fmt.Errorf("list all segment names: %w", err)
	}
	if len(names) <= 1 {
		return "", nil
	}

	sort.Strings(names)
	// Exclude the newest segment (assumed to be the last after sorting)
	names = names[:len(names)-1]

	if from == "" {
		return names[0], nil
	}

	idx := sort.SearchStrings(names, from)
	if idx < len(names) && names[idx] == from {
		// Found. Return the next one
		if idx+1 < len(names) {
			return names[idx+1], nil
		}
		return "", nil // no next segment
	}
	return "", nil // 'from' not found, or no next segment
}

func (sd *SegmentDirectory) LastSegmentName() (string, error) {
	names, err := fsutils.ListDirNames(sd.dirPath)
	if err != nil {
		return "", fmt.Errorf("list all segment names: %w", err)
	}
	if len(names) == 0 {
		return "", nil
	}
	sort.Strings(names)
	return names[len(names)-1], nil
}

func (sd *SegmentDirectory) Save(name string, data []byte) error {
	path := filepath.Join(sd.dirPath, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}
