package application

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/project"
	"github.com/ProvisionLabs/provisionhub/internal/projectlock"
)

func TestLocalProjectLockPlanAndApplyIntegrity(t *testing.T) {
	base := t.TempDir()
	t.Setenv("PROVISIONHUB_DATA_DIR", filepath.Join(base, "data"))
	folder := filepath.Join(base, "plugins", "nestjs")
	if err := os.MkdirAll(filepath.Join(folder, "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	definition := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nestjs
version: 1.1.0
technologyId: nestjs
category: backend
name: NestJS
description: Test generator
capability: generate_backend
files:
  - type: template
    source: templates/package.tmpl
    destination: backend/package.json
`
	if err := os.WriteFile(filepath.Join(folder, "plugin.yaml"), []byte(definition), 0644); err != nil {
		t.Fatal(err)
	}
	template := filepath.Join(folder, "templates", "package.tmpl")
	if err := os.WriteFile(template, []byte(`{"name":"{{componentName}}"}`), 0644); err != nil {
		t.Fatal(err)
	}
	service, err := NewProjectServiceWithPlugins(filepath.Join(base, "plugins"))
	if err != nil {
		t.Fatal(err)
	}
	components := []project.Component{{Name: "api", Type: "backend", Technology: "nestjs"}, {Name: "billing", Type: "backend", Technology: "nestjs"}}
	created, err := service.CreateComponentProject("demo", filepath.Join(base, "demo"), components)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.PreviewProjectLock(created.ManifestPath)
	if err != nil || preview.Found || len(preview.Proposed.Plugins) != 1 {
		t.Fatalf("initial preview: %+v %v", preview, err)
	}
	lockPath, err := projectlock.Path(created.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("preview wrote lock")
	}
	// A cancelled preview leaves the project unlocked.
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("cancel wrote lock")
	}
	lock, err := service.LockProject(created.ManifestPath, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.Plugins) != 1 || lock.Plugins[0].Source != "local" {
		t.Fatal(lock)
	}
	if _, err := service.LockProject(created.ManifestPath, false, false); err == nil {
		t.Fatal("lock overwritten without --update")
	}
	conflicting, err := service.WithPluginVersions([]string{"local/official-nestjs@1.2.0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conflicting.PlanProject(created.ManifestPath); err == nil || !strings.Contains(err.Error(), "conflicts with lock") {
		t.Fatalf("selection conflict: %v", err)
	}
	plan, err := service.PlanProject(created.ManifestPath)
	if err != nil || !plan.Executable {
		t.Fatal(plan, err)
	}
	if _, err := service.ApplyProject(created.ManifestPath, plan); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(template, []byte(`{"name":"changed"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.PlanProject(created.ManifestPath); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("modified plugin accepted: %v", err)
	}
	if _, err := service.LockProject(created.ManifestPath, true, false); err == nil {
		t.Fatal("modified lock accepted")
	}
	oldBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	update, err := service.PreviewProjectLock(created.ManifestPath)
	if err != nil || !update.Found || update.Proposed.Plugins[0].ContentSHA256 == update.Existing.Plugins[0].ContentSHA256 {
		t.Fatalf("update preview: %+v %v", update, err)
	}
	stillOld, err := os.ReadFile(lockPath)
	if err != nil || !bytes.Equal(oldBytes, stillOld) {
		t.Fatal("update preview changed lock")
	}
	if err := service.ApplyProjectLockPreview(created.ManifestPath, update); err != nil {
		t.Fatal(err)
	}
	newBytes, err := os.ReadFile(lockPath)
	if err != nil || bytes.Equal(oldBytes, newBytes) {
		t.Fatal("confirmed update did not write lock")
	}
	if _, err := service.LockProject(created.ManifestPath, true, false); err != nil {
		t.Fatalf("updated lock invalid: %v", err)
	}
	if err := service.ApplyProjectLockPreview(created.ManifestPath, update); err == nil {
		t.Fatal("stale update accepted")
	}
}
