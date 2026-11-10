package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/planning"
)

func TestConnectionFlagsAndGeneration(t *testing.T) {
	root := t.TempDir()
	provider := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: provider
version: 1.0.0
technologyId: nestjs
category: backend
name: NestJS
description: Provider fixture
capability: generate_backend
configuration:
  properties:
    port: {type: integer, default: 3000}
connections:
  provides:
    - id: service
      type: transport
      metadata:
        port: {configuration: port}
files:
  - {type: template, source: base, destination: package.json}
`
	consumer := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: consumer
version: 1.0.0
technologyId: nextjs
category: frontend
name: Next.js
description: Consumer fixture
capability: generate_frontend
connections:
  consumes:
    - id: remote
      type: transport
      requires:
        properties:
          port: {type: integer, required: true}
      configuration:
        properties:
          host: {type: string, format: host, required: true}
          enabled: {type: boolean, default: true}
      files:
        - {type: template, source: endpoint, destination: .env.local}
files:
  - {type: template, source: base, destination: package.json}
`
	for name, manifest := range map[string]string{"service": provider, "client": consumer} {
		folder := filepath.Join(root, name)
		if err := os.Mkdir(folder, 0755); err != nil {
			t.Fatal(err)
		}
		for filename, data := range map[string]string{"plugin.yaml": manifest, "base": `{"name":"{{componentName}}"}`, "endpoint": "ENDPOINT=http://{{connection.host|host}}:{{target.port}}\nENABLED={{connection.enabled}}\n"} {
			if err := os.WriteFile(filepath.Join(folder, filename), []byte(data), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	destination := filepath.Join(t.TempDir(), "connected project")
	base := []string{"--name", "demo", "--path", destination, "--plugins", root, "--component", "api-a:backend:nestjs", "--component", "api-b:backend:nestjs", "--component", "web:frontend:nextjs", "--component", "admin:frontend:nextjs", "--set", "api-a.port=4001", "--set", "api-b.port=4002"}
	connected := append(append([]string{}, base...), "--connect", "web:api-a:transport", "--connect", "admin:api-b:transport", "--connection-set", "web:api-a:transport.host=::1", "--connection-set", "admin:api-b:transport.host=127.0.0.1")
	output, err := executeCLI(append([]string{"project", "plan"}, connected...)...)
	if err != nil || !strings.Contains(output, "web -> api-a (transport)") || !strings.Contains(output, `"port":4002`) {
		t.Fatal(output, err)
	}
	jsonArgs := append(append([]string{"project", "plan"}, connected...), "--output", "json")
	raw, err := executeCLI(jsonArgs...)
	if err != nil {
		t.Fatal(raw, err)
	}
	var plan planning.Plan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil || !plan.Executable || len(plan.Connections) != 2 {
		t.Fatal(plan, err)
	}
	for range 5 {
		again, err := executeCLI(jsonArgs...)
		if err != nil || again != raw {
			t.Fatal("nondeterministic connection plan", err)
		}
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("planning wrote output")
	}
	if _, err := executeCLI(append([]string{"project", "apply"}, connected...)...); err == nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("unapproved generation wrote output")
	}
	if _, err := executeCLI(append(append([]string{"project", "apply"}, connected...), "--yes")...); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"web": "ENDPOINT=http://[::1]:4001", "admin": "ENDPOINT=http://127.0.0.1:4002"} {
		data, err := os.ReadFile(filepath.Join(destination, name, ".env.local"))
		if err != nil || !strings.Contains(string(data), want) {
			t.Fatal(string(data), err)
		}
	}
	for _, tc := range []struct {
		extra []string
		want  string
	}{
		{[]string{"--connect", "web:api-a"}, "source:target:type"},
		{[]string{"--connect", "web:absent:transport"}, "unknown component"},
		{[]string{"--connect", "web:web:transport"}, "itself"},
		{[]string{"--connect", "web:api-a:transport"}, "duplicate connection"},
		{[]string{"--connect", "web:api-b:transport"}, "supports one target"},
		{[]string{"--connection-set", "web:api-a:transport.enabled=yes"}, "true or false"},
		{[]string{"--connection-set", "web:api-a:transport.unknown=1"}, "unknown key"},
		{[]string{"--connection-set", "web:api-b:transport.host=localhost"}, "unknown connection"},
		{[]string{"--connection-set", "web:api-a:transport.host=localhost"}, "duplicate connection setting"},
	} {
		args := append(append(append([]string{"project", "plan"}, connected...), tc.extra...), "--output", "json")
		if _, err := executeCLI(args...); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatal(tc, err)
		}
	}
	for _, values := range [][]string{
		{"--connect", "web:api-a:transport"}, // required connection host
		{"--connect", "web:api-a:unsupported"},
		{"--connect", "web:api-a:transport", "--connection-set", "web:api-a:transport.host=http://localhost"},
	} {
		if _, err := executeCLI(append(append([]string{"project", "plan"}, base...), values...)...); err == nil {
			t.Fatal("accepted invalid connection values", values)
		}
	}
}
