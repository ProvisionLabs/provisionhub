package planning

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestCombinedFrontendAndBackendPlan(t *testing.T) {
	root := t.TempDir()
	for _, item := range []struct{ directory, manifest, output string }{
		{"nestjs", `apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nestjs
version: 1.0.0
technologyId: nestjs
category: backend
name: NestJS
description: Backend
capability: generate_backend
files:
  - type: template
    source: templates/package.json.tmpl
    destination: backend/package.json
`, "backend/package.json"},
		{"nextjs", `apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nextjs
version: 1.0.0
technologyId: nextjs
category: frontend
name: Next.js
description: Frontend
capability: generate_frontend
files:
  - type: template
    source: templates/package.json.tmpl
    destination: frontend/package.json
`, "frontend/package.json"},
	} {
		folder := filepath.Join(root, item.directory, "templates")
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, item.directory, "plugin.yaml"), []byte(item.manifest), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "package.json.tmpl"), []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	set, err := plugins.LoadDirectory(root, catalog.Builtin())
	if err != nil {
		t.Fatal(err)
	}
	manifest := testManifest()
	manifest.Spec.Frontend = &project.TechnologySelection{Technology: "nextjs"}
	plan := BuildWithPlugins(manifest, catalog.Builtin(), set)
	if !plan.Executable || len(plan.Diagnostics) != 0 || len(plan.Operations) != 4 {
		t.Fatalf("combined plan = %+v", plan)
	}
	if !reflect.DeepEqual(plan, BuildWithPlugins(manifest, catalog.Builtin(), set)) {
		t.Fatal("nondeterministic combined plan")
	}
	for _, operation := range plan.Operations {
		if operation.Status != OperationSupported {
			t.Fatalf("unsupported operation: %+v", operation)
		}
	}
	// Cross-plugin collisions must be rejected even though each plugin is valid by itself.
	definition := filepath.Join(root, "nextjs", "plugin.yaml")
	data, err := os.ReadFile(definition)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(definition, []byte(strings.Replace(string(data), "frontend/package.json", "backend/package.json", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err = plugins.LoadDirectory(root, catalog.Builtin())
	if err != nil {
		t.Fatal(err)
	}
	plan = BuildWithPlugins(manifest, catalog.Builtin(), set)
	if plan.Executable || plan.Status != StatusInvalid {
		t.Fatalf("accepted conflicting output: %+v", plan)
	}
}
