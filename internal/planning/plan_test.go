package planning

import (
	"reflect"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func testManifest() project.Manifest {
	return project.Manifest{
		APIVersion: project.APIVersion, Kind: project.Kind,
		Metadata: project.Metadata{Name: "my-application"},
		Spec:     project.Spec{Backend: &project.TechnologySelection{Technology: "nestjs"}},
	}
}

func TestBuildDescribesUnavailableBackendGeneration(t *testing.T) {
	manifest := testManifest()
	plan := Build(manifest, catalog.Builtin())
	if plan.ProjectName != manifest.Metadata.Name || plan.ManifestVersion != project.APIVersion || plan.Status != StatusPlanned || plan.Executable {
		t.Fatalf("plan metadata = %+v", plan)
	}
	wantTechnologies := []Technology{{ID: "nestjs", Name: "NestJS", Category: catalog.Backend}}
	if !reflect.DeepEqual(plan.Technologies, wantTechnologies) {
		t.Fatalf("technologies = %+v", plan.Technologies)
	}
	wantOperation := Operation{
		ID: "backend:nestjs", Description: "Generate backend project", Type: GenerateBackend,
		Technology: "nestjs", TargetPaths: []string{}, Dependencies: []string{}, Status: OperationUnavailable,
	}
	if !reflect.DeepEqual(plan.Operations, []Operation{wantOperation}) {
		t.Fatalf("operations = %+v", plan.Operations)
	}
	if len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != MissingOperationProvider || plan.Diagnostics[0].Severity != SeverityWarning || plan.Diagnostics[0].OperationID != wantOperation.ID {
		t.Fatalf("diagnostics = %+v", plan.Diagnostics)
	}
}
