package project

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	APIVersion   = "provisionhub.dev/v1alpha1"
	Kind         = "Project"
	ManifestName = "provisionhub.yaml"
)

// Manifest describes a desired project configuration, not provisioned state.
type Manifest struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name string `yaml:"name"`
}

type TechnologySelection struct {
	Technology string `yaml:"technology"`
}

type Spec struct {
	Backend  TechnologySelection  `yaml:"backend"`
	Frontend *TechnologySelection `yaml:"frontend,omitempty"`
	Database *TechnologySelection `yaml:"database,omitempty"`
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
	if strings.TrimSpace(m.Spec.Backend.Technology) == "" {
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

// Marshal returns a consistently ordered YAML representation.
func Marshal(m Manifest) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return yaml.Marshal(m)
}

// Unmarshal reads exactly one YAML document and rejects unknown fields.
func Unmarshal(data []byte) (Manifest, error) {
	var m Manifest
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&m); err != nil {
		if errors.Is(err, io.EOF) {
			return Manifest{}, errors.New("manifest is empty")
		}
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return Manifest{}, fmt.Errorf("decode manifest: %w", err)
		}
		return Manifest{}, errors.New("manifest must contain exactly one YAML document")
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}
