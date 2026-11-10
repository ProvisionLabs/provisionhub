// Package connections resolves plugin data capabilities without I/O or execution.
package connections

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

var ErrCycle = errors.New("connection generation dependency cycle")

type Resolved struct {
	project.Connection
	ProviderID string               `json:"providerId,omitempty"`
	ConsumerID string               `json:"consumerId,omitempty"`
	Metadata   configuration.Values `json:"metadata,omitempty"`
	Outputs    []string             `json:"outputs,omitempty"`
}

func (r Resolved) OperationID() string { return "connection/" + r.ID() }

type Match struct {
	ProviderID string
	Consumer   plugins.Consumer
	Metadata   configuration.Values
}

type Option struct {
	Connection  project.Connection
	Description string
	Schema      configuration.Schema
}

// Select identifies exactly one compatible slot pair on the named instances.
// It intentionally does not require connection values yet, so interfaces can
// inspect the selected schema before collecting its required fields.
func Select(components []project.Component, connection project.Connection, loaded plugins.Set) (Match, error) {
	var source, target project.Component
	for _, c := range components {
		if c.Name == connection.From {
			source = c
		}
		if c.Name == connection.To {
			target = c
		}
	}
	if source.Name == "" || target.Name == "" {
		return Match{}, fmt.Errorf("connection %s: source or target component does not exist", connection.ID())
	}
	if source.Name == target.Name {
		return Match{}, fmt.Errorf("connection %s: self-connections are unsupported", connection.ID())
	}
	sourcePlugin, sourceOK := loaded.ForTechnology(source.Technology)
	targetPlugin, targetOK := loaded.ForTechnology(target.Technology)
	if !sourceOK || !targetOK {
		return Match{}, fmt.Errorf("connection %s: both components require installed generators with connection capabilities", connection.ID())
	}
	for _, plugin := range []plugins.Manifest{sourcePlugin, targetPlugin} {
		if err := plugin.ValidateConnections(); err != nil {
			return Match{}, fmt.Errorf("connection %s plugin %q: %w", connection.ID(), plugin.ID, err)
		}
	}
	providers := slices.Clone(targetPlugin.Connections.Provides)
	consumers := slices.Clone(sourcePlugin.Connections.Consumes)
	slices.SortFunc(providers, func(a, b plugins.Provider) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(consumers, func(a, b plugins.Consumer) int { return strings.Compare(a.ID, b.ID) })
	matches := []Match{}
	reasons := []string{}
	for _, consumer := range consumers {
		if consumer.Type != connection.Type {
			continue
		}
		for _, provider := range providers {
			if provider.Type != connection.Type {
				continue
			}
			metadata, err := providerMetadata(target, targetPlugin, provider)
			if err == nil {
				// Providers may export extra fields; consumers validate only the
				// subset they declare, using the existing scalar resolver.
				required := configuration.Values{}
				for _, key := range consumer.Requires.Keys() {
					if value, ok := metadata[key]; ok {
						required[key] = value
					}
				}
				metadata, err = configuration.Resolve(consumer.Requires, required, connection.ID()+" metadata")
			}
			if err != nil {
				reasons = append(reasons, fmt.Sprintf("%s/%s: %v", consumer.ID, provider.ID, err))
				continue
			}
			matches = append(matches, Match{ProviderID: provider.ID, Consumer: consumer, Metadata: metadata})
		}
	}
	if len(matches) == 0 {
		detail := "source must consume and target must provide the same type"
		if len(reasons) > 0 {
			detail = reasons[0]
		}
		return Match{}, fmt.Errorf("connection %s: no compatible capabilities: %s", connection.ID(), detail)
	}
	if len(matches) != 1 {
		pairs := []string{}
		for _, match := range matches {
			pairs = append(pairs, match.Consumer.ID+"/"+match.ProviderID)
		}
		return Match{}, fmt.Errorf("connection %s: ambiguous capability pairs %v; select plugin definitions with one compatible pair", connection.ID(), pairs)
	}
	return matches[0], nil
}

func providerMetadata(component project.Component, plugin plugins.Manifest, provider plugins.Provider) (configuration.Values, error) {
	values, err := configuration.Resolve(plugin.Configuration, component.Configuration, component.Name)
	if err != nil {
		return nil, err
	}
	metadata := configuration.Values{}
	for _, key := range slices.Sorted(maps.Keys(provider.Metadata)) {
		binding := provider.Metadata[key]
		value := binding.Value
		if binding.Configuration != "" {
			var ok bool
			value, ok = values[binding.Configuration]
			if !ok {
				return nil, fmt.Errorf("provider %s on %s: configuration.%s has no value for metadata.%s", provider.ID, component.Name, binding.Configuration, key)
			}
		}
		if n, ok := value.(int); ok {
			value = int64(n)
		}
		metadata[key] = value
	}
	return metadata, nil
}

// Resolve validates topology before resolving connection configuration. No file
// operation can be contributed from a partially resolved connection set.
func Resolve(components []project.Component, declarations []project.Connection, loaded plugins.Set) ([]Resolved, error) {
	if len(declarations) == 0 {
		return nil, nil
	}
	ordered, matches, err := topology(components, declarations, loaded)
	if err != nil {
		return nil, err
	}
	result := make([]Resolved, 0, len(ordered))
	byName := map[string]project.Component{}
	for _, c := range components {
		byName[c.Name] = c
	}
	for i, c := range ordered {
		match := matches[i]
		values, err := configuration.Resolve(match.Consumer.Configuration, c.Configuration, "connection "+c.ID())
		if err != nil {
			return nil, err
		}
		c.Configuration = values
		outputs := []string{}
		for _, file := range match.Consumer.Files {
			outputs = append(outputs, plugins.DestinationFor(byName[c.From], file))
		}
		slices.Sort(outputs)
		result = append(result, Resolved{Connection: c, ProviderID: match.ProviderID, ConsumerID: match.Consumer.ID, Metadata: match.Metadata, Outputs: outputs})
	}
	return result, nil
}

func topology(components []project.Component, declarations []project.Connection, loaded plugins.Set) ([]project.Connection, []Match, error) {
	spec := project.Specification{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "connections"}, Spec: project.Spec{Components: components, Connections: declarations}}
	if err := spec.Validate(); err != nil {
		return nil, nil, err
	}
	ordered := spec.CanonicalConnections()
	matches := make([]Match, 0, len(ordered))
	used := map[string]string{}
	for _, c := range ordered {
		match, err := Select(components, c, loaded)
		if err != nil {
			return nil, nil, err
		}
		slot := c.From + "/" + match.Consumer.ID
		if previous, ok := used[slot]; ok {
			return nil, nil, fmt.Errorf("connection %s: consumer slot %s supports one target; already used by %s", c.ID(), slot, previous)
		}
		used[slot] = c.ID()
		matches = append(matches, match)
	}
	if err := validateCycles(ordered); err != nil {
		return nil, nil, err
	}
	return ordered, matches, nil
}

