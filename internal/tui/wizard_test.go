package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func typeText(m *Model, value string) {
	for _, r := range value {
		press(m, r, string(r))
	}
}

func openWizard(m *Model, name, destination string) {
	press(m, tea.KeyEnter, "")
	typeText(m, name)
	press(m, tea.KeyEnter, "")
	typeText(m, destination)
	press(m, tea.KeyEnter, "")
}

func TestCreateProjectOpensNamedComponentsDirectly(t *testing.T) {
	m := newTestModel(t)
	press(m, tea.KeyEnter, "")
	if strings.Contains(m.View().Content, "F2") || strings.Contains(m.View().Content, "Legacy categories") {
		t.Fatal("legacy mode still appears in project information")
	}
	typeText(m, "demo")
	press(m, tea.KeyEnter, "")
	typeText(m, filepath.Join(t.TempDir(), "demo"))
	press(m, tea.KeyF2, "")
	press(m, tea.KeyEnter, "")
	if m.wizard.step != componentListStep || !strings.Contains(m.View().Content, "No components yet") {
		t.Fatalf("wizard did not open named components: %v, %s", m.wizard.step, m.View().Content)
	}
}

func TestWizardComponentReviewWithoutGenerator(t *testing.T) {
	m := newTestModel(t)
	destination := filepath.Join(t.TempDir(), "new project")
	openWizard(m, "my-app", destination)
	if m.wizard.step != componentListStep {
		t.Fatalf("step after project information = %v; error = %q", m.wizard.step, m.wizard.err)
	}
	addComponent(t, m, "api", 0, 1)  // NestJS.
	addComponent(t, m, "web", 1, 0)  // Next.js.
	addComponent(t, m, "data", 2, 1) // PostgreSQL.
	press(m, tea.KeyEnter, "")
	if m.wizard.step != reviewStep {
		t.Fatalf("step = %v, error = %q", m.wizard.step, m.wizard.err)
	}
	if m.wizard.destination.Value() != destination {
		t.Fatalf("destination input lost text: %q", m.wizard.destination.Value())
	}
	for _, part := range []string{"my-app", "new project", "api", "web", "data", "NestJS", "Next.js", "PostgreSQL", "Generation unavailable"} {
		if !strings.Contains(m.View().Content, part) {
			t.Errorf("review missing %q", part)
		}
	}
	press(m, tea.KeyEscape, "")
	if m.wizard.step != componentListStep || len(m.wizard.components) != 3 {
		t.Fatal("back navigation lost components")
	}
	press(m, tea.KeyEnter, "")
	cmd := press(m, tea.KeyEnter, "")
	if cmd != nil || m.wizard.step != reviewStep || !strings.Contains(m.wizard.err, "unavailable") {
		t.Fatalf("unsupported generation was approved: %q", m.wizard.err)
	}
	press(m, tea.KeyEscape, "")
	if m.wizard.step != componentListStep {
		t.Fatal("review did not return to components")
	}
}

func TestWizardInputValidationAndTextShortcuts(t *testing.T) {
	m := newTestModel(t)
	press(m, tea.KeyEnter, "")
	typeText(m, "qjkBad Name")
	if m.nav.current != createProject || !strings.HasPrefix(m.wizard.name.Value(), "qjk") {
		t.Fatal("q, j, and k should be text while editing a field")
	}
	press(m, tea.KeyEnter, "")
	typeText(m, filepath.Join(t.TempDir(), "demo"))
	press(m, tea.KeyEnter, "")
	if m.wizard.step != projectInfoStep || !strings.Contains(m.View().Content, "project name") {
		t.Fatalf("invalid information advanced: step %v, error %q", m.wizard.step, m.wizard.err)
	}
	press(m, tea.KeyEscape, "")
	if m.nav.current != home {
		t.Fatal("Esc on first step should return Home")
	}
	openWizard(m, "valid-name", "")
	if m.wizard.step != projectInfoStep || !strings.Contains(m.wizard.err, "destination path") {
		t.Fatalf("empty destination advanced: step %v, error %q", m.wizard.step, m.wizard.err)
	}
}

func TestWizardResizeFits(t *testing.T) {
	m := newTestModel(t)
	press(m, tea.KeyEnter, "")
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 40, Height: 17}, {Width: 20, Height: 5}, {Width: 1, Height: 1}} {
		m.Update(size)
		view := m.View()
		if lipgloss.Width(view.Content) > size.Width || lipgloss.Height(view.Content) > size.Height {
			t.Errorf("wizard overflows %dx%d terminal", size.Width, size.Height)
		}
	}
}
