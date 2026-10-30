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
	var source string
	initCommand := &cobra.Command{
		Use:   "init",
		Short: "Create a provisionhub.yaml without generating source code",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("path") {
				return fmt.Errorf("required flag(s) \"path\" not set")
			}
			service := application.NewProjectService()
			var result application.CreateProjectResult
			var err error
			if cmd.Flags().Changed("from") {
				if source == "" {
					return fmt.Errorf("--from requires a manifest path")
				}
				for _, flag := range []string{"name", "backend", "frontend", "database"} {
					if cmd.Flags().Changed(flag) {
						return fmt.Errorf("--from cannot be combined with --%s", flag)
					}
				}
				result, err = service.CreateProjectFromManifest(source, input.Destination)
			} else {
				if !cmd.Flags().Changed("name") || !cmd.Flags().Changed("backend") {
					return fmt.Errorf("required flag(s) \"name\", \"backend\" not set (or use --from)")
				}
				result, err = service.CreateProject(input)
			}
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
	initCommand.Flags().StringVar(&source, "from", "", "Existing project manifest for a multi-component configuration")
	projectCommand.AddCommand(initCommand, newPlanCommand(), newApplyCommand())
	return projectCommand
}
