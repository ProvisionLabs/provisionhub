package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
)

var keys = struct {
	up, down, selectItem, back, quit key.Binding
	menuHelp, backHelp               key.Binding
}{
	up:         key.NewBinding(key.WithKeys("up", "k")),
	down:       key.NewBinding(key.WithKeys("down", "j")),
	selectItem: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	back:       key.NewBinding(key.WithKeys("esc")),
	quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	menuHelp:   key.NewBinding(key.WithKeys("up", "down", "j", "k"), key.WithHelp("↑↓/jk", "navigate")),
	backHelp:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
}

type screenText struct {
	title       string
	description string
}

type workspacePreferenceMsg struct {
	request   uint64
	directory string
	err       error
}

var secondaryScreens = map[screen]screenText{
	pluginMarketplace: {
		title:       "Plugins",
		description: "Discover official plugins and manage local installations.",
	},
	settings: {
		title:       "Settings",
		description: "Local preferences and Git provider connections.",
	},
}

// Model contains terminal presentation state; application metadata comes from
// the shared application layer.
type Model struct {
	info             application.AppInfo
	service          application.ProjectService
	nav              navigation
	wizard           wizard
	help             help.Model
	width            int
	height           int
	preferences      preferences
	workspaceRequest uint64
	pluginBrowser    pluginBrowser
	localSettings    *application.LocalSettings
}

func NewModel(info application.AppInfo) *Model {
	service := application.NewProjectService()
	return newModelWithPersistence(info, service, application.NewLocalSettings())
}

func newModelWithPersistence(info application.AppInfo, service application.ProjectService, localSettings *application.LocalSettings) *Model {
	h := help.New()
	h.SetWidth(48)
	return &Model{info: info, service: service, localSettings: localSettings, help: h, width: 80, height: 24, preferences: newPreferences(localSettings), pluginBrowser: newPluginBrowser(application.PluginManager{})}
}

