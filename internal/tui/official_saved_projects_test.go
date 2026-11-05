package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/project"
	"github.com/ProvisionLabs/provisionhub/internal/projectlock"
)

// This opt-in test runs against the adjacent official plugin definitions.
// Core CI remains self-contained and offline.
func TestOfficialSavedProjectLifecycle(t *testing.T) {
	plugins := os.Getenv("PROVISIONHUB_OFFICIAL_PLUGINS")
	if plugins == "" {
		t.Skip("set PROVISIONHUB_OFFICIAL_PLUGINS")
	}
	root := t.TempDir()
	t.Setenv("PROVISIONHUB_DATA_DIR", filepath.Join(root, "data"))
	savedPath := filepath.Join(root, "data", "provisionhub.db")
	service, err := application.NewProjectServiceWithPlugins(plugins)
	if err != nil {
		t.Fatal(err)
	}
	components := []project.Component{{Name: "api", Type: "backend", Technology: "nestjs"}, {Name: "billing", Type: "backend", Technology: "nestjs"}, {Name: "web", Type: "frontend", Technology: "nextjs"}, {Name: "admin", Type: "frontend", Technology: "nextjs"}}
	created, err := service.CreateComponentProject("commerce", filepath.Join(root, "commerce"), components)
	if err != nil {
		t.Fatal(err)
	}
	if err := registerCreatedProject(application.SavedProjects{DatabasePath: savedPath, Projects: service}, nil, created.ManifestPath); err != nil {
		t.Fatal(err)
	}
	// Construct a new browser as a process restart would; all state comes from SQLite and files.
	restarted, err := application.NewProjectServiceWithPlugins(plugins)
	if err != nil {
		t.Fatal(err)
	}
	b := newSavedBrowser(application.SavedProjects{DatabasePath: savedPath, Projects: restarted})
	deliverSaved(&b, b.load())
	if len(b.views) != 1 || len(b.views[0].Manifest.CanonicalComponents()) != 4 {
		t.Fatalf("reopen listing: %+v", b.views)
	}
	id := b.views[0].Reference.ID
	deliverSaved(&b, savedKey(&b, "enter"))
	if b.state != "details" || b.current.Manifest.Metadata.Name != "commerce" {
		t.Fatal("details unavailable", b.err)
	}
	savedKey(&b, "l")
	deliverSaved(&b, savedKey(&b, "c"))
	if b.state != "lockPreview" {
		t.Fatal("lock preview unavailable", b.err)
	}
	savedKey(&b, "enter")
	deliverSaved(&b, deliverSaved(&b, savedKey(&b, "y")))
	lock, found, err := projectlock.Load(created.ManifestPath)
	if err != nil || !found || len(lock.Plugins) != 2 {
		t.Fatalf("official lock: %+v %t %v", lock, found, err)
	}
	deliverSaved(&b, savedKey(&b, "p"))
	if b.state != "plan" || !b.plan.Executable {
		t.Fatalf("saved plan: %s %s", b.state, b.err)
	}
	savedKey(&b, "g")
	if b.state != "applyConfirm" {
		t.Fatalf("approval missing: %s", b.err)
	}
	deliverSaved(&b, savedKey(&b, "y"))
	for _, name := range []string{"api", "billing", "web", "admin"} {
		if _, err := os.Stat(filepath.Join(root, "commerce", name, "package.json")); err != nil {
			t.Fatalf("missing generated %s: %v", name, err)
		}
	}
	deliverSaved(&b, savedKey(&b, "esc"))
	moved := filepath.Join(root, "moved")
	if err := os.Rename(filepath.Join(root, "commerce"), moved); err != nil {
		t.Fatal(err)
	}
	deliverSaved(&b, b.load())
	if b.views[0].ManifestError == nil {
		t.Fatal("moved project not shown unavailable")
	}
	savedKey(&b, "m")
	b.input.SetValue(moved)
	deliverSaved(&b, deliverSaved(&b, savedKey(&b, "enter")))
	if len(b.views) != 1 || b.views[0].Reference.ID != id || b.views[0].ManifestError != nil {
		t.Fatalf("relocation: %+v %s", b.views, b.err)
	}
	deliverSaved(&b, savedKey(&b, "enter"))
	if b.state != "details" || len(b.current.Manifest.CanonicalComponents()) != 4 {
		t.Fatal("relocated open failed", b.err)
	}
	savedKey(&b, "f")
	deliverSaved(&b, deliverSaved(&b, savedKey(&b, "y")))
	if len(b.views) != 0 {
		t.Fatal("forget did not clear saved reference")
	}
	if _, err := os.Stat(filepath.Join(moved, ".provisionhub", "plugins.lock.yaml")); err != nil {
		t.Fatalf("forget removed lockfile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(moved, "api", "package.json")); err != nil {
		t.Fatalf("forget removed generated files: %v", err)
	}
}
