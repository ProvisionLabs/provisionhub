package planning

import (
	"encoding/hex"
	"fmt"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

// Build constructs a conceptual plan. Catalog membership does not imply a generator exists.
func Build(manifest project.Manifest, technologies catalog.Catalog) Plan {
	return BuildWithPlugins(manifest, technologies, plugins.Set{})
}

// BuildWithPlugins adds concrete file operations for locally loaded generators.
// Plugin files have already been read and validated; this function has no I/O.
func BuildWithPlugins(manifest project.Manifest, technologies catalog.Catalog, loaded plugins.Set) Plan {
	plan := Plan{
		ProjectName: manifest.Metadata.Name, ManifestVersion: manifest.APIVersion,
		Technologies: []Technology{}, Operations: []Operation{}, Diagnostics: []Diagnostic{}, Status: StatusInvalid,
	}
	if err := manifest.Validate(); err != nil {
		plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Severity: SeverityError, Code: InvalidManifest, Message: err.Error()})
		return finalizePlan(plan)
	}
	components := manifest.CanonicalComponents()
	if manifest.UsesComponents() {
		plan.Components = []Component{}
		plan.Connections = manifest.CanonicalConnections()
	}
	var ids []string
	for _, component := range components {
		technology, err := technologies.ValidateSelection(catalog.Category(component.Type), component.Technology)
		if err != nil {
			field := "spec." + component.Type + ".technology"
			componentID := ""
			if !component.Legacy {
				field = "spec.components." + component.Name + ".technology"
				componentID = component.ID()
			}
			plan.Diagnostics = append(plan.Diagnostics, Diagnostic{
				Severity: SeverityError, Code: InvalidTechnology, Message: err.Error(), Technology: component.Technology,
				ComponentID: componentID, Field: field,
			})
			continue
		}
		ids = append(ids, technology.ID)
		entry := Technology{ID: technology.ID, Name: technology.Name, Category: technology.Category}
		if !component.Legacy {
			entry.ComponentID = component.ID()
		}
		plan.Technologies = append(plan.Technologies, entry)
	}
	if len(plan.Diagnostics) != 0 {
		return finalizePlan(plan)
	}
	if err := technologies.ValidateCombination(ids...); err != nil {
		plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Severity: SeverityError, Code: IncompatibleTechnologies, Message: err.Error()})
		return finalizePlan(plan)
	}
	plan.Status = StatusPlanned
	for i, technology := range plan.Technologies {
		component := components[i]
		componentID := ""
		operationID := string(technology.Category) + ":" + technology.ID
		if !component.Legacy {
			componentID = component.ID()
			operationID = componentID
		}
		operation := Operation{
			ID: operationID, Technology: technology.ID, ComponentID: componentID,
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
			if component.Legacy {
				operation.Dependencies = []string{"backend:" + manifest.Spec.Backend.Technology}
			}
			message = fmt.Sprintf("No database integration provider is registered for %s.", technology.Name)
		}
		planComponent := Component{ID: component.ID(), Name: component.Name, Type: technology.Category, Technology: technology.ID}
		if plugin, ok := loaded.ForTechnology(technology.ID); ok &&
			(technology.Category == catalog.Backend || technology.Category == catalog.Frontend) {
			operation.Status = OperationSupported
			operation.PluginID = plugin.ID
			operation.PluginDigest = hex.EncodeToString(plugin.Digest[:])
			planComponent.PluginID = plugin.ID
			for _, file := range plugin.Files {
				target := plugins.DestinationFor(component, file)
				operation.TargetPaths = append(operation.TargetPaths, target)
				plan.Operations = append(plan.Operations, Operation{
					ID:          "file:" + plugin.ID + ":" + target,
					Description: "Create " + target,
					Type:        GenerateFile, Technology: technology.ID, ComponentID: componentID, PluginID: plugin.ID,
					PluginDigest: hex.EncodeToString(plugin.Digest[:]),
					Source:       file.Source, SourceType: file.Type,
					SourceDigest: hex.EncodeToString(file.Digest[:]),
					TargetPaths:  []string{target}, Dependencies: []string{operation.ID}, Status: OperationSupported,
				})
			}
		} else {
			plan.Diagnostics = append(plan.Diagnostics, Diagnostic{
				Severity: SeverityWarning, Code: MissingOperationProvider,
				Message: message, OperationID: operation.ID, Technology: technology.ID, ComponentID: componentID,
			})
		}
		plan.Operations = append(plan.Operations, operation)
		if manifest.UsesComponents() {
			plan.Components = append(plan.Components, planComponent)
		}
	}
	return finalizePlan(plan)
}
