package plugins

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

// expand recognizes only literal references and fixed escaping modes. It never
// recursively expands substituted values or evaluates template programs.
func expand(content []byte, lookup func(string) (string, error)) ([]byte, error) {
	s := string(content)
	var out strings.Builder
	for strings.Contains(s, "{{") {
		before, rest, _ := strings.Cut(s, "{{")
		if strings.Contains(before, "}}") {
			return nil, fmt.Errorf("unmatched template delimiter")
		}
		ref, after, ok := strings.Cut(rest, "}}")
		if !ok {
			return nil, fmt.Errorf("unclosed template reference %q", rest)
		}
		value, err := lookup(ref)
		if err != nil {
			return nil, err
		}
		out.WriteString(before)
		out.WriteString(value)
		s = after
	}
	if strings.Contains(s, "}}") {
		return nil, fmt.Errorf("unmatched template delimiter")
	}
	out.WriteString(s)
	return []byte(out.String()), nil
}

func reference(ref string, schema configuration.Schema) (key, escape string, err error) {
	key, escape, _ = strings.Cut(ref, "|")
	if !strings.HasPrefix(key, "config.") {
		return "", "", fmt.Errorf("unsupported template variable %q", ref)
	}
	key = strings.TrimPrefix(key, "config.")
	p, ok := schema.Properties[key]
	if !ok {
		return "", "", fmt.Errorf("undeclared template reference config.%s", key)
	}
	if escape != "" && escape != "json" && escape != "html" {
		return "", "", fmt.Errorf("unsupported template escape %q", escape)
	}
	if (p.Type == "string" || p.Type == "enum") && escape == "" {
		return "", "", fmt.Errorf("template reference config.%s requires |json or |html escaping", key)
	}
	return key, escape, nil
}

func validateTemplate(content []byte, schema configuration.Schema) error {
	_, err := expand(content, func(ref string) (string, error) {
		if ref == "projectName" || ref == "componentName" || ref == "component.name" {
			return "", nil
		}
		_, _, err := reference(ref, schema)
		return "", err
	})
	return err
}

// Render is shared by planning and execution. Strings must request a JSON
// literal (including quotes) or HTML text escaping appropriate to their context.
// Configuration never participates in source or destination paths.
func Render(file File, projectName string, component project.Component, schema configuration.Schema, values configuration.Values) ([]byte, error) {
	if file.Type == "copy" {
		return file.Content, nil
	}
	return expand(file.Content, func(ref string) (string, error) {
		if ref == "projectName" {
			return projectName, nil
		}
		if ref == "component.name" {
			return component.Name, nil
		}
		if ref == "componentName" {
			if component.Legacy {
				return projectName, nil
			}
			return component.Name, nil
		}
		key, escape, err := reference(ref, schema)
		if err != nil {
			return "", err
		}
		value, ok := values[key]
		if !ok {
			return "", fmt.Errorf("component %q template reference config.%s has no value", component.Name, key)
		}
		value, err = schema.Properties[key].ValidateValue(value)
		if err != nil {
			return "", fmt.Errorf("component %q template reference config.%s: %w", component.Name, key, err)
		}
		if escape == "html" {
			return html.EscapeString(fmt.Sprint(value)), nil
		}
		data, err := json.Marshal(value)
		return string(data), err
	})
}
