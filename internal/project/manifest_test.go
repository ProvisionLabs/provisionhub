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
			Backend:  TechnologySelection{Technology: "nestjs"},
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
	for _, name := range []string{"", "Upper", "contains space", "-leading", "trailing-", strings.Repeat("a", 64)} {
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
