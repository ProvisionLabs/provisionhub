package tui

import (
	"fmt"
	"maps"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/connections"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

type connectionRequest struct{ spec project.Specification }
type connectionOptionsMsg struct {
	request *connectionRequest
	options []connections.Option
	err     error
}

func (w *wizard) specification(service application.ProjectService) (project.Specification, error) {
	spec, err := service.ValidateComponentsInput(w.name.Value(), w.destination.Value(), w.components)
	if err != nil {
		return project.Specification{}, err
	}
	spec.Spec.Connections = slices.Clone(w.connections)
	for i := range spec.Spec.Connections {
		spec.Spec.Connections[i].Configuration = maps.Clone(spec.Spec.Connections[i].Configuration)
	}
	if err := spec.Validate(); err != nil {
		return project.Specification{}, err
	}
	return spec, nil
}

func connectionLabel(c project.Connection) string {
	return fmt.Sprintf("%s -> %s (%s)", c.From, c.To, c.Type)
}

func (w *wizard) beginConnectionEdit(index int, service application.ProjectService) tea.Cmd {
	spec, err := w.specification(service)
	if err != nil {
		w.err = err.Error()
		return nil
	}
	w.editingConnection = index
	w.connectionDraft = project.Connection{}
	if index >= 0 {
		w.connectionDraft = spec.Spec.Connections[index]
		spec.Spec.Connections = append(spec.Spec.Connections[:index], spec.Spec.Connections[index+1:]...)
	}
	w.configurationRaw = nil
	w.configurationTechnology = ""
	w.configuringConnection = false
	w.connectionOptions = nil
	w.connectionOptionCursor = 0
	w.step = connectionTargetStep
	request := &connectionRequest{spec: spec}
	w.connectionRequest = request
	return func() tea.Msg {
		options, err := service.ConnectionOptions(spec)
		return connectionOptionsMsg{request: request, options: options, err: err}
	}
}

func (w *wizard) connectionOptionsLoaded(msg connectionOptionsMsg) tea.Cmd {
	if msg.request == nil || msg.request != w.connectionRequest || w.step != connectionTargetStep {
		return nil
	}
	w.connectionRequest = nil
	if msg.err != nil {
		w.err = msg.err.Error()
		w.step = connectionListStep
		return nil
	}
	if len(msg.options) == 0 {
		w.err = "No compatible unused connection targets; check component capabilities and existing connections"
		w.step = connectionListStep
		return nil
	}
	w.connectionOptions = msg.options
	for i, option := range w.connectionOptions {
		if option.Connection.ID() == w.connectionDraft.ID() {
			w.connectionOptionCursor = i
			break
		}
	}
	return nil
}

func (w *wizard) updateConnections(msg tea.KeyPressMsg, service application.ProjectService) tea.Cmd {
	if w.step == connectionListStep {
		switch msg.String() {
		case "esc", "enter":
			w.step = componentListStep
		case "a":
			return w.beginConnectionEdit(-1, service)
		case "e":
			if len(w.connections) > 0 {
				return w.beginConnectionEdit(w.connectionCursor, service)
			}
		case "d":
			if len(w.connections) > 0 {
				w.connections = append(w.connections[:w.connectionCursor], w.connections[w.connectionCursor+1:]...)
				w.connectionCursor = max(0, min(w.connectionCursor, len(w.connections)-1))
			}
		case "up", "k":
			if len(w.connections) > 0 {
				w.connectionCursor = (w.connectionCursor - 1 + len(w.connections)) % len(w.connections)
			}
		case "down", "j":
			if len(w.connections) > 0 {
				w.connectionCursor = (w.connectionCursor + 1) % len(w.connections)
			}
		}
		return nil
	}
	if msg.String() == "esc" {
		w.connectionRequest = nil
		w.step = connectionListStep
		return nil
	}
	if w.connectionRequest != nil || len(w.connectionOptions) == 0 {
		return nil
	}
	switch msg.String() {
	case "up", "k":
		w.connectionOptionCursor = (w.connectionOptionCursor - 1 + len(w.connectionOptions)) % len(w.connectionOptions)
	case "down", "j":
		w.connectionOptionCursor = (w.connectionOptionCursor + 1) % len(w.connectionOptions)
	case "enter":
		option := w.connectionOptions[w.connectionOptionCursor]
		connection := option.Connection
		if connection.ID() == w.connectionDraft.ID() {
			connection.Configuration = maps.Clone(w.connectionDraft.Configuration)
		}
		w.connectionDraft = connection
		w.configuringConnection = true
		w.configurationDraft = project.Component{Name: "connection " + connection.ID(), Configuration: maps.Clone(connection.Configuration)}
		w.step = componentConfigurationStep
		identity := "connection/" + connection.ID()
		if w.configurationRaw != nil && w.configurationTechnology == identity {
			return w.focusConfiguration()
		}
		w.configurationTechnology = identity
		return w.openConfiguration(option.Schema)
	}
	return nil
}

func (w *wizard) backFromConfiguration() tea.Cmd {
	w.configurationInput.Blur()
	if w.configuringConnection {
		w.step = connectionTargetStep
		return nil
	}
	w.step = componentNameStep
	return w.componentName.Focus()
}

func (w *wizard) saveConnection(values configuration.Values) {
	w.connectionDraft.Configuration = values
	if w.editingConnection >= 0 {
		w.connections[w.editingConnection] = w.connectionDraft
		w.connectionCursor = w.editingConnection
	} else {
		w.connections = append(w.connections, w.connectionDraft)
		w.connectionCursor = len(w.connections) - 1
	}
	w.configuringConnection = false
	w.configurationInput.Blur()
	w.step = connectionListStep
}

func (w *wizard) removeComponentConnections(name string) {
	before := len(w.connections)
	w.connections = slices.DeleteFunc(w.connections, func(c project.Connection) bool { return c.From == name || c.To == name })
	w.connectionCursor = max(0, min(w.connectionCursor, len(w.connections)-1))
	if removed := before - len(w.connections); removed > 0 {
		w.err = fmt.Sprintf("Removed %d connection(s) for deleted component %s", removed, name)
	}
}

func (w *wizard) connectionLines() []string {
	lines := []string{"Connections:"}
	if w.step == connectionListStep {
		for i, c := range w.connections {
			prefix := "  "
			if i == w.connectionCursor {
				prefix = "> "
			}
			lines = append(lines, prefix+connectionLabel(c))
			lines = append(lines, configurationSummary(c.Configuration)...)
		}
		if len(w.connections) == 0 {
			lines = append(lines, "  No connections yet")
		}
		return append(lines, "", "a add · e edit · d remove · enter/esc components")
	}
	if w.connectionRequest != nil {
		return append(lines, "Loading compatible targets…", "esc cancel")
	}
	lines = append(lines, "Choose source -> target:")
	pageSize := max(1, w.height-14)
	start := max(0, w.connectionOptionCursor-pageSize+1)
	for i := start; i < min(len(w.connectionOptions), start+pageSize); i++ {
		prefix := "  "
		if i == w.connectionOptionCursor {
			prefix = "> "
		}
		lines = append(lines, prefix+connectionLabel(w.connectionOptions[i].Connection))
	}
	if len(w.connectionOptions) > 0 {
		lines = append(lines, w.connectionOptions[w.connectionOptionCursor].Description)
	}
	return append(lines, "", "up/down choose · enter configure · esc back")
}

func (w *wizard) compactConnectionLines() []string {
	lines := []string{"Connections"}
	if w.connectionRequest != nil {
		lines = append(lines, "Loading targets…")
	} else if w.step == connectionTargetStep && len(w.connectionOptions) > 0 {
		lines = append(lines, "> "+connectionLabel(w.connectionOptions[w.connectionOptionCursor].Connection))
	} else if len(w.connections) > 0 {
		lines = append(lines, "> "+connectionLabel(w.connections[w.connectionCursor]))
	} else {
		lines = append(lines, "No connections yet")
	}
	if w.err != "" {
		lines = append(lines, "Error: "+w.err)
	}
	return append(lines, "a/e/d edit · enter select · esc back")
}
