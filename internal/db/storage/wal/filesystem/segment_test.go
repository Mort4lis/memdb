package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSegment_UsesExistingFile(t *testing.T) {
	dir := t.TempDir()
	// Create the file in dir so it is not empty
	f, err := os.Create(filepath.Join(dir, "2025-05-01_1746100860543867"))
	require.NoError(t, err)

	_, err = f.WriteString("abc")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	seg, err := NewSegment(dir, 50)
	require.NoError(t, err)
	defer seg.Close()

	assert.NotNil(t, seg.file)
	assert.Equal(t, len("abc"), seg.curSize)
}

func TestNewSegment_FailCreateDir(t *testing.T) {
	// Use a file as a dir, so MkdirAll fails
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "not_a_dir")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0644))

	_, err := NewSegment(filePath, 7)
	assert.Error(t, err)
}

func TestNewSegment_CreatesNewSegmentOnEmpty(t *testing.T) {
	dir := t.TempDir()
	// No files in dir
	seg, err := NewSegment(dir, 10)
	require.NoError(t, err)
	assert.Nil(t, seg.file)
	assert.Equal(t, 10, seg.maxSize)
}

func TestSegment_Write_WritesDataAndSyncs(t *testing.T) {
	dir := t.TempDir()
	seg, err := NewSegment(dir, 100)
	require.NoError(t, err)
	defer seg.Close()

	data := []byte("hello")
	err = seg.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), seg.curSize)
}

func TestSegment_Write_FailsIfTooBigForSegment(t *testing.T) {
	dir := t.TempDir()
	seg, err := NewSegment(dir, 8)
	require.NoError(t, err)
	defer seg.Close()

	err = seg.Write([]byte("this is more than 8 bytes"))
	assert.ErrorContains(t, err, "too big")
}

func TestSegment_Write_RotatesOnOverflow(t *testing.T) {
	dir := t.TempDir()
	seg, err := NewSegment(dir, 6)
	require.NoError(t, err)
	defer seg.Close()

	require.NoError(t, seg.Write([]byte("123")))
	require.NoError(t, seg.Write([]byte("456"))) // reaches exactly maxSize
	oldFileName := seg.file.Name()

	// this should trigger rotation
	err = seg.Write([]byte("hey"))
	require.NoError(t, err)

	assert.NotEqual(t, oldFileName, seg.file.Name())
	assert.Equal(t, 2, countRegularFiles(t, dir))
}

func TestSegment_Write_CreatesMultipleSegmentsAsNeeded(t *testing.T) {
	dir := t.TempDir()
	seg, err := NewSegment(dir, 2)
	require.NoError(t, err)
	defer seg.Close()

	err = seg.Write([]byte("a"))
	require.NoError(t, err)
	err = seg.Write([]byte("b"))
	require.NoError(t, err)
	err = seg.Write([]byte("x"))
	require.NoError(t, err)
	err = seg.Write([]byte("y"))
	require.NoError(t, err)
	err = seg.Write([]byte("z"))
	require.NoError(t, err)

	assert.Equal(t, countRegularFiles(t, dir), 3)
}

func TestSegment_Write_ErrorIfClosed(t *testing.T) {
	dir := t.TempDir()
	seg, err := NewSegment(dir, 10)
	require.NoError(t, err)
	require.NoError(t, seg.Close())

	err = seg.Write([]byte("some"))
	assert.ErrorContains(t, err, "closed")
}

func TestSegment_Close_DoubleClose(t *testing.T) {
	dir := t.TempDir()
	seg, err := NewSegment(dir, 42)
	require.NoError(t, err)
	require.NoError(t, seg.Close())

	// Close again is fine
	require.NoError(t, seg.Close())
}

func countRegularFiles(t *testing.T, dir string) int {
	t.Helper()

	var n int
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, _ error) error {
		if d != nil && d.Type().IsRegular() {
			n++
		}
		return nil
	})
	require.NoError(t, err)

	return n
}
