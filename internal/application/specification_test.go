package application

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestInMemoryGenerationWithoutProjectMetadataOrDatabase(t *testing.T) {
	base := t.TempDir()
	blocked := filepath.Join(base, "not-a-directory")
	if err := os.WriteFile(blocked, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROVISIONHUB_DATA_DIR", blocked)
	pluginRoot := filepath.Join(base, "plugins")
	for _, entry := range []struct{ name, display, technology, category, capability string }{
		{"nestjs", "NestJS", "nestjs", "backend", "generate_backend"},
		{"nextjs", "Next.js", "nextjs", "frontend", "generate_frontend"},
	} {
		directory := filepath.Join(pluginRoot, entry.name)
		if err := os.MkdirAll(filepath.Join(directory, "templates"), 0755); err != nil {
			t.Fatal(err)
		}
		definition := "apiVersion: provisionhub.dev/plugin/v1alpha1\nid: official-" + entry.name + "\nversion: 1.0.0\ntechnologyId: " + entry.technology + "\ncategory: " + entry.category + "\nname: " + entry.display + "\ndescription: local generator\ncapability: " + entry.capability + "\nfiles:\n  - type: template\n    source: templates/package.json.tmpl\n    destination: package.json\n"
		if err := os.WriteFile(filepath.Join(directory, "plugin.yaml"), []byte(definition), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "templates", "package.json.tmpl"), []byte(`{"name":"{{componentName}}"}`), 0644); err != nil {
			t.Fatal(err)
		}
	}
	service, err := NewProjectServiceWithPlugins(pluginRoot)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(base, "new project")
	spec, err := service.ValidateComponentsInput("demo", destination, []project.Component{
		{Name: "api", Type: "backend", Technology: "nestjs"},
		{Name: "billing", Type: "backend", Technology: "nestjs"},
		{Name: "web", Type: "frontend", Technology: "nextjs"},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := service.PlanSpecification(spec)
	if err != nil || !plan.Executable {
		t.Fatal(plan, err)
	}
	again, err := service.PlanSpecification(spec)
	if err != nil || !reflect.DeepEqual(plan, again) {
		t.Fatal("planning is not deterministic", err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("planning wrote output")
	}
	if err := service.PreflightSpecification(spec, destination, plan); err != nil {
		t.Fatal(err)
	}
	result, err := service.ApplySpecification(spec, destination, plan)
	if err != nil || len(result.Created) != 3 {
		t.Fatal(result, err)
	}
	for _, component := range []string{"api", "billing", "web"} {
		if _, err := os.Stat(filepath.Join(destination, component, "package.json")); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{project.ManifestName, filepath.Join(".provisionhub", "plugins.lock.yaml")} {
		if _, err := os.Stat(filepath.Join(destination, name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected project metadata %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(blocked, "provisionhub.db")); !os.IsNotExist(err) {
		t.Fatal("generation accessed SQLite")
	}
}
