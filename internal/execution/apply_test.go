package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func fixture(t *testing.T) (string, string, project.Manifest, plugins.Set, planning.Plan) {
	t.Helper()
	base := t.TempDir()
	destination := filepath.Join(base, "my app café")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	m := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "my-app"}, Spec: project.Spec{Backend: project.TechnologySelection{Technology: "nestjs"}}}
	if _, err := project.Save(destination, m); err != nil {
		t.Fatal(err)
	}
	pluginDir := filepath.Join(base, "plugins")
	if err := os.MkdirAll(filepath.Join(pluginDir, "nestjs", "templates", "src"), 0o755); err != nil {
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
  - type: template
    source: templates/src/main.ts.tmpl
    destination: src/main.ts
`
	folder := filepath.Join(pluginDir, "nestjs")
	if err := os.WriteFile(filepath.Join(folder, "plugin.yaml"), []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "templates", "package.json.tmpl"), []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "templates", "src", "main.ts.tmpl"), []byte("export const name = '{{projectName}}';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := plugins.LoadDirectory(pluginDir, catalog.Builtin())
	if err != nil {
		t.Fatal(err)
	}
	return destination, pluginDir, m, set, planning.BuildWithPlugins(m, catalog.Builtin(), set)
}

func TestApplyCreatesReviewedFilesWithoutCommands(t *testing.T) {
	destination, _, m, set, plan := fixture(t)
	result, err := Apply(plan, m, set, destination)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 2 {
		t.Fatalf("created: %+v", result)
	}
	data, err := os.ReadFile(filepath.Join(destination, "package.json"))
	if err != nil || string(data) != `{"name":"my-app"}` {
		t.Fatalf("package: %s, %v", data, err)
	}
	data, err = os.ReadFile(filepath.Join(destination, "src", "main.ts"))
	if err != nil || !strings.Contains(string(data), "my-app") {
		t.Fatalf("source: %s, %v", data, err)
	}
	entries, err := os.ReadDir(destination)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".provisionhub-stage-") {
			t.Fatal("staging directory left behind")
		}
	}
	if _, err := Apply(plan, m, set, destination); err == nil {
		t.Fatal("second apply overwrote files")
	}
}

func TestApplyRejectsConflictBeforePublishingAnyFile(t *testing.T) {
	destination, _, m, set, plan := fixture(t)
	path := filepath.Join(destination, "src", "main.ts")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(plan, m, set, destination); err == nil {
		t.Fatal("accepted conflict")
	}
	if _, err := os.Stat(filepath.Join(destination, "package.json")); !os.IsNotExist(err) {
		t.Fatalf("partial output: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "existing" {
		t.Fatal("overwrote existing file")
	}
}

func TestApplyRejectsStalePlanAndSymlinkedOutputParent(t *testing.T) {
	destination, pluginDir, m, set, plan := fixture(t)
	folder := filepath.Join(pluginDir, "nestjs", "templates")
	if err := os.WriteFile(filepath.Join(folder, "package.json.tmpl"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := plugins.LoadDirectory(pluginDir, catalog.Builtin())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(plan, m, changed, destination); err == nil {
		t.Fatal("accepted stale plugin content")
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(destination, "src")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Apply(plan, m, set, destination); err == nil {
		t.Fatal("accepted symlinked output parent")
	}
	if _, err := os.Stat(filepath.Join(outside, "main.ts")); !os.IsNotExist(err) {
		t.Fatal("wrote outside project")
	}
}
