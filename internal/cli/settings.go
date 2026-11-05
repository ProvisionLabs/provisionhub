package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/settings"
	"github.com/spf13/cobra"
)

func newSettingsCommand(backend *application.LocalSettings) *cobra.Command {
	command := &cobra.Command{Use: "settings", Short: "View or save local application preferences"}
	show := &cobra.Command{Use: "show", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		values, _, err := backend.Load(cmd.Context())
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(values)
	}}
	var values settings.Values
	save := &cobra.Command{Use: "set", Short: "Save specified preferences (empty clears a preference)", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		existing, _, err := backend.Load(cmd.Context())
		if err != nil && !errors.Is(err, settings.ErrInvalidStored) {
			return err
		}
		if cmd.Flags().Changed("workspace") {
			existing.Workspace = values.Workspace
		}
		if cmd.Flags().Changed("provider") {
			existing.PreferredProvider = values.PreferredProvider
		}
		if err := backend.Save(cmd.Context(), existing); err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "Settings saved.")
		return err
	}}
	save.Flags().StringVar(&values.Workspace, "workspace", "", "Default workspace (absolute existing directory)")
	save.Flags().StringVar(&values.PreferredProvider, "provider", "", "Preferred provider: github, gitlab, bitbucket, or empty")
	var validate bool
	status := &cobra.Command{Use: "git-status", Short: "List account state without revealing credentials", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		statuses, err := backend.Status(cmd.Context(), validate)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(statuses)
	}}
	status.Flags().BoolVar(&validate, "validate", false, "Explicitly validate credentials with the provider (requires network)")
	disconnect := &cobra.Command{Use: "git-disconnect <account-id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return backend.Disconnect(cmd.Context(), args[0])
	}}
	command.AddCommand(show, save, status, disconnect)
	return command
}
