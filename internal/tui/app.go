package tui

import (
	"strings"

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

var secondaryScreens = map[screen]screenText{
	openProject: {
		title:       "Open Existing Project",
		description: "Opening and editing local projects is planned. No manifests are read yet.",
	},
	pluginMarketplace: {
		title:       "Plugin Marketplace",
		description: "Plugin discovery and installation are planned. No marketplace is connected.",
	},
	settings: {
		title:       "Settings",
		description: "Application settings are planned. No settings are stored yet.",
	},
}

// Model contains terminal presentation state; application metadata comes from
// the shared application layer.
type Model struct {
	info    application.AppInfo
	service application.ProjectService
	nav     navigation
	wizard  wizard
	help    help.Model
	width   int
	height  int
}

func NewModel(info application.AppInfo) *Model {
	h := help.New()
	h.SetWidth(48)
	return &Model{info: info, service: application.NewProjectService(), help: h, width: 80, height: 24}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetWidth(max(1, min(msg.Width-4, 54)-6))
		m.wizard.resize(msg.Width, msg.Height)
	case projectCreatedMsg:
		m.wizard.created(msg)
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.nav.current == createProject && m.wizard.step == projectInfoStep {
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
			if m.nav.current == createProject {
				var cmd tea.Cmd
				m.wizard, cmd = newWizard()
				m.wizard.resize(m.width, m.height)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m *Model) View() tea.View {
	width, height := max(1, m.width), max(1, m.height)
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
		runes := []rune(lines[i])
		if len(runes) > width {
			lines[i] = string(runes[:width])
		}
	}
	return strings.Join(lines, "\n")
}

// Run owns the terminal session and returns after the user quits or an error occurs.
func Run(info application.AppInfo) error {
	_, err := tea.NewProgram(NewModel(info)).Run()
	return err
}
