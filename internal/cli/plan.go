package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
	"github.com/spf13/cobra"
)

func newPlanCommand() *cobra.Command {
	var file, output string
	command := &cobra.Command{
		Use: "plan", Short: "Preview generation capabilities without writing or executing anything",
		Long: "Preview a generation plan without writing project files or executing operations.\nExit 0 means a plan was constructed, even when execution is unavailable.\nExit 1 means an input, loading, validation, planning, or output failure.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if output != "text" && output != "json" {
				return fmt.Errorf("unsupported output format %q (expected text or json)", output)
			}
			plan, planErr := application.NewProjectService().PlanProject(file)
			if output == "json" {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(plan); err != nil {
					return err
				}
			} else if _, err := fmt.Fprint(cmd.OutOrStdout(), formatPlan(plan)); err != nil {
				return err
			}
			return planErr
		},
	}
	command.Flags().StringVar(&file, "file", "", "Path to provisionhub.yaml (required)")
	command.Flags().StringVar(&output, "output", "text", "Output format: text or json")
	_ = command.MarkFlagRequired("file")
	return command
}

func formatPlan(plan planning.Plan) string {
	var out strings.Builder
	fmt.Fprintf(&out, "ProvisionHub — Generation Plan\n\nProject: %s\nManifest version: %s\nStatus: %s\nExecution: unavailable (no execution engine)\n", plan.ProjectName, plan.ManifestVersion, plan.Status)
	fmt.Fprintln(&out, "\nSelected Technologies:")
	for _, technology := range plan.Technologies {
		fmt.Fprintf(&out, "  %s: %s (%s)\n", technology.Category, technology.Name, technology.ID)
	}
	fmt.Fprintln(&out, "\nPlanned Capabilities:")
	for _, operation := range plan.Operations {
		fmt.Fprintf(&out, "  %s: %s [%s]\n", operation.ID, operation.Description, operation.Status)
		if len(operation.Dependencies) != 0 {
			fmt.Fprintf(&out, "    Depends on: %s\n", strings.Join(operation.Dependencies, ", "))
		}
	}
	fmt.Fprintln(&out, "\nDiagnostics:")
	for _, diagnostic := range plan.Diagnostics {
		fmt.Fprintf(&out, "  %s [%s]: %s\n", diagnostic.Severity, diagnostic.Code, diagnostic.Message)
	}
	return out.String()
}
