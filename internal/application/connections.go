package application

import (
	"fmt"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/connections"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

// ConnectionOptions discovers capabilities using the same integrity/version
// checks as planning. The wizard calls this service from a Bubble Tea command.
func (s ProjectService) ConnectionOptions(spec project.Specification) ([]connections.Option, error) {
	resolved, err := s.ConfigureSpecification(spec, nil)
	if err != nil {
		return nil, err
	}
	loaded, err := s.loadPluginsFor(resolved)
	if err != nil {
		return nil, err
	}
	return connections.Options(resolved.CanonicalComponents(), resolved.CanonicalConnections(), loaded), nil
}

// ConfigureConnections parses source:target:type.key=value using the selected
// consumer's schema. Resolution is shared with the planner and TUI.
func (s ProjectService) ConfigureConnections(spec project.Specification, settings []string) (project.Specification, error) {
	if err := spec.Validate(); err != nil {
		return project.Specification{}, err
	}
	if len(spec.Spec.Connections) == 0 && len(settings) == 0 {
		return spec, nil
	}
	loaded, err := s.loadPluginsFor(spec)
	if err != nil {
		return project.Specification{}, err
	}
	declarations := spec.CanonicalConnections()
	components := spec.CanonicalComponents()
	byID := map[string]int{}
	for i, c := range declarations {
		byID[c.ID()] = i
	}
	seen := map[string]bool{}
	for _, setting := range settings {
		left, raw, hasValue := strings.Cut(setting, "=")
		id, key, hasKey := strings.Cut(left, ".")
		if !hasValue || !hasKey || key == "" {
			return project.Specification{}, fmt.Errorf("invalid connection setting %q; expected source:target:type.key=value", setting)
		}
		i, ok := byID[id]
		if !ok {
			return project.Specification{}, fmt.Errorf("setting references unknown connection %q", id)
		}
		if seen[left] {
			return project.Specification{}, fmt.Errorf("duplicate connection setting %s", left)
		}
		seen[left] = true
		match, err := connections.Select(components, declarations[i], loaded)
		if err != nil {
			return project.Specification{}, err
		}
		property, ok := match.Consumer.Configuration.Properties[key]
		if !ok {
			return project.Specification{}, fmt.Errorf("connection %s configuration.%s: unknown key", id, key)
		}
		value, err := property.Parse(raw)
		if err != nil {
			return project.Specification{}, fmt.Errorf("connection %s configuration.%s: %w", id, key, err)
		}
		if declarations[i].Configuration == nil {
			declarations[i].Configuration = configuration.Values{}
		}
		declarations[i].Configuration[key] = value
	}
	resolved, err := connections.Resolve(components, declarations, loaded)
	if err != nil {
		return project.Specification{}, err
	}
	spec.Spec.Connections = make([]project.Connection, 0, len(resolved))
	for _, c := range resolved {
		spec.Spec.Connections = append(spec.Spec.Connections, c.Connection)
	}
	return spec, nil
}
