package tui

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

type configurationRequest struct{ component project.Component }
type configurationLoadedMsg struct {
	request *configurationRequest
	schema  configuration.Schema
	err     error
}

func (w *wizard) beginConfiguration(component project.Component, service application.ProjectService) tea.Cmd {
	if component.Technology == w.configurationDraft.Technology {
		component.Configuration = maps.Clone(w.configurationDraft.Configuration)
	}
	w.configurationDraft = component
	w.step = componentConfigurationStep
	if w.configurationRaw != nil && w.configurationTechnology == component.Technology {
		return w.focusConfiguration()
	}
	request := &configurationRequest{component: component}
	w.configurationRequest = request
	return func() tea.Msg {
		schema, err := service.ConfigurationSchema(component)
		return configurationLoadedMsg{request: request, schema: schema, err: err}
	}
}

func (w *wizard) configurationLoaded(msg configurationLoadedMsg) tea.Cmd {
	if msg.request == nil || w.step != componentConfigurationStep || msg.request != w.configurationRequest {
		return nil
	}
	w.configurationRequest = nil
	if msg.err != nil {
		w.err = msg.err.Error()
		w.step = componentNameStep
		return w.componentName.Focus()
	}
	w.configurationTechnology = w.configurationDraft.Technology
	return w.openConfiguration(msg.schema)
}

func (w *wizard) openConfiguration(schema configuration.Schema) tea.Cmd {
	w.configurationSchema = schema
	w.configurationKeys = schema.Keys()
	w.configurationRaw = map[string]string{}
	w.configurationPresent = map[string]bool{}
	w.configurationCursor = 0
	for _, key := range w.configurationKeys {
		property := schema.Properties[key]
		value, present := w.configurationDraft.Configuration[key]
		if !present && property.Default != nil {
			value, present = property.Default, true
		}
		if present {
			w.configurationRaw[key] = fmt.Sprint(value)
			w.configurationPresent[key] = true
		}
	}
	if len(w.configurationKeys) == 0 {
		values, err := configuration.Resolve(schema, w.configurationDraft.Configuration, w.configurationDraft.Name)
		if err != nil {
			w.err = err.Error()
			return w.backFromConfiguration()
		}
		w.saveConfiguredComponent(values)
		return nil
	}
	return w.focusConfiguration()
}

func (w *wizard) focusConfiguration() tea.Cmd {
	w.configurationInput = textinput.New()
	w.configurationInput.Prompt = ""
	w.configurationInput.CharLimit = 1 << 20
	w.configurationInput.SetWidth(max(1, min(w.width-4, 48)))
	w.configurationInput.SetValue(w.configurationRaw[w.configurationKeys[w.configurationCursor]])
	p := w.configurationSchema.Properties[w.configurationKeys[w.configurationCursor]]
	if p.Type == "string" || p.Type == "integer" {
		return w.configurationInput.Focus()
	}
	return nil
}

func (w *wizard) saveConfiguredComponent(values configuration.Values) {
	if w.configuringConnection {
		w.saveConnection(values)
		return
	}
	component := w.configurationDraft
	component.Configuration = values
	if w.editingComponent >= 0 {
		previous := w.components[w.editingComponent].Name
		for i := range w.connections {
			if w.connections[i].From == previous {
				w.connections[i].From = component.Name
			}
			if w.connections[i].To == previous {
				w.connections[i].To = component.Name
			}
		}
		w.components[w.editingComponent] = component
		w.componentCursor = w.editingComponent
	} else {
		w.components = append(w.components, component)
		w.componentCursor = len(w.components) - 1
	}
	w.configurationInput.Blur()
	w.step = componentListStep
}

func (w *wizard) configurationValues() (configuration.Values, error) {
	values := maps.Clone(w.configurationDraft.Configuration)
	if values == nil {
		values = configuration.Values{}
	}
	for _, key := range w.configurationKeys {
		if !w.configurationPresent[key] {
			delete(values, key)
			continue
		}
		v, err := w.configurationSchema.Properties[key].Parse(w.configurationRaw[key])
		if err != nil {
			return nil, fmt.Errorf("configuration.%s: %w", key, err)
		}
		values[key] = v
	}
	return configuration.Resolve(w.configurationSchema, values, w.configurationDraft.Name)
}

