package tui

import (
	"fmt"
	"reflect"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

var componentCategories = []catalog.Category{catalog.Backend, catalog.Frontend, catalog.Database}

func (w *wizard) componentCategoryValue() catalog.Category {
	return componentCategories[w.componentCategory]
}

func (w *wizard) beginComponentEdit(index int, service application.ProjectService) {
	w.editingComponent = index
	w.componentCategory = 0
	w.componentTechnology = 0
	w.componentName.SetValue("")
	if index >= 0 {
		component := w.components[index]
		for i, category := range componentCategories {
			if string(category) == component.Type {
				w.componentCategory = i
				break
			}
		}
		for i, technology := range service.Technologies(w.componentCategoryValue()) {
			if technology.ID == component.Technology {
				w.componentTechnology = i
				break
			}
		}
		w.componentName.SetValue(component.Name)
	}
	w.step = componentCategoryStep
}

func (w *wizard) updateComponents(msg tea.KeyPressMsg, service application.ProjectService) (tea.Cmd, bool) {
	key := msg.String()
	switch w.step {
	case componentListStep:
		switch key {
		case "esc":
			w.step = projectInfoStep
			return w.focusField(w.focused), false
		case "up", "k":
			if len(w.components) > 0 {
				w.componentCursor = (w.componentCursor - 1 + len(w.components)) % len(w.components)
			}
		case "down", "j":
			if len(w.components) > 0 {
				w.componentCursor = (w.componentCursor + 1) % len(w.components)
			}
		case "a":
			w.beginComponentEdit(-1, service)
		case "e":
			if len(w.components) > 0 {
				w.beginComponentEdit(w.componentCursor, service)
			}
		case "d":
			if len(w.components) > 0 {
				w.components = append(w.components[:w.componentCursor], w.components[w.componentCursor+1:]...)
				w.componentCursor = max(0, min(w.componentCursor, len(w.components)-1))
			}
		case "enter":
			manifest, err := service.ValidateComponentsInput(w.name.Value(), w.destination.Value(), w.components)
			if err != nil {
				w.err = err.Error()
				return nil, false
			}
			w.plan, err = service.PlanManifest(manifest)
			if err != nil {
				w.err = err.Error()
				return nil, false
			}
			w.step = reviewStep
		}
	case componentCategoryStep:
		switch key {
		case "esc":
			w.step = componentListStep
		case "up", "k":
			w.componentCategory = (w.componentCategory - 1 + len(componentCategories)) % len(componentCategories)
			w.componentTechnology = 0
		case "down", "j":
			w.componentCategory = (w.componentCategory + 1) % len(componentCategories)
			w.componentTechnology = 0
		case "enter":
			w.step = componentTechnologyStep
		}
	case componentTechnologyStep:
		options := service.Technologies(w.componentCategoryValue())
		if len(options) == 0 {
			w.err = "No technologies in this category"
			return nil, false
		}
		switch key {
		case "esc":
			w.step = componentCategoryStep
		case "up", "k":
			w.componentTechnology = (w.componentTechnology - 1 + len(options)) % len(options)
		case "down", "j":
			w.componentTechnology = (w.componentTechnology + 1) % len(options)
		case "enter":
			w.step = componentNameStep
			return w.componentName.Focus(), false
		}
	case componentNameStep:
		switch key {
		case "esc":
			w.componentName.Blur()
			w.step = componentTechnologyStep
		case "enter":
			name := w.componentName.Value()
			if err := project.ValidateName(name); err != nil {
				w.err = fmt.Sprintf("component name: %v", err)
				return nil, false
			}
			for i, component := range w.components {
				if i != w.editingComponent && component.Name == name {
					w.err = fmt.Sprintf("duplicate component name %q", name)
					return nil, false
				}
			}
			options := service.Technologies(w.componentCategoryValue())
			component := project.Component{Name: name, Type: string(w.componentCategoryValue()), Technology: options[w.componentTechnology].ID}
			if w.editingComponent >= 0 {
				w.components[w.editingComponent] = component
				w.componentCursor = w.editingComponent
			} else {
				w.components = append(w.components, component)
				w.componentCursor = len(w.components) - 1
			}
			w.componentName.Blur()
			w.step = componentListStep
		default:
			var cmd tea.Cmd
			w.componentName, cmd = w.componentName.Update(msg)
			return cmd, false
		}
	case reviewStep:
		switch key {
		case "esc":
			w.step = componentListStep
		case "p":
			w.preview = true
			w.planOffset = 0
		case "enter":
			components := append([]project.Component{}, w.components...)
			name, destination := w.name.Value(), w.destination.Value()
			manifest, err := service.ValidateComponentsInput(name, destination, components)
			if err != nil {
				w.err = err.Error()
				return nil, false
			}
			fresh, err := service.PlanManifest(manifest)
			if err != nil || !sameReviewedPlan(w.plan, fresh) {
				w.err = "generation plan changed; review a new plan"
				return nil, false
			}
			if w.plan.Executable {
				w.applying = true
				plan := w.plan
				return func() tea.Msg {
					created, applied, err := service.CreateAndApplyComponentProject(name, destination, components, plan)
					return projectCreatedMsg{result: created, applied: applied, err: err}
				}, false
			}
			w.saving = true
			return func() tea.Msg {
				result, err := service.CreateComponentProject(name, destination, components)
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
		} else if key == "esc" || key == "enter" {
			return nil, true
		}
	}
	return nil, false
}

func (w *wizard) componentLines(service application.ProjectService, contentWidth int) []string {
	lines := []string{"PROVISIONHUB · Create Project", "Named components · " + w.name.Value(), "", strings.Repeat("─", contentWidth), ""}
	switch w.step {
	case componentListStep:
		lines = append(lines, "Components:")
		for i, component := range w.components {
			prefix := "  "
			if i == w.componentCursor {
				prefix = "> "
			}
			lines = append(lines, fmt.Sprintf("%s%s  %s  %s", prefix, component.Name, component.Type, service.TechnologyName(component.Technology)))
		}
		if len(w.components) == 0 {
			lines = append(lines, "  No components yet")
		}
		lines = append(lines, "", "a add · e edit · d remove · enter review · esc back")
	case componentCategoryStep:
		lines = append(lines, "Choose component category:")
		for i, category := range componentCategories {
			prefix := "  "
			if i == w.componentCategory {
				prefix = "> "
			}
			lines = append(lines, prefix+string(category))
		}
	case componentTechnologyStep:
		lines = append(lines, "Choose "+string(w.componentCategoryValue())+" technology:")
		for i, technology := range service.Technologies(w.componentCategoryValue()) {
			prefix := "  "
			if i == w.componentTechnology {
				prefix = "> "
			}
			lines = append(lines, prefix+technology.Name+" · "+technology.Description)
		}
	case componentNameStep:
		lines = append(lines, "Component name:", "> "+w.componentName.View())
	case reviewStep, resultStep:
		lines = append(lines, "Project: "+w.name.Value(), "Destination: "+fitTail(w.destination.Value(), contentWidth-13), "", "Components:")
		for _, component := range w.components {
			lines = append(lines, fmt.Sprintf("  %s  %s  %s", component.Name, component.Type, service.TechnologyName(component.Technology)))
		}
		if w.step == reviewStep {
			lines = append(lines, "", "[ "+w.reviewAction()+" ]", "p preview plan · enter approve · esc edit")
		} else {
			status := "Manifest created successfully"
			if len(w.applyResult.Created) > 0 {
				status = "Project generated successfully"
			}
			lines = append(lines, "", status, "Manifest: "+fitTail(w.result.ManifestPath, contentWidth-10), "p plan · enter/esc home")
		}
	}
	if w.err != "" {
		lines = append(lines, "", "Error: "+w.err)
	}
	return append(lines, "", strings.Repeat("─", contentWidth))
}

func (w *wizard) compactComponentLines(service application.ProjectService) []string {
	return w.componentLines(service, 32)
}

func sameReviewedPlan(a, b planning.Plan) bool { return reflect.DeepEqual(a, b) }
