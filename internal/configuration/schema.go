// Package configuration validates flat, plugin-defined component options without I/O.
package configuration

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"unicode/utf8"
)

type Values map[string]any

type Schema struct {
	Properties map[string]Property `yaml:"properties"`
}

type Property struct {
	Type        string   `yaml:"type"`
	Format      string   `yaml:"format,omitempty"`
	Label       string   `yaml:"label,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Required    bool     `yaml:"required,omitempty"`
	Default     any      `yaml:"default,omitempty"`
	Options     []string `yaml:"options,omitempty"`
	Minimum     *int64   `yaml:"minimum,omitempty"`
	Maximum     *int64   `yaml:"maximum,omitempty"`
	MinLength   *int     `yaml:"minLength,omitempty"`
	MaxLength   *int     `yaml:"maxLength,omitempty"`
}

var keyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,62}$`)

func ValidKey(key string) bool { return keyPattern.MatchString(key) }

func (s Schema) Keys() []string { return slices.Sorted(maps.Keys(s.Properties)) }

func (s Schema) Validate() error {
	for _, key := range s.Keys() {
		p := s.Properties[key]
		fail := func(message string) error { return fmt.Errorf("configuration.%s: %s", key, message) }
		if !keyPattern.MatchString(key) {
			return fail("key must start with an ASCII letter and contain at most 63 letters, digits, or underscores")
		}
		switch p.Type {
		case "string", "boolean", "integer", "enum":
		default:
			return fail("type must be string, boolean, integer, or enum")
		}
		if p.Format != "" && (p.Type != "string" || p.Format != "host") {
			return fail("format supports only host on string properties")
		}
		if p.Type == "enum" {
			if len(p.Options) == 0 {
				return fail("enum requires options")
			}
			seen := map[string]bool{}
			for _, option := range p.Options {
				if option == "" || seen[option] {
					return fail("enum options must be nonempty and unique")
				}
				seen[option] = true
			}
		} else if p.Options != nil {
			return fail("options are only supported for enum")
		}
		if p.Minimum != nil || p.Maximum != nil {
			if p.Type != "integer" {
				return fail("minimum and maximum require integer type")
			}
			if p.Minimum != nil && p.Maximum != nil && *p.Minimum > *p.Maximum {
				return fail("minimum exceeds maximum")
			}
		}
		if p.MinLength != nil || p.MaxLength != nil {
			if p.Type != "string" {
				return fail("minLength and maxLength require string type")
			}
			if p.MinLength != nil && *p.MinLength < 0 || p.MaxLength != nil && *p.MaxLength < 0 {
				return fail("length constraints must be nonnegative")
			}
			if p.MinLength != nil && p.MaxLength != nil && *p.MinLength > *p.MaxLength {
				return fail("minLength exceeds maxLength")
			}
		}
		if p.Default != nil {
			if _, err := p.ValidateValue(p.Default); err != nil {
				return fail("invalid default: " + err.Error())
			}
		}
	}
	return nil
}

// ValidateValue returns a scalar value; integers use int64 on every platform.
func (p Property) ValidateValue(value any) (any, error) {
	switch p.Type {
	case "string", "enum":
		v, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected %s", p.Type)
		}
		if !utf8.ValidString(v) {
			return nil, fmt.Errorf("string must be valid UTF-8")
		}
		if p.Format == "host" {
			if _, err := HostLiteral(v); err != nil {
				return nil, err
			}
		}
		if p.Type == "enum" && !slices.Contains(p.Options, v) {
			return nil, fmt.Errorf("expected one of %v", p.Options)
		}
		length := utf8.RuneCountInString(v)
		if p.MinLength != nil && length < *p.MinLength {
			return nil, fmt.Errorf("requires at least %d characters", *p.MinLength)
		}
		if p.MaxLength != nil && length > *p.MaxLength {
			return nil, fmt.Errorf("allows at most %d characters", *p.MaxLength)
		}
		return v, nil
	case "boolean":
		if v, ok := value.(bool); ok {
			return v, nil
		}
		return nil, fmt.Errorf("expected boolean")
	case "integer":
		var v int64
		switch n := value.(type) {
		case int:
			v = int64(n)
		case int64:
			v = n
		default:
			return nil, fmt.Errorf("expected signed 64-bit integer")
		}
		if p.Minimum != nil && v < *p.Minimum {
			return nil, fmt.Errorf("must be at least %d", *p.Minimum)
		}
		if p.Maximum != nil && v > *p.Maximum {
			return nil, fmt.Errorf("must be at most %d", *p.Maximum)
		}
		return v, nil
	}
	return nil, fmt.Errorf("unsupported type %q", p.Type)
}

// Parse converts presentation text according to the declared type, never YAML inference.
func (p Property) Parse(raw string) (any, error) {
	var value any = raw
	switch p.Type {
	case "boolean":
		if raw != "true" && raw != "false" {
			return nil, fmt.Errorf("expected true or false")
		}
		value = raw == "true"
	case "integer":
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected signed 64-bit decimal integer")
		}
		value = n
	}
	return p.ValidateValue(value)
}

// Resolve applies defaults into a new map and emits errors in sorted key order.
// The schema and caller's values are never mutated or shared with the result.
func Resolve(s Schema, user Values, component string) (Values, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("component %q: %w", component, err)
	}
	for _, key := range slices.Sorted(maps.Keys(user)) {
		if _, ok := s.Properties[key]; !ok {
			return nil, fmt.Errorf("component %q configuration.%s: unknown key", component, key)
		}
	}
	resolved := Values{}
	for _, key := range s.Keys() {
		p := s.Properties[key]
		value, present := user[key]
		if !present && p.Default != nil {
			value, present = p.Default, true
		}
		if !present {
			if p.Required {
				return nil, fmt.Errorf("component %q configuration.%s: required value is missing", component, key)
			}
			continue
		}
		v, err := p.ValidateValue(value)
		if err != nil {
			return nil, fmt.Errorf("component %q configuration.%s: %w", component, key, err)
		}
		resolved[key] = v
	}
	return resolved, nil
}
