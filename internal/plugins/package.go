package plugins

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
)

const MaxArchiveSize int64 = 16 << 20
const MaxExpandedSize int64 = 32 << 20
const MaxPackageFiles = 1024

// LoadFolder uses precisely the same validation as local discovery.
func LoadFolder(folder string) (Manifest, error) {
	info, err := os.Lstat(folder)
	if err != nil {
		return Manifest{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return Manifest{}, errors.New("plugin folder must be a real directory")
	}
	root, err := os.OpenRoot(folder)
	if err != nil {
		return Manifest{}, err
	}
	defer root.Close()
	m, found, err := loadPluginFolder(root, folder, catalog.Builtin())
	if err == nil && !found {
		err = errors.New("plugin.yaml is required")
	}
	return m, err
}

// Package includes only validated declared inputs. Metadata and ordering are fixed.
func Package(folder string) ([]byte, Manifest, error) {
	m, err := LoadFolder(folder)
	if err != nil {
		return nil, m, err
	}
	root, err := os.OpenRoot(folder)
	if err != nil {
		return nil, m, err
	}
	defer root.Close()
	manifest, err := readBounded(root, "plugin.yaml", 1<<20)
	if err != nil {
		return nil, m, err
	}
	if sha256.Sum256(manifest) != m.Digest {
		return nil, m, errors.New("manifest changed during packaging")
	}
	files := map[string][]byte{"plugin.yaml": manifest}
	for _, f := range m.Files {
		files[f.Source] = f.Content
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	if len(names) > MaxPackageFiles {
		return nil, m, errors.New("too many package files")
	}
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	gz.Header.OS = 255
	tw := tar.NewWriter(gz)
	var total int64
	for _, name := range names {
		content := files[name]
		total += int64(len(content))
		if total > MaxExpandedSize {
			return nil, m, errors.New("package exceeds expanded size limit")
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}); err != nil {
			return nil, m, err
		}
		if _, err := tw.Write(content); err != nil {
			return nil, m, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, m, err
	}
	if err := gz.Close(); err != nil {
		return nil, m, err
	}
	if int64(buffer.Len()) > MaxArchiveSize {
		return nil, m, errors.New("compressed package exceeds limit")
	}
	return buffer.Bytes(), m, nil
}
func Checksum(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// ContentChecksum is independent of gzip implementations and archive metadata.
// The manifest digest binds identity/version/paths, and sorted file digests bind
// every generator input. No file needs to be recompressed during local loading.
func ContentChecksum(manifest Manifest) string {
	hash := sha256.New()
	hash.Write(manifest.Digest[:])
	for _, file := range manifest.Files {
		fmt.Fprintf(hash, "%d:%s", len(file.Source), file.Source)
		hash.Write(file.Digest[:])
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// Extract accepts only portable regular files; callers supply an empty isolated stage.
// It does not activate packages or run downloaded content.
func Extract(data []byte, destination string) (Manifest, error) {
	if int64(len(data)) > MaxArchiveSize {
		return Manifest{}, errors.New("compressed archive exceeds limit")
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		return Manifest{}, err
	}
	if len(entries) != 0 {
		return Manifest{}, errors.New("extraction destination must be empty")
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return Manifest{}, err
	}
	defer root.Close()
	reader := bytes.NewReader(data)
	gz, err := gzip.NewReader(reader)
	if err != nil {
		return Manifest{}, fmt.Errorf("invalid gzip: %w", err)
	}
	defer gz.Close()
	gz.Multistream(false)
	expanded, err := io.ReadAll(io.LimitReader(gz, MaxExpandedSize+(MaxPackageFiles*1024)+1))
	if err != nil {
		return Manifest{}, fmt.Errorf("invalid compressed archive: %w", err)
	}
	if int64(len(expanded)) > MaxExpandedSize+(MaxPackageFiles*1024) || reader.Len() != 0 {
		return Manifest{}, errors.New("archive exceeds limit or contains trailing streams")
	}
	tr := tar.NewReader(bytes.NewReader(expanded))
	seen := map[string]bool{}
	var total int64
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Manifest{}, fmt.Errorf("invalid tar: %w", err)
		}
		if header.Typeflag != tar.TypeReg || header.Linkname != "" || len(header.PAXRecords) > 0 {
			return Manifest{}, errors.New("archive may contain only regular files without extended headers")
		}
		if err := validatePath(header.Name); err != nil {
			return Manifest{}, fmt.Errorf("unsafe archive path %q: %w", header.Name, err)
		}
		key := strings.ToLower(header.Name)
		if seen[key] {
			return Manifest{}, fmt.Errorf("duplicate archive path %q", header.Name)
		}
		seen[key] = true
		if len(seen) > MaxPackageFiles || header.Size < 0 || header.Size > 1<<20 {
			return Manifest{}, errors.New("archive file limit exceeded")
		}
		total += header.Size
		if total > MaxExpandedSize {
			return Manifest{}, errors.New("archive expanded size limit exceeded")
		}
		if err := root.MkdirAll(filepath.FromSlash(path.Dir(header.Name)), 0o700); err != nil {
			return Manifest{}, err
		}
		file, err := root.OpenFile(filepath.FromSlash(header.Name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return Manifest{}, err
		}
		_, copyErr := io.CopyN(file, tr, header.Size)
		closeErr := file.Close()
		if copyErr != nil {
			return Manifest{}, copyErr
		}
		if closeErr != nil {
			return Manifest{}, closeErr
		}
	}
	m, err := LoadFolder(destination)
	if err != nil {
		return Manifest{}, err
	}
	allowed := map[string]bool{"plugin.yaml": true}
	for _, f := range m.Files {
		allowed[strings.ToLower(f.Source)] = true
	}
	if len(allowed) != len(seen) {
		return Manifest{}, errors.New("package contains undeclared or missing files")
	}
	for name := range seen {
		if !allowed[name] {
			return Manifest{}, fmt.Errorf("undeclared package file %q", name)
		}
	}
	return m, nil
}
