package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
)

const testPlugin = `apiVersion: provisionhub.dev/plugin/v1alpha1
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

func writeTestPlugin(t *testing.T, root, name, manifest string) string {
	t.Helper()
	folder := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Join(folder, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "templates", "package.json.tmpl"), []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return folder
}

func TestLoadLocalPlugin(t *testing.T) {
	root := t.TempDir()
	writeTestPlugin(t, root, "nestjs", testPlugin)
	set, err := LoadDirectory(root, catalog.Builtin())
	if err != nil {
		t.Fatal(err)
	}
	m, ok := set.ForTechnology("nestjs")
	if !ok || m.ID != "official-nestjs" || len(m.Files) != 1 || m.Files[0].Digest == [32]byte{} {
		t.Fatalf("loaded plugin = %+v", m)
	}
	if _, ok := set.ForTechnology("nextjs"); ok {
		t.Fatal("invented generator")
	}
}

func TestLoadFrontendPlugin(t *testing.T) {
	root := t.TempDir()
	frontend := strings.NewReplacer(
		"official-nestjs", "official-nextjs", "technologyId: nestjs", "technologyId: nextjs",
		"category: backend", "category: frontend", "name: NestJS", "name: Next.js",
		"generate_backend", "generate_frontend",
	).Replace(testPlugin)
	writeTestPlugin(t, root, "nextjs", frontend)
	set, err := LoadDirectory(root, catalog.Builtin())
	if err != nil {
		t.Fatal(err)
	}
	if plugin, ok := set.ForTechnology("nextjs"); !ok || plugin.Capability != "generate_frontend" {
		t.Fatalf("frontend plugin = %+v, %v", plugin, ok)
	}
	for _, invalid := range []string{
		strings.Replace(frontend, "generate_frontend", "generate_backend", 1),
		strings.Replace(frontend, "category: frontend", "category: database", 1),
	} {
		writeTestPlugin(t, root, "nextjs", invalid)
		if _, err := LoadDirectory(root, catalog.Builtin()); err == nil {
			t.Fatal("accepted mismatched frontend capability or category")
		}
	}
}

func TestRejectInvalidPluginDefinitions(t *testing.T) {
	tests := []struct{ name, manifest, content string }{
		{"schema", strings.Replace(testPlugin, APIVersion, "old", 1), ""},
		{"version", strings.Replace(testPlugin, "version: 1.0.0", "version: unknown", 1), ""},
		{"unknown technology", strings.Replace(testPlugin, "technologyId: nestjs", "technologyId: unknown", 1), ""},
		{"wrong category", strings.Replace(testPlugin, "category: backend", "category: frontend", 1), ""},
		{"traversal source", strings.Replace(testPlugin, "templates/package.json.tmpl", "../secret", 1), ""},
		{"traversal destination", strings.Replace(testPlugin, "destination: package.json", "destination: ../escape", 1), ""},
		{"absolute destination", strings.Replace(testPlugin, "destination: package.json", "destination: /escape", 1), ""},
		{"reserved destination", strings.Replace(testPlugin, "destination: package.json", "destination: CON.txt", 1), ""},
		{"manifest destination", strings.Replace(testPlugin, "destination: package.json", "destination: ProvisionHub.yaml", 1), ""},
		{"command field", testPlugin + "command: npm install\n", ""},
		{"duplicate outputs", testPlugin + "  - type: copy\n    source: templates/package.json.tmpl\n    destination: package.json\n", ""},
		{"unknown field", testPlugin + "surprise: true\n", ""},
		{"unsupported variable", testPlugin, "{{secret}}"},
		{"unclosed variable", testPlugin, "{{projectName"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			folder := writeTestPlugin(t, root, "nestjs", tc.manifest)
			if tc.content != "" {
				if err := os.WriteFile(filepath.Join(folder, "templates", "package.json.tmpl"), []byte(tc.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := LoadDirectory(root, catalog.Builtin()); err == nil {
				t.Fatal("accepted invalid plugin")
			}
		})
	}
}

func TestRejectDuplicateAndSymlinkedPlugins(t *testing.T) {
	root := t.TempDir()
	first := writeTestPlugin(t, root, "one", testPlugin)
	writeTestPlugin(t, root, "two", testPlugin)
	if _, err := LoadDirectory(root, catalog.Builtin()); err == nil || !strings.Contains(err.Error(), "duplicate plugin ID") {
		t.Fatalf("duplicate ID: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "two", "plugin.yaml"), []byte(strings.Replace(testPlugin, "id: official-nestjs", "id: alternate-nestjs", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDirectory(root, catalog.Builtin()); err == nil || !strings.Contains(err.Error(), "duplicate generator") {
		t.Fatalf("duplicate generator: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(root, "two")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(first, "templates", "package.json.tmpl")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(first, "plugin.yaml"), filepath.Join(first, "templates", "package.json.tmpl")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := LoadDirectory(root, catalog.Builtin()); err == nil {
		t.Fatal("accepted symlinked template")
	}
}
