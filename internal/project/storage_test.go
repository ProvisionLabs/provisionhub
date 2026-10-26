package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadInUnicodeDirectory(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "meu projeto café", "nested")
	path, err := Save(destination, exampleManifest())
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(destination, ManifestName) {
		t.Fatalf("path = %q", path)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Name != "my-application" || got.Spec.Backend.Technology != "nestjs" {
		t.Fatalf("persisted manifest = %+v", got)
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != ManifestName {
		t.Fatalf("temporary file left behind: %v", entries)
	}
}

func TestSaveNeverOverwritesExistingManifest(t *testing.T) {
	destination := t.TempDir()
	path := filepath.Join(destination, ManifestName)
	if err := os.WriteFile(path, []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(destination, exampleManifest()); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("save conflict error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "sentinel" {
		t.Fatalf("existing manifest was changed: %q, %v", data, err)
	}
}

func TestCheckDestinationRejectsFileAncestor(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "file")
	if err := os.WriteFile(parent, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CheckDestination(filepath.Join(parent, "child")); err == nil || !strings.Contains(err.Error(), "contains a file") {
		t.Fatalf("file ancestor error = %v", err)
	}
	if _, _, err := CheckDestination("bad\x00path"); err == nil {
		t.Fatal("accepted NUL in path")
	}
}
