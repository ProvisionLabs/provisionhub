package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func validInput(destination string) CreateProjectInput {
	return CreateProjectInput{
		Name: "my-app", Destination: destination,
		Backend: "nestjs", Frontend: "nextjs", Database: "postgresql",
	}
}

func TestCreateProjectAndLoad(t *testing.T) {
	service := NewProjectService()
	input := validInput(filepath.Join(t.TempDir(), "new app"))
	result, err := service.CreateProject(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.ManifestPath != filepath.Join(input.Destination, project.ManifestName) {
		t.Fatalf("manifest path = %q", result.ManifestPath)
	}
	loaded, err := service.LoadProject(result.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Metadata.Name != input.Name || loaded.Spec.Frontend.Technology != input.Frontend || loaded.Spec.Database.Technology != input.Database {
		t.Fatalf("loaded manifest = %+v", loaded)
	}
	if _, err := service.CreateProject(input); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("second creation error = %v", err)
	}
}

func TestOptionalSelections(t *testing.T) {
	service := NewProjectService()
	input := validInput(filepath.Join(t.TempDir(), "backend-only"))
	input.Frontend, input.Database = "none", ""
	result, err := service.CreateProject(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.Spec.Frontend != nil || result.Manifest.Spec.Database != nil {
		t.Fatalf("optional categories persisted: %+v", result.Manifest.Spec)
	}
	data, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "frontend:") || strings.Contains(string(data), "database:") {
		t.Fatalf("optional fields emitted: %s", data)
	}
}

func TestProjectValidationErrors(t *testing.T) {
	service := NewProjectService()
	base := t.TempDir()
	for _, test := range []struct {
		name string
		edit func(*CreateProjectInput)
		want string
	}{
		{"invalid name", func(i *CreateProjectInput) { i.Name = "Invalid Name" }, "project name"},
		{"missing destination", func(i *CreateProjectInput) { i.Destination = "" }, "destination path"},
		{"NUL path", func(i *CreateProjectInput) { i.Destination = "bad\x00path" }, "NUL"},
		{"missing backend", func(i *CreateProjectInput) { i.Backend = "" }, "backend technology is required"},
		{"unknown backend", func(i *CreateProjectInput) { i.Backend = "unknown" }, "unknown backend"},
		{"wrong category", func(i *CreateProjectInput) { i.Database = "nextjs" }, "belongs to frontend"},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := validInput(filepath.Join(base, test.name))
			test.edit(&input)
			if _, err := service.CreateProject(input); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestFileAncestorFailureAndUnknownLoadedTechnology(t *testing.T) {
	service := NewProjectService()
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	input := validInput(filepath.Join(file, "project"))
	if _, err := service.CreateProject(input); err == nil || !strings.Contains(err.Error(), "contains a file") {
		t.Fatalf("filesystem error = %v", err)
	}

	manifest := project.Manifest{
		APIVersion: project.APIVersion, Kind: project.Kind,
		Metadata: project.Metadata{Name: "demo"},
		Spec:     project.Spec{Backend: project.TechnologySelection{Technology: "unknown"}},
	}
	data, err := project.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, project.ManifestName)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.LoadProject(path); err == nil || !strings.Contains(err.Error(), "unknown backend") {
		t.Fatalf("load error = %v", err)
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
	input := CreateProjectInput{Name: "demo", Destination: filepath.Join(t.TempDir(), "demo"), Backend: "backend-a", Database: "database-b"}
	if _, _, err := service.ValidateInput(input); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("compatibility error = %v", err)
	}
}
