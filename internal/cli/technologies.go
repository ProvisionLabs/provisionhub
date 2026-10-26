package cli

import (
	"fmt"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/spf13/cobra"
)

func newTechnologiesCommand() *cobra.Command {
	technologies := &cobra.Command{
		Use:   "technologies",
		Short: "Inspect available technology definitions",
	}
	technologies.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List built-in technologies",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			service := application.NewProjectService()
			for _, category := range []catalog.Category{catalog.Backend, catalog.Frontend, catalog.Database} {
				for _, technology := range service.Technologies(category) {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", category, technology.ID, technology.Name); err != nil {
						return err
					}
				}
			}
			return nil
		},
	})
	return technologies
}
