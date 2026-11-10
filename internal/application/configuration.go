package application

import (
	"fmt"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

// ConfigurationSchema uses the same local/managed/versioned plugin discovery
// and integrity verification as planning. An unavailable generator has no fields.
func (s ProjectService) ConfigurationSchema(component project.Component) (configuration.Schema, error) {
	if _, err := s.catalog.ValidateSelection(catalog.Category(component.Type), component.Technology); err != nil {
		return configuration.Schema{}, err
	}
	loaded, err := s.loadPluginsFor(project.Specification{Spec: project.Spec{Components: []project.Component{component}}})
	if err != nil {
		return configuration.Schema{}, err
	}
	plugin, _ := loaded.ForTechnology(component.Technology)
	return plugin.Configuration, nil
}

// ConfigureSpecification parses component.key=value settings using the plugin
// schema, then resolves every component into a private in-memory value map.
func (s ProjectService) ConfigureSpecification(spec project.Specification, settings []string) (project.Specification, error) {
	if err := spec.Validate(); err != nil {
		return project.Specification{}, err
	}
	if !spec.UsesComponents() {
		return project.Specification{}, fmt.Errorf("configuration requires named components")
	}
	loaded, err := s.loadPluginsFor(spec)
	if err != nil {
		return project.Specification{}, err
	}
	components := spec.CanonicalComponents()
	byName := map[string]int{}
	for i, c := range components {
		byName[c.Name] = i
	}
	seen := map[string]bool{}
	for _, setting := range settings {
		left, raw, ok := strings.Cut(setting, "=")
		name, key, hasKey := strings.Cut(left, ".")
		if !ok || !hasKey || key == "" {
			return project.Specification{}, fmt.Errorf("invalid setting %q; expected component.key=value", setting)
		}
		i, exists := byName[name]
		if !exists {
			return project.Specification{}, fmt.Errorf("setting references unknown component %q", name)
		}
		if seen[left] {
			return project.Specification{}, fmt.Errorf("duplicate setting %s", left)
		}
		seen[left] = true
		plugin, _ := loaded.ForTechnology(components[i].Technology)
		property, exists := plugin.Configuration.Properties[key]
		if !exists {
			return project.Specification{}, fmt.Errorf("component %q configuration.%s: unknown key", name, key)
		}
		value, err := property.Parse(raw)
		if err != nil {
			return project.Specification{}, fmt.Errorf("component %q configuration.%s: %w", name, key, err)
		}
		if components[i].Configuration == nil {
			components[i].Configuration = configuration.Values{}
		}
		components[i].Configuration[key] = value
	}
	for i := range components {
		c := &components[i]
		plugin, _ := loaded.ForTechnology(c.Technology)
		c.Configuration, err = configuration.Resolve(plugin.Configuration, c.Configuration, c.Name)
		if err != nil {
			return project.Specification{}, err
		}
	}
	spec.Spec.Components = components
	return spec, nil
}
