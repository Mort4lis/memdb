package fsutils

import (
	"fmt"
	"os"
	"path/filepath"
)

func ListDirNames(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dirPath, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	return names, nil
}

func ListDirPaths(dirPath string) ([]string, error) {
	names, err := ListDirNames(dirPath)
	if err != nil {
		return nil, err
	}

	paths := make([]string, len(names))
	for i := range paths {
		paths[i] = filepath.Join(dirPath, names[i])
	}
	return paths, nil
}
