package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComponentSettings(t *testing.T) {
	base := t.TempDir()
	folder := filepath.Join(base, "plugins", "generator")
	if err := os.MkdirAll(folder, 0755); err != nil {
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
    mode: {type: enum, options: [small, large], default: small}
files:
  - {type: template, source: content, destination: config.json}
`
	for name, content := range map[string]string{"plugin.yaml": manifest, "content": `{"title":{{config.title|json}},"enabled":{{config.enabled}},"port":{{config.port}},"mode":{{config.mode|json}}}`} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	destination := filepath.Join(base, "output")
	args := []string{"--name", "demo", "--path", destination, "--plugins", filepath.Dir(folder), "--component", "api:backend:nestjs", "--component", "billing:backend:nestjs"}
	settings := []string{"--set", "api.enabled=false", "--set", "api.port=4000", "--set", "api.mode=large", "--set", "api.title=true=a,b"}
	planArgs := append(append([]string{"project", "plan"}, args...), settings...)
	output, err := executeCLI(planArgs...)
	if err != nil || !strings.Contains(output, `"port":4000`) || !strings.Contains(output, `"port":3000`) {
		t.Fatal(output, err)
	}
	for range 5 {
		next, err := executeCLI(planArgs...)
		if err != nil || next != output {
			t.Fatal(next, err)
		}
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("planning wrote output")
	}
	applyArgs := append(append(append([]string{"project", "apply"}, args...), settings...), "--yes")
	if _, err := executeCLI(applyArgs...); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "api", "config.json"))
	if err != nil || !strings.Contains(string(data), `"title":"true=a,b","enabled":false,"port":4000,"mode":"large"`) {
		t.Fatal(string(data), err)
	}
	for _, tc := range []struct {
		settings []string
		want     string
	}{
		{[]string{"api.enabled=yes"}, "true or false"}, {[]string{"api.port=4.2"}, "decimal integer"}, {[]string{"api.port=0"}, "at least"},
		{[]string{"api.mode=other"}, "one of"}, {[]string{"other.port=3"}, "unknown component"}, {[]string{"api.unknown=x"}, "unknown key"},
		{[]string{"api.enabled=true", "api.enabled=true"}, "duplicate setting"}, {[]string{"api.port=3", "api.port=4"}, "duplicate setting"}, {[]string{"api.port"}, "component.key=value"},
	} {
		invalid := append([]string{"project", "plan"}, args...)
		for _, setting := range tc.settings {
			invalid = append(invalid, "--set", setting)
		}
		if _, err := executeCLI(invalid...); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatal(tc, err)
		}
	}
}
