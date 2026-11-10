package plugins

import (
	"fmt"
	"go.yaml.in/yaml/v3"
)

func validateConnectionsYAML(contract *yaml.Node) error {
	if contract.Kind != yaml.MappingNode {
		return fmt.Errorf("connections must be a mapping")
	}
	for i := 0; i+1 < len(contract.Content); i += 2 {
		kind, list := contract.Content[i].Value, contract.Content[i+1]
		if kind != "provides" && kind != "consumes" {
			continue
		} // strict decoder rejects unknown fields
		if list.Kind != yaml.SequenceNode {
			return fmt.Errorf("connections.%s must be a sequence", kind)
		}
		for _, entry := range list.Content {
			if entry.Kind != yaml.MappingNode {
				return fmt.Errorf("connection capability must be a mapping")
			}
			for j := 0; j+1 < len(entry.Content); j += 2 {
				field, node := entry.Content[j].Value, entry.Content[j+1]
				switch field {
				case "id", "type", "description":
					if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
						return fmt.Errorf("connection capability %s must be a string", field)
					}
				case "configuration", "requires":
					if err := validateSchemaNode(node); err != nil {
						return fmt.Errorf("connection %s: %w", field, err)
					}
				case "metadata":
					if node.Kind != yaml.MappingNode {
						return fmt.Errorf("provider metadata must be a mapping")
					}
					for n := 0; n+1 < len(node.Content); n += 2 {
						key, binding := node.Content[n], node.Content[n+1]
						if key.Tag != "!!str" || binding.Kind != yaml.MappingNode {
							return fmt.Errorf("metadata requires string keys and scalar binding definitions")
						}
						for b := 0; b+1 < len(binding.Content); b += 2 {
							name, value := binding.Content[b].Value, binding.Content[b+1]
							if name == "configuration" && (value.Kind != yaml.ScalarNode || value.Tag != "!!str") {
								return fmt.Errorf("metadata configuration binding must be a string key")
							}
							if name == "value" && (value.Kind != yaml.ScalarNode || (value.Tag != "!!str" && value.Tag != "!!int" && value.Tag != "!!bool")) {
								return fmt.Errorf("metadata literal must be a string, integer, or boolean")
							}
						}
					}
				}
			}
		}
	}
	return nil
}
