package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
)

func newTestModel() *Model { return NewModel(application.Info()) }

func press(m *Model, code rune, text string) tea.Cmd {
	_, cmd := m.Update(tea.KeyPressMsg{Code: code, Text: text})
	return cmd
}

func TestInitialView(t *testing.T) {
	m := newTestModel()
	if m.Init() != nil {
		t.Fatal("initial model should have no command")
	}
	view := m.View()
	if !view.AltScreen {
		t.Fatal("TUI must use the alternate screen")
	}
	for _, part := range []string{"PROVISIONHUB", application.Info().Tagline, "› Create Project", "Open Existing Project", "Plugin Marketplace", "Settings"} {
		if !strings.Contains(view.Content, part) {
			t.Errorf("initial view missing %q", part)
		}
	}
	if got := strings.Count(view.Content, strings.Repeat("─", 48)); got < 2 {
		t.Fatalf("home dividers wrapped or disappeared: found %d full-width rules", got)
	}
}

func TestNavigationAndBack(t *testing.T) {
	for index, item := range homeMenu {
		m := newTestModel()
		for range index {
			press(m, tea.KeyDown, "")
		}
		press(m, tea.KeyEnter, "")
		if m.nav.current != item.screen {
			t.Fatalf("selection %q did not open its screen", item.label)
		}
		if item.screen == createProject {
			if !strings.Contains(m.View().Content, "Project information") {
				t.Fatal("create project did not open the wizard")
			}
		} else if !strings.Contains(m.View().Content, secondaryScreens[item.screen].title) {
			t.Fatalf("selection %q did not open its screen", item.label)
		}
		press(m, tea.KeyEscape, "")
		if m.nav.current != home || m.nav.selected != index {
			t.Fatalf("return from %q lost home selection", item.label)
		}
	}
}

func TestMenuKeysWrapAndSecondaryIgnoresSelection(t *testing.T) {
	m := newTestModel()
	press(m, 'k', "k")
	if m.nav.selected != len(homeMenu)-1 {
		t.Fatalf("up from first item selected %d", m.nav.selected)
	}
	press(m, 'j', "j")
	if m.nav.selected != 0 {
		t.Fatalf("down from last item selected %d", m.nav.selected)
	}
	press(m, tea.KeyEnter, "")
	if m.nav.current != createProject || m.wizard.step != projectInfoStep {
		t.Fatal("create project should open at its information step")
	}
	press(m, tea.KeyEscape, "")
	press(m, tea.KeyEscape, "")
	if m.nav.current != home {
		t.Fatal("Esc on home should leave the home screen visible")
	}
}

func TestQuitKeys(t *testing.T) {
	for _, msg := range []tea.KeyPressMsg{
		{Code: 'q', Text: "q"},
		{Code: 'c', Mod: tea.ModCtrl},
	} {
		m := newTestModel()
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatalf("key %q did not return quit command", msg.String())
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("key %q returned %T, want QuitMsg", msg.String(), cmd())
		}
	}
}

func TestResizeFitsViewport(t *testing.T) {
	m := newTestModel()
	for _, size := range []tea.WindowSizeMsg{
		{Width: 80, Height: 24},
		{Width: 40, Height: 17},
		{Width: 39, Height: 16},
		{Width: 20, Height: 5},
		{Width: 1, Height: 1},
	} {
		m.Update(size)
		view := m.View()
		if got := lipgloss.Width(view.Content); got > size.Width {
			t.Errorf("%dx%d view width = %d", size.Width, size.Height, got)
		}
		if got := lipgloss.Height(view.Content); got > size.Height {
			t.Errorf("%dx%d view height = %d", size.Width, size.Height, got)
		}
	}
	press(m, tea.KeyEnter, "")
	m.Update(tea.WindowSizeMsg{Width: 42, Height: 17})
	if !strings.Contains(m.View().Content, "Create Project") {
		t.Fatal("resizing a secondary screen lost its content")
	}
}
