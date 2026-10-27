package planning

import (
	"reflect"
	"slices"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestDependencyDiagnosticsAreStableAcrossGraphPermutations(t *testing.T) {
	input := []Operation{
		{ID: "z", Dependencies: []string{"missing-b", "missing-a"}},
		{ID: "a", Dependencies: []string{"missing-c"}},
	}
	want := []Diagnostic{
		{Severity: SeverityError, Code: InvalidDependency, OperationID: "a", Message: `Operation "a" depends on unknown operation "missing-c".`},
		{Severity: SeverityError, Code: InvalidDependency, OperationID: "z", Message: `Operation "z" depends on unknown operation "missing-a".`},
		{Severity: SeverityError, Code: InvalidDependency, OperationID: "z", Message: `Operation "z" depends on unknown operation "missing-b".`},
	}
	for range 20 {
		plan := finalizePlan(Plan{Status: StatusPlanned, Operations: input})
		if plan.Status != StatusInvalid || len(plan.Operations) != 0 || !reflect.DeepEqual(plan.Diagnostics, want) {
			t.Fatalf("diagnostics lost or reordered: %+v", plan)
		}
		slices.Reverse(input)
		for i := range input {
			slices.Reverse(input[i].Dependencies)
		}
	}
}

func TestMultipleInvalidTechnologyDiagnosticsRemainStable(t *testing.T) {
	manifest := testManifest()
	manifest.Spec.Backend.Technology = "unknown-z"
	manifest.Spec.Frontend = &project.TechnologySelection{Technology: "unknown-a"}
	manifest.Spec.Database = &project.TechnologySelection{Technology: "unknown-m"}
	first := Build(manifest, catalog.Builtin())
	var fields []string
	for _, diagnostic := range first.Diagnostics {
		if diagnostic.Code != InvalidTechnology || diagnostic.Severity != SeverityError {
			t.Fatalf("diagnostic = %+v", diagnostic)
		}
		fields = append(fields, diagnostic.Field)
	}
	if !reflect.DeepEqual(fields, []string{"spec.frontend.technology", "spec.database.technology", "spec.backend.technology"}) {
		t.Fatalf("missing or unordered error fields: %v", fields)
	}
	for range 25 {
		if again := Build(manifest, catalog.Builtin()); !reflect.DeepEqual(first, again) {
			t.Fatalf("unstable invalid plan: %+v", again)
		}
	}
}

func TestBuildResolvesEveryBuiltinTechnologyWithoutInventingSupport(t *testing.T) {
	c := catalog.Builtin()
	for _, backend := range c.List(catalog.Backend) {
		for _, frontend := range append([]catalog.Technology{{}}, c.List(catalog.Frontend)...) {
			for _, database := range append([]catalog.Technology{{}}, c.List(catalog.Database)...) {
				manifest := testManifest()
				manifest.Spec.Backend.Technology = backend.ID
				count := 1
				if frontend.ID != "" {
					manifest.Spec.Frontend = &project.TechnologySelection{Technology: frontend.ID}
					count++
				}
				if database.ID != "" {
					manifest.Spec.Database = &project.TechnologySelection{Technology: database.ID}
					count++
				}
				plan := Build(manifest, c)
				if plan.Status != StatusPlanned || plan.Executable || len(plan.Technologies) != count || len(plan.Operations) != count || len(plan.Diagnostics) != count {
					t.Fatalf("built-in selections rejected: %+v", plan)
				}
				for _, operation := range plan.Operations {
					if operation.Status != OperationUnavailable || len(operation.TargetPaths) != 0 {
						t.Fatalf("invented support/targets: %+v", operation)
					}
				}
			}
		}
	}
}
