package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func addNestComponent(t *testing.T, m *Model, name string) {
	t.Helper()
	press(m, 'a', "a")
	if m.wizard.step != componentCategoryStep {
		t.Fatal("add did not open category")
	}
	press(m, tea.KeyEnter, "") // Backend.
	press(m, tea.KeyDown, "")  // NestJS after Go.
	press(m, tea.KeyEnter, "")
	typeText(m, name)
	press(m, tea.KeyEnter, "")
	if m.wizard.step != componentListStep {
		t.Fatalf("component not saved: %s", m.wizard.err)
	}
}

func TestMultiComponentWizardAddEditRemovePreviewAndGenerate(t *testing.T) {
	m, destination := generatorModel(t)
	press(m, tea.KeyEnter, "")
	typeText(m, "multi-demo")
	press(m, tea.KeyEnter, "")
	typeText(m, destination)
	press(m, tea.KeyF2, "")
	press(m, tea.KeyEnter, "")
	if m.wizard.step != componentListStep || !m.wizard.multiMode {
		t.Fatalf("multi-component mode did not open: %+v", m.wizard)
	}
	press(m, tea.KeyEnter, "")
	if !strings.Contains(m.wizard.err, "at least one") {
		t.Fatalf("empty list accepted: %s", m.wizard.err)
	}
	addNestComponent(t, m, "api")
	press(m, 'a', "a")
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyDown, "")
	press(m, tea.KeyEnter, "")
	typeText(m, "api")
	press(m, tea.KeyEnter, "")
	if !strings.Contains(m.wizard.err, "duplicate") {
		t.Fatal("duplicate component name accepted")
	}
	m.wizard.componentName.SetValue("billing")
	press(m, tea.KeyEnter, "")
	if len(m.wizard.components) != 2 {
		t.Fatal("second component missing")
	}
	press(m, 'e', "e")
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "")
	m.wizard.componentName.SetValue("worker")
	press(m, tea.KeyEnter, "")
	if m.wizard.components[1].Name != "worker" {
		t.Fatal("edit did not preserve selected technology")
	}
	press(m, 'd', "d")
	if len(m.wizard.components) != 1 || m.wizard.components[0].Name != "api" {
		t.Fatal("remove changed another component")
	}
	addNestComponent(t, m, "billing")
	press(m, tea.KeyEscape, "")
	if m.wizard.step != projectInfoStep || m.wizard.name.Value() != "multi-demo" || m.wizard.destination.Value() != destination {
		t.Fatal("back navigation lost project information")
	}
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyEnter, "")
	if m.wizard.step != reviewStep || !m.wizard.plan.Executable {
		t.Fatalf("review plan: %+v, %s", m.wizard.plan, m.wizard.err)
	}
	press(m, 'p', "p")
	preview := strings.Join(m.wizard.planLines(80), "\n")
	if !strings.Contains(preview, "component/api") || !strings.Contains(preview, "billing/package.json") {
		t.Fatalf("preview lacks component operations: %s", preview)
	}
	if _, err := os.Stat(filepath.Join(destination, "api", "package.json")); !os.IsNotExist(err) {
		t.Fatal("preview wrote a file")
	}
	press(m, tea.KeyEnter, "")
	cmd := press(m, tea.KeyEnter, "")
	if cmd == nil || !m.wizard.applying {
		t.Fatal("review did not start approved generation")
	}
	m.Update(cmd())
	if m.wizard.step != resultStep || len(m.wizard.applyResult.Created) != 2 {
		t.Fatalf("generation result: %+v", m.wizard)
	}
	manifest, err := project.Load(filepath.Join(destination, project.ManifestName))
	if err != nil || len(manifest.Spec.Components) != 2 {
		t.Fatalf("manifest: %+v, %v", manifest, err)
	}
	for _, name := range []string{"api", "billing"} {
		if _, err := os.Stat(filepath.Join(destination, name, "package.json")); err != nil {
			t.Fatal(err)
		}
	}
}
