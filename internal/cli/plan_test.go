package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/project"
)

type unreadableInput struct{ t *testing.T }

func (r unreadableInput) Read([]byte) (int, error) {
	r.t.Fatal("plan command read interactive input")
	return 0, nil
}

func planFixture(t *testing.T) string {
	t.Helper()
	manifest := project.Manifest{
		APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "demo"},
		Spec: project.Spec{
			Backend:  &project.TechnologySelection{Technology: "nestjs"},
			Frontend: &project.TechnologySelection{Technology: "nextjs"},
			Database: &project.TechnologySelection{Technology: "postgresql"},
		},
	}
	path, err := project.Save(filepath.Join(t.TempDir(), "project café"), manifest)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProjectPlanPreviewIsNonInteractiveAndReadOnly(t *testing.T) {
	path := planFixture(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(unreadableInput{t})
	cmd.SetArgs([]string{"project", "plan", "--file", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("valid unsupported configuration must succeed: %v", err)
	}
	for _, part := range []string{"Generation Plan", "demo", "NestJS", "Next.js", "PostgreSQL", "backend:nestjs", "database:postgresql", "frontend:nextjs", "unavailable", "Status: planned", "Execution: unavailable", "missing_operation_provider"} {
		if !strings.Contains(out.String(), part) {
			t.Errorf("preview missing %q: %s", part, out.String())
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("manifest changed: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 || entries[0].Name() != project.ManifestName {
		t.Fatalf("planning wrote files: %v, %v", entries, err)
	}
}