// NewModelWithPlugins loads an explicit local plugin directory for this session.
func NewModelWithPlugins(info application.AppInfo, directory string) (*Model, error) {
	service, err := application.NewProjectServiceWithPlugins(directory)
	if err != nil {
		return nil, err
	}
	return newModelWithPersistence(info, service, application.NewLocalSettings()), nil
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(pluginResultMsg); ok {
		cmd, _ := m.pluginBrowser.update(msg)
		return m, cmd
	}
	if result, ok := msg.(workspacePreferenceMsg); ok {
		if result.request == m.workspaceRequest && m.nav.current == createProject && m.wizard.step == projectInfoStep {
			if result.err != nil {
				m.wizard.err = "Workspace preference unavailable; enter destination manually."
			} else {
				m.wizard.defaultWorkspace = result.directory
				if m.wizard.focused == 1 && m.wizard.destination.Value() == "" && m.wizard.name.Value() != "" && result.directory != "" {
					m.wizard.destination.SetValue(filepath.Join(result.directory, m.wizard.name.Value()))
				}
			}
		}
		return m, nil
	}
	if _, ok := msg.(preferencesMsg); ok {
		cmd, _ := m.preferences.update(msg)
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetWidth(max(1, min(msg.Width-4, 54)-6))
		m.wizard.resize(msg.Width, msg.Height)
		m.preferences.workspace.SetWidth(max(1, min(46, msg.Width-4)))
		m.preferences.provider.SetWidth(max(1, min(40, msg.Width-4)))
	case projectCreatedMsg:
		m.wizard.created(msg)
	case projectAppliedMsg:
		m.wizard.applied(msg)
	case configurationLoadedMsg:
		if m.nav.current == createProject {
			return m, m.wizard.configurationLoaded(msg)
		}
	case connectionOptionsMsg:
		if m.nav.current == createProject {
			return m, m.wizard.connectionOptionsLoaded(msg)
		}
	case tea.KeyPressMsg:
		if m.nav.current == createProject && m.wizard.applying {
			return m, nil
		}
		if msg.String() == "ctrl+c" {
			m.preferences.stop()
			m.pluginBrowser.stop()
			return m, tea.Quit
		}
		if m.nav.current == pluginMarketplace {
			cmd, back := m.pluginBrowser.update(msg)
			if back {
				values := []string{}
				for id, version := range m.pluginBrowser.selections {
					values = append(values, id+"@"+version)
				}
				service, err := m.service.WithPluginVersions(values)
				if err == nil {
					m.service = service
				}
				m.nav.back()
			}
			return m, cmd
		}
		if m.nav.current == settings {
			cmd, back := m.preferences.update(msg)
			if back {
				m.nav.back()
			}
			return m, cmd
		}
		if m.nav.current == createProject && (m.wizard.step == projectInfoStep || m.wizard.step == componentNameStep || m.wizard.step == componentConfigurationStep) {
			cmd, back := m.wizard.update(msg, m.service)
			if back {
				m.nav.back()
			}
			return m, cmd
		}
		switch {
		case key.Matches(msg, keys.quit):
			return m, tea.Quit
		case m.nav.current == createProject:
			cmd, back := m.wizard.update(msg, m.service)
			if back {
				m.nav.back()
			}
			return m, cmd
		case key.Matches(msg, keys.back):
			m.nav.back()
		case key.Matches(msg, keys.up):
			m.nav.move(-1)
		case key.Matches(msg, keys.down):
			m.nav.move(1)
		case key.Matches(msg, keys.selectItem):
			m.nav.selectItem()
			if m.nav.current == pluginMarketplace {
				return m, m.pluginBrowser.start("load", "", "")
			}
			if m.nav.current == settings {
				return m, m.preferences.load()
			}
			if m.nav.current == createProject {
				var cmd tea.Cmd
				m.wizard, cmd = newWizard()
				m.wizard.resize(m.width, m.height)
				m.workspaceRequest++
				request := m.workspaceRequest
				return m, tea.Batch(cmd, func() tea.Msg {
					ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
					defer cancel()
					directory, err := m.localSettings.DefaultWorkspace(ctx)
					return workspacePreferenceMsg{request: request, directory: directory, err: err}
				})
			}
		}
	default:
		if m.nav.current == settings {
			cmd, _ := m.preferences.update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) View() tea.View {
	width, height := max(1, m.width), max(1, m.height)
	if m.nav.current == pluginMarketplace {
		view := tea.NewView(m.pluginBrowser.view(width, height))
		view.AltScreen = true
		return view
	}
	if m.nav.current == settings {
		view := tea.NewView(m.preferences.view(width, height))
		view.AltScreen = true
		return view
	}
	if m.nav.current == createProject && m.wizard.preview {
		view := tea.NewView(m.wizard.planView(width, height))
		view.AltScreen = true
		return view
	}
	if width < 40 || height < 17 {
		view := tea.NewView(m.compactView(width, height))
		view.AltScreen = true
		return view
	}

	frameWidth := min(width-4, 54)
	if m.nav.current == createProject {
		frameWidth = min(width-4, 72)
	}
	contentWidth := frameWidth - 6
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD3FC"))
	selected := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD3FC"))

	var lines []string
	if m.nav.current == createProject {
		lines = m.wizard.lines(m.service, contentWidth)
	} else {
		lines = []string{
			lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, title.Render(strings.ToUpper(m.info.Name))),
			lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, muted.Render(m.info.Tagline)),
			"",
			strings.Repeat("─", contentWidth),
			"",
		}
		if m.nav.current == home {
			for i, item := range homeMenu {
				line := "  " + item.label
				if i == m.nav.selected {
					line = selected.Render("› " + item.label)
				}
				lines = append(lines, line)
			}
		} else {
			info := secondaryScreens[m.nav.current]
			lines = append(lines, title.Render(info.title), "", info.description)
		}
		lines = append(lines, "", strings.Repeat("─", contentWidth))
		if m.nav.current == home {
			lines = append(lines, m.help.ShortHelpView([]key.Binding{keys.menuHelp, keys.selectItem, keys.quit}))
		} else {
			lines = append(lines, m.help.ShortHelpView([]key.Binding{keys.backHelp, keys.quit}))
		}
		lines = append(lines, muted.Render(m.info.Version))
	}

	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#64748B")).
		Padding(1, 2).
		Width(frameWidth).
		Render(strings.Join(lines, "\n"))
	if lipgloss.Width(frame) > width || lipgloss.Height(frame) > height {
		view := tea.NewView(m.compactView(width, height))
		view.AltScreen = true
		return view
	}

	view := tea.NewView(lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, frame))
	view.AltScreen = true
	return view
}

func (m *Model) compactView(width, height int) string {
	var lines []string
	if m.nav.current == createProject {
		lines = m.wizard.compactLines(m.service)
	} else {
		lines = []string{strings.ToUpper(m.info.Name)}
		if m.nav.current == home {
			lines = append(lines, "> "+homeMenu[m.nav.selected].label, "↑↓/jk navigate · enter select · q quit")
		} else {
			info := secondaryScreens[m.nav.current]
			lines = append(lines, info.title, info.description, "esc back · q quit")
		}
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for i := range lines {
		lines[i] = lipgloss.NewStyle().MaxWidth(width).Render(lines[i])
	}
	return strings.Join(lines, "\n")
}

// Run owns the terminal session and returns after the user quits or an error occurs.
func Run(info application.AppInfo) error {
	directory, err := localPluginDirectory()
	if err != nil {
		return err
	}
	model, err := NewModelWithPlugins(info, directory)
	if err != nil {
		return err
	}
	defer model.preferences.stop()
	defer model.pluginBrowser.stop()
	_, err = tea.NewProgram(model).Run()
	return err
}

func localPluginDirectory() (string, error) {
	if directory := os.Getenv("PROVISIONHUB_PLUGINS"); directory != "" {
		return directory, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return adjacentPluginDirectory(executable)
}

func adjacentPluginDirectory(executable string) (string, error) {
	directory := filepath.Join(filepath.Dir(executable), "plugins")
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return directory, nil
}
