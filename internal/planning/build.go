package planning

import (
	"fmt"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

// Build constructs a conceptual plan. Catalog membership does not imply a generator exists.
func Build(manifest project.Manifest, technologies catalog.Catalog) Plan {
	plan := Plan{
		ProjectName: manifest.Metadata.Name, ManifestVersion: manifest.APIVersion,
		Technologies: []Technology{}, Operations: []Operation{}, Diagnostics: []Diagnostic{}, Status: StatusInvalid,
	}
	if err := manifest.Validate(); err != nil {
		plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Severity: SeverityError, Code: InvalidManifest, Message: err.Error()})
		return finalizePlan(plan)
	}
	type selection struct {
		category catalog.Category
		id       string
	}
	selections := []selection{{catalog.Backend, manifest.Spec.Backend.Technology}}
	if manifest.Spec.Frontend != nil {
		selections = append(selections, selection{catalog.Frontend, manifest.Spec.Frontend.Technology})
	}
	if manifest.Spec.Database != nil {
		selections = append(selections, selection{catalog.Database, manifest.Spec.Database.Technology})
	}
	var ids []string
	for _, selection := range selections {
		technology, err := technologies.ValidateSelection(selection.category, selection.id)
		if err != nil {
			plan.Diagnostics = append(plan.Diagnostics, Diagnostic{
				Severity: SeverityError, Code: InvalidTechnology, Message: err.Error(), Technology: selection.id,
				Field: "spec." + string(selection.category) + ".technology",
			})
			continue
		}
		ids = append(ids, technology.ID)
		plan.Technologies = append(plan.Technologies, Technology{ID: technology.ID, Name: technology.Name, Category: technology.Category})
	}
	if len(plan.Diagnostics) != 0 {
		return finalizePlan(plan)
	}
	if err := technologies.ValidateCombination(ids...); err != nil {
		plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Severity: SeverityError, Code: IncompatibleTechnologies, Message: err.Error()})
		return finalizePlan(plan)
	}
	plan.Status = StatusPlanned
	for _, technology := range plan.Technologies {
		operation := Operation{
			ID: string(technology.Category) + ":" + technology.ID, Technology: technology.ID,
			TargetPaths: []string{}, Dependencies: []string{}, Status: OperationUnavailable,
		}
		var message string
		switch technology.Category {
		case catalog.Backend:
			operation.Type, operation.Description = GenerateBackend, "Generate backend project"
			message = fmt.Sprintf("No backend generator is registered for %s.", technology.Name)
		case catalog.Frontend:
			operation.Type, operation.Description = GenerateFrontend, "Generate frontend project"
			message = fmt.Sprintf("No frontend generator is registered for %s.", technology.Name)
		case catalog.Database:
			operation.Type, operation.Description = IntegrateDatabase, "Integrate database with backend"
			operation.Dependencies = []string{"backend:" + manifest.Spec.Backend.Technology}
			message = fmt.Sprintf("No database integration provider is registered for %s.", technology.Name)
		}
		plan.Operations = append(plan.Operations, operation)
		plan.Diagnostics = append(plan.Diagnostics, Diagnostic{
			Severity: SeverityWarning, Code: MissingOperationProvider,
			Message:     message,
			OperationID: operation.ID, Technology: technology.ID,
		})
	}
	return finalizePlan(plan)
}
