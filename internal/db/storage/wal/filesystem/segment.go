package filesystem

import (
	"errors"
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
	closed  bool
}

func NewSegment(dirPath string, maxSize int) (*Segment, error) {
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return nil, fmt.Errorf("create segment directory: %w", err)
	}

	paths, err := fsutils.ListDirPaths(dirPath)
	if err != nil {
		return nil, fmt.Errorf("list all segment paths: %w", err)
	}

	sort.Strings(paths)
	if len(paths) == 0 {
		return &Segment{dirPath: dirPath, maxSize: maxSize}, nil
	}

	lastPath := paths[len(paths)-1]
	file, err := os.OpenFile(lastPath, os.O_WRONLY|os.O_APPEND, 0o644)
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
	if s.closed {
		return errors.New("segment is closed")
	}
	if len(b) > s.maxSize {
		return errors.New("write data is too big")
	}
	if s.file == nil || s.curSize+len(b) > s.maxSize {
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

	if s.file != nil {
		err = s.file.Close()
		if err != nil {
			err = fmt.Errorf("close file %s: %w", s.file.Name(), err)
		}
	}

	s.file = file
	s.curSize = 0
	return err
}

func (s *Segment) Close() error {
	if s.file == nil {
		s.closed = true
		return nil
	}
	if err := s.file.Close(); err != nil {
		return fmt.Errorf("close file %s: %w", s.file.Name(), err)
	}

	s.file = nil
	s.closed = true
	return nil
}

func newSegmentPath(dirPath string) string {
	now := time.Now()
	filename := fmt.Sprintf("%s_%d", now.Format("2006-01-02"), now.UnixMicro())
	return filepath.Join(dirPath, filename)
}
