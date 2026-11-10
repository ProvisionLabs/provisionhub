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

func connectionModel(t *testing.T) (*Model, string) {
	t.Helper()
	m, destination := generatorModel(t)
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
      description: Remote service endpoint
      requires:
        properties:
          port: {type: integer, required: true}
      configuration:
        properties:
          host: {type: string, format: host, label: Server hostname, required: true}
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
		for file, content := range map[string]string{"plugin.yaml": manifest, "base": `{"name":"{{componentName}}"}`, "endpoint": "# target={{target.name}}\nAPI=http://{{connection.host|host}}:{{target.port}}\n"} {
			if err := os.WriteFile(filepath.Join(folder, file), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	s, err := application.NewProjectServiceWithPlugins(root)
	if err != nil {
		t.Fatal(err)
	}
	m.service = s
	return m, destination
}

func addConnectionBackend(t *testing.T, m *Model, name, port string) {
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
	m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	typeText(m, port)
	press(m, tea.KeyEnter, "")
	if m.wizard.step != componentListStep {
		t.Fatal(m.wizard.err)
	}
}

func chooseConnection(t *testing.T, m *Model, id string) {
	t.Helper()
	for i, option := range m.wizard.connectionOptions {
		if option.Connection.ID() == id {
			for m.wizard.connectionOptionCursor != i {
				press(m, tea.KeyDown, "")
			}
			press(m, tea.KeyEnter, "")
			return
		}
	}
	t.Fatal("compatible target not offered", id, m.wizard.connectionOptions)
}

func TestWizardConnectionCreateEditRemoveReviewAndGenerate(t *testing.T) {
	m, destination := connectionModel(t)
	openWizard(m, "demo", destination)
	addConnectionBackend(t, m, "api-a", "4001")
	addConnectionBackend(t, m, "api-b", "4002")
	addComponent(t, m, "web", 1, 0)
	addComponent(t, m, "admin", 1, 0)
	press(m, 'c', "c")
	press(m, 'a', "a")
	if m.wizard.step != connectionTargetStep || len(m.wizard.connectionOptions) != 4 {
		t.Fatal(m.wizard.err, m.wizard.connectionOptions)
	}
	for _, option := range m.wizard.connectionOptions {
		if option.Connection.From != "web" && option.Connection.From != "admin" {
			t.Fatal("incompatible source exposed", option)
		}
	}
	chooseConnection(t, m, "web:api-b:transport")
	if !strings.Contains(m.View().Content, "Server hostname") {
		t.Fatal(m.View().Content)
	}
	press(m, tea.KeyEnter, "")
	if !strings.Contains(m.wizard.err, "required") {
		t.Fatal(m.wizard.err)
	}
	typeText(m, "http://invalid")
	press(m, tea.KeyEnter, "")
	if !strings.Contains(m.wizard.err, "DNS hostname") {
		t.Fatal(m.wizard.err)
	}
	m.Update(tea.WindowSizeMsg{Width: 20, Height: 5})
	if !strings.Contains(m.View().Content, "Error:") {
		t.Fatal("compact connection error hidden", m.View().Content)
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	typeText(m, "localhost")
	press(m, tea.KeyEscape, "")
	press(m, tea.KeyEnter, "")
	if m.wizard.configurationInput.Value() != "localhost" {
		t.Fatal("back lost connection draft")
	}
	press(m, tea.KeyEnter, "")
	if len(m.wizard.connections) != 1 || m.wizard.connections[0].To != "api-b" {
		t.Fatal(m.wizard.err, m.wizard.connections)
	}
	press(m, 'e', "e")
	chooseConnection(t, m, "web:api-b:transport")
	if m.wizard.configurationInput.Value() != "localhost" {
		t.Fatal("edit lost connection settings")
	}
	press(m, tea.KeyEscape, "")
	chooseConnection(t, m, "web:api-a:transport")
	typeText(m, "127.0.0.1")
	press(m, tea.KeyEnter, "")
	press(m, 'a', "a")
	chooseConnection(t, m, "admin:api-a:transport")
	typeText(m, "127.0.0.1")
	press(m, tea.KeyEnter, "")
	if len(m.wizard.connections) != 2 {
		t.Fatal(m.wizard.err)
	}
	press(m, 'd', "d")
	if len(m.wizard.connections) != 1 {
		t.Fatal("remove failed")
	}
	press(m, 'a', "a")
	chooseConnection(t, m, "admin:api-a:transport")
	typeText(m, "127.0.0.1")
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "")
	if m.wizard.step != reviewStep || !m.wizard.plan.Executable {
		t.Fatal(m.wizard.err, m.wizard.plan)
	}
	if !strings.Contains(strings.Join(m.wizard.componentLines(m.service, 66), "\n"), "web -> api-a") {
		t.Fatal("review omitted connections")
	}
	press(m, 'p', "p")
	if !strings.Contains(strings.Join(m.wizard.planLines(80), "\n"), "Target metadata:") {
		t.Fatal("preview omitted resolved connection")
	}
	press(m, tea.KeyEscape, "")
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("review wrote files")
	}
	m.wizard.connections[0].Configuration["host"] = "other.example"
	if cmd := press(m, tea.KeyEnter, ""); cmd != nil || !strings.Contains(m.wizard.err, "plan changed") {
		t.Fatal("accepted stale connection review", m.wizard.err)
	}
	m.wizard.connections[0].Configuration["host"] = "127.0.0.1"
	press(m, tea.KeyEscape, "")
	press(m, tea.KeyEnter, "")
	cmd := press(m, tea.KeyEnter, "")
	if cmd == nil {
		t.Fatal(m.wizard.err)
	}
	m.Update(cmd())
	if m.wizard.err != "" || len(m.wizard.applyResult.Created) != 6 {
		t.Fatal(m.wizard.err, m.wizard.applyResult)
	}
	for _, name := range []string{"web", "admin"} {
		data, err := os.ReadFile(filepath.Join(destination, name, ".env.local"))
		if err != nil || !strings.Contains(string(data), "target=api-a") || !strings.Contains(string(data), ":4001") {
			t.Fatal(string(data), err)
		}
	}
}

func TestWizardConnectionCancellationAndComponentChanges(t *testing.T) {
	m, destination := connectionModel(t)
	openWizard(m, "demo", destination)
	addConnectionBackend(t, m, "api", "4001")
	addComponent(t, m, "web", 1, 0)
	press(m, 'c', "c")
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if cmd == nil || m.wizard.connectionRequest == nil {
		t.Fatal("schema discovery was not asynchronous")
	}
	press(m, tea.KeyEscape, "")
	m.Update(cmd())
	if m.wizard.step != connectionListStep || len(m.wizard.connections) != 0 {
		t.Fatal("cancelled result changed wizard")
	}
	press(m, 'a', "a")
	chooseConnection(t, m, "web:api:transport")
	typeText(m, "localhost")
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "") // components
	m.wizard.componentCursor = 0
	press(m, 'e', "e")
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "")
	m.wizard.componentName.SetValue("renamed")
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "")
	if m.wizard.connections[0].To != "renamed" {
		t.Fatal("rename lost connection identity", m.wizard.connections)
	}
	press(m, 'd', "d")
	if len(m.wizard.connections) != 0 || !strings.Contains(m.wizard.err, "Removed 1") {
		t.Fatal("component removal left dangling connection", m.wizard.err)
	}
	press(m, 'c', "c")
	press(m, 'a', "a")
	if !strings.Contains(m.wizard.err, "No compatible") || m.wizard.step != connectionListStep {
		t.Fatal(m.wizard.err)
	}
	// Consumer capabilities with no options skip the shared form.
	w, _ := newWizard()
	w.configuringConnection = true
	w.editingConnection = -1
	w.connectionDraft = project.Connection{From: "web", To: "api", Type: "transport"}
	w.configurationDraft = project.Component{Name: "connection web:api:transport"}
	w.openConfiguration(configuration.Schema{})
	if w.step != connectionListStep || len(w.connections) != 1 {
		t.Fatal(w.connections)
	}
}
