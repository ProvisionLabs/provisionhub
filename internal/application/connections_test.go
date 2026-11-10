package application

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func connectionService(t *testing.T) (ProjectService, string) {
	t.Helper()
	root := t.TempDir()
	provider := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: service-fixture
version: 1.0.0
technologyId: nestjs
category: backend
name: NestJS
description: Generic provider fixture
capability: generate_backend
configuration:
  properties:
    port: {type: integer, default: 3000}
connections:
  provides:
    - id: endpoint
      type: transport
      metadata:
        port: {configuration: port}
files:
  - {type: template, source: base, destination: package.json}
`
	consumer := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: client-fixture
version: 1.0.0
technologyId: nextjs
category: frontend
name: Next.js
description: Generic consumer fixture
capability: generate_frontend
connections:
  consumes:
    - id: remote
      type: transport
      description: Consume a remote endpoint
      requires:
        properties:
          port: {type: integer, required: true}
      configuration:
        properties:
          host: {type: string, format: host, required: true}
          enabled: {type: boolean, default: true}
          optional: {type: string}
      files:
        - {type: template, source: endpoint, destination: endpoint.json}
files:
  - {type: template, source: base, destination: package.json}
`
	for name, manifest := range map[string]string{"nestjs": provider, "nextjs": consumer} {
		folder := filepath.Join(root, name)
		if err := os.Mkdir(folder, 0755); err != nil {
			t.Fatal(err)
		}
		for file, content := range map[string]string{"plugin.yaml": manifest, "base": `{"name":"{{componentName}}"}`} {
			if err := os.WriteFile(filepath.Join(folder, file), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(root, "nextjs", "endpoint"), []byte(`{"source":"{{source.name}}","target":"{{target.name}}","host":{{connection.host|json}},"port":{{target.port}},"enabled":{{connection.enabled}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(t.TempDir(), "blocked-data")
	if err := os.WriteFile(blocked, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROVISIONHUB_DATA_DIR", blocked)
	service, err := NewProjectServiceWithPlugins(root)
	if err != nil {
		t.Fatal(err)
	}
	return service, root
}

func connectionSpec(t *testing.T, s ProjectService, destination string) project.Specification {
	t.Helper()
	components := []project.Component{{Name: "api-a", Type: "backend", Technology: "nestjs", Configuration: configuration.Values{"port": 4001}}, {Name: "api-b", Type: "backend", Technology: "nestjs", Configuration: configuration.Values{"port": 4002}}, {Name: "web", Type: "frontend", Technology: "nextjs"}, {Name: "admin", Type: "frontend", Technology: "nextjs"}}
	spec, err := s.ValidateComponentsInput("demo", destination, components)
	if err != nil {
		t.Fatal(err)
	}
	spec.Spec.Connections = []project.Connection{{From: "web", To: "api-a", Type: "transport", Configuration: configuration.Values{"host": "127.0.0.1"}}, {From: "admin", To: "api-a", Type: "transport", Configuration: configuration.Values{"host": "127.0.0.1"}}}
	return spec
}

func TestConnectionPlanAndExecutionFreshness(t *testing.T) {
	s, root := connectionService(t)
	destination := filepath.Join(t.TempDir(), "connected café")
	spec := connectionSpec(t, s, destination)
	plan, err := s.PlanSpecification(spec)
	if err != nil || !plan.Executable || len(plan.Connections) != 2 {
		t.Fatal(plan, err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("planning wrote output")
	}
	if plan.Connections[0].Metadata["port"] != int64(4001) || len(spec.Spec.Connections[0].Configuration) != 1 {
		t.Fatal("resolution leaked", plan.Connections)
	}
	for range 10 {
		fresh, err := s.PlanSpecification(spec)
		if err != nil || !reflect.DeepEqual(plan, fresh) {
			t.Fatal(fresh, err)
		}
	}
	positions := map[string]int{}
	for i, op := range plan.Operations {
		positions[op.ID] = i
		for _, dependency := range op.Dependencies {
			if _, ok := positions[dependency]; !ok {
				t.Fatal("dependency not before operation", op)
			}
		}
	}
	spec.Spec.Connections[0].To = "api-b"
	if _, err := s.ApplySpecification(spec, destination, plan); err == nil {
		t.Fatal("accepted changed target")
	}
	spec.Spec.Connections[0].To = "api-a"
	spec.Spec.Connections[0].Configuration["host"] = "other.example"
	if _, err := s.ApplySpecification(spec, destination, plan); err == nil {
		t.Fatal("accepted changed connection configuration")
	}
	spec.Spec.Connections[0].Configuration["host"] = "127.0.0.1"
	spec.Spec.Components[0].Configuration["port"] = 5000
	if _, err := s.ApplySpecification(spec, destination, plan); err == nil {
		t.Fatal("accepted changed target metadata")
	}
	spec.Spec.Components[0].Configuration["port"] = 4001
	result, err := s.ApplySpecification(spec, destination, plan)
	if err != nil || len(result.Created) != 6 {
		t.Fatal(result, err)
	}
	for _, name := range []string{"web", "admin"} {
		data, err := os.ReadFile(filepath.Join(destination, name, "endpoint.json"))
		if err != nil || !strings.Contains(string(data), `"target":"api-a"`) || !strings.Contains(string(data), `"port":4001,"enabled":true`) {
			t.Fatal(string(data), err)
		}
	}
	if _, err := s.ApplySpecification(spec, destination, plan); err == nil {
		t.Fatal("overwrote connection files")
	}
	spec.Spec.Connections[1].To = "api-b"
	distinct, err := s.PlanSpecification(spec)
	if err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(t.TempDir(), "distinct")
	if _, err := s.ApplySpecification(spec, second, distinct); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(second, "admin", "endpoint.json"))
	if err != nil || !strings.Contains(string(data), `"target":"api-b"`) || !strings.Contains(string(data), `"port":4002`) {
		t.Fatal(string(data), err)
	}
	// Missing optional references are rejected before any generation operations.
	if err := os.WriteFile(filepath.Join(root, "nextjs", "endpoint"), []byte(`{{connection.optional|json}}`), 0644); err != nil {
		t.Fatal(err)
	}
	invalid, err := s.PlanSpecification(spec)
	if err == nil || invalid.Executable || len(invalid.Operations) != 0 || !strings.Contains(err.Error(), "has no value") {
		t.Fatal(invalid, err)
	}
}

func TestConnectionOutputConflictPublishesNothing(t *testing.T) {
	s, _ := connectionService(t)
	destination := t.TempDir()
	if err := os.Mkdir(filepath.Join(destination, "web"), 0755); err != nil {
		t.Fatal(err)
	}
	conflict := filepath.Join(destination, "web", "endpoint.json")
	if err := os.WriteFile(conflict, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	spec := connectionSpec(t, s, destination)
	plan, err := s.PlanSpecification(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApplySpecification(spec, destination, plan); err == nil {
		t.Fatal("ignored connection output conflict")
	}
	if _, err := os.Stat(filepath.Join(destination, "api-a")); !os.IsNotExist(err) {
		t.Fatal("published partial component")
	}
	data, err := os.ReadFile(conflict)
	if err != nil || string(data) != "keep" {
		t.Fatal(string(data), err)
	}
}
