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
	if escape != "" && escape != "json" && escape != "html" && escape != "host" {
		return "", "", fmt.Errorf("unsupported template escape %q", escape)
	}
	if escape == "host" && (p.Type != "string" || p.Format != "host") {
		return "", "", fmt.Errorf("|host requires a string property with format: host")
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
		return renderScalar(ref, schema, values, component.Name)
	})
}

func renderScalar(ref string, schema configuration.Schema, values configuration.Values, identity string) (string, error) {
	key, escape, err := reference(ref, schema)
	if err != nil {
		return "", err
	}
	value, ok := values[key]
	if !ok {
		return "", fmt.Errorf("component %q template reference config.%s has no value", identity, key)
	}
	value, err = schema.Properties[key].ValidateValue(value)
	if err != nil {
		return "", fmt.Errorf("component %q template reference config.%s: %w", identity, key, err)
	}
	if escape == "html" {
		return html.EscapeString(fmt.Sprint(value)), nil
	}
	if escape == "host" {
		return configuration.HostLiteral(value.(string))
	}
	data, err := json.Marshal(value)
	return string(data), err
}

func integrationScope(ref string, schema configuration.Schema, consumer Consumer) (string, configuration.Schema, error) {
	for _, scope := range []struct {
		prefix string
		schema configuration.Schema
	}{{"config.", schema}, {"connection.", consumer.Configuration}, {"target.", consumer.Requires}} {
		if strings.HasPrefix(ref, scope.prefix) {
			return "config." + strings.TrimPrefix(ref, scope.prefix), scope.schema, nil
		}
	}
	return "", configuration.Schema{}, fmt.Errorf("unsupported integration template reference %q", ref)
}

func validateIntegrationTemplate(content []byte, schema configuration.Schema, consumer Consumer) error {
	_, err := expand(content, func(ref string) (string, error) {
		if ref == "source.name" || ref == "target.name" || ref == "componentName" || ref == "component.name" || ref == "projectName" {
			return "", nil
		}
		ref, schema, err := integrationScope(ref, schema, consumer)
		if err != nil {
			return "", err
		}
		_, _, err = reference(ref, schema)
		return "", err
	})
	return err
}

// RenderIntegration adds fixed connection/target scopes to the same literal
// renderer. The metadata has been validated against the consumer requirements.
func RenderIntegration(file File, projectName string, source, target project.Component, componentSchema configuration.Schema, consumer Consumer, config, metadata configuration.Values) ([]byte, error) {
	if file.Type == "copy" {
		return file.Content, nil
	}
	return expand(file.Content, func(ref string) (string, error) {
		switch ref {
		case "projectName":
			return projectName, nil
		case "source.name", "componentName", "component.name":
			return source.Name, nil
		case "target.name":
			return target.Name, nil
		}
		values := source.Configuration
		if strings.HasPrefix(ref, "connection.") {
			values = config
		} else if strings.HasPrefix(ref, "target.") {
			values = metadata
		}
		normalized, schema, err := integrationScope(ref, componentSchema, consumer)
		if err != nil {
			return "", err
		}
		value, err := renderScalar(normalized, schema, values, source.Name+" -> "+target.Name)
		if err != nil {
			return "", fmt.Errorf("integration reference %q: %w", ref, err)
		}
		return value, nil
	})
}