func validateCycles(ordered []project.Connection) error {
	edges := map[string][]string{}
	for _, c := range ordered {
		edges[c.From] = append(edges[c.From], c.To)
	}
	state := map[string]int{}
	var visit func(string, []string) error
	visit = func(name string, path []string) error {
		if state[name] == 2 {
			return nil
		}
		if state[name] == 1 {
			start := slices.Index(path, name)
			return fmt.Errorf("%w: %s", ErrCycle, strings.Join(append(slices.Clone(path[start:]), name), " -> "))
		}
		state[name] = 1
		for _, target := range edges[name] {
			if err := visit(target, append(path, name)); err != nil {
				return err
			}
		}
		state[name] = 2
		return nil
	}
	for _, name := range slices.Sorted(maps.Keys(edges)) {
		if err := visit(name, nil); err != nil {
			return err
		}
	}
	return nil
}

// Options hides unsupported/ambiguous pairs and additions that would duplicate
// a declaration, occupy an existing slot, or create a dependency cycle. Required
// configuration is collected only after selection.
func Options(components []project.Component, existing []project.Connection, loaded plugins.Set) []Option {
	result := []Option{}
	for _, source := range components {
		plugin, _ := loaded.ForTechnology(source.Technology)
		types := map[string]bool{}
		for _, consumer := range plugin.Connections.Consumes {
			types[consumer.Type] = true
		}
		for _, kind := range slices.Sorted(maps.Keys(types)) {
			for _, target := range components {
				if source.Name == target.Name {
					continue
				}
				candidate := project.Connection{From: source.Name, To: target.Name, Type: kind}
				declarations := append(slices.Clone(existing), candidate)
				if _, _, err := topology(components, declarations, loaded); err != nil {
					continue
				}
				match, err := Select(components, candidate, loaded)
				if err != nil {
					continue
				}
				result = append(result, Option{Connection: candidate, Description: match.Consumer.Description, Schema: match.Consumer.Configuration})
			}
		}
	}
	slices.SortFunc(result, func(a, b Option) int { return strings.Compare(a.Connection.ID(), b.Connection.ID()) })
	return result
}
