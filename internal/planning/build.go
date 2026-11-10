package planning

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/connections"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

// Build constructs a conceptual plan. Catalog membership does not imply a generator exists.
func Build(manifest project.Specification, technologies catalog.Catalog) Plan {
	return BuildWithPlugins(manifest, technologies, plugins.Set{})
}

// BuildWithPlugins adds concrete file operations for locally loaded generators.
// Plugin files have already been read and validated; this function has no I/O.
func BuildWithPlugins(manifest project.Specification, technologies catalog.Catalog, loaded plugins.Set) Plan {
	plan := Plan{
		ProjectName: manifest.Metadata.Name, SpecificationVersion: manifest.APIVersion,
		Technologies: []Technology{}, Operations: []Operation{}, Diagnostics: []Diagnostic{}, Status: StatusInvalid,
	}
	for _, c := range manifest.CanonicalConnections() {
		plan.Connections = append(plan.Connections, connections.Resolved{Connection: c})
	}
	if err := manifest.Validate(); err != nil {
		plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Severity: SeverityError, Code: InvalidSpecification, Message: err.Error()})
		return finalizePlan(plan)
	}
	components := manifest.CanonicalComponents()
	if manifest.UsesComponents() {
		plan.Components = []Component{}
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
	// Resolve all instances before contributing operations. Plugin definitions
	// remain immutable, including when multiple components share a technology.
	renderedDigests := map[string]map[string][32]byte{}
	for i := range components {
		component := &components[i]
		renderedDigests[component.ID()] = map[string][32]byte{}
		plugin, _ := loaded.ForTechnology(component.Technology)
		values, err := configuration.Resolve(plugin.Configuration, component.Configuration, component.Name)
		if err == nil {
			component.Configuration = values
			for _, file := range plugin.Files {
				data, renderErr := plugins.Render(file, manifest.Metadata.Name, *component, plugin.Configuration, values)
				if renderErr != nil {
					err = fmt.Errorf("component %q file %q: %w", component.Name, file.Source, renderErr)
					break
				}
				renderedDigests[component.ID()][file.Destination] = sha256.Sum256(data)
			}
		}
		if err != nil {
			plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Severity: SeverityError, Code: InvalidConfiguration, Message: err.Error(), ComponentID: component.ID(), Field: "spec.components." + component.Name + ".configuration"})
		}
	}
	if len(plan.Diagnostics) > 0 {
		return finalizePlan(plan)
	}
	if manifest.UsesComponents() {
		for _, component := range components {
			plugin, exists := loaded.ForTechnology(component.Technology)
			var values configuration.Values
			if exists {
				values = component.Configuration
			}
			plan.Components = append(plan.Components, Component{
				ID: component.ID(), Name: component.Name, Type: catalog.Category(component.Type),
				Technology: component.Technology, PluginID: plugin.ID, Configuration: values,
			})
		}
	}
	resolved, err := connections.Resolve(components, manifest.CanonicalConnections(), loaded)
	if err != nil {
		code := InvalidConnection
		if errors.Is(err, connections.ErrCycle) {
			code = DependencyCycle
		}
		plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Severity: SeverityError, Code: code, Field: "spec.connections", Message: err.Error()})
		return finalizePlan(plan)
	}
	plan.Connections = resolved
	connectionOperations, err := buildConnectionOperations(manifest.Metadata.Name, components, resolved, loaded)
	if err != nil {
		plan.Diagnostics = append(plan.Diagnostics, Diagnostic{Severity: SeverityError, Code: InvalidConnection, Field: "spec.connections", Message: err.Error()})
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
		for _, connection := range resolved {
			if connection.From == component.Name {
				operation.Dependencies = append(operation.Dependencies, "component/"+connection.To)
			}
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
		if plugin, ok := loaded.ForTechnology(technology.ID); ok &&
			(technology.Category == catalog.Backend || technology.Category == catalog.Frontend) {
			operation.Status = OperationSupported
			operation.PluginID = plugin.ID
			operation.PluginDigest = hex.EncodeToString(plugin.Digest[:])
			for _, file := range plugin.Files {
				renderedDigest := renderedDigests[component.ID()][file.Destination]
				target := plugins.DestinationFor(component, file)
				operation.TargetPaths = append(operation.TargetPaths, target)
				plan.Operations = append(plan.Operations, Operation{
					ID:          "file:" + plugin.ID + ":" + target,
					Description: "Create " + target,
					Type:        GenerateFile, Technology: technology.ID, ComponentID: componentID, PluginID: plugin.ID,
					PluginDigest: hex.EncodeToString(plugin.Digest[:]),
					Source:       file.Source, SourceType: file.Type,
					SourceDigest:   hex.EncodeToString(file.Digest[:]),
					RenderedDigest: hex.EncodeToString(renderedDigest[:]),
					TargetPaths:    []string{target}, Dependencies: []string{operation.ID}, Status: OperationSupported,
				})
			}
		} else {
			plan.Diagnostics = append(plan.Diagnostics, Diagnostic{
				Severity: SeverityWarning, Code: MissingOperationProvider,
				Message: message, OperationID: operation.ID, Technology: technology.ID, ComponentID: componentID,
			})
		}
		plan.Operations = append(plan.Operations, operation)
	}
	plan.Operations = append(plan.Operations, connectionOperations...)
	return finalizePlan(plan)
}
