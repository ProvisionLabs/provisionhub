package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/execution"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
)

type wizardStep uint8

const (
	projectInfoStep wizardStep = iota
	backendStep
	frontendStep
	databaseStep
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
	step        wizardStep
	name        textinput.Model
	destination textinput.Model
	focused     int
	selected    [3]int
	err         string
	saving      bool
	result      application.CreateProjectResult
	preview     bool
	confirm     bool
	applying    bool
	applyResult execution.Result
	plan        planning.Plan
	planOffset  int
	width       int
	height      int
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
	w := wizard{name: name, destination: destination, width: 80, height: 24}
	cmd := w.name.Focus()
	return w, cmd
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
}

func (w *wizard) input(service application.ProjectService) application.CreateProjectInput {
	return application.CreateProjectInput{
		Name:        w.name.Value(),
		Destination: w.destination.Value(),
		Backend:     w.selectedID(service, backendStep),
		Frontend:    w.selectedID(service, frontendStep),
		Database:    w.selectedID(service, databaseStep),
	}
}

func (w *wizard) selectedID(service application.ProjectService, step wizardStep) string {
	category := stepCategory(step)
	index := w.selected[step-backendStep]
	if step != backendStep {
		if index == 0 {
			return ""
		}
		index--
	}
	options := service.Technologies(category)
	if index < 0 || index >= len(options) {
		return ""
	}
	return options[index].ID
}

func stepCategory(step wizardStep) catalog.Category {
	switch step {
	case backendStep:
		return catalog.Backend
	case frontendStep:
		return catalog.Frontend
	default:
		return catalog.Database
	}
}

func (w *wizard) optionCount(service application.ProjectService) int {
	count := len(service.Technologies(stepCategory(w.step)))
	if w.step != backendStep {
		count++ // The first optional choice is no technology.
	}
	return count
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
	switch w.step {
	case projectInfoStep:
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
			w.step = backendStep
			return nil, false
		}
		if w.focused == 0 {
			var cmd tea.Cmd
			w.name, cmd = w.name.Update(msg)
			return cmd, false
		} else {
			var cmd tea.Cmd
			w.destination, cmd = w.destination.Update(msg)
			return cmd, false
		}
	case backendStep, frontendStep, databaseStep:
		switch key {
		case "esc":
			w.step--
			if w.step == projectInfoStep {
				return w.focusField(w.focused), false
			}
		case "up", "k":
			index := w.step - backendStep
			w.selected[index] = (w.selected[index] - 1 + w.optionCount(service)) % w.optionCount(service)
		case "down", "j":
			index := w.step - backendStep
			w.selected[index] = (w.selected[index] + 1) % w.optionCount(service)
		case "enter":
			if w.step == databaseStep {
				manifest, _, err := service.ValidateInput(w.input(service))
				if err != nil {
					w.err = err.Error()
					return nil, false
				}
				w.plan, err = service.PlanManifest(manifest)
				if err != nil {
					w.err = err.Error()
				}
			}
			w.step++
		}
	case reviewStep:
		switch key {
		case "esc":
			w.step = databaseStep
		case "p":
			manifest, _, err := service.ValidateInput(w.input(service))
			if err != nil {
				w.err = err.Error()
				return nil, false
			}
			w.plan, err = service.PlanManifest(manifest)
			if err != nil {
				w.err = err.Error()
			}
			w.preview = true
			w.planOffset = 0
		case "enter":
			input := w.input(service)
			if _, _, err := service.ValidateInput(input); err != nil {
				w.err = err.Error()
				return nil, false
			}
			if w.plan.Executable {
				w.applying = true
				plan := w.plan
				return func() tea.Msg {
					created, applied, err := service.CreateAndApplyProject(input, plan)
					return projectCreatedMsg{result: created, applied: applied, err: err}
				}, false
			}
			w.saving = true
			return func() tea.Msg {
				result, err := service.CreateProject(input)
				return projectCreatedMsg{result: result, err: err}
			}, false
		}
	case resultStep:
		if key == "p" {
			plan, err := service.PlanProject(w.result.ManifestPath)
			if err != nil {
				w.err = err.Error()
				return nil, false
			}
			w.plan, w.preview, w.planOffset = plan, true, 0
			return nil, false
		}
		if key == "esc" || key == "enter" {
			return nil, true
		}
	}
	return nil, false
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
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD3FC"))
	selected := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD3FC"))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	lines := []string{
		title.Render("PROVISIONHUB · Create Project"),
		muted.Render(fmt.Sprintf("Step %d of 6 · %s", int(w.step)+1, w.stepTitle())),
		"",
		strings.Repeat("─", contentWidth),
		"",
	}
	switch w.step {
	case projectInfoStep:
		namePrefix, pathPrefix := "  ", "  "
		if w.focused == 0 {
			namePrefix = "› "
		} else {
			pathPrefix = "› "
		}
		lines = append(lines, namePrefix+"Name: "+w.name.View(), "", pathPrefix+"Destination: "+w.destination.View())
	case backendStep, frontendStep, databaseStep:
		options := service.Technologies(stepCategory(w.step))
		if w.step != backendStep {
			options = append([]catalog.Technology{{ID: "", Name: "None", Description: "Skip this optional category"}}, options...)
		}
		for index, option := range options {
			line := "  " + option.Name + " · " + option.Description
			if index == w.selected[w.step-backendStep] {
				line = selected.Render("› " + option.Name + " · " + option.Description)
			}
			lines = append(lines, line)
		}
	case reviewStep, resultStep:
		input := w.input(service)
		lines = append(lines,
			"Project:      "+input.Name,
			"Destination:  "+fitTail(input.Destination, contentWidth-len("Destination:  ")),
			"",
			"Backend:      "+service.TechnologyName(input.Backend),
			"Frontend:     "+service.TechnologyName(input.Frontend),
			"Database:     "+service.TechnologyName(input.Database),
		)
		if w.step == resultStep {
			status := "Manifest created successfully"
			if len(w.applyResult.Created) > 0 {
				status = "Project generated successfully"
			} else if w.err != "" {
				status = "Manifest saved; generation incomplete"
			}
			lines = append(lines, "", selected.Render(status), "Manifest: "+fitTail(w.result.ManifestPath, contentWidth-len("Manifest: ")))
			if len(w.applyResult.Created) > 0 {
				lines = append(lines, "", selected.Render(fmt.Sprintf("Generated %d project files", len(w.applyResult.Created))))
			} else if w.applying {
				lines = append(lines, "", "Generating project files...")
			} else {
				lines = append(lines, "", "Press p to review generation (local plugins required)")
			}
		} else if w.applying {
			lines = append(lines, "", "Generating project files...")
		} else if w.saving {
			lines = append(lines, "", "Creating manifest...")
		} else {
			lines = append(lines, "", selected.Render("[ "+w.reviewAction()+" ]"))
			if w.plan.Executable {
				files := 0
				for _, operation := range w.plan.Operations {
					if operation.Type == planning.GenerateFile {
						files++
					}
				}
				lines = append(lines, fmt.Sprintf("Creates manifest and %d project files; press p for the full plan", files))
			} else {
				lines = append(lines, "Generation unavailable for this selection; manifest only")
			}
		}
	}
	if w.err != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("#FCA5A5")).Render("Error: "+w.err))
	}
	lines = append(lines, "", strings.Repeat("─", contentWidth), w.footer())
	return lines
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

