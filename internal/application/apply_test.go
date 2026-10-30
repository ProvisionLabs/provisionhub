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
	file, err := project.Save(filepath.Join(base, "demo"), project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "demo"}, Spec: project.Spec{Backend: &project.TechnologySelection{Technology: "nestjs"}}})
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

func TestCreateAndApplyProjectChecksPlanBeforeWriting(t *testing.T) {
	base := t.TempDir()
	folder := filepath.Join(base, "plugins", "nestjs", "templates")
	if err := os.MkdirAll(folder, 0o755); err != nil {
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
    destination: backend/package.json
`
	if err := os.WriteFile(filepath.Join(base, "plugins", "nestjs", "plugin.yaml"), []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	template := filepath.Join(folder, "package.json.tmpl")
	if err := os.WriteFile(template, []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	service, err := NewProjectServiceWithPlugins(filepath.Join(base, "plugins"))
	if err != nil {
		t.Fatal(err)
	}
	input := CreateProjectInput{Name: "demo", Destination: filepath.Join(base, "demo"), Backend: "nestjs"}
	manifest, _, err := service.ValidateInput(input)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := service.PlanManifest(manifest)
	if err != nil || !plan.Executable {
		t.Fatalf("reviewed plan: %v %+v", err, plan)
	}
	if err := os.WriteFile(template, []byte(`{"name":"changed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.CreateAndApplyProject(input, plan); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("stale plan: %v", err)
	}
	if _, err := os.Stat(input.Destination); !os.IsNotExist(err) {
		t.Fatalf("stale plan created destination: %v", err)
	}
	if err := os.WriteFile(template, []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	created, applied, err := service.CreateAndApplyProject(input, plan)
	if err != nil || len(applied.Created) != 1 {
		t.Fatalf("generation: %v, %+v", err, applied)
	}
	if _, err := os.Stat(created.ManifestPath); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(input.Destination, "backend", "package.json"))
	if err != nil || string(data) != `{"name":"demo"}` {
		t.Fatalf("generated package: %v %s", err, data)
	}
	if _, _, err := service.CreateAndApplyProject(input, plan); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("repeated generation: %v", err)
	}
}
