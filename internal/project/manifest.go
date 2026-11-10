package project

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/configuration"
)

const (
	APIVersion = "provisionhub.dev/v1alpha1"
	Kind       = "Project"
)

// Manifest is the existing in-memory project model. It is never written as
// project metadata during generation.
type Manifest struct {
	APIVersion string
	Kind       string
	Metadata   Metadata
	Spec       Spec
}

// Specification names the in-memory input used by planning and execution.
type Specification = Manifest

type Metadata struct {
	Name string
}

type TechnologySelection struct {
	Technology string
}

type Spec struct {
	Backend     *TechnologySelection
	Frontend    *TechnologySelection
	Database    *TechnologySelection
	Components  []Component
	Connections []Connection
}

// Component is one named instance of a catalog technology. Legacy is internal
// normalization metadata and is never serialized in a project manifest.
type Component struct {
	Configuration configuration.Values
	Name          string
	Type          string
	Technology    string
	Legacy        bool
}

func (c Component) ID() string { return "component/" + c.Name }

type Connection struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

func (s Spec) usesComponents() bool {
	return s.Components != nil || s.Connections != nil
}

func (m Manifest) UsesComponents() bool { return m.Spec.usesComponents() }

// CanonicalComponents is the common instance model for legacy and new manifests.
// Names, rather than YAML positions, determine the order and identity.
func (m Manifest) CanonicalComponents() []Component {
	if m.Spec.usesComponents() {
		components := slices.Clone(m.Spec.Components)
		for i := range components {
			components[i].Configuration = maps.Clone(components[i].Configuration)
		}
		slices.SortFunc(components, func(a, b Component) int { return strings.Compare(a.Name, b.Name) })
		return components
	}
	components := []Component{}
	if m.Spec.Backend != nil {
		components = append(components, Component{Name: "backend", Type: "backend", Technology: m.Spec.Backend.Technology, Legacy: true})
	}
	if m.Spec.Frontend != nil {
		components = append(components, Component{Name: "frontend", Type: "frontend", Technology: m.Spec.Frontend.Technology, Legacy: true})
	}
	if m.Spec.Database != nil {
		components = append(components, Component{Name: "database", Type: "database", Technology: m.Spec.Database.Technology, Legacy: true})
	}
	return components
}

func (m Manifest) CanonicalConnections() []Connection {
	connections := slices.Clone(m.Spec.Connections)
	slices.SortFunc(connections, func(a, b Connection) int {
		return strings.Compare(a.From+"\x00"+a.To+"\x00"+a.Type, b.From+"\x00"+b.To+"\x00"+b.Type)
	})
	return connections
}

// ValidateName accepts a portable, lowercase project identifier.
func ValidateName(name string) error {
	if name == "" || len(name) > 63 {
		return errors.New("project name must contain 1 to 63 lowercase letters, digits, or hyphens")
	}
	if name[0] == '-' || name[len(name)-1] == '-' {
		return errors.New("project name must start and end with a letter or digit")
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return fmt.Errorf("project name %q contains an unsupported character; use lowercase letters, digits, and hyphens", name)
		}
	}
	switch strings.ToUpper(name) {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return fmt.Errorf("name %q is reserved on Windows", name)
	}
	return nil
}

// Validate checks the manifest's version and required structural fields.
// Technology membership is checked by the application catalog.
func (m Manifest) Validate() error {
	if m.APIVersion != APIVersion {
		return fmt.Errorf("unsupported apiVersion %q (expected %q)", m.APIVersion, APIVersion)
	}
	if m.Kind != Kind {
		return fmt.Errorf("unsupported kind %q (expected %q)", m.Kind, Kind)
	}
	if err := ValidateName(m.Metadata.Name); err != nil {
		return err
	}
	if m.Spec.usesComponents() {
		if m.Spec.Backend != nil || m.Spec.Frontend != nil || m.Spec.Database != nil {
			return errors.New("spec.components cannot be mixed with legacy backend, frontend, or database fields")
		}
		if len(m.Spec.Components) == 0 {
			return errors.New("spec.components must contain at least one component")
		}
		names := map[string]bool{}
		for _, component := range m.Spec.Components {
			if err := ValidateName(component.Name); err != nil {
				return fmt.Errorf("component name: %w", err)
			}
			if names[component.Name] {
				return fmt.Errorf("duplicate component name %q", component.Name)
			}
			names[component.Name] = true
			if component.Type != "backend" && component.Type != "frontend" && component.Type != "database" {
				return fmt.Errorf("component %q has unsupported type %q", component.Name, component.Type)
			}
			if strings.TrimSpace(component.Technology) == "" {
				return fmt.Errorf("component %q requires a technology", component.Name)
			}
		}
		connections := map[Connection]bool{}
		for _, connection := range m.Spec.Connections {
			if !names[connection.From] || !names[connection.To] {
				return fmt.Errorf("connection %q to %q references an unknown component", connection.From, connection.To)
			}
			if connection.From == connection.To {
				return fmt.Errorf("connection %q cannot target itself", connection.From)
			}
			if connection.Type != "http" {
				return fmt.Errorf("unsupported connection type %q", connection.Type)
			}
			if connections[connection] {
				return fmt.Errorf("duplicate connection from %q to %q", connection.From, connection.To)
			}
			connections[connection] = true
		}
		return nil
	}
	if m.Spec.Backend == nil || strings.TrimSpace(m.Spec.Backend.Technology) == "" {
		return errors.New("spec.backend.technology is required")
	}
	if m.Spec.Frontend != nil && strings.TrimSpace(m.Spec.Frontend.Technology) == "" {
		return errors.New("spec.frontend.technology must be omitted or nonempty")
	}
	if m.Spec.Database != nil && strings.TrimSpace(m.Spec.Database.Technology) == "" {
		return errors.New("spec.database.technology must be omitted or nonempty")
	}
	return nil
}
