package tui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/gitprovider"
	appsettings "github.com/ProvisionLabs/provisionhub/internal/settings"
)

type settingsBackend interface {
	Configured() bool
	Load(context.Context) (appsettings.Values, []gitprovider.Status, error)
	Save(context.Context, appsettings.Values) error
	Begin(context.Context) (gitprovider.Authorization, error)
	Complete(context.Context, gitprovider.Authorization) error
	Status(context.Context, bool) ([]gitprovider.Status, error)
	Disconnect(context.Context, string) error
}
type preferences struct {
	backend              settingsBackend
	workspace, provider  textinput.Model
	tab, focus, selected int
	statuses             []gitprovider.Status
	busy                 bool
	feedback             string
	auth                 gitprovider.Authorization
	generation           uint64
	cancel               context.CancelFunc
	ctx                  context.Context
	ready                bool
	offset               int
}
type preferencesMsg struct {
	generation uint64
	kind       string
	values     appsettings.Values
	statuses   []gitprovider.Status
	auth       gitprovider.Authorization
	err        error
	feedback   string
}

func newPreferences(backend settingsBackend) preferences {
	workspace := textinput.New()
	workspace.Prompt = ""
	workspace.Placeholder = "No default workspace"
	workspace.SetWidth(46)
	provider := textinput.New()
	provider.Prompt = ""
	provider.Placeholder = "github, gitlab, bitbucket, or empty"
	provider.SetWidth(40)
	provider.CharLimit = 20
	return preferences{backend: backend, workspace: workspace, provider: provider}
}
func (p *preferences) stop() {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	p.generation++
	p.busy = false
	p.auth = gitprovider.Authorization{}
}
func (p *preferences) start() (context.Context, uint64) {
	p.stop()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.ctx = ctx
	p.busy = true
	return ctx, p.generation
}
func (p *preferences) load() tea.Cmd {
	if p.backend == nil {
		p.backend = application.NewLocalSettings()
	}
	ctx, generation := p.start()
	backend := p.backend
	return func() tea.Msg {
		values, statuses, err := backend.Load(ctx)
		return preferencesMsg{generation: generation, kind: "load", values: values, statuses: statuses, err: err}
	}
}

func (p *preferences) update(msg tea.Msg) (tea.Cmd, bool) {
	if result, ok := msg.(preferencesMsg); ok {
		if result.generation != p.generation {
			return nil, false
		}
		if result.kind == "browser" {
			if result.err != nil {
				p.feedback = result.err.Error()
			} else {
				p.feedback = "Verification page opened."
			}
			return nil, false
		}
		p.busy = false
		if result.err != nil {
			if result.kind == "load" && errors.Is(result.err, appsettings.ErrInvalidStored) {
				p.workspace.SetValue(result.values.Workspace)
				p.provider.SetValue(result.values.PreferredProvider)
				p.ready = true
			}
			p.feedback = result.err.Error()
			p.auth = gitprovider.Authorization{}
			if result.kind == "disconnect" || result.kind == "complete" {
				for index := range p.statuses {
					p.statuses[index].State = gitprovider.Unverified
				}
				ctx, generation := p.start()
				backend, feedback := p.backend, p.feedback
				return func() tea.Msg {
					statuses, err := backend.Status(ctx, false)
					return preferencesMsg{generation: generation, kind: "status", statuses: statuses, err: err, feedback: feedback}
				}, false
			}
			return nil, false
		}
		switch result.kind {
		case "load":
			p.workspace.SetValue(result.values.Workspace)
			p.provider.SetValue(result.values.PreferredProvider)
			p.statuses = result.statuses
			p.ready = true
			p.feedback = ""
			return p.workspace.Focus(), false
		case "save":
			p.feedback = "Settings saved."
		case "begin":
			p.auth = result.auth
			p.busy = true
			p.feedback = string(gitprovider.Authenticating)
			ctx := p.ctx
			backend, auth, generation := p.backend, p.auth, p.generation
			return func() tea.Msg {
				return preferencesMsg{generation: generation, kind: "complete", err: backend.Complete(ctx, auth)}
			}, false
		case "complete", "disconnect":
			p.auth = gitprovider.Authorization{}
			p.feedback = "Account updated."
			return p.refresh(true), false
		case "status":
			p.statuses = result.statuses
			p.feedback = "Account status loaded."
			if result.feedback != "" {
				p.feedback = result.feedback
			}
			p.selected = min(p.selected, max(0, len(p.statuses)-1))
		case "browser":
			p.feedback = "Verification page opened."
		}
		return nil, false
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		if p.tab == 0 && !p.busy {
			var cmd tea.Cmd
			if p.focus == 0 {
				p.workspace, cmd = p.workspace.Update(msg)
			} else {
				p.provider, cmd = p.provider.Update(msg)
			}
			return cmd, false
		}
		return nil, false
	}
	switch key.String() {
	case "pgup":
		p.offset = max(0, p.offset-5)
		return nil, false
	case "pgdown":
		p.offset += 5
		return nil, false
	case "esc":
		if p.busy {
			stay := p.auth.UserCode != ""
			p.stop()
			p.feedback = "Operation cancelled."
			return nil, !stay
		}
		p.stop()
		return nil, true
	case "f1":
		p.tab = 0
		p.offset = 0
		return nil, false
	case "f2":
		p.tab = 1
		p.offset = 0
		return nil, false
	}
	if p.busy {
		if key.String() == "o" && p.auth.VerificationURL != "" {
			return p.openBrowser(), false
		}
		return nil, false
	}
	if p.tab == 0 {
		switch key.String() {
		case "tab", "shift+tab", "up", "down":
			p.focus = 1 - p.focus
			p.workspace.Blur()
			p.provider.Blur()
			if p.focus == 0 {
				return p.workspace.Focus(), false
			}
			return p.provider.Focus(), false
		case "enter", "ctrl+s":
			if !p.ready {
				p.feedback = "Settings are not loaded; press r in Git Providers to retry loading."
				return nil, false
			}
			values := appsettings.Values{Workspace: strings.TrimSpace(p.workspace.Value()), PreferredProvider: strings.TrimSpace(p.provider.Value())}
			ctx, generation := p.start()
			backend := p.backend
			return func() tea.Msg {
				return preferencesMsg{generation: generation, kind: "save", err: backend.Save(ctx, values)}
			}, false
		}
		var cmd tea.Cmd
		if p.focus == 0 {
			p.workspace, cmd = p.workspace.Update(msg)
		} else {
			p.provider, cmd = p.provider.Update(msg)
		}
		return cmd, false
	}
	switch key.String() {
	case "up", "k":
		p.selected = max(0, p.selected-1)
	case "down", "j":
		p.selected = min(max(0, len(p.statuses)-1), p.selected+1)
	case "c":
		ctx, generation := p.start()
		backend := p.backend
		p.feedback = "Starting GitHub device authorization..."
		return func() tea.Msg {
			auth, err := backend.Begin(ctx)
			return preferencesMsg{generation: generation, kind: "begin", auth: auth, err: err}
		}, false
	case "r":
		if !p.ready {
			return p.load(), false
		}
		return p.refresh(true), false
	case "d":
		if len(p.statuses) > 0 {
			account := p.statuses[p.selected].Account
			ctx, generation := p.start()
			backend := p.backend
			return func() tea.Msg {
				return preferencesMsg{generation: generation, kind: "disconnect", err: backend.Disconnect(ctx, account.ID)}
			}, false
		}
	}
	return nil, false
}
func (p *preferences) refresh(validate bool) tea.Cmd {
	ctx, generation := p.start()
	backend := p.backend
	return func() tea.Msg {
		statuses, err := backend.Status(ctx, validate)
		return preferencesMsg{generation: generation, kind: "status", statuses: statuses, err: err}
	}
}
func (p *preferences) openBrowser() tea.Cmd {
	generation := p.generation
	url := p.auth.VerificationURL
	return func() tea.Msg {
		// Never pass an arbitrary provider URL to the shell or OS protocol handler.
		if url != "https://github.com/login/device" {
			return preferencesMsg{generation: generation, kind: "browser", err: fmt.Errorf("untrusted verification URL")}
		}
		var command *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			command = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
		case "darwin":
			command = exec.Command("open", url)
		default:
			command = exec.Command("xdg-open", url)
		}
		err := command.Run()
		if err != nil {
			err = fmt.Errorf("browser could not open; visit the displayed verification URL manually")
		}
		return preferencesMsg{generation: generation, kind: "browser", err: err}
	}
}

