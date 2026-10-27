package tui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestPlanPreviewRendersScrollableDiagnosticsWithoutWrites(t *testing.T) {
	m := newTestModel()
	destination := filepath.Join(t.TempDir(), "not-created")
	reviewWizard(t, m, destination)
	press(m, 'p', "p")
	initial := m.View().Content
	for _, part := range []string{"Generation Plan", "my-app", "NestJS", "Next.js", "PostgreSQL", "Status: planned", "Execution: unavailable"} {
		if !strings.Contains(initial, part) {
			t.Errorf("preview missing %q: %s", part, initial)
		}
	}
	var seen strings.Builder
	for range 75 {
		seen.WriteString(m.View().Content)
		if cmd := press(m, tea.KeyDown, ""); cmd != nil {
			t.Fatal("scroll started an operation")
		}
	}
	for _, part := range []string{"backend:nestjs", "frontend:nextjs", "database:postgresql", "Depends on: backend:nestjs", "missing_operation_provider", "No backend generator", "No frontend generator", "No database integration provider"} {
		if !strings.Contains(seen.String(), part) {
			t.Errorf("scroll could not reach %q", part)
		}
	}
	bottom := m.View().Content
	press(m, tea.KeyDown, "")
	if bottom != m.View().Content {
		t.Fatal("scrolling beyond end changed the view")
	}
	for range 75 {
		press(m, tea.KeyUp, "")
	}
	if m.View().Content != initial {
		t.Fatal("could not scroll back to initial preview")
	}
	if _, err := os.Stat(destination); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("rendering/scrolling wrote destination: %v", err)
	}
}

func TestPlanPreviewResizesWithinTerminalCells(t *testing.T) {
	m := newTestModel()
	reviewWizard(t, m, filepath.Join(t.TempDir(), "not-created"))
	press(m, 'p', "p")
	m.wizard.plan.Technologies[0].Name = strings.Repeat("界é", 40)
	for range 75 {
		press(m, tea.KeyDown, "")
	}
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 80, Height: 24}, {Width: 40, Height: 17}, {Width: 20, Height: 5}, {Width: 1, Height: 1}, {Width: 2, Height: 2}, {Width: 80, Height: 24}} {
		m.Update(size)
		last := max(0, len(m.wizard.planLines(size.Width))-planPageHeight(size.Height))
		if m.wizard.planOffset < 0 || m.wizard.planOffset > last {
			t.Fatalf("resize left offset %d outside [0,%d]", m.wizard.planOffset, last)
		}
		for range 4 {
			view := m.View()
			if !view.AltScreen || lipgloss.Width(view.Content) > size.Width || lipgloss.Height(view.Content) > size.Height {
				t.Fatalf("preview overflows %dx%d: %q", size.Width, size.Height, view.Content)
			}
			press(m, 'j', "j")
		}
	}
	for range len(m.wizard.planLines(80)) {
		press(m, tea.KeyUp, "")
	}
	if !strings.Contains(m.View().Content, "Status: planned") {
		t.Fatal("could not reach plan status after bottom-scroll/resize")
	}
	press(m, tea.KeyEscape, "")
	if m.wizard.preview || m.wizard.step != reviewStep {
		t.Fatal("resize/scroll broke return to review")
	}
}
