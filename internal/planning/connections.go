package planning

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/ProvisionLabs/provisionhub/internal/connections"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func buildConnectionOperations(projectName string, components []project.Component, resolved []connections.Resolved, loaded plugins.Set) ([]Operation, error) {
	operations := []Operation{}
	byName := map[string]project.Component{}
	for _, c := range components {
		byName[c.Name] = c
	}
	for _, connection := range resolved {
		source, target := byName[connection.From], byName[connection.To]
		plugin, _ := loaded.ForTechnology(source.Technology)
		consumer, ok := plugin.Consumer(connection.ConsumerID)
		if !ok {
			return nil, fmt.Errorf("connection %s consumer changed", connection.ID())
		}
		op := Operation{
			ID: connection.OperationID(), Type: ConnectComponents,
			Description: fmt.Sprintf("Connect %s to %s (%s)", source.Name, target.Name, connection.Type),
			ComponentID: source.ID(), ConnectionID: connection.ID(), Technology: source.Technology,
			PluginID: plugin.ID, PluginDigest: hex.EncodeToString(plugin.Digest[:]),
			TargetPaths: connection.Outputs, Dependencies: []string{source.ID(), target.ID()}, Status: OperationSupported,
		}
		for _, component := range []project.Component{source, target} {
			definition, _ := loaded.ForTechnology(component.Technology)
			for _, file := range definition.Files {
				op.Dependencies = append(op.Dependencies, "file:"+definition.ID+":"+plugins.DestinationFor(component, file))
			}
		}
		for _, file := range consumer.Files {
			data, err := plugins.RenderIntegration(file, projectName, source, target, plugin.Configuration, consumer, connection.Configuration, connection.Metadata)
			if err != nil {
				return nil, fmt.Errorf("connection %s file %q: %w", connection.ID(), file.Source, err)
			}
			rendered := sha256.Sum256(data)
			targetPath := plugins.DestinationFor(source, file)
			operations = append(operations, Operation{
				ID: "file:" + plugin.ID + ":" + targetPath, Type: GenerateFile, Description: "Create " + targetPath,
				ComponentID: source.ID(), ConnectionID: connection.ID(), Technology: source.Technology,
				PluginID: plugin.ID, PluginDigest: hex.EncodeToString(plugin.Digest[:]),
				Source: file.Source, SourceType: file.Type, SourceDigest: hex.EncodeToString(file.Digest[:]),
				RenderedDigest: hex.EncodeToString(rendered[:]), TargetPaths: []string{targetPath},
				Dependencies: []string{op.ID}, Status: OperationSupported,
			})
		}
		operations = append(operations, op)
	}
	return operations, nil
}
