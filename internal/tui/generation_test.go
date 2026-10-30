package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
)

func generatorModel(t *testing.T) (*Model, string) {
	t.Helper()
	base := t.TempDir()
	folder := filepath.Join(base, "plugins", "nestjs", "templates")
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
	if err := os.WriteFile(filepath.Join(base, "plugins", "nestjs", "plugin.yaml"), []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "package.json.tmpl"), []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := NewModelWithPlugins(application.Info(), filepath.Join(base, "plugins"))
	if err != nil {
		t.Fatal(err)
	}
	return m, filepath.Join(base, "project with spaces")
}

func reviewGeneratorWizard(t *testing.T, m *Model, destination string) {
	t.Helper()
	openWizard(m, "my-app", destination)
	press(m, tea.KeyDown, "") // NestJS backend.
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "") // No frontend.
	press(m, tea.KeyEnter, "") // No database.
	if m.wizard.step != reviewStep {
		t.Fatalf("step %v: %s", m.wizard.step, m.wizard.err)
	}
	if !m.wizard.plan.Executable {
		t.Fatalf("generator unavailable: %+v", m.wizard.plan)
	}
}

func TestTUIReviewGeneratesProjectAfterExplicitEnter(t *testing.T) {
	m, destination := generatorModel(t)
	reviewGeneratorWizard(t, m, destination)
	output := filepath.Join(destination, "package.json")
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("review generated source")
	}
	if !strings.Contains(m.View().Content, "Generate Project") {
		t.Fatal("review does not label the generation action")
	}
	if cmd := press(m, 'p', "p"); cmd != nil || !m.wizard.preview || !m.wizard.plan.Executable {
		t.Fatalf("preview: %v %+v", cmd, m.wizard.plan)
	}
	if !strings.Contains(m.View().Content, "package.json") {
		t.Fatal("preview omits concrete target")
	}
	m.Update(tea.WindowSizeMsg{Width: 17, Height: 5})
	for _, line := range strings.Split(m.View().Content, "\n") {
		if lipgloss.Width(line) > 17 {
			t.Fatalf("narrow preview overflow: %q", line)
		}
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	press(m, tea.KeyEnter, "") // Return from read-only preview.
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("preview generated source")
	}
	press(m, tea.KeyEscape, "") // Return to selections without writing.
	if m.wizard.step != databaseStep {
		t.Fatal("review cancellation lost navigation")
	}
	press(m, tea.KeyEnter, "")
	cmd := press(m, tea.KeyEnter, "") // Explicit Generate Project action.
	if cmd == nil || !m.wizard.applying {
		t.Fatal("generation did not start from review")
	}
	if _, quit := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}); quit != nil {
		t.Fatal("quit interrupted in-progress publication")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("confirmation wrote before command")
	}
	m.Update(cmd())
	if m.wizard.step != resultStep || m.wizard.preview || m.wizard.applying || len(m.wizard.applyResult.Created) != 1 {
		t.Fatalf("apply result: %+v", m.wizard)
	}
	if !strings.Contains(m.View().Content, "Project generated successfully") {
		t.Fatal("success screen did not report generation")
	}
	if data, err := os.ReadFile(output); err != nil || !strings.Contains(string(data), "my-app") {
		t.Fatalf("generated file: %v %s", err, data)
	}
}

func TestTUIApplyFailureKeepsManifestAndAllowsReviewRetry(t *testing.T) {
	m, destination := generatorModel(t)
	reviewGeneratorWizard(t, m, destination)
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "package.json"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := press(m, tea.KeyEnter, "")
	if cmd == nil {
		t.Fatal("generation command missing")
	}
	m.Update(cmd())
	if m.wizard.step != resultStep || !strings.Contains(m.wizard.err, "already exists") || !strings.Contains(strings.Join(m.wizard.lines(m.service, 64), "\n"), "generation incomplete") {
		t.Fatalf("failure state: step %v, error %q", m.wizard.step, m.wizard.err)
	}
	if data, err := os.ReadFile(filepath.Join(destination, "package.json")); err != nil || string(data) != "existing" {
		t.Fatalf("conflicting output changed: %v %s", err, data)
	}
	if _, err := os.Stat(m.wizard.result.ManifestPath); err != nil {
		t.Fatal("manifest lost")
	}
	if err := os.Remove(filepath.Join(destination, "package.json")); err != nil {
		t.Fatal(err)
	}
	press(m, 'p', "p")
	if !m.wizard.preview {
		t.Fatal("cannot review after failure")
	}
	press(m, 'g', "g")
	if !m.wizard.confirm {
		t.Fatal("retry did not request confirmation")
	}
	cmd = press(m, 'y', "y")
	if cmd == nil {
		t.Fatal("retry command missing")
	}
	m.Update(cmd())
	if m.wizard.err != "" || len(m.wizard.applyResult.Created) != 1 {
		t.Fatalf("retry failed: %+v", m.wizard)
	}
}

func TestTUICombinedFrontendGenerationFromReview(t *testing.T) {
	m, destination := generatorModel(t)
	plugin := filepath.Join(filepath.Dir(destination), "plugins", "nextjs")
	if err := os.MkdirAll(filepath.Join(plugin, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	definition := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nextjs
version: 1.0.0
technologyId: nextjs
category: frontend
name: Next.js
description: Minimal Next.js frontend
capability: generate_frontend
files:
  - type: template
    source: templates/package.json.tmpl
    destination: frontend/package.json
`
	if err := os.WriteFile(filepath.Join(plugin, "plugin.yaml"), []byte(definition), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plugin, "templates", "package.json.tmpl"), []byte(`{"name":"{{projectName}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	openWizard(m, "my-app", destination)
	press(m, tea.KeyDown, "") // NestJS backend.
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyDown, "") // Next.js frontend.
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "") // No database.
	if m.wizard.step != reviewStep {
		t.Fatalf("review step: %v, %s", m.wizard.step, m.wizard.err)
	}
	press(m, 'p', "p")
	if !m.wizard.preview || !m.wizard.plan.Executable {
		t.Fatalf("combined preview: %+v", m.wizard.plan)
	}
	preview := strings.Join(m.wizard.planLines(80), "\n")
	for _, part := range []string{"frontend:nextjs", "official-nextjs", "frontend/package.json", "backend:nestjs"} {
		if !strings.Contains(preview, part) {
			t.Fatalf("preview missing %q", part)
		}
	}
	press(m, tea.KeyEnter, "") // Close read-only preview.
	if _, err := os.Stat(filepath.Join(destination, "frontend", "package.json")); !os.IsNotExist(err) {
		t.Fatal("preview generated frontend")
	}
	cmd := press(m, tea.KeyEnter, "")
	if cmd == nil || !m.wizard.applying {
		t.Fatal("review did not start combined generation")
	}
	m.Update(cmd())
	if m.wizard.step != resultStep || len(m.wizard.applyResult.Created) != 2 {
		t.Fatalf("combined result: %+v", m.wizard)
	}
	for _, target := range []string{"package.json", "frontend/package.json"} {
		data, err := os.ReadFile(filepath.Join(destination, target))
		if err != nil || !strings.Contains(string(data), "my-app") {
			t.Fatalf("generated %q: %v %s", target, err, data)
		}
	}
}