func (w *wizard) stepTitle() string {
	switch w.step {
	case projectInfoStep:
		return "Project information"
	case backendStep:
		return "Backend"
	case frontendStep:
		return "Frontend"
	case databaseStep:
		return "Database"
	case reviewStep:
		return "Review configuration"
	default:
		if len(w.applyResult.Created) > 0 {
			return "Project generated"
		}
		return "Manifest created"
	}
}

func (w *wizard) reviewAction() string {
	if w.plan.Executable {
		return "Generate Project"
	}
	return "Save Manifest Only"
}

func (w *wizard) footer() string {
	switch w.step {
	case projectInfoStep:
		return "tab/up/down field · enter next · esc back · ctrl+c quit"
	case backendStep, frontendStep, databaseStep:
		return "↑↓/jk choose · enter select · esc back · q quit"
	case reviewStep:
		return "p plan · enter " + strings.ToLower(w.reviewAction()) + " · esc back · q quit"
	default:
		return "p plan · enter/esc home · q quit"
	}
}

func (w *wizard) compactLines(service application.ProjectService) []string {
	lines := []string{"CREATE PROJECT: " + w.stepTitle()}
	switch w.step {
	case projectInfoStep:
		namePrefix, pathPrefix := "  ", "  "
		if w.focused == 0 {
			namePrefix = "> "
		} else {
			pathPrefix = "> "
		}
		lines = append(lines, namePrefix+"Name: "+w.name.Value(), pathPrefix+"Path: "+w.destination.Value())
	case backendStep, frontendStep, databaseStep:
		lines = append(lines, "> "+service.TechnologyName(w.selectedID(service, w.step)))
	case reviewStep, resultStep:
		input := w.input(service)
		lines = append(lines, input.Name+" → "+input.Destination,
			service.TechnologyName(input.Backend)+" / "+service.TechnologyName(input.Frontend)+" / "+service.TechnologyName(input.Database))
		if w.step == resultStep {
			lines = append(lines, "Saved: "+w.result.ManifestPath)
			if len(w.applyResult.Created) > 0 {
				lines = append(lines, fmt.Sprintf("Generated %d files", len(w.applyResult.Created)))
			}
		} else {
			lines = append(lines, w.reviewAction())
		}
	}
	if w.err != "" {
		lines = append(lines, "Error: "+w.err)
	}
	return append(lines, w.footer())
}
