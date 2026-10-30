package project

import (
	"reflect"
	"strings"
	"testing"
)

func exampleManifest() Manifest {
	return Manifest{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata:   Metadata{Name: "my-application"},
		Spec: Spec{
			Backend:  &TechnologySelection{Technology: "nestjs"},
			Frontend: &TechnologySelection{Technology: "nextjs"},
			Database: &TechnologySelection{Technology: "postgresql"},
		},
	}
}

func TestManifestRoundTrip(t *testing.T) {
	want := exampleManifest()
	data, err := Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"apiVersion: provisionhub.dev/v1alpha1", "kind: Project", "name: my-application", "technology: nestjs"} {
		if !strings.Contains(string(data), field) {
			t.Errorf("manifest missing %q: %s", field, data)
		}
	}
	got, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip changed manifest: got %+v, want %+v", got, want)
	}
}

func TestOptionalCategoriesOmitted(t *testing.T) {
	m := exampleManifest()
	m.Spec.Frontend, m.Spec.Database = nil, nil
	data, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "frontend:") || strings.Contains(string(data), "database:") {
		t.Fatalf("optional categories should be omitted: %s", data)
	}
	got, err := Unmarshal(data)
	if err != nil || got.Spec.Frontend != nil || got.Spec.Database != nil {
		t.Fatalf("optional category round trip: %+v, %v", got, err)
	}
}

func TestUnmarshalRejectsInvalidDocuments(t *testing.T) {
	valid, err := Marshal(exampleManifest())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, yaml, want string
	}{
		{"unsupported version", strings.Replace(string(valid), APIVersion, "provisionhub.dev/v9", 1), "unsupported apiVersion"},
		{"malformed YAML", "metadata: [", "decode manifest"},
		{"unknown field", string(valid) + "unexpected: true\n", "field unexpected not found"},
		{"multiple documents", string(valid) + "---\nkind: Project\n", "exactly one"},
		{"empty", "", "manifest is empty"},
		{"missing backend", "apiVersion: provisionhub.dev/v1alpha1\nkind: Project\nmetadata:\n  name: demo\nspec: {}\n", "spec.backend.technology"},
		{"null components", "apiVersion: provisionhub.dev/v1alpha1\nkind: Project\nmetadata: {name: demo}\nspec: {components: null}\n", "spec.components must contain"},
		{"null components mixed with backend", "apiVersion: provisionhub.dev/v1alpha1\nkind: Project\nmetadata: {name: demo}\nspec: {backend: {technology: nestjs}, components: null}\n", "cannot be mixed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Unmarshal([]byte(test.yaml))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

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

func TestComponentManifestAndDeterministicSerialization(t *testing.T) {
	frontendOnly := Manifest{APIVersion: APIVersion, Kind: Kind, Metadata: Metadata{Name: "web-only"}, Spec: Spec{
		Components: []Component{{Name: "web", Type: "frontend", Technology: "nextjs"}},
	}}
	if err := frontendOnly.Validate(); err != nil {
		t.Fatalf("frontend-only named project rejected: %v", err)
	}
	manifest := Manifest{APIVersion: APIVersion, Kind: Kind, Metadata: Metadata{Name: "commerce-platform"}, Spec: Spec{
		Components: []Component{
			{Name: "web", Type: "frontend", Technology: "nextjs"},
			{Name: "billing", Type: "backend", Technology: "nestjs"},
			{Name: "api", Type: "backend", Technology: "nestjs"},
		},
		Connections: []Connection{{From: "web", To: "api", Type: "http"}, {From: "api", To: "web", Type: "http"}},
	}}
	data, err := Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "backend:") || strings.Index(string(data), "name: api") > strings.Index(string(data), "name: billing") {
		t.Fatalf("noncanonical component YAML: %s", data)
	}
	again := manifest
	again.Spec.Components = []Component{manifest.Spec.Components[2], manifest.Spec.Components[0], manifest.Spec.Components[1]}
	again.Spec.Connections = []Connection{manifest.Spec.Connections[1], manifest.Spec.Connections[0]}
	encoded, err := Marshal(again)
	if err != nil || string(encoded) != string(data) {
		t.Fatalf("reordered YAML differs: %v\n%s", err, encoded)
	}
	parsed, err := Unmarshal(data)
	if err != nil || len(parsed.CanonicalComponents()) != 3 || parsed.CanonicalComponents()[0].ID() != "component/api" {
		t.Fatalf("component round trip: %+v, %v", parsed, err)
	}
	legacy := exampleManifest().CanonicalComponents()
	if len(legacy) != 3 || !legacy[0].Legacy || legacy[0].Name != "backend" {
		t.Fatalf("legacy normalization: %+v", legacy)
	}
}

func TestComponentManifestRejectsInvalidDefinitions(t *testing.T) {
	base := Manifest{APIVersion: APIVersion, Kind: Kind, Metadata: Metadata{Name: "demo"}, Spec: Spec{
		Components: []Component{{Name: "api", Type: "backend", Technology: "nestjs"}},
	}}
	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{"empty", func(m *Manifest) { m.Spec.Components = []Component{} }},
		{"duplicate", func(m *Manifest) { m.Spec.Components = append(m.Spec.Components, m.Spec.Components[0]) }},
		{"unsafe name", func(m *Manifest) { m.Spec.Components[0].Name = "../escape" }},
		{"wrong type", func(m *Manifest) { m.Spec.Components[0].Type = "worker" }},
		{"missing technology", func(m *Manifest) { m.Spec.Components[0].Technology = "" }},
		{"mixed", func(m *Manifest) { m.Spec.Backend = &TechnologySelection{Technology: "nestjs"} }},
		{"unknown connection source", func(m *Manifest) { m.Spec.Connections = []Connection{{From: "missing", To: "api", Type: "http"}} }},
		{"self connection", func(m *Manifest) { m.Spec.Connections = []Connection{{From: "api", To: "api", Type: "http"}} }},
		{"unsupported connection", func(m *Manifest) {
			m.Spec.Components = append(m.Spec.Components, Component{Name: "web", Type: "frontend", Technology: "nextjs"})
			m.Spec.Connections = []Connection{{From: "web", To: "api", Type: "grpc"}}
		}},
		{"duplicate connection", func(m *Manifest) {
			m.Spec.Components = append(m.Spec.Components, Component{Name: "web", Type: "frontend", Technology: "nextjs"})
			m.Spec.Connections = []Connection{{From: "web", To: "api", Type: "http"}, {From: "web", To: "api", Type: "http"}}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := base
			m.Spec.Components = append([]Component{}, base.Spec.Components...)
			tc.mutate(&m)
			if err := m.Validate(); err == nil {
				t.Fatal("accepted invalid manifest")
			}
		})
	}
	if _, err := Unmarshal([]byte("apiVersion: provisionhub.dev/v1alpha1\nkind: Project\nmetadata: {name: demo}\nspec:\n  components:\n    - {name: api, type: backend, technology: nestjs, unknown: true}\n")); err == nil {
		t.Fatal("accepted unknown component field")
	}
}
