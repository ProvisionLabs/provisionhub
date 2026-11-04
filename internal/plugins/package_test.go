package plugins

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageDeterministicAndComplete(t *testing.T) {
	folder := writeTestPlugin(t, t.TempDir(), "nestjs", testPlugin)
	os.WriteFile(filepath.Join(folder, ".env"), []byte("never package"), 0o600)
	first, m, err := Package(folder)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := Package(folder)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("non deterministic", err)
	}
	got, err := Extract(first, t.TempDir())
	if err != nil || got.ID != m.ID || got.Digest != m.Digest {
		t.Fatal(got, err)
	}
	if len(Checksum(first)) != 64 {
		t.Fatal("checksum")
	}
	os.Remove(filepath.Join(folder, "templates", "package.json.tmpl"))
	if _, _, err := Package(folder); err == nil {
		t.Fatal("missing input accepted")
	}
}

func TestDecompressionBombAndFileCountLimits(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	writer.Write([]byte(strings.Repeat("x", int(MaxExpandedSize)+MaxPackageFiles*1024+1)))
	writer.Close()
	if _, err := Extract(compressed.Bytes(), t.TempDir()); err == nil {
		t.Fatal("decompression bomb accepted")
	}
	headers := make([]tar.Header, MaxPackageFiles+1)
	for i := range headers {
		headers[i] = tar.Header{Name: fmt.Sprintf("file-%04d", i), Typeflag: tar.TypeReg}
	}
	if _, err := Extract(maliciousArchive(t, headers...), t.TempDir()); err == nil {
		t.Fatal("file count limit ignored")
	}
}
func maliciousArchive(t *testing.T, headers ...tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	for _, h := range headers {
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			tw.Write(make([]byte, h.Size))
		}
	}
	tw.Close()
	gz.Close()
	return b.Bytes()
}
func TestArchiveSecurity(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", "C:/absolute", "a\\escape", "CON", "trailing.", "a/../b"} {
		t.Run(name, func(t *testing.T) {
			if _, err := Extract(maliciousArchive(t, tar.Header{Name: name, Mode: 0o644, Typeflag: tar.TypeReg}), t.TempDir()); err == nil {
				t.Fatal("unsafe path accepted")
			}
		})
	}
	cases := [][]tar.Header{
		{{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "../escape"}},
		{{Name: "hard", Typeflag: tar.TypeLink, Linkname: "../escape"}},
		{{Name: "a", Typeflag: tar.TypeReg}, {Name: "A", Typeflag: tar.TypeReg}},
		{{Name: "huge", Typeflag: tar.TypeReg, Size: (1 << 20) + 1}},
		{{Name: "a", Typeflag: tar.TypeReg}, {Name: "a/b", Typeflag: tar.TypeReg}},
	}
	for _, headers := range cases {
		if _, err := Extract(maliciousArchive(t, headers...), t.TempDir()); err == nil {
			t.Fatal("unsafe archive accepted", headers)
		}
	}
	for _, data := range [][]byte{[]byte("broken"), make([]byte, MaxArchiveSize+1), maliciousArchive(t)} {
		if _, err := Extract(data, t.TempDir()); err == nil {
			t.Fatal("invalid archive accepted")
		}
	}
	folder := writeTestPlugin(t, t.TempDir(), "nestjs", testPlugin)
	data, _, _ := Package(folder)
	if _, err := Extract(data[:len(data)-6], t.TempDir()); err == nil {
		t.Fatal("truncated gzip accepted")
	}
	if _, err := Extract(data, folder); err == nil {
		t.Fatal("nonempty destination accepted")
	}
}
