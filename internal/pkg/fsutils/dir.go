package fsutils

import (
	"fmt"
	"os"
	"path/filepath"
)

func ListDir(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dirPath, err)
	}

	var filePaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePaths = append(filePaths, filepath.Join(dirPath, entry.Name()))
	}

	return filePaths, nil
}
