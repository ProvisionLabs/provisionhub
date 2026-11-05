package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/application"
)

func executeCLI(args ...string) (string, error) {
	return executeCLIWithDatabase(testDatabasePath, args...)
}

func executeCLIWithDatabase(databasePath string, args ...string) (string, error) {
	settings := application.NewLocalSettings()
	settings.Path = databasePath
	command := newRootCommand(settings)
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs(args)
	err := command.Execute()
	return output.String(), err
}

func TestProjectPlanAndApplyWithoutMetadata(t *testing.T) {
	base := t.TempDir()
	blocked := filepath.Join(base, "blocked-data")
	if err := os.WriteFile(blocked, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROVISIONHUB_DATA_DIR", blocked)
	plugins := filepath.Join(base, "plugins", "nestjs")
	if err := os.MkdirAll(filepath.Join(plugins, "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	definition := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nestjs
version: 1.0.0
technologyId: nestjs
category: backend
name: NestJS
description: Local generator
capability: generate_backend
files:
  - type: template
    source: templates/package.json.tmpl
    destination: package.json
`
	if err := os.WriteFile(filepath.Join(plugins, "plugin.yaml"), []byte(definition), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plugins, "templates", "package.json.tmpl"), []byte(`{"name":"{{componentName}}"}`), 0644); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(base, "generated project")
	flags := []string{"--name", "demo", "--path", destination, "--component", "api:backend:nestjs", "--component", "billing:backend:nestjs", "--plugins", filepath.Dir(plugins)}
	output, err := executeCLI(append([]string{"project", "plan"}, flags...)...)
	if err != nil || !strings.Contains(output, "component/api") || !strings.Contains(output, "component/billing") {
		t.Fatal(output, err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("planning wrote output")
	}
	if _, err := executeCLI(append([]string{"project", "apply"}, flags...)...); err == nil {
		t.Fatal("apply skipped approval", err)
	}
	output, err = executeCLI(append(append([]string{"project", "apply"}, flags...), "--yes")...)
	if err != nil || !strings.Contains(output, "Created 2 files") {
		t.Fatal(output, err)
	}
	for _, name := range []string{"api", "billing"} {
		if _, err := os.Stat(filepath.Join(destination, name, "package.json")); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"provisionhub.yaml", filepath.Join(".provisionhub", "plugins.lock.yaml")} {
		if _, err := os.Stat(filepath.Join(destination, name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected metadata %s: %v", name, err)
		}
	}
}

func TestProjectFlagsValidateComponents(t *testing.T) {
	for _, tc := range []struct{ component, want string }{
		{"api:nestjs", "name:type:technology"},
		{"api:backend:unknown", "unknown backend"},
		{"Bad:backend:nestjs", "component name"},
	} {
		_, err := executeCLI("project", "plan", "--name", "demo", "--path", filepath.Join(t.TempDir(), "out"), "--component", tc.component)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%q: %v", tc.component, err)
		}
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
