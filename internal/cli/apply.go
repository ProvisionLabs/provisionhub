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
	var file, pluginDirectory string
	var yes bool
	command := &cobra.Command{
		Use: "apply", Short: "Review and generate local project files from installed plugins",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := application.NewProjectServiceWithPlugins(pluginDirectory)
			if err != nil {
				return err
			}
			plan, err := service.PlanProject(file)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprint(cmd.OutOrStdout(), formatPlan(plan)); err != nil {
				return err
			}
			if !plan.Executable {
				return errors.New("plan is not executable; install generators for every selected technology")
			}
			if err := service.PreflightProject(file, plan); err != nil {
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
			result, err := service.ApplyProject(file, plan)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "\nCreated %d files in %s\n", len(result.Created), result.Destination)
			return err
		},
	}
	command.Flags().StringVar(&file, "file", "", "Path to provisionhub.yaml (required)")
	command.Flags().StringVar(&pluginDirectory, "plugins", "", "Local plugin directory (required for generation)")
	command.Flags().BoolVar(&yes, "yes", false, "Approve the reviewed plan without an interactive prompt")
	_ = command.MarkFlagRequired("file")
	return command
}
