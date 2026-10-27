package planning

import (
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestBuildRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*project.Manifest)
		code DiagnosticCode
	}{
		{"version", func(m *project.Manifest) { m.APIVersion = "unknown" }, InvalidManifest},
		{"kind", func(m *project.Manifest) { m.Kind = "Unknown" }, InvalidManifest},
		{"name", func(m *project.Manifest) { m.Metadata.Name = "Bad Name" }, InvalidManifest},
		{"missing backend", func(m *project.Manifest) { m.Spec.Backend.Technology = "" }, InvalidManifest},
		{"unknown backend", func(m *project.Manifest) { m.Spec.Backend.Technology = "missing" }, InvalidTechnology},
		{"wrong category", func(m *project.Manifest) { m.Spec.Backend.Technology = "react" }, InvalidTechnology},
		{"unknown frontend", func(m *project.Manifest) { m.Spec.Frontend = &project.TechnologySelection{Technology: "missing"} }, InvalidTechnology},
		{"unknown database", func(m *project.Manifest) { m.Spec.Database = &project.TechnologySelection{Technology: "missing"} }, InvalidTechnology},
		{"empty optional selection", func(m *project.Manifest) { m.Spec.Frontend = &project.TechnologySelection{} }, InvalidManifest},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest := testManifest()
			test.edit(&manifest)
			plan := Build(manifest, catalog.Builtin())
			if plan.Status != StatusInvalid || plan.Executable || len(plan.Operations) != 0 {
				t.Fatalf("invalid configuration produced a plan: %+v", plan)
			}
			if len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != test.code || plan.Diagnostics[0].Severity != SeverityError {
				t.Fatalf("diagnostics = %+v", plan.Diagnostics)
			}
		})
	}
}

func TestBuildRejectsExplicitTechnologyConflicts(t *testing.T) {
	c, err := catalog.New([]catalog.Technology{
		{ID: "nestjs", Name: "NestJS", Category: catalog.Backend, Description: "Backend", IncompatibleWith: []string{"postgresql"}},
		{ID: "postgresql", Name: "PostgreSQL", Category: catalog.Database, Description: "Database"},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := testManifest()
	manifest.Spec.Database = &project.TechnologySelection{Technology: "postgresql"}
	plan := Build(manifest, c)
	if plan.Status != StatusInvalid || plan.Executable || len(plan.Operations) != 0 || len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != IncompatibleTechnologies || plan.Diagnostics[0].Severity != SeverityError {
		t.Fatalf("conflicting selection = %+v", plan)
	}
}
