package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Mort4lis/memdb/internal/pkg/fsutils"
)

type Segment struct {
	file    *os.File
	dirPath string
	curSize int
	maxSize int
}

func NewSegment(dirPath string, maxSize int) (*Segment, error) {
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return nil, fmt.Errorf("create segment directory: %w", err)
	}

	paths, err := fsutils.ListDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("list all segments: %w", err)
	}

	sort.Strings(paths)

	var lastPath string
	if len(paths) == 0 {
		lastPath = newSegmentPath(dirPath)
	} else {
		lastPath = paths[len(paths)-1]
	}

	file, err := os.OpenFile(lastPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", lastPath, err)
	}

	fi, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("get file info %s: %w", lastPath, err)
	}

	return &Segment{
		file:    file,
		dirPath: dirPath,
		maxSize: maxSize,
		curSize: int(fi.Size()),
	}, nil
}

func (s *Segment) Write(b []byte) error {
	if s.file == nil {
		return fmt.Errorf("segment is closed")
	}

	if len(b) > s.maxSize {
		return fmt.Errorf("write data is too big")
	}
	if s.curSize+len(b) > s.maxSize {
		if err := s.rotate(); err != nil {
			return fmt.Errorf("rotate segment: %w", err)
		}
	}

	if _, err := s.file.Write(b); err != nil {
		return fmt.Errorf("write file %s: %w", s.file.Name(), err)
	}
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("sync file %s: %w", s.file.Name(), err)
	}

	s.curSize += len(b)
	return nil
}

func (s *Segment) rotate() error {
	path := newSegmentPath(s.dirPath)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open file %s: %w", path, err)
	}

	oldFilename := s.file.Name()
	err = s.file.Close()

	s.file = file
	s.curSize = 0

	if err != nil {
		return fmt.Errorf("close file %s: %w", oldFilename, err)
	}
	return nil
}

func (s *Segment) Close() error {
	if s.file == nil {
		return nil
	}
	if err := s.file.Close(); err != nil {
		return fmt.Errorf("close file %s: %w", s.file.Name(), err)
	}

	s.file = nil
	return nil
}

func newSegmentPath(dirPath string) string {
	now := time.Now()
	filename := fmt.Sprintf("%s_%d", now.Format("2006-01-02"), now.UnixMicro())
	return filepath.Join(dirPath, filename)
}
