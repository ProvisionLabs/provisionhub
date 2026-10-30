package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/execution"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

type wizardStep uint8

const (
	projectInfoStep wizardStep = iota
	componentListStep
	componentCategoryStep
	componentTechnologyStep
	componentNameStep
	reviewStep
	resultStep
)

type projectCreatedMsg struct {
	result  application.CreateProjectResult
	applied execution.Result
	err     error
}

type projectAppliedMsg struct {
	result execution.Result
	err    error
}

type wizard struct {
	step                wizardStep
	name                textinput.Model
	destination         textinput.Model
	focused             int
	err                 string
	saving              bool
	result              application.CreateProjectResult
	preview             bool
	confirm             bool
	applying            bool
	applyResult         execution.Result
	plan                planning.Plan
	planOffset          int
	width               int
	height              int
	components          []project.Component
	componentCursor     int
	componentCategory   int
	componentTechnology int
	componentName       textinput.Model
	editingComponent    int
}

func newWizard() (wizard, tea.Cmd) {
	name := textinput.New()
	name.Prompt = ""
	name.Placeholder = "my-application"
	name.CharLimit = 63
	name.SetWidth(38)
	destination := textinput.New()
	destination.Prompt = ""
	destination.Placeholder = "./my-application"
	destination.SetWidth(38)
	componentName := textinput.New()
	componentName.Prompt = ""
	componentName.Placeholder = "api"
	componentName.CharLimit = 63
	componentName.SetWidth(38)
	w := wizard{name: name, destination: destination, componentName: componentName, editingComponent: -1, width: 80, height: 24}
	return w, w.name.Focus()
}

func (w *wizard) resize(width, height int) {
	w.width, w.height = max(1, width), max(1, height)
	if w.preview {
		w.clampPlanOffset()
	}
	frameWidth := min(width-4, 72)
	inputWidth := max(4, min(frameWidth-24, 48))
	w.name.SetWidth(inputWidth)
	w.destination.SetWidth(inputWidth)
	w.componentName.SetWidth(inputWidth)
}

func (w *wizard) focusField(index int) tea.Cmd {
	w.focused = index
	w.name.Blur()
	w.destination.Blur()
	if index == 0 {
		return w.name.Focus()
	}
	return w.destination.Focus()
}

// update returns a command and whether the wizard should return to Home.
func (w *wizard) update(msg tea.KeyPressMsg, service application.ProjectService) (tea.Cmd, bool) {
	if w.saving || w.applying {
		return nil, false
	}
	w.err = ""
	key := msg.String()
	if w.preview {
		if w.confirm {
			switch key {
			case "esc", "n":
				w.confirm = false
			case "y":
				w.confirm = false
				w.applying = true
				file, plan := w.result.ManifestPath, w.plan
				return func() tea.Msg {
					result, err := service.ApplyProject(file, plan)
					return projectAppliedMsg{result: result, err: err}
				}, false
			}
			return nil, false
		}
		switch key {
		case "esc", "enter", "p":
			w.preview = false
		case "g":
			if w.step == resultStep && w.plan.Executable && w.applyResult.Created == nil {
				if err := service.PreflightProject(w.result.ManifestPath, w.plan); err != nil {
					w.err = err.Error()
				} else {
					w.confirm = true
				}
			}
		case "down", "j":
			w.planOffset++
		case "up", "k":
			w.planOffset--
		}
		w.clampPlanOffset()
		return nil, false
	}
	if w.step != projectInfoStep {
		return w.updateComponents(msg, service)
	}
	switch key {
	case "esc":
		return nil, true
	case "tab", "down":
		return w.focusField(1), false
	case "shift+tab", "up":
		return w.focusField(0), false
	case "enter":
		if w.focused == 0 {
			return w.focusField(1), false
		}
		if _, err := service.ValidateProjectInfo(w.name.Value(), w.destination.Value()); err != nil {
			w.err = err.Error()
			return nil, false
		}
		w.name.Blur()
		w.destination.Blur()
		w.step = componentListStep
		return nil, false
	}
	if w.focused == 0 {
		var cmd tea.Cmd
		w.name, cmd = w.name.Update(msg)
		return cmd, false
	}
	var cmd tea.Cmd
	w.destination, cmd = w.destination.Update(msg)
	return cmd, false
}

func (w *wizard) applied(msg projectAppliedMsg) {
	w.applying = false
	w.preview = false
	if msg.err != nil {
		w.err = msg.err.Error()
		return
	}
	w.applyResult = msg.result
	w.err = ""
}

func (w *wizard) created(msg projectCreatedMsg) {
	w.saving = false
	w.applying = false
	if msg.result.ManifestPath == "" {
		if msg.err != nil {
			w.err = msg.err.Error()
		}
		return
	}
	w.result = msg.result
	w.applyResult = msg.applied
	w.step = resultStep
	if msg.err != nil {
		w.err = msg.err.Error()
		return
	}
	w.err = ""
}

func (w *wizard) lines(service application.ProjectService, contentWidth int) []string {
	if w.step != projectInfoStep {
		return w.componentLines(service, contentWidth)
	}
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD3FC"))
	namePrefix, pathPrefix := "  ", "  "
	if w.focused == 0 {
		namePrefix = "› "
	} else {
		pathPrefix = "› "
	}
	lines := []string{
		title.Render("PROVISIONHUB · Create Project"),
		muted.Render("Project information"),
		"", strings.Repeat("─", contentWidth), "",
		namePrefix + "Name: " + w.name.View(),
		"", pathPrefix + "Destination: " + w.destination.View(),
	}
	if w.err != "" {
		lines = append(lines, "", "Error: "+w.err)
	}
	return append(lines, "", strings.Repeat("─", contentWidth), "tab/up/down field · enter components · esc back · ctrl+c quit")
}

func fitTail(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width("…"+string(runes)) > width {
		runes = runes[1:]
	}
	return "…" + string(runes)
}

func (w *wizard) reviewAction() string {
	if w.plan.Executable {
		return "Generate Project"
	}
	return "Save Manifest Only"
}

func (w *wizard) compactLines(service application.ProjectService) []string {
	if w.step != projectInfoStep {
		return w.compactComponentLines(service)
	}
	namePrefix, pathPrefix := "  ", "  "
	if w.focused == 0 {
		namePrefix = "> "
	} else {
		pathPrefix = "> "
	}
	lines := []string{"Create Project: Project information", namePrefix + "Name: " + w.name.Value(), pathPrefix + "Path: " + w.destination.Value()}
	if w.err != "" {
		lines = append(lines, "Error: "+w.err)
	}
	return append(lines, "enter components · esc back")
}
