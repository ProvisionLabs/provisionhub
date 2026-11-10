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
	var output, pluginDirectory string
	var specification specificationFlags
	var pluginVersions []string
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
			service, err = service.WithPluginVersions(pluginVersions)
			if err != nil {
				return err
			}
			spec, err := specification.build(service)
			if err != nil {
				return err
			}
			plan, planErr := service.PlanSpecification(spec)
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
	specification.bind(command)
	command.Flags().StringVar(&output, "output", "text", "Output format: text or json")
	command.Flags().StringVar(&pluginDirectory, "plugins", "", "Local plugin directory (optional)")
	command.Flags().StringArrayVar(&pluginVersions, "plugin-version", nil, "Select installed namespace/name@x.y.z (repeatable)")
	return command
}

func formatPlan(plan planning.Plan) string {
	var out strings.Builder
	execution := "unavailable (missing generation capabilities)"
	if plan.Executable {
		execution = "available after explicit approval"
	}
	fmt.Fprintf(&out, "ProvisionHub — Generation Plan\n\nProject: %s\nStatus: %s\nExecution: %s\n", plan.ProjectName, plan.Status, execution)
	if len(plan.Components) > 0 {
		fmt.Fprintln(&out, "\nComponents:")
		for _, component := range plan.Components {
			fmt.Fprintf(&out, "  %s: %s (%s, %s)\n", component.ID, component.Name, component.Type, component.Technology)
			if len(component.Configuration) > 0 {
				data, _ := json.Marshal(component.Configuration)
				fmt.Fprintf(&out, "    Configuration: %s\n", data)
			}
		}
	}
	if len(plan.Connections) > 0 {
		fmt.Fprintln(&out, "\nConnections:")
		for _, connection := range plan.Connections {
			fmt.Fprintf(&out, "  %s -> %s (%s)\n", connection.From, connection.To, connection.Type)
			if connection.ConsumerID != "" {
				fmt.Fprintf(&out, "    Capabilities: %s consumes %s\n", connection.ConsumerID, connection.ProviderID)
			}
			if len(connection.Configuration) > 0 {
				data, _ := json.Marshal(connection.Configuration)
				fmt.Fprintf(&out, "    Configuration: %s\n", data)
			}
			if len(connection.Metadata) > 0 {
				data, _ := json.Marshal(connection.Metadata)
				fmt.Fprintf(&out, "    Target metadata: %s\n", data)
			}
			for _, output := range connection.Outputs {
				fmt.Fprintf(&out, "    Output: %s\n", output)
			}
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
