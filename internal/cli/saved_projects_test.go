package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestProjectSavedCommandsAndInitRegistrationFailure(t *testing.T) {
	base := t.TempDir()
	t.Setenv("PROVISIONHUB_DATA_DIR", filepath.Join(base, "data"))
	output := filepath.Join(base, "project")
	if _, err := executeCLI("project", "init", "--name", "demo", "--path", output, "--backend", "nestjs"); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(output, project.ManifestName)
	canonicalFile, err := application.CanonicalManifestPath(file)
	if err != nil {
		t.Fatal(err)
	}
	listed, err := executeCLI("project", "list")
	if err != nil || !strings.Contains(listed, canonicalFile) || !strings.Contains(listed, "demo") {
		t.Fatalf("list: %s %v", listed, err)
	}
	if _, err := executeCLI("project", "register", "--file", file); err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("duplicate register: %v", err)
	}
	if _, err := executeCLI("project", "forget", "--file", file); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatal("forget deleted manifest", err)
	}
	listed, err = executeCLI("project", "list")
	if err != nil || strings.Contains(listed, canonicalFile) {
		t.Fatalf("forget list: %s %v", listed, err)
	}
	if _, err := executeCLI("project", "register", "--file", output); err != nil {
		t.Fatal(err)
	}
	listed, err = executeCLI("project", "list")
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Fields(listed)[0]
	moved := filepath.Join(base, "moved")
	if err := os.Rename(output, moved); err != nil {
		t.Fatal(err)
	}
	if _, err := executeCLI("project", "relocate", "--id", id, "--file", moved); err != nil {
		t.Fatal(err)
	}
	canonicalMoved, err := application.CanonicalManifestPath(moved)
	if err != nil {
		t.Fatal(err)
	}
	listed, err = executeCLI("project", "list")
	if err != nil || !strings.Contains(listed, canonicalMoved) || !strings.HasPrefix(listed, id) {
		t.Fatalf("relocated list: %s %v", listed, err)
	}
	if _, err := executeCLI("project", "forget", "--id", id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(moved, project.ManifestName)); err != nil {
		t.Fatal("forget removed moved manifest", err)
	}
	blocked := filepath.Join(base, "blocked")
	if err := os.WriteFile(blocked, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROVISIONHUB_DATA_DIR", blocked)
	failed := filepath.Join(base, "created-even-if-registration-fails")
	if _, err := executeCLI("project", "init", "--name", "demo", "--path", failed, "--backend", "nestjs"); err == nil || !strings.Contains(err.Error(), "registration failed") {
		t.Fatalf("registration error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(failed, project.ManifestName)); err != nil {
		t.Fatalf("registration failure lost manifest: %v", err)
	}
}
