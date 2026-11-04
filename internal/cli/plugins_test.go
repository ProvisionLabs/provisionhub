package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

type cliRegistry struct{ index registry.Index }

func (s cliRegistry) Fetch(context.Context) (registry.Snapshot, error) {
	return registry.Snapshot{Index: s.index}, nil
}
func TestPluginCLI(t *testing.T) {
	directory := t.TempDir()
	os.WriteFile(filepath.Join(directory, "file"), []byte("hello"), 0o600)
	os.WriteFile(filepath.Join(directory, "plugin.yaml"), []byte(`apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nestjs
version: 1.1.0
technologyId: nestjs
category: backend
name: NestJS
description: test
capability: generate_backend
files:
  - type: copy
    source: file
    destination: backend/file
`), 0o600)
	data, _, err := plugins.Package(directory)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer server.Close()
	index := registry.Index{SchemaVersion: 1, Plugins: []registry.Plugin{{ID: "official/nestjs", ManifestID: "official-nestjs", Name: "NestJS", Publisher: "ProvisionLabs", Description: "test", Category: "backend", TechnologyID: "nestjs", Versions: []registry.Release{{Version: "1.1.0", MinimumCoreVersion: "0.1.0", Artifact: server.URL + "/releases/download/official-nestjs-v1.1.0/plugin.tar.gz", SHA256: plugins.Checksum(data)}}}}}
	manager := application.PluginManager{DataDirectory: t.TempDir(), Source: cliRegistry{index}, HTTP: server.Client()}
	run := func(args ...string) (string, error) {
		command := pluginCommand(manager)
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetErr(&output)
		command.SetIn(strings.NewReader(""))
		command.SetArgs(args)
		err := command.Execute()
		return output.String(), err
	}
	for _, args := range [][]string{{"search", "backend"}, {"info", "official/nestjs"}, {"install", "official/nestjs@1.1.0"}, {"list"}, {"update", "official/nestjs"}, {"updates"}} {
		out, err := run(args...)
		if err != nil || !strings.Contains(out, "1.1.0") && args[0] != "updates" {
			t.Fatalf("%v: %s %v", args, out, err)
		}
	}
	if _, err := run("remove", "official/nestjs"); err == nil {
		t.Fatal("unapproved removal")
	}
	if _, err := run("install", "official/nestjs@"); err == nil {
		t.Fatal("invalid version")
	}
	if _, err := run("remove", "official/nestjs", "--yes"); err != nil {
		t.Fatal(err)
	}
	if _, err := run("remove", "official/nestjs", "--yes"); err == nil {
		t.Fatal("missing removal")
	}
}
