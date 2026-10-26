package cli

import (
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/tui"
	"github.com/spf13/cobra"
)

// NewRootCommand constructs the command tree without starting an interactive session.
func NewRootCommand() *cobra.Command {
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
	return root
}
