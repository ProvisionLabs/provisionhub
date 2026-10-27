package planning

import (
	"reflect"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestBuildResolvesOptionalCapabilities(t *testing.T) {
	for _, optional := range []struct{ frontend, database bool }{{}, {frontend: true}, {database: true}, {true, true}} {
		manifest := testManifest()
		wantIDs := []string{"backend:nestjs"}
		if optional.frontend {
			manifest.Spec.Frontend = &project.TechnologySelection{Technology: "nextjs"}
			wantIDs = append(wantIDs, "frontend:nextjs")
		}
		if optional.database {
			manifest.Spec.Database = &project.TechnologySelection{Technology: "postgresql"}
			wantIDs = append(wantIDs, "database:postgresql")
		}
		plan := Build(manifest, catalog.Builtin())
		if plan.Status != StatusPlanned || plan.Executable || len(plan.Technologies) != len(wantIDs) || len(plan.Operations) != len(wantIDs) || len(plan.Diagnostics) != len(wantIDs) {
			t.Fatalf("optional=%+v: plan = %+v", optional, plan)
		}
		for _, id := range wantIDs {
			found := false
			for _, operation := range plan.Operations {
				if operation.ID != id {
					continue
				}
				found = true
				if operation.Status != OperationUnavailable || len(operation.TargetPaths) != 0 {
					t.Fatalf("unsupported operation must not invent targets/support: %+v", operation)
				}
				if operation.Type == IntegrateDatabase && !reflect.DeepEqual(operation.Dependencies, []string{"backend:nestjs"}) {
					t.Fatalf("database dependencies = %v", operation.Dependencies)
				}
			}
			if !found {
				t.Fatalf("missing stable operation ID %q in %+v", id, plan.Operations)
			}
		}
		encoded, err := project.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		equivalent, err := project.Unmarshal(encoded)
		if err != nil {
			t.Fatal(err)
		}
		for range 25 {
			if again := Build(equivalent, catalog.Builtin()); !reflect.DeepEqual(plan, again) {
				t.Fatalf("equivalent manifest produced a different plan: %+v", again)
			}
		}
	}
}
