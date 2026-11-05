package tui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ProvisionLabs/provisionhub/internal/gitprovider"
	appsettings "github.com/ProvisionLabs/provisionhub/internal/settings"
)

type fakePreferences struct {
	values    appsettings.Values
	statuses  []gitprovider.Status
	failure   error
	cancelled bool
}

func (f *fakePreferences) Configured() bool { return true }
func (f *fakePreferences) Load(context.Context) (appsettings.Values, []gitprovider.Status, error) {
	return f.values, f.statuses, f.failure
}
func (f *fakePreferences) Save(_ context.Context, v appsettings.Values) error {
	if f.failure != nil {
		return f.failure
	}
	if err := v.Validate(); err != nil {
		return err
	}
	f.values = v
	return nil
}
func (f *fakePreferences) Begin(context.Context) (gitprovider.Authorization, error) {
	return gitprovider.Authorization{UserCode: "USER-CODE", VerificationURL: "https://github.com/login/device"}, f.failure
}
func (f *fakePreferences) Complete(ctx context.Context, _ gitprovider.Authorization) error {
	if ctx.Err() != nil {
		f.cancelled = true
		return ctx.Err()
	}
	if f.failure != nil {
		return f.failure
	}
	f.statuses = []gitprovider.Status{{Account: gitprovider.Account{ID: "test-id", Username: "sample"}, State: gitprovider.Connected}}
	return nil
}
func (f *fakePreferences) Status(context.Context, bool) ([]gitprovider.Status, error) {
	return f.statuses, f.failure
}
func (f *fakePreferences) Disconnect(context.Context, string) error {
	if f.failure != nil {
		return f.failure
	}
	f.statuses = nil
	return nil
}

func settingsModel(t *testing.T) (*Model, *fakePreferences) {
	t.Helper()
	m := newTestModel(t)
	backend := &fakePreferences{}
	m.preferences = newPreferences(backend)
	m.nav.selected = len(homeMenu) - 1
	cmd := press(m, tea.KeyEnter, "")
	if cmd == nil {
		t.Fatal("settings load must be asynchronous")
	}
	m.Update(cmd())
	return m, backend
}
func TestSettingsEditingFeedbackAndNoBlockingWork(t *testing.T) {
	m, backend := settingsModel(t)
	workspace := t.TempDir()
	typeText(m, workspace)
	press(m, tea.KeyTab, "")
	typeText(m, "github")
	cmd := press(m, tea.KeyEnter, "")
	if backend.values.Workspace != "" {
		t.Fatal("save ran in event loop")
	}
	m.Update(cmd())
	if backend.values.Workspace != workspace || backend.values.PreferredProvider != "github" || !strings.Contains(m.View().Content, "Settings saved") {
		t.Fatal(m.View().Content)
	}
	backend.failure = errors.New("database unavailable")
	cmd = press(m, tea.KeyEnter, "")
	m.Update(cmd())
	if !strings.Contains(m.View().Content, "database unavailable") {
		t.Fatal("missing persistence feedback")
	}
	// q is literal input, not quit, in General.
	if cmd := press(m, 'q', "q"); cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("text input quit application")
		}
	}
	press(m, tea.KeyEscape, "")
	if m.nav.current != home {
		t.Fatal("settings back did not work")
	}
}
func TestSettingsDeviceConnectionCancellationAndDisconnect(t *testing.T) {
	m, backend := settingsModel(t)
	press(m, tea.KeyF2, "")
	for _, text := range []string{"GitHub", "GitLab: Coming soon", "Bitbucket: Coming soon"} {
		if !strings.Contains(m.View().Content, text) {
			t.Fatal(m.View().Content)
		}
	}
	begin := press(m, 'c', "c")
	_, poll := m.Update(begin())
	if poll == nil || !strings.Contains(m.View().Content, "USER-CODE") {
		t.Fatal("missing device instructions")
	}
	generation := m.preferences.generation
	m.Update(preferencesMsg{generation: generation, kind: "browser"})
	if !m.preferences.busy {
		t.Fatal("opening browser stopped authentication state")
	}
	press(m, tea.KeyEscape, "")
	m.Update(poll())
	if !backend.cancelled || len(m.preferences.statuses) != 0 {
		t.Fatal("cancelled authentication connected")
	}
	begin = press(m, 'c', "c")
	_, poll = m.Update(begin())
	_, status := m.Update(poll())
	m.Update(status())
	if !strings.Contains(m.View().Content, "sample — connected") {
		t.Fatal(m.View().Content)
	}
	cmd := press(m, 'd', "d")
	_, status = m.Update(cmd())
	m.Update(status())
	if len(m.preferences.statuses) != 0 {
		t.Fatal("disconnect did not update state")
	}
	backend.failure = gitprovider.ErrUnavailable
	begin = press(m, 'c', "c")
	m.Update(begin())
	if !strings.Contains(m.View().Content, "unavailable") {
		t.Fatal("missing provider error")
	}
}
func TestSettingsNarrowLayoutsAndWorkspaceDefault(t *testing.T) {
	m, _ := settingsModel(t)
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 20, Height: 10}, {Width: 1, Height: 1}} {
		m.Update(size)
		view := m.View().Content
		if lipgloss.Width(view) > size.Width || lipgloss.Height(view) > size.Height {
			t.Fatalf("overflow %dx%d: %q", size.Width, size.Height, view)
		}
	}
	press(m, tea.KeyEscape, "")
	m.nav.selected = 0
	press(m, tea.KeyEnter, "")
	directory := t.TempDir()
	m.Update(workspacePreferenceMsg{request: m.workspaceRequest, directory: directory})
	typeText(m, "demo")
	press(m, tea.KeyEnter, "")
	if m.wizard.destination.Value() != filepath.Join(directory, "demo") {
		t.Fatal("workspace default not used")
	}
}

func TestSettingsDisconnectErrorRefreshesStaleConnectedState(t *testing.T) {
	m, backend := settingsModel(t)
	press(m, tea.KeyF2, "")
	m.preferences.statuses = []gitprovider.Status{{Account: gitprovider.Account{ID: "test-id", Username: "sample"}, State: gitprovider.Connected}}
	backend.statuses = []gitprovider.Status{{Account: gitprovider.Account{ID: "test-id", Username: "sample", Disconnected: true}, State: gitprovider.NotConnected}}
	_, refresh := m.Update(preferencesMsg{generation: m.preferences.generation, kind: "disconnect", err: errors.New("credential removed; metadata cleanup failed")})
	if m.preferences.statuses[0].State == gitprovider.Connected {
		t.Fatal("stale connected state remained after disconnect failure")
	}
	m.Update(refresh())
	if m.preferences.statuses[0].State != gitprovider.NotConnected || !strings.Contains(m.preferences.feedback, "metadata cleanup failed") {
		t.Fatal("lost cleanup failure or disconnected state")
	}
}
