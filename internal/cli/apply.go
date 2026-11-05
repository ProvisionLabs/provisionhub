package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/spf13/cobra"
)

func newApplyCommand() *cobra.Command {
	var pluginDirectory string
	var specification specificationFlags
	var pluginVersions []string
	var yes bool
	command := &cobra.Command{
		Use: "apply", Short: "Review and generate local project files from installed plugins",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := application.NewProjectServiceWithPlugins(pluginDirectory)
			if err != nil {
				return err
			}
			service, err = service.WithPluginVersions(pluginVersions)
			if err != nil {
				return err
			}
			spec, err := specification.build(service)
			if err != nil {
				return err
			}
			plan, err := service.PlanSpecification(spec)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprint(cmd.OutOrStdout(), formatPlan(plan)); err != nil {
				return err
			}
			if !plan.Executable {
				return errors.New("plan is not executable; install generators for every selected technology")
			}
			if err := service.PreflightSpecification(spec, specification.destination, plan); err != nil {
				return err
			}
			if !yes {
				input, ok := cmd.InOrStdin().(*os.File)
				if !ok {
					return errors.New("interactive approval requires a terminal; use --yes after reviewing the plan")
				}
				info, err := input.Stat()
				if err != nil || info.Mode()&os.ModeCharDevice == 0 {
					return errors.New("interactive approval requires a terminal; use --yes after reviewing the plan")
				}
				if _, err := fmt.Fprint(cmd.OutOrStdout(), "\nType yes to generate these files: "); err != nil {
					return err
				}
				line, err := bufio.NewReader(input).ReadString('\n')
				if err != nil {
					return fmt.Errorf("read approval: %w", err)
				}
				if strings.TrimSpace(line) != "yes" {
					return errors.New("generation cancelled")
				}
			}
			result, err := service.ApplySpecification(spec, specification.destination, plan)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "\nCreated %d files in %s\n", len(result.Created), result.Destination)
			return err
		},
	}
	specification.bind(command)
	command.Flags().StringVar(&pluginDirectory, "plugins", "", "Local plugin directory (required for generation)")
	command.Flags().StringArrayVar(&pluginVersions, "plugin-version", nil, "Select installed namespace/name@x.y.z (repeatable)")
	command.Flags().BoolVar(&yes, "yes", false, "Approve the reviewed plan without an interactive prompt")
	return command
}
