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
	var file, output, pluginDirectory string
	command := &cobra.Command{
		Use: "plan", Short: "Preview generation capabilities without writing or executing anything",
		Long: "Preview a generation plan without writing project files or executing operations.\nExit 0 means a plan was constructed, even when execution is unavailable.\nExit 1 means an input, loading, validation, planning, or output failure.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if output != "text" && output != "json" {
				return fmt.Errorf("unsupported output format %q (expected text or json)", output)
			}
			service, err := application.NewProjectServiceWithPlugins(pluginDirectory)
			if err != nil {
				return err
			}
			plan, planErr := service.PlanProject(file)
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
	command.Flags().StringVar(&pluginDirectory, "plugins", "", "Local plugin directory (optional)")
	_ = command.MarkFlagRequired("file")
	return command
}

func formatPlan(plan planning.Plan) string {
	var out strings.Builder
	execution := "unavailable (missing generation capabilities)"
	if plan.Executable {
		execution = "available after explicit approval"
	}
	fmt.Fprintf(&out, "ProvisionHub — Generation Plan\n\nProject: %s\nManifest version: %s\nStatus: %s\nExecution: %s\n", plan.ProjectName, plan.ManifestVersion, plan.Status, execution)
	if len(plan.Components) > 0 {
		fmt.Fprintln(&out, "\nComponents:")
		for _, component := range plan.Components {
			fmt.Fprintf(&out, "  %s: %s (%s, %s)\n", component.ID, component.Name, component.Type, component.Technology)
		}
	}
	fmt.Fprintln(&out, "\nSelected Technologies:")
	for _, technology := range plan.Technologies {
		fmt.Fprintf(&out, "  %s: %s (%s)\n", technology.Category, technology.Name, technology.ID)
	}
	fmt.Fprintln(&out, "\nPlanned Capabilities:")
	for _, operation := range plan.Operations {
		fmt.Fprintf(&out, "  %s: %s [%s]\n", operation.ID, operation.Description, operation.Status)
		if operation.PluginID != "" {
			fmt.Fprintf(&out, "    Plugin: %s\n", operation.PluginID)
			fmt.Fprintf(&out, "    Plugin digest: %s\n", operation.PluginDigest)
		}
		if operation.Source != "" {
			fmt.Fprintf(&out, "    Source: %s (%s, sha256:%s)\n", operation.Source, operation.SourceType, operation.SourceDigest)
		}
		for _, target := range operation.TargetPaths {
			fmt.Fprintf(&out, "    Target: %s\n", target)
		}
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
