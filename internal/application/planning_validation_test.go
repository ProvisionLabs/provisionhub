package application

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestPlanProjectFailuresPreserveDiagnosticsAndFiles(t *testing.T) {
	valid := "apiVersion: provisionhub.dev/v1alpha1\nkind: Project\nmetadata:\n  name: demo\nspec:\n  backend:\n    technology: go\n"
	for _, test := range []struct {
		name, data string
		code       planning.DiagnosticCode
	}{
		{"missing", "", planning.ManifestLoadFailed},
		{"malformed", "metadata: [", planning.InvalidManifest},
		{"unknown field", valid + "unexpected: true\n", planning.InvalidManifest},
		{"multiple documents", valid + "---\nkind: Project\n", planning.InvalidManifest},
		{"incomplete", "apiVersion: provisionhub.dev/v1alpha1\nkind: Project\nmetadata:\n  name: demo\nspec: {}\n", planning.InvalidManifest},
		{"unknown technology", "apiVersion: provisionhub.dev/v1alpha1\nkind: Project\nmetadata:\n  name: demo\nspec:\n  backend:\n    technology: missing\n", planning.InvalidTechnology},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, project.ManifestName)
			if test.name != "missing" {
				if err := os.WriteFile(path, []byte(test.data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			before := projectSnapshot(t, root)
			plan, err := NewProjectService().PlanProject(path)
			if err == nil || plan.Status != planning.StatusInvalid || plan.Executable || len(plan.Operations) != 0 || len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != test.code || plan.Diagnostics[0].Severity != planning.SeverityError {
				t.Fatalf("plan=%+v, error=%v", plan, err)
			}
			if after := projectSnapshot(t, root); !reflect.DeepEqual(before, after) {
				t.Fatal("failed planning changed files")
			}
		})
	}
}

func TestPlanManifestRetainsExplicitConflictValidation(t *testing.T) {
	c, err := catalog.New([]catalog.Technology{
		{ID: "go", Name: "Go", Category: catalog.Backend, Description: "Backend", IncompatibleWith: []string{"mysql"}},
		{ID: "mysql", Name: "MySQL", Category: catalog.Database, Description: "Database"},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "demo"}, Spec: project.Spec{
		Backend: project.TechnologySelection{Technology: "go"}, Database: &project.TechnologySelection{Technology: "mysql"},
	}}
	service := ProjectService{catalog: c}
	plan, err := service.PlanManifest(manifest)
	if err == nil || plan.Status != planning.StatusInvalid || len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != planning.IncompatibleTechnologies {
		t.Fatalf("conflict plan=%+v, error=%v", plan, err)
	}
}
