package planning

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func testLocalPlugins(t *testing.T) plugins.Set {
	t.Helper()
	root := t.TempDir()
	folder := filepath.Join(root, "nestjs")
	if err := os.MkdirAll(filepath.Join(folder, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nestjs
version: 1.0.0
technologyId: nestjs
category: backend
name: NestJS
description: Backend generator
capability: generate_backend
files:
  - type: template
    source: templates/package.json.tmpl
    destination: package.json
`
	if err := os.WriteFile(filepath.Join(folder, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "templates", "package.json.tmpl"), []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := plugins.LoadDirectory(root, catalog.Builtin())
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestConcretePlanAndMixedUnavailableCapabilities(t *testing.T) {
	set := testLocalPlugins(t)
	manifest := testManifest()
	plan := BuildWithPlugins(manifest, catalog.Builtin(), set)
	if plan.Status != StatusPlanned || !plan.Executable || len(plan.Operations) != 2 {
		t.Fatalf("concrete plan = %+v", plan)
	}
	if plan.Operations[0].ID != "backend:nestjs" || plan.Operations[1].ID != "file:official-nestjs:package.json" || plan.Operations[1].SourceDigest == "" {
		t.Fatalf("operations = %+v", plan.Operations)
	}
	for range 10 {
		if again := BuildWithPlugins(manifest, catalog.Builtin(), set); !reflect.DeepEqual(plan, again) {
			t.Fatal("non-deterministic concrete plan")
		}
	}
	manifest.Spec.Frontend = &project.TechnologySelection{Technology: "nextjs"}
	mixed := BuildWithPlugins(manifest, catalog.Builtin(), set)
	if mixed.Executable || len(mixed.Diagnostics) != 1 || len(mixed.Operations) != 3 {
		t.Fatalf("mixed plan = %+v", mixed)
	}
	without := Build(manifest, catalog.Builtin())
	if without.Executable || len(without.Diagnostics) != 2 {
		t.Fatalf("conceptual regression = %+v", without)
	}
}
