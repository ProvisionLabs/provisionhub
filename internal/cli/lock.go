package cli

import (
	"fmt"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/spf13/cobra"
)

func newLockCommand() *cobra.Command {
	var file, pluginDirectory string
	var versions []string
	var check, update, restore bool
	command := &cobra.Command{Use: "lock", Short: "Create, check, update, or restore a project plugin lock", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if (check && update) || (check && restore) || (update && restore) {
			return fmt.Errorf("--check, --update, and --restore are mutually exclusive")
		}
		service, err := application.NewProjectServiceWithPlugins(pluginDirectory)
		if err != nil {
			return err
		}
		service, err = service.WithPluginVersions(versions)
		if err != nil {
			return err
		}
		if restore {
			if err := service.RestoreLockedProject(cmd.Context(), file); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Locked plugins restored")
			return err
		}
		lock, err := service.LockProject(file, check, update)
		if err != nil {
			return err
		}
		action := "Created"
		if check {
			action = "Validated"
		}
		if update {
			action = "Updated"
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s project lock (%d plugins)\n", action, len(lock.Plugins))
		return err
	}}
	command.Flags().StringVar(&file, "file", "", "Path to provisionhub.yaml (required)")
	command.Flags().StringVar(&pluginDirectory, "plugins", "", "Local plugin directory")
	command.Flags().StringArrayVar(&versions, "plugin-version", nil, "Select installed namespace/name@x.y.z (repeatable)")
	command.Flags().BoolVar(&check, "check", false, "Validate the existing lock and installed plugins")
	command.Flags().BoolVar(&update, "update", false, "Intentionally regenerate the lock from installed selections")
	command.Flags().BoolVar(&restore, "restore", false, "Explicitly restore missing exact official releases")
	_ = command.MarkFlagRequired("file")
	return command
}
