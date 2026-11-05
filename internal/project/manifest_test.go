package project

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	for _, name := range []string{"", "Upper", "contains space", "-leading", "trailing-", "con", "lpt1", strings.Repeat("a", 64)} {
		if err := ValidateName(name); err == nil {
			t.Errorf("accepted invalid name %q", name)
		}
	}
	for _, name := range []string{"a", "my-app-2", strings.Repeat("a", 63)} {
		if err := ValidateName(name); err != nil {
			t.Errorf("rejected valid name %q: %v", name, err)
		}
	}
}

func TestSpecificationValidationAndOrdering(t *testing.T) {
	spec := Specification{APIVersion: APIVersion, Kind: Kind, Metadata: Metadata{Name: "commerce-platform"}, Spec: Spec{
		Components: []Component{
			{Name: "web", Type: "frontend", Technology: "nextjs"},
			{Name: "billing", Type: "backend", Technology: "nestjs"},
			{Name: "api", Type: "backend", Technology: "nestjs"},
		},
		Connections: []Connection{{From: "web", To: "api", Type: "http"}, {From: "api", To: "web", Type: "http"}},
	}}
	if err := spec.Validate(); err != nil {
		t.Fatal(err)
	}
	components := spec.CanonicalComponents()
	if components[0].Name != "api" || components[1].Name != "billing" || components[2].Name != "web" {
		t.Fatal(components)
	}
	connections := spec.CanonicalConnections()
	if connections[0].From != "api" || connections[1].From != "web" {
		t.Fatal(connections)
	}
	if spec.Spec.Components[0].Name != "web" {
		t.Fatal("canonical ordering mutated the specification")
	}
	again := spec
	again.Spec.Components = []Component{components[2], components[0], components[1]}
	if !reflect.DeepEqual(components, again.CanonicalComponents()) {
		t.Fatal("input order changed canonical components")
	}
}

func TestSpecificationRejectsInvalidComponentsAndConnections(t *testing.T) {
	base := Specification{APIVersion: APIVersion, Kind: Kind, Metadata: Metadata{Name: "demo"}, Spec: Spec{
		Components: []Component{{Name: "api", Type: "backend", Technology: "nestjs"}},
	}}
	tests := []struct {
		name   string
		mutate func(*Specification)
	}{
		{"empty", func(s *Specification) { s.Spec.Components = []Component{} }},
		{"duplicate", func(s *Specification) { s.Spec.Components = append(s.Spec.Components, s.Spec.Components[0]) }},
		{"unsafe name", func(s *Specification) { s.Spec.Components[0].Name = "../escape" }},
		{"wrong type", func(s *Specification) { s.Spec.Components[0].Type = "worker" }},
		{"missing technology", func(s *Specification) { s.Spec.Components[0].Technology = "" }},
		{"mixed", func(s *Specification) { s.Spec.Backend = &TechnologySelection{Technology: "nestjs"} }},
		{"unknown connection", func(s *Specification) { s.Spec.Connections = []Connection{{From: "missing", To: "api", Type: "http"}} }},
		{"self connection", func(s *Specification) { s.Spec.Connections = []Connection{{From: "api", To: "api", Type: "http"}} }},
		{"unsupported connection", func(s *Specification) {
			s.Spec.Components = append(s.Spec.Components, Component{Name: "web", Type: "frontend", Technology: "nextjs"})
			s.Spec.Connections = []Connection{{From: "web", To: "api", Type: "grpc"}}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := base
			s.Spec.Components = append([]Component{}, base.Spec.Components...)
			tc.mutate(&s)
			if err := s.Validate(); err == nil {
				t.Fatal("accepted invalid specification")
			}
		})
	}
}
