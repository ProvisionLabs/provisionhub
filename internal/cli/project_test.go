package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func executeCLI(args ...string) (string, error) {
	command := NewRootCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs(args)
	err := command.Execute()
	return output.String(), err
}

func TestProjectInitCommand(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "new project")
	args := []string{"project", "init", "--name", "my-app", "--path", destination, "--backend", "nestjs", "--frontend", "nextjs", "--database", "postgresql"}
	output, err := executeCLI(args...)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(destination, project.ManifestName)
	if strings.TrimSpace(output) != path {
		t.Fatalf("output = %q, want path %q", output, path)
	}
	manifest, err := project.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Metadata.Name != "my-app" || manifest.Spec.Frontend.Technology != "nextjs" {
		t.Fatalf("manifest = %+v", manifest)
	}
	if _, err := executeCLI(args...); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing manifest error = %v", err)
	}
}

func TestProjectInitRejectsInvalidInputs(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"missing required", []string{"project", "init", "--backend", "nestjs"}, "required flag"},
		{"unknown technology", []string{"project", "init", "--name", "demo", "--path", filepath.Join(t.TempDir(), "demo"), "--backend", "missing"}, "unknown backend"},
		{"invalid name", []string{"project", "init", "--name", "Bad Name", "--path", filepath.Join(t.TempDir(), "demo"), "--backend", "nestjs"}, "project name"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := executeCLI(test.args...)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestProjectInitOptionalCategories(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "backend-only")
	_, err := executeCLI("project", "init", "--name", "backend-only", "--path", destination, "--backend", "go")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, project.ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "frontend:") || strings.Contains(string(data), "database:") {
		t.Fatalf("optional categories should be omitted: %s", data)
	}
}

func TestProjectInitFromMultiComponentManifest(t *testing.T) {
	base := t.TempDir()
	sourceDir := filepath.Join(base, "source")
	manifest := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind,
		Metadata: project.Metadata{Name: "multi-service-demo"}, Spec: project.Spec{Components: []project.Component{
			{Name: "web", Type: "frontend", Technology: "nextjs"},
			{Name: "api", Type: "backend", Technology: "nestjs"},
			{Name: "billing", Type: "backend", Technology: "nestjs"},
		}}}
	source, err := project.Save(sourceDir, manifest)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(base, "output")
	output, err := executeCLI("project", "init", "--from", source, "--path", destination)
	if err != nil {
		t.Fatal(err)
	}
	created := filepath.Join(destination, project.ManifestName)
	if strings.TrimSpace(output) != created {
		t.Fatalf("output = %q", output)
	}
	copy, err := project.Load(created)
	if err != nil || len(copy.Spec.Components) != 3 || copy.Metadata.Name != "multi-service-demo" {
		t.Fatalf("copied manifest = %+v, %v", copy, err)
	}
	plan, err := executeCLI("project", "plan", "--file", created)
	if err != nil || !strings.Contains(plan, "component/api") || !strings.Contains(plan, "component/billing") || !strings.Contains(plan, "component/web") {
		t.Fatalf("component plan = %s, %v", plan, err)
	}
	if _, err := executeCLI("project", "init", "--from", source, "--path", destination); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("overwritten manifest: %v", err)
	}
	if _, err := executeCLI("project", "init", "--from", source, "--path", filepath.Join(base, "conflict"), "--backend", "nestjs"); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("mixed flags accepted: %v", err)
	}
}

func TestTechnologiesListCommand(t *testing.T) {
	output, err := executeCLI("technologies", "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"backend\tgo\tGo", "backend\tnestjs\tNestJS", "frontend\tnextjs\tNext.js", "database\tpostgresql\tPostgreSQL"} {
		if !strings.Contains(output, line) {
			t.Errorf("catalog output missing %q: %s", line, output)
		}
	}
}
