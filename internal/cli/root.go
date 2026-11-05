package cli

import (
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/tui"
	"github.com/spf13/cobra"
)

// NewRootCommand constructs the command tree without starting an interactive session.
func NewRootCommand() *cobra.Command {
	return newRootCommand(func() (application.SavedProjects, error) {
		return application.NewSavedProjects(application.NewProjectService())
	}, application.NewLocalSettings())
}

func newRootCommand(saved func() (application.SavedProjects, error), localSettings *application.LocalSettings) *cobra.Command {
	info := application.Info()
	root := &cobra.Command{
		Use:           "provisionhub",
		Short:         info.Tagline,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return tui.Run(application.Info())
		},
	}
	root.AddCommand(newVersionCommand())
	root.AddCommand(newProjectCommand(saved))
	root.AddCommand(newTechnologiesCommand())
	root.AddCommand(newSettingsCommand(localSettings))
	root.AddCommand(newPluginCommand())
	return root
}
