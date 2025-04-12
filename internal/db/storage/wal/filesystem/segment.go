package filesystem

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Mort4lis/memdb/internal/pkg/fsutils"
)

const segmentFileNameFormat = "2006-01-02T15:04:05.999999999"

type Segment struct {
	f *os.File
	r *bufio.Reader
	w *bufio.Writer
}

func openSegment(path string) (*Segment, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return &Segment{
		f: f,
		r: bufio.NewReader(f),
		w: bufio.NewWriter(f),
	}, nil
}

func newSegment(dirPath string) (*Segment, error) {
	fileName := time.Now().UTC().Format(segmentFileNameFormat) + ".wal"
	filePath := filepath.Join(dirPath, fileName)

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return &Segment{
		f: f,
		r: bufio.NewReader(f),
		w: bufio.NewWriter(f),
	}, nil
}

func (s *Segment) Size() (int64, error) {
	info, err := s.f.Stat()
	if err != nil {
		return 0, fmt.Errorf("get file info: %w", err)
	}
	return info.Size(), nil
}

func (s *Segment) Write(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	if _, err := s.w.Write(b); err != nil {
		return fmt.Errorf("write to buffer: %w", err)
	}
	if b[len(b)-1] == '\n' {
		return nil
	}
	if err := s.w.WriteByte('\n'); err != nil {
		return fmt.Errorf("write to buffer: %w", err)
	}
	return nil
}

func (s *Segment) ReadLine() ([]byte, error) {
	var line []byte
	for {
		l, more, err := s.r.ReadLine()
		if err != nil {
			return nil, err //nolint:wrapcheck // ignore
		}
		// avoid the copy if the first call produced a full line
		if line == nil && !more {
			return l, nil
		}
		line = append(line, l...)
		if !more {
			break
		}
	}
	return line, nil
}

func (s *Segment) Flush() error {
	if err := s.w.Flush(); err != nil {
		return fmt.Errorf("flush buffer: %w", err)
	}
	if err := s.f.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}
	return nil
}

func (s *Segment) Close() error {
	if err := s.f.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	return nil
}

type SegmentController struct {
	*Segment
	dirPath        string
	maxSegmentSize int64
}

func NewSegmentController(dirPath string, maxSegmentSize int) (*SegmentController, error) {
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return nil, fmt.Errorf("create segments directory: %w", err)
	}

	segmentPaths, err := fsutils.ListDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("list segments: %w", err)
	}

	var lastSegment *Segment
	if len(segmentPaths) != 0 {
		lastSegment, err = openSegment(segmentPaths[len(segmentPaths)-1])
		if err != nil {
			return nil, fmt.Errorf("open last segment: %w", err)
		}
	}

	if lastSegment == nil {
		lastSegment, err = newSegment(dirPath)
		if err != nil {
			return nil, fmt.Errorf("create new segment: %w", err)
		}
	}

	return &SegmentController{
		Segment:        lastSegment,
		dirPath:        dirPath,
		maxSegmentSize: int64(maxSegmentSize),
	}, nil
}

func (sc *SegmentController) WalkLines(fn func(line []byte) error) error {
	segmentPaths, err := fsutils.ListDir(sc.dirPath)
	if err != nil {
		return fmt.Errorf("list segments: %w", err)
	}
	for _, path := range segmentPaths {
		if err = walkSegmentLines(path, fn); err != nil {
			return fmt.Errorf("read segment %s: %w", path, err)
		}
	}
	return nil
}

func (sc *SegmentController) Flush() error {
	if err := sc.Segment.Flush(); err != nil {
		return fmt.Errorf("flush segment: %w", err)
	}

	segmentSize, err := sc.Segment.Size()
	if err != nil {
		return fmt.Errorf("get segment size: %w", err)
	}
	if segmentSize < sc.maxSegmentSize {
		return nil
	}

	if err = sc.Segment.Close(); err != nil {
		return fmt.Errorf("close segment: %w", err)
	}

	sc.Segment, err = newSegment(sc.dirPath)
	if err != nil {
		return fmt.Errorf("create new segment: %w", err)
	}
	return nil
}

func walkSegmentLines(path string, fn func(line []byte) error) error {
	seg, err := openSegment(path)
	if err != nil {
		return fmt.Errorf("open segment: %w", err)
	}
	defer seg.Close()

	var line []byte
	for {
		line, err = seg.ReadLine()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err = fn(line); err != nil {
			return fmt.Errorf("call callback for segment line: %w", err)
		}
	}
}
