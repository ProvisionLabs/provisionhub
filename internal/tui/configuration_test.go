package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func configuredModel(t *testing.T) (*Model, string) {
	t.Helper()
	m, destination := generatorModel(t)
	root := t.TempDir()
	folder := filepath.Join(root, "fixture")
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
    enabled: {type: boolean, label: Enable feature, description: Enable the starter feature., default: true}
    mode: {type: enum, options: [small, large], default: small}
    port: {type: integer, default: 3000, minimum: 1, maximum: 65535}
    title: {type: string, default: Hello, minLength: 1}
files:
  - {type: template, source: content, destination: config.json}
`
	for name, content := range map[string]string{"plugin.yaml": manifest, "content": `{"enabled":{{config.enabled}},"mode":{{config.mode|json}},"port":{{config.port}},"title":{{config.title|json}}}`} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	service, err := application.NewProjectServiceWithPlugins(root)
	if err != nil {
		t.Fatal(err)
	}
	m.service = service
	return m, destination
}

func startConfiguredComponent(t *testing.T, m *Model, name string) {
	t.Helper()
	press(m, 'a', "a")
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyDown, "")
	press(m, tea.KeyEnter, "")
	typeText(m, name)
	press(m, tea.KeyEnter, "")
	if m.wizard.step != componentConfigurationStep {
		t.Fatal(m.wizard.step, m.wizard.err)
	}
}

func TestWizardDynamicConfigurationEditReviewAndGenerate(t *testing.T) {
	m, destination := configuredModel(t)
	openWizard(m, "demo", destination)
	startConfiguredComponent(t, m, "api")
	view := m.View().Content
	if !strings.Contains(view, "Enable feature") || !strings.Contains(view, "Enable the starter feature.") || !strings.Contains(view, "Default: true") {
		t.Fatal(view)
	}
	press(m, tea.KeySpace, "") // true -> false
	press(m, tea.KeyEnter, "") // enum
	press(m, tea.KeyRight, "") // small -> large
	press(m, tea.KeyEnter, "") // integer
	m.wizard.configurationInput.SetValue("70000")
	m.wizard.configurationRaw["port"] = "70000"
	press(m, tea.KeyEnter, "")
	if !strings.Contains(m.wizard.err, "at most") || m.wizard.configurationCursor != 2 {
		t.Fatal(m.wizard.err)
	}
	m.Update(tea.WindowSizeMsg{Width: 20, Height: 5})
	// The cursor adds ANSI styling around the final digit.
	if !strings.Contains(m.View().Content, "7000") || !strings.Contains(m.View().Content, "Error:") {
		t.Fatal("compact configuration hid input/error", m.View().Content)
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.wizard.configurationInput.SetValue("4000")
	m.wizard.configurationRaw["port"] = "4000"
	press(m, tea.KeyEnter, "")  // string
	press(m, tea.KeyEscape, "") // name
	press(m, tea.KeyEnter, "")  // return to the same draft
	if m.wizard.configurationInput.Value() != "Hello" || m.wizard.configurationRaw["port"] != "4000" {
		t.Fatal("back lost draft")
	}
	typeText(m, "q") // text q must not quit the app
	press(m, tea.KeyEnter, "")
	if m.wizard.step != componentListStep || m.wizard.components[0].Configuration["title"] != "Helloq" {
		t.Fatal(m.wizard.err, m.wizard.components)
	}
	startConfiguredComponent(t, m, "billing")
	for range 4 {
		press(m, tea.KeyEnter, "")
	}
	if m.wizard.components[1].Configuration["enabled"] != true || m.wizard.components[1].Configuration["port"] != int64(3000) {
		t.Fatal("instances leaked")
	}
	// Edit api, preserving the existing configuration for every field.
	m.wizard.componentCursor = 0
	press(m, 'e', "e")
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "")
	if m.wizard.configurationRaw["enabled"] != "false" || m.wizard.configurationRaw["mode"] != "large" {
		t.Fatal("edit lost values")
	}
	for range 4 {
		press(m, tea.KeyEnter, "")
	}
	press(m, tea.KeyEnter, "")
	if m.wizard.step != reviewStep || !strings.Contains(m.View().Content, `"port":4000`) {
		t.Fatal(m.wizard.err, m.View().Content)
	}
	press(m, 'p', "p")
	if !strings.Contains(strings.Join(m.wizard.planLines(80), "\n"), "Configuration:") {
		t.Fatal("preview omits config")
	}
	press(m, tea.KeyEscape, "")
	cmd := press(m, tea.KeyEnter, "")
	if cmd == nil {
		t.Fatal("generation did not start")
	}
	m.Update(cmd())
	if m.wizard.err != "" {
		t.Fatal(m.wizard.err)
	}
	for name, want := range map[string]string{"api": `"enabled":false,"mode":"large","port":4000`, "billing": `"enabled":true,"mode":"small","port":3000`} {
		data, err := os.ReadFile(filepath.Join(destination, name, "config.json"))
		if err != nil || !strings.Contains(string(data), want) {
			t.Fatal(string(data), err)
		}
	}
}

func TestWizardConfigurationLoadingCancellationAndOptionalFields(t *testing.T) {
	w, _ := newWizard()
	w.configurationDraft = project.Component{Name: "api", Type: "backend", Technology: "nestjs"}
	w.step = componentConfigurationStep
	request := &configurationRequest{component: w.configurationDraft}
	w.configurationRequest = request
	w.updateConfiguration(tea.KeyPressMsg{Code: tea.KeyEscape})
	w.configurationLoaded(configurationLoadedMsg{request: request})
	if w.step != componentNameStep || len(w.components) != 0 {
		t.Fatal("stale result changed wizard")
	}
	w.step = componentConfigurationStep
	w.configurationRequest = request
	schema := configuration.Schema{Properties: map[string]configuration.Property{"required": {Type: "string", Required: true}, "optional": {Type: "integer"}}}
	w.configurationLoaded(configurationLoadedMsg{request: request, schema: schema})
	// Sorted optional is first; a blank optional integer is omitted.
	w.updateConfiguration(tea.KeyPressMsg{Code: tea.KeyEnter})
	w.updateConfiguration(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.Contains(w.err, "required") {
		t.Fatal(w.err)
	}
	w.configurationRaw["required"] = "yes"
	w.configurationPresent["required"] = true
	w.updateConfiguration(tea.KeyPressMsg{Code: tea.KeyEnter})
	if w.step != componentListStep || len(w.components[0].Configuration) != 1 {
		t.Fatal(w.components)
	}
}
