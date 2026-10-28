package cli

import (
	"fmt"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/spf13/cobra"
)

func newProjectCommand() *cobra.Command {
	projectCommand := &cobra.Command{
		Use:   "project",
		Short: "Manage local project configurations",
	}
	var input application.CreateProjectInput
	initCommand := &cobra.Command{
		Use:   "init",
		Short: "Create a provisionhub.yaml without generating source code",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := application.NewProjectService().CreateProject(input)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), result.ManifestPath)
			return err
		},
	}
	initCommand.Flags().StringVar(&input.Name, "name", "", "Project name (lowercase letters, digits, hyphens)")
	initCommand.Flags().StringVar(&input.Destination, "path", "", "Destination directory")
	initCommand.Flags().StringVar(&input.Backend, "backend", "", "Backend technology ID")
	initCommand.Flags().StringVar(&input.Frontend, "frontend", "", "Frontend technology ID or none (optional)")
	initCommand.Flags().StringVar(&input.Database, "database", "", "Database technology ID or none (optional)")
	for _, flag := range []string{"name", "path", "backend"} {
		_ = initCommand.MarkFlagRequired(flag)
	}
	projectCommand.AddCommand(initCommand, newPlanCommand(), newApplyCommand())
	return projectCommand
}