func safeText(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
}
func (p *preferences) view(width, height int) string {
	lines := []string{"Settings", "F1 General · F2 Git Providers · Esc back", ""}
	if p.tab == 0 {
		lines = append(lines, "General", "Default workspace:", p.workspace.View(), "Preferred Git provider:", p.provider.View(), "Tab change field · Enter save")
	} else {
		state := string(gitprovider.NotConnected)
		if p.backend != nil && !p.backend.Configured() {
			state = string(gitprovider.NotConfigured)
		}
		if len(p.statuses) > 0 {
			state = "accounts listed below"
		}
		lines = append(lines, "Git Providers", "GitHub: "+state)
		if p.auth.UserCode != "" {
			lines = append(lines, "Verification: "+p.auth.VerificationURL, "Code: "+safeText(p.auth.UserCode), "o open browser · Esc cancel")
		}
		for index, status := range p.statuses {
			prefix := "  "
			if index == p.selected {
				prefix = "› "
			}
			lines = append(lines, prefix+safeText(status.Account.Username)+" — "+string(status.State))
		}
		lines = append(lines, "GitLab: Coming soon", "Bitbucket: Coming soon", "c connect GitHub · r validate/retry", "↑↓ select account · d disconnect")
	}
	if p.busy {
		lines = append(lines, "Working... (Esc cancels)")
	}
	if p.feedback != "" {
		lines = append(lines, safeText(p.feedback))
	}
	// Wrap rather than truncate device instructions or errors on narrow terminals.
	wrapped := []string{}
	for _, line := range lines {
		wrapped = append(wrapped, strings.Split(lipgloss.NewStyle().MaxWidth(max(1, width)).Render(line), "\n")...)
	}
	capacity := max(1, height-1)
	offset := min(p.offset, max(0, len(wrapped)-capacity))
	visible := wrapped[offset:min(len(wrapped), offset+capacity)]
	if height > 1 {
		visible = append(visible, fitPlanLine("PgUp/PgDn scroll · Ctrl+C quit", width))
	}
	content := strings.Join(visible, "\n")
	if width >= 60 && height >= 24 {
		frame := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#64748B")).Padding(1, 2).Render(content)
		if lipgloss.Width(frame) <= width && lipgloss.Height(frame) <= height {
			return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, frame)
		}
	}
	return content
}
