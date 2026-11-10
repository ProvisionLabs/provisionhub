package cli

import (
	"fmt"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/project"
	"github.com/spf13/cobra"
)

func newProjectCommand() *cobra.Command {
	command := &cobra.Command{Use: "project", Short: "Plan and generate local source files"}
	command.AddCommand(newPlanCommand(), newApplyCommand())
	return command
}

type specificationFlags struct {
	name, destination string
	components        []string
	settings          []string
}

func (f *specificationFlags) bind(command *cobra.Command) {
	command.Flags().StringVar(&f.name, "name", "", "Project name")
	command.Flags().StringVar(&f.destination, "path", "", "Output directory")
	command.Flags().StringArrayVar(&f.components, "component", nil, "Named component name:type:technology (repeatable)")
	command.Flags().StringArrayVar(&f.settings, "set", nil, "Component configuration component.key=value (repeatable)")
	_ = command.MarkFlagRequired("name")
	_ = command.MarkFlagRequired("path")
	_ = command.MarkFlagRequired("component")
}

func (f specificationFlags) build(service application.ProjectService) (project.Specification, error) {
	components := make([]project.Component, 0, len(f.components))
	for _, value := range f.components {
		parts := strings.Split(value, ":")
		if len(parts) != 3 {
			return project.Specification{}, fmt.Errorf("invalid component %q; expected name:type:technology", value)
		}
		components = append(components, project.Component{Name: parts[0], Type: parts[1], Technology: parts[2]})
	}
	spec, err := service.ValidateComponentsInput(f.name, f.destination, components)
	if err != nil {
		return project.Specification{}, err
	}
	return service.ConfigureSpecification(spec, f.settings)
}
