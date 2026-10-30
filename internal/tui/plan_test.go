package tui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
)

func reviewWizard(t *testing.T, m *Model, destination string) {
	t.Helper()
	openWizard(m, "my-app", destination)
	press(m, tea.KeyDown, "") // NestJS.
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyDown, "") // Next.js.
	press(m, tea.KeyEnter, "")
	press(m, tea.KeyDown, "")
	press(m, tea.KeyDown, "") // PostgreSQL.
	press(m, tea.KeyEnter, "")
	if m.wizard.step != reviewStep {
		t.Fatalf("review step = %v; %s", m.wizard.step, m.wizard.err)
	}
}

func TestWizardPlanPreviewPreservesSelectionsAndSaveBehavior(t *testing.T) {
	m := newTestModel()
	root := t.TempDir()
	destination := filepath.Join(root, "new project café")
	reviewWizard(t, m, destination)
	selected := m.wizard.selected
	if cmd := press(m, 'p', "p"); cmd != nil {
		t.Fatal("preview must not start an execution/save command")
	}
	if !m.wizard.preview || m.wizard.plan.Status != planning.StatusPlanned || m.wizard.plan.Executable {
		t.Fatalf("preview = %+v", m.wizard)
	}
	manifest, _, err := m.service.ValidateInput(m.wizard.input(m.service))
	if err != nil {
		t.Fatal(err)
	}
	shared, err := m.service.PlanManifest(manifest)
	if err != nil || !reflect.DeepEqual(shared, m.wizard.plan) {
		t.Fatalf("preview bypasses shared planning: %v", err)
	}
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyEscape}, {Code: tea.KeyEnter}, {Code: 'p', Text: "p"}} {
		if _, cmd := m.Update(key); cmd != nil {
			t.Fatal("leaving preview must not save")
		}
		if m.wizard.preview || m.wizard.step != reviewStep || m.wizard.selected != selected {
			t.Fatal("leaving preview lost review/selections")
		}
		if _, err := os.Stat(destination); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("preview created destination: %v", err)
		}
		press(m, 'p', "p")
	}
	press(m, tea.KeyEscape, "") // Preview -> Review.
	press(m, tea.KeyEscape, "") // Review -> database selection.
	if m.wizard.step != databaseStep || m.wizard.selected != selected {
		t.Fatal("back to configuration lost selections")
	}
	press(m, tea.KeyEnter, "")
	cmd := press(m, tea.KeyEnter, "") // Unsupported selection saves only the manifest.
	if cmd == nil || !m.wizard.saving {
		t.Fatal("Review Enter no longer saves the manifest")
	}
	m.Update(cmd())
	if m.wizard.step != resultStep {
		t.Fatalf("save after preview failed: %s", m.wizard.err)
	}
}

func TestWizardPreviewShortcutRemainsTextWhileEditing(t *testing.T) {
	m := newTestModel()
	press(m, tea.KeyEnter, "")
	typeText(m, "pqjk")
	if m.wizard.name.Value() != "pqjk" || m.wizard.preview {
		t.Fatal("preview key interferes with text input")
	}
	if !strings.Contains(m.View().Content, "pqjk") {
		t.Fatal("text input not rendered")
	}
}

func TestPlanPreviewQuitKeysNeverSave(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{{Code: 'q', Text: "q"}, {Code: 'c', Mod: tea.ModCtrl}} {
		m := newTestModel()
		destination := filepath.Join(t.TempDir(), "not-created")
		reviewWizard(t, m, destination)
		press(m, 'p', "p")
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Fatalf("%q did not quit preview", key.String())
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%q started a non-quit operation", key.String())
		}
		if m.wizard.saving {
			t.Fatal("quitting preview started a save")
		}
		if _, err := os.Stat(destination); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("quit created destination: %v", err)
		}
	}
}
