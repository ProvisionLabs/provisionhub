package plugins

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/configuration"
)

// ConnectionContract declares data capabilities and connection-owned files.
// IDs identify slots within a plugin; Type is the shared compatibility identifier.
type ConnectionContract struct {
	Provides []Provider `yaml:"provides,omitempty"`
	Consumes []Consumer `yaml:"consumes,omitempty"`
}

type Provider struct {
	ID          string                   `yaml:"id"`
	Type        string                   `yaml:"type"`
	Description string                   `yaml:"description,omitempty"`
	Metadata    map[string]MetadataValue `yaml:"metadata"`
}

// Exactly one literal value or component configuration key supplies a scalar.
type MetadataValue struct {
	Value         any    `yaml:"value,omitempty"`
	Configuration string `yaml:"configuration,omitempty"`
}

type Consumer struct {
	ID            string               `yaml:"id"`
	Type          string               `yaml:"type"`
	Description   string               `yaml:"description,omitempty"`
	Requires      configuration.Schema `yaml:"requires"`
	Configuration configuration.Schema `yaml:"configuration,omitempty"`
	Files         []File               `yaml:"files"`
}

// AllFiles includes integration inputs even when no connection is selected.
// Packaging and installed-content integrity must cover every declared source.
func (m Manifest) AllFiles() []File {
	files := slices.Clone(m.Files)
	for _, consumer := range m.Connections.Consumes {
		files = append(files, consumer.Files...)
	}
	return files
}

func (m Manifest) ValidateConnections() error {
	ids := map[string]bool{}
	for _, provider := range m.Connections.Provides {
		if !validID(provider.ID) || !validID(provider.Type) || ids[provider.ID] {
			return fmt.Errorf("invalid or duplicate provider ID/type %q", provider.ID)
		}
		ids[provider.ID] = true
		if len(provider.Metadata) == 0 {
			return fmt.Errorf("provider %q requires metadata", provider.ID)
		}
		for _, key := range sortedMetadataKeys(provider.Metadata) {
			binding := provider.Metadata[key]
			if !configuration.ValidKey(key) {
				return fmt.Errorf("provider %q invalid metadata key %q", provider.ID, key)
			}
			if (binding.Configuration != "") == (binding.Value != nil) {
				return fmt.Errorf("provider %q metadata %q requires exactly one value or configuration key", provider.ID, key)
			}
			if binding.Configuration != "" {
				if _, ok := m.Configuration.Properties[binding.Configuration]; !ok {
					return fmt.Errorf("provider %q metadata %q references unknown configuration %q", provider.ID, key, binding.Configuration)
				}
			} else {
				switch binding.Value.(type) {
				case string, bool, int, int64:
				default:
					return fmt.Errorf("provider %q metadata %q requires a scalar literal", provider.ID, key)
				}
			}
		}
	}
	ids = map[string]bool{}
	for _, consumer := range m.Connections.Consumes {
		if !validID(consumer.ID) || !validID(consumer.Type) || ids[consumer.ID] {
			return fmt.Errorf("invalid or duplicate consumer ID/type %q", consumer.ID)
		}
		ids[consumer.ID] = true
		if len(consumer.Files) == 0 {
			return fmt.Errorf("consumer %q requires integration files", consumer.ID)
		}
		if err := consumer.Requires.Validate(); err != nil {
			return fmt.Errorf("consumer %q requires: %w", consumer.ID, err)
		}
		if err := consumer.Configuration.Validate(); err != nil {
			return fmt.Errorf("consumer %q: %w", consumer.ID, err)
		}
	}
	return nil
}

func sortedMetadataKeys(metadata map[string]MetadataValue) []string {
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func (m Manifest) Consumer(id string) (Consumer, bool) {
	for _, c := range m.Connections.Consumes {
		if c.ID == id {
			return c, true
		}
	}
	return Consumer{}, false
}

func sortContract(contract *ConnectionContract) {
	slices.SortFunc(contract.Provides, func(a, b Provider) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(contract.Consumes, func(a, b Consumer) int { return strings.Compare(a.ID, b.ID) })
}