func (w *wizard) updateConfiguration(msg tea.KeyPressMsg) tea.Cmd {
	if w.configurationRequest != nil {
		if msg.String() == "esc" {
			w.configurationRequest = nil
			w.step = componentNameStep
			return w.componentName.Focus()
		}
		return nil
	}
	key := w.configurationKeys[w.configurationCursor]
	p := w.configurationSchema.Properties[key]
	switch msg.String() {
	case "esc":
		return w.backFromConfiguration()
	case "ctrl+u":
		delete(w.configurationPresent, key)
		w.configurationRaw[key] = ""
		w.configurationInput.SetValue("")
	case "left", "right", "space":
		if p.Type == "boolean" || p.Type == "enum" {
			options := p.Options
			if p.Type == "boolean" {
				options = []string{"false", "true"}
			}
			i := slices.Index(options, w.configurationRaw[key])
			delta := 1
			if msg.String() == "left" {
				delta = -1
			}
			if i < 0 {
				i = 0
			} else {
				i = (i + delta + len(options)) % len(options)
			}
			w.configurationRaw[key] = options[i]
			w.configurationPresent[key] = true
			return nil
		}
		var cmd tea.Cmd
		w.configurationInput, cmd = w.configurationInput.Update(msg)
		w.configurationRaw[key] = w.configurationInput.Value()
		if msg.String() == "space" {
			w.configurationPresent[key] = true
		}
		return cmd
	case "enter", "tab", "down", "shift+tab", "up":
		if w.configurationPresent[key] {
			if _, err := p.Parse(w.configurationRaw[key]); err != nil {
				w.err = fmt.Sprintf("configuration.%s: %v", key, err)
				return nil
			}
		} else if p.Required && p.Default == nil {
			w.err = "configuration." + key + ": required value is missing"
			return nil
		}
		if msg.String() == "up" || msg.String() == "shift+tab" {
			w.configurationCursor = max(0, w.configurationCursor-1)
		} else if w.configurationCursor < len(w.configurationKeys)-1 {
			w.configurationCursor++
		} else {
			values, err := w.configurationValues()
			if err != nil {
				w.err = err.Error()
				return nil
			}
			w.saveConfiguredComponent(values)
			return nil
		}
		return w.focusConfiguration()
	default:
		if p.Type == "string" || p.Type == "integer" {
			var cmd tea.Cmd
			w.configurationInput, cmd = w.configurationInput.Update(msg)
			w.configurationRaw[key] = w.configurationInput.Value()
			w.configurationPresent[key] = true
			return cmd
		}
	}
	return nil
}

func (w *wizard) configurationLines() []string {
	if w.configurationRequest != nil {
		return []string{"Loading component configuration…", "esc back"}
	}
	key := w.configurationKeys[w.configurationCursor]
	p := w.configurationSchema.Properties[key]
	label := p.Label
	if label == "" {
		label = key
	}
	lines := []string{fmt.Sprintf("Configure %s · %d/%d", w.configurationDraft.Name, w.configurationCursor+1, len(w.configurationKeys)), label + " (" + key + ", " + p.Type + ")", p.Description}
	if p.Default != nil {
		value, _ := json.Marshal(p.Default)
		lines = append(lines, fmt.Sprintf("Default: %s", value))
	}
	if p.Required {
		lines = append(lines, "Required")
	} else {
		lines = append(lines, "Optional · ctrl+u unset/use default")
	}
	if p.Type == "boolean" || p.Type == "enum" {
		options := p.Options
		if p.Type == "boolean" {
			options = []string{"false", "true"}
		}
		for _, option := range options {
			prefix := "[ ] "
			if w.configurationPresent[key] && option == w.configurationRaw[key] {
				prefix = "[x] "
			}
			lines = append(lines, prefix+option)
		}
		lines = append(lines, "left/right/space select")
	} else {
		lines = append(lines, "> "+w.configurationInput.View())
	}
	return append(lines, "enter/tab next or save · up previous · esc name")
}

func configurationSummary(values configuration.Values) []string {
	if len(values) == 0 {
		return nil
	}
	data, _ := json.Marshal(values)
	return []string{"    Configuration: " + string(data)}
}

// Keep the active control and its validation error visible on small terminals.
func (w *wizard) compactConfigurationLines() []string {
	if w.configurationRequest != nil {
		return []string{"Loading configuration…", "esc back"}
	}
	key := w.configurationKeys[w.configurationCursor]
	p := w.configurationSchema.Properties[key]
	label := p.Label
	if label == "" {
		label = key
	}
	value := w.configurationRaw[key]
	if p.Type == "string" || p.Type == "integer" {
		value = w.configurationInput.View()
	}
	lines := []string{label, "> " + value}
	if w.err != "" {
		lines = append(lines, "Error: "+w.err)
	}
	return append(lines, fmt.Sprintf("%s %d/%d", w.configurationDraft.Name, w.configurationCursor+1, len(w.configurationKeys)), "enter next · esc back")
}
