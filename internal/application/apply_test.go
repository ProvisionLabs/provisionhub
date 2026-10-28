package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestApplyProjectRechecksManifestAndPluginAfterReview(t *testing.T) {
	base := t.TempDir()
	file, err := project.Save(filepath.Join(base, "demo"), project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "demo"}, Spec: project.Spec{Backend: project.TechnologySelection{Technology: "nestjs"}}})
	if err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(base, "plugins", "nestjs")
	if err := os.MkdirAll(filepath.Join(folder, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	definition := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nestjs
version: 1.0.0
technologyId: nestjs
category: backend
name: NestJS
description: Minimal NestJS project
capability: generate_backend
files:
  - type: template
    source: templates/package.json.tmpl
    destination: package.json
`
	if err := os.WriteFile(filepath.Join(folder, "plugin.yaml"), []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	template := filepath.Join(folder, "templates", "package.json.tmpl")
	if err := os.WriteFile(template, []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	service, err := NewProjectServiceWithPlugins(filepath.Join(base, "plugins"))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := service.PlanProject(file)
	if err != nil || !plan.Executable {
		t.Fatalf("plan: %v %+v", err, plan)
	}
	if err := os.WriteFile(template, []byte(`{"name":"changed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplyProject(file, plan); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("stale apply: %v", err)
	}
	if err := os.WriteFile(template, []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "plugin.yaml"), []byte(strings.Replace(definition, "version: 1.0.0", "version: 1.0.1", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ApplyProject(file, plan); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("stale plugin manifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(file), "package.json")); !os.IsNotExist(err) {
		t.Fatal("stale apply wrote output")
	}
}
