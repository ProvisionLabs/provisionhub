package application

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/localdb"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func savedFixture(t *testing.T, directory string) string {
	t.Helper()
	manifest := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "same-name"}, Spec: project.Spec{Backend: &project.TechnologySelection{Technology: "nestjs"}}}
	path, err := project.Save(directory, manifest)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSavedProjectsRestartRelocateForgetAndFiles(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	service := SavedProjects{DatabasePath: filepath.Join(root, "data", "app.db"), Projects: NewProjectService()}
	first := savedFixture(t, filepath.Join(root, "first"))
	second := savedFixture(t, filepath.Join(root, "second"))
	before, _ := os.ReadFile(first)
	a, err := service.Register(ctx, filepath.Dir(first))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Register(ctx, first); !errors.Is(err, localdb.ErrSavedProjectExists) {
		t.Fatalf("duplicate: %v", err)
	}
	b, err := service.Register(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Fatal("duplicate local identities")
	}
	views, err := service.List(ctx)
	if err != nil || len(views) != 2 {
		t.Fatalf("restart listing: %d %v", len(views), err)
	}
	for _, view := range views {
		if view.Manifest.Metadata.Name != "same-name" {
			t.Fatal("name not read from manifest")
		}
	}
	opened, err := service.Open(ctx, a.ID)
	if err != nil || opened.Manifest.Metadata.Name != "same-name" {
		t.Fatalf("open: %+v %v", opened, err)
	}
	if err := os.Rename(filepath.Dir(first), filepath.Join(root, "moved")); err != nil {
		t.Fatal(err)
	}
	views, err = service.List(ctx)
	if err != nil || views[0].ManifestError == nil {
		t.Fatalf("moved project not shown unavailable: %v", err)
	}
	moved := filepath.Join(root, "moved", project.ManifestName)
	if err := service.Relocate(ctx, a.ID, second); !errors.Is(err, localdb.ErrSavedProjectExists) {
		t.Fatalf("duplicate relocation: %v", err)
	}
	if err := service.Relocate(ctx, a.ID, moved); err != nil {
		t.Fatal(err)
	}
	opened, err = service.Open(ctx, a.ID)
	canonicalMoved, canonicalErr := CanonicalManifestPath(moved)
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	if err != nil || opened.Reference.ID != a.ID || opened.Reference.ManifestPath != canonicalMoved {
		t.Fatalf("relocated open: %+v %v", opened, err)
	}
	if err := service.Forget(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Open(ctx, a.ID); !errors.Is(err, localdb.ErrSavedProjectMissing) {
		t.Fatalf("forgotten open: %v", err)
	}
	after, err := os.ReadFile(moved)
	if err != nil || string(before) != string(after) {
		t.Fatalf("forget altered manifest: %v", err)
	}
	if _, err := os.Stat(second); err != nil {
		t.Fatal(err)
	}
}

func TestSavedProjectValidationAndCanonicalSymlink(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	service := SavedProjects{DatabasePath: filepath.Join(root, "data.db"), Projects: NewProjectService()}
	file := savedFixture(t, filepath.Join(root, "real"))
	if _, err := service.Register(ctx, filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing accepted")
	}
	bad := filepath.Join(root, "bad.yaml")
	if err := os.WriteFile(bad, []byte("not: a project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Register(ctx, bad); err == nil {
		t.Fatal("invalid accepted")
	}
	if _, err := service.Register(ctx, file); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias.yaml")
	if err := os.Symlink(file, alias); err == nil {
		if _, err := service.Register(ctx, alias); !errors.Is(err, localdb.ErrSavedProjectExists) {
			t.Fatalf("symlink duplicate: %v", err)
		}
	}
	views, err := service.List(ctx)
	if err != nil || len(views) != 1 {
		t.Fatalf("list after failures: %v %+v", err, views)
	}
}

func TestExplicitSavedProjectDatabaseDoesNotTouchSentinel(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	projectService := NewProjectService()
	sentinel := SavedProjects{DatabasePath: filepath.Join(root, "sentinel", "provisionhub.db"), Projects: projectService}
	isolated := SavedProjects{DatabasePath: filepath.Join(root, "isolated", "provisionhub.db"), Projects: projectService}
	legitimate := savedFixture(t, filepath.Join(root, "legitimate"))
	if _, err := sentinel.Register(ctx, legitimate); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(sentinel.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	temporary := savedFixture(t, filepath.Join(root, "temporary"))
	if _, err := isolated.Register(ctx, temporary); err != nil {
		t.Fatal(err)
	}
	store, err := localdb.Open(ctx, isolated.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	views, err := isolated.List(ctx)
	if err != nil || len(views) != 1 || views[0].ManifestError != nil {
		t.Fatalf("isolated reopen: %+v %v", views, err)
	}
	after, err := os.ReadFile(sentinel.DatabasePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("sentinel changed: %v", err)
	}
	views, err = sentinel.List(ctx)
	canonical, canonicalErr := CanonicalManifestPath(legitimate)
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	if err != nil || len(views) != 1 || views[0].Reference.ManifestPath != canonical {
		t.Fatalf("sentinel registration changed: %+v %v", views, err)
	}
}
