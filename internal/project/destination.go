package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ValidateDestination resolves an output directory and rejects file ancestors.
// It does not create files or require project metadata.
func ValidateDestination(destination string) (string, error) {
	if strings.TrimSpace(destination) == "" {
		return "", errors.New("destination path is required")
	}
	if strings.ContainsRune(destination, 0) {
		return "", errors.New("destination path contains a NUL character")
	}
	directory, err := filepath.Abs(destination)
	if err != nil {
		return "", fmt.Errorf("resolve destination %q: %w", destination, err)
	}
	directory = filepath.Clean(directory)
	for current := directory; ; current = filepath.Dir(current) {
		info, statErr := os.Lstat(current)
		if statErr == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("destination path %q contains a file or symlink where a directory is required", current)
			}
			break
		}
		if !errors.Is(statErr, fs.ErrNotExist) {
			return "", fmt.Errorf("inspect destination %q: %w", current, statErr)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("destination %q has no accessible parent directory", directory)
		}
	}
	return directory, nil
}
