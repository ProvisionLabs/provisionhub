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

func configurationService(t *testing.T) ProjectService {
	t.Helper()
	root := t.TempDir()
	folder := filepath.Join(root, "generator")
	if err := os.Mkdir(folder, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: configured
version: 1.0.0
technologyId: nestjs
category: backend
name: NestJS
description: Configuration fixture
capability: generate_backend
configuration:
  properties:
    title: {type: string, default: Hello}
    enabled: {type: boolean, default: true}
    port: {type: integer, default: 3000, minimum: 1, maximum: 65535}
files:
  - {type: template, source: content, destination: config.json}
`
	for name, content := range map[string]string{"plugin.yaml": manifest, "content": `{"name":"{{component.name}}","title":{{config.title|json}},"enabled":{{config.enabled}},"port":{{config.port}}}`} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROVISIONHUB_DATA_DIR", blocked)
	s, err := NewProjectServiceWithPlugins(root)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestConfiguredPlanningAndExecution(t *testing.T) {
	s := configurationService(t)
	destination := filepath.Join(t.TempDir(), "generated café")
	components := []project.Component{
		{Name: "api", Type: "backend", Technology: "nestjs", Configuration: configuration.Values{"enabled": false, "port": 4000}},
		{Name: "billing", Type: "backend", Technology: "nestjs"},
	}
	spec, err := s.ValidateComponentsInput("demo", destination, components)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := s.PlanSpecification(spec)
	if err != nil || !plan.Executable {
		t.Fatal(plan, err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("planning wrote output", err)
	}
	if len(spec.Spec.Components[0].Configuration) != 2 || spec.Spec.Components[1].Configuration != nil {
		t.Fatal("planner mutated input")
	}
	for range 10 {
		fresh, err := s.PlanSpecification(spec)
		if err != nil || !reflect.DeepEqual(plan, fresh) {
			t.Fatal(fresh, err)
		}
	}
	spec.Spec.Components[0].Configuration["enabled"] = true
	if _, err := s.ApplySpecification(spec, destination, plan); err == nil {
		t.Fatal("accepted stale config")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("stale plan wrote destination")
	}
	spec.Spec.Components[0].Configuration["enabled"] = false
	result, err := s.ApplySpecification(spec, destination, plan)
	if err != nil || len(result.Created) != 2 {
		t.Fatal(result, err)
	}
	for name, want := range map[string]string{"api": `"enabled":false,"port":4000`, "billing": `"enabled":true,"port":3000`} {
		data, err := os.ReadFile(filepath.Join(destination, name, "config.json"))
		if err != nil || !strings.Contains(string(data), want) {
			t.Fatal(string(data), err)
		}
	}
	if _, err := s.ApplySpecification(spec, destination, plan); err == nil {
		t.Fatal("overwrote output")
	}
	for _, name := range []string{"provisionhub.yaml", ".provisionhub"} {
		if _, err := os.Stat(filepath.Join(destination, name)); !os.IsNotExist(err) {
			t.Fatal("project metadata created")
		}
	}
	for _, values := range []configuration.Values{{"unknown": true}, {"port": "3000"}, {"port": 70000}} {
		spec.Spec.Components[0].Configuration = values
		invalid, err := s.PlanSpecification(spec)
		if err == nil || invalid.Executable || len(invalid.Operations) != 0 || !strings.Contains(err.Error(), `component "api"`) {
			t.Fatal(invalid, err)
		}
	}
}
