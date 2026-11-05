package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestSpecificationValidationDoesNotWrite(t *testing.T) {
	service := NewProjectService()
	root := t.TempDir()
	destination := filepath.Join(root, "new app")
	components := []project.Component{{Name: "api", Type: "backend", Technology: "nestjs"}, {Name: "billing", Type: "backend", Technology: "nestjs"}}
	spec, err := service.ValidateComponentsInput("demo", destination, components)
	if err != nil || len(spec.CanonicalComponents()) != 2 {
		t.Fatal(spec, err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("validation wrote destination")
	}
	for _, tc := range []struct {
		name, destination string
		components        []project.Component
		want              string
	}{
		{"Bad Name", destination, components, "project name"},
		{"demo", "", components, "destination path"},
		{"demo", "bad\x00path", components, "NUL"},
		{"demo", destination, nil, "at least one"},
		{"demo", destination, []project.Component{{Name: "api", Type: "backend", Technology: "unknown"}}, "unknown backend"},
		{"demo", destination, []project.Component{{Name: "api", Type: "database", Technology: "nextjs"}}, "belongs to frontend"},
		{"demo", destination, []project.Component{components[0], components[0]}, "duplicate"},
	} {
		if _, err := service.ValidateComponentsInput(tc.name, tc.destination, tc.components); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%q: %v, want %q", tc.name, err, tc.want)
		}
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateComponentsInput("demo", filepath.Join(file, "project"), components); err == nil {
		t.Fatal("accepted file ancestor")
	}
}

func TestTechnologiesAreSharedWithService(t *testing.T) {
	service := NewProjectService()
	if len(service.Technologies(catalog.Backend)) != 3 {
		t.Fatal("backend catalog not exposed")
	}
	if service.TechnologyName("postgresql") != "PostgreSQL" || service.TechnologyName("") != "None" {
		t.Fatal("unexpected display names")
	}
}

func TestServiceRejectsExplicitCompatibilityConflict(t *testing.T) {
	c, err := catalog.New([]catalog.Technology{
		{ID: "backend-a", Name: "Backend A", Category: catalog.Backend, Description: "Backend", IncompatibleWith: []string{"database-b"}},
		{ID: "database-b", Name: "Database B", Category: catalog.Database, Description: "Database"},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := ProjectService{catalog: c}
	components := []project.Component{{Name: "api", Type: "backend", Technology: "backend-a"}, {Name: "data", Type: "database", Technology: "database-b"}}
	if _, err := service.ValidateComponentsInput("demo", filepath.Join(t.TempDir(), "demo"), components); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("compatibility error = %v", err)
	}
}
