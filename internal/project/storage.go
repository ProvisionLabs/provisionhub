package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CheckDestination resolves a directory and verifies that a new manifest can
// be placed there. Save checks again because the filesystem can change later.
func CheckDestination(destination string) (directory, manifestPath string, err error) {
	if strings.TrimSpace(destination) == "" {
		return "", "", errors.New("destination path is required")
	}
	if strings.ContainsRune(destination, 0) {
		return "", "", errors.New("destination path contains a NUL character")
	}
	directory, err = filepath.Abs(destination)
	if err != nil {
		return "", "", fmt.Errorf("resolve destination %q: %w", destination, err)
	}
	directory = filepath.Clean(directory)
	for current := directory; ; current = filepath.Dir(current) {
		info, statErr := os.Stat(current)
		if statErr == nil {
			if !info.IsDir() {
				return "", "", fmt.Errorf("destination path %q contains a file where a directory is required", current)
			}
			break
		}
		if !errors.Is(statErr, fs.ErrNotExist) {
			if ancestor := nonDirectoryAncestor(current); ancestor != "" {
				return "", "", fmt.Errorf("destination path %q contains a file where a directory is required", ancestor)
			}
			return "", "", fmt.Errorf("inspect destination %q: %w", current, statErr)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", "", fmt.Errorf("destination %q has no accessible parent directory", directory)
		}
	}
	manifestPath = filepath.Join(directory, ManifestName)
	if _, err = os.Lstat(manifestPath); err == nil {
		return "", "", fmt.Errorf("manifest already exists at %q; choose a different destination", manifestPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", "", fmt.Errorf("inspect manifest %q: %w", manifestPath, err)
	}
	return directory, manifestPath, nil
}

// On Unix, stat of a child of a regular file reports ENOTDIR rather than
// fs.ErrNotExist. Walk upward to identify that file for a portable error.
func nonDirectoryAncestor(path string) string {
	for current := filepath.Dir(path); ; current = filepath.Dir(current) {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() {
				return current
			}
			return ""
		}
		if parent := filepath.Dir(current); parent == current {
			return ""
		}
	}
}

// Save atomically publishes a complete manifest without replacing an existing file.
func Save(destination string, manifest Manifest) (string, error) {
	data, err := Marshal(manifest)
	if err != nil {
		return "", err
	}
	directory, path, err := CheckDestination(destination)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("create destination %q: %w", directory, err)
	}
	temporary, err := os.CreateTemp(directory, ".provisionhub-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary manifest in %q: %w", directory, err)
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return "", fmt.Errorf("write temporary manifest: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return "", fmt.Errorf("sync temporary manifest: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close temporary manifest: %w", err)
	}
	// A hard link within the same directory is atomic and fails if path exists.
	// Rename cannot provide the no-overwrite guarantee on every supported OS.
	if err := os.Link(temporary.Name(), path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("manifest already exists at %q; choose a different destination", path)
		}
		return "", fmt.Errorf("publish manifest at %q: %w", path, err)
	}
	return path, nil
}

func Load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest %q: %w", path, err)
	}
	manifest, err := Unmarshal(data)
	if err != nil {
		return Manifest{}, fmt.Errorf("manifest %q: %w", path, err)
	}
	return manifest, nil
}
