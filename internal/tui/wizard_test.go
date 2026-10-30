package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ProvisionLabs/provisionhub/internal/project"
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

func TestWizardSelectionBackReviewAndSuccess(t *testing.T) {
	m := newTestModel()
	destination := filepath.Join(t.TempDir(), "new project")
	openWizard(m, "my-app", destination)
	if m.wizard.step != backendStep {
		t.Fatalf("step after valid project information = %v; error = %q", m.wizard.step, m.wizard.err)
	}
	press(m, tea.KeyDown, "") // NestJS, after Go.
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyDown, "") // Next.js, after None.
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyDown, "") // MySQL.
	press(m, tea.KeyDown, "") // PostgreSQL.
	press(m, tea.KeyEnter, "")
	if m.wizard.step != reviewStep {
		t.Fatalf("step = %v, error = %q", m.wizard.step, m.wizard.err)
	}
	if m.wizard.destination.Value() != destination {
		t.Fatalf("destination input lost text: %q", m.wizard.destination.Value())
	}
	for _, part := range []string{"my-app", "new project", "NestJS", "Next.js", "PostgreSQL", "Save Manifest Only"} {
		if !strings.Contains(m.View().Content, part) {
			t.Errorf("review missing %q", part)
		}
	}
	press(m, tea.KeyEscape, "")
	if m.wizard.step != databaseStep || m.wizard.selected[2] != 2 {
		t.Fatal("back navigation lost database selection")
	}
	press(m, tea.KeyEnter, "")
	cmd := press(m, tea.KeyEnter, "")
	if cmd == nil || !m.wizard.saving {
		t.Fatal("review did not start manifest creation")
	}
	if m.wizard.step == resultStep {
		t.Fatal("success shown before persistence completed")
	}
	msg := cmd()
	m.Update(msg)
	if m.wizard.step != resultStep || !strings.Contains(m.View().Content, "Manifest created successfully") {
		t.Fatalf("result step = %v, error = %q", m.wizard.step, m.wizard.err)
	}
	manifest, err := project.Load(filepath.Join(destination, project.ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Spec.Backend.Technology != "nestjs" || manifest.Spec.Frontend.Technology != "nextjs" || manifest.Spec.Database.Technology != "postgresql" {
		t.Fatalf("saved selections = %+v", manifest.Spec)
	}
	press(m, tea.KeyEscape, "")
	if m.nav.current != home {
		t.Fatal("result did not return to Home")
	}
}

func TestWizardInputValidationAndTextShortcuts(t *testing.T) {
	m := newTestModel()
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

func TestWizardFailureCanRetry(t *testing.T) {
	m := newTestModel()
	destination := filepath.Join(t.TempDir(), "project")
	openWizard(m, "demo", destination)
	for range 3 {
		press(m, tea.KeyEnter, "") // Backend, Frontend None, Database None.
	}
	if m.wizard.step != reviewStep {
		t.Fatalf("expected review, got %v: %q", m.wizard.step, m.wizard.err)
	}
	cmd := press(m, tea.KeyEnter, "")
	if cmd == nil {
		t.Fatal("expected persistence command")
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	conflict := filepath.Join(destination, project.ManifestName)
	if err := os.WriteFile(conflict, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	m.Update(cmd())
	if m.wizard.step != reviewStep || !strings.Contains(m.wizard.err, "already exists") {
		t.Fatalf("failure state = %v, %q", m.wizard.step, m.wizard.err)
	}
	if err := os.Remove(conflict); err != nil {
		t.Fatal(err)
	}
	cmd = press(m, tea.KeyEnter, "")
	m.Update(cmd())
	if m.wizard.step != resultStep {
		t.Fatalf("retry did not succeed: %q", m.wizard.err)
	}
}

func TestWizardResizeFits(t *testing.T) {
	m := newTestModel()
	press(m, tea.KeyEnter, "")
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 40, Height: 17}, {Width: 20, Height: 5}, {Width: 1, Height: 1}} {
		m.Update(size)
		view := m.View()
		if lipgloss.Width(view.Content) > size.Width || lipgloss.Height(view.Content) > size.Height {
			t.Errorf("wizard overflows %dx%d terminal", size.Width, size.Height)
		}
	}
}
