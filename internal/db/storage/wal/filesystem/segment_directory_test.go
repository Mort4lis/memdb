package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSegmentDirectory_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	segDir, err := NewSegmentDirectory(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, segDir.dirPath)
}

func TestNewSegmentDirectory_FailCreateDir(t *testing.T) {
	// Simulate by passing a file path, not a dir
	f, err := os.CreateTemp(t.TempDir(), "f")
	require.NoError(t, err)
	defer f.Close()

	_, err = NewSegmentDirectory(f.Name())
	assert.Error(t, err)
}

func TestSegmentDirectory_List_SortedByName(t *testing.T) {
	dir := t.TempDir()
	sfs := []segmentFile{
		{name: "2025-05-01_11", data: []byte("bbb")},
		{name: "2025-05-02_01", data: []byte("ccc")},
		{name: "2025-05-01_01", data: []byte("aaa")},
	}
	createSegmentFiles(t, dir, sfs)

	segDir, err := NewSegmentDirectory(dir)
	require.NoError(t, err)

	var yieldErr error
	got := make([][]byte, 0, len(sfs))
	segDir.List()(func(data []byte, err error) bool {
		if err != nil {
			yieldErr = err
		} else {
			got = append(got, data)
		}
		return true
	})

	want := [][]byte{[]byte("aaa"), []byte("bbb"), []byte("ccc")}
	assert.Equal(t, want, got)
	assert.NoError(t, yieldErr)
}

func TestSegmentDirectory_List_Interruption(t *testing.T) {
	dir := t.TempDir()
	sfs := []segmentFile{
		{name: "2025-05-01_11", data: []byte("bbb")},
		{name: "2025-05-02_01", data: []byte("ccc")},
		{name: "2025-05-01_01", data: []byte("aaa")},
	}
	createSegmentFiles(t, dir, sfs)

	segDir, err := NewSegmentDirectory(dir)
	require.NoError(t, err)

	var loopCounter int
	var yieldErr error
	var got [][]byte
	segDir.List()(func(data []byte, err error) bool {
		loopCounter++
		if err != nil {
			yieldErr = err
		} else {
			got = append(got, data)
		}

		if loopCounter == 2 {
			return false
		}
		return true
	})

	assert.NoError(t, yieldErr)
	assert.Equal(t, 2, loopCounter)
	want := [][]byte{[]byte("aaa"), []byte("bbb")}
	assert.Equal(t, want, got)
}

func TestSegmentDirectory_List_FailReadFile(t *testing.T) {
	dir := t.TempDir()
	sfs := []segmentFile{
		{name: "2025-05-01_01", data: []byte("aaa")},
		{name: "2025-05-01_02", data: []byte("bbb")},
	}
	createSegmentFiles(t, dir, sfs)

	segDir, err := NewSegmentDirectory(dir)
	require.NoError(t, err)

	// change permission to cause an error
	err = os.Chmod(filepath.Join(dir, sfs[0].name), 0000)
	require.NoError(t, err)

	var loopCounter int
	var yieldErr error
	segDir.List()(func(_ []byte, err error) bool {
		loopCounter++
		if err != nil {
			yieldErr = err
		}
		return true
	})

	assert.Error(t, yieldErr)
	assert.Equal(t, 1, loopCounter)
}

type segmentFile struct {
	name string
	data []byte
}

func createSegmentFiles(t *testing.T, dir string, sfs []segmentFile) {
	t.Helper()
	for _, sf := range sfs {
		path := filepath.Join(dir, sf.name)
		require.NoError(t, os.WriteFile(path, sf.data, 0666))
	}
}
