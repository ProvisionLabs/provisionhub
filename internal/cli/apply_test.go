package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/planning"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func cliPluginFixture(t *testing.T) (string, string) {
	t.Helper()
	base := t.TempDir()
	manifest := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "demo"}, Spec: project.Spec{Backend: project.TechnologySelection{Technology: "nestjs"}}}
	file, err := project.Save(filepath.Join(base, "project café"), manifest)
	if err != nil {
		t.Fatal(err)
	}
	pluginDir := filepath.Join(base, "plugins")
	folder := filepath.Join(pluginDir, "nestjs", "templates")
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
    destination: package.json
`
	if err := os.WriteFile(filepath.Join(pluginDir, "nestjs", "plugin.yaml"), []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "package.json.tmpl"), []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return file, pluginDir
}

func TestPlanAndApplyCLIWithLocalPlugin(t *testing.T) {
	file, plugins := cliPluginFixture(t)
	out, err := executeCLI("project", "plan", "--file", file, "--plugins", plugins, "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	var plan planning.Plan
	if err := json.Unmarshal([]byte(out), &plan); err != nil || !plan.Executable {
		t.Fatalf("JSON plan: %v, %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(file), "package.json")); !os.IsNotExist(err) {
		t.Fatal("planning generated a file")
	}
	cmd := NewRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetIn(strings.NewReader("yes\n")) // A pipe must never be treated as a terminal.
	cmd.SetArgs([]string{"project", "apply", "--file", file, "--plugins", plugins})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "requires a terminal") {
		t.Fatalf("non-interactive approval = %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(file), "package.json")); !os.IsNotExist(err) {
		t.Fatal("unapproved apply generated a file")
	}
	out, err = executeCLI("project", "apply", "--file", file, "--plugins", plugins, "--yes")
	if err != nil || !strings.Contains(out, "Created 1 files") {
		t.Fatalf("apply: %v, %s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "package.json"))
	if err != nil || string(data) != `{"name":"demo"}` {
		t.Fatalf("output: %v %s", err, data)
	}
	if _, err := executeCLI("project", "apply", "--file", file, "--plugins", plugins, "--yes"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("overwrite = %v", err)
	}
}

func TestApplyCLIRejectsMissingGenerator(t *testing.T) {
	file, _ := cliPluginFixture(t)
	if _, err := executeCLI("project", "apply", "--file", file, "--yes"); err == nil || !strings.Contains(err.Error(), "not executable") {
		t.Fatalf("missing plugin = %v", err)
	}
}
