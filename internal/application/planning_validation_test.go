package application

import (
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestPlanSpecificationRetainsExplicitConflictValidation(t *testing.T) {
	c, err := catalog.New([]catalog.Technology{
		{ID: "go", Name: "Go", Category: catalog.Backend, Description: "Backend", IncompatibleWith: []string{"mysql"}},
		{ID: "mysql", Name: "MySQL", Category: catalog.Database, Description: "Database"},
	})
	if err != nil {
		t.Fatal(err)
	}
	spec := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "demo"}, Spec: project.Spec{Components: []project.Component{
		{Name: "api", Type: "backend", Technology: "go"}, {Name: "data", Type: "database", Technology: "mysql"},
	}}}
	service := ProjectService{catalog: c, localOnly: true}
	plan, err := service.PlanSpecification(spec)
	if err == nil || plan.Status != planning.StatusInvalid || len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != planning.IncompatibleTechnologies {
		t.Fatalf("conflict plan=%+v, error=%v", plan, err)
	}
}
