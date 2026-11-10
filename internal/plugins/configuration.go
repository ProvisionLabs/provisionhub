package plugins

import (
	"fmt"
	"go.yaml.in/yaml/v3"
)

// Check scalar tags before YAML's typed decoder can convert numbers to strings
// or fractional bounds to integers. The normal strict decoder checks field names
// and duplicates; this pass also refuses aliases and explicit null defaults.
func validateConfigurationYAML(data []byte) error {
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return err
	}
	if len(document.Content) == 0 {
		return nil
	}
	root := document.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "configuration" {
			continue
		}
		config := root.Content[i+1]
		if config.Kind != yaml.MappingNode {
			return fmt.Errorf("configuration must be a mapping")
		}
		for j := 0; j+1 < len(config.Content); j += 2 {
			if config.Content[j].Value != "properties" {
				continue
			}
			properties := config.Content[j+1]
			if properties.Kind != yaml.MappingNode {
				return fmt.Errorf("configuration.properties must be a mapping")
			}
			for k := 0; k+1 < len(properties.Content); k += 2 {
				key, property := properties.Content[k], properties.Content[k+1]
				if key.Tag != "!!str" || property.Kind != yaml.MappingNode {
					return fmt.Errorf("configuration property must have a string key and mapping definition")
				}
				for n := 0; n+1 < len(property.Content); n += 2 {
					field, value := property.Content[n], property.Content[n+1]
					tag := "!!str"
					switch field.Value {
					case "required":
						tag = "!!bool"
					case "minimum", "maximum", "minLength", "maxLength":
						tag = "!!int"
					case "default":
						if value.Kind != yaml.ScalarNode || (value.Tag != "!!str" && value.Tag != "!!int" && value.Tag != "!!bool") {
							return fmt.Errorf("configuration.%s: default must be a string, boolean, or integer", key.Value)
						}
						continue
					case "options":
						if value.Kind != yaml.SequenceNode {
							return fmt.Errorf("configuration.%s: options must be a sequence", key.Value)
						}
						for _, option := range value.Content {
							if option.Kind != yaml.ScalarNode || option.Tag != "!!str" {
								return fmt.Errorf("configuration.%s: enum options must be strings", key.Value)
							}
						}
						continue
					}
					if field.Tag != "!!str" || value.Kind != yaml.ScalarNode || value.Tag != tag {
						return fmt.Errorf("configuration.%s.%s: expected %s", key.Value, field.Value, tag)
					}
				}
			}
		}
	}
	return nil
}
