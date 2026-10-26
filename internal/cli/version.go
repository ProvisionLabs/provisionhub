package cli

import (
	"fmt"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the ProvisionHub version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := application.Info()
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", info.Name, info.Version)
			return err
		},
	}
}
