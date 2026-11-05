package cli

import (
	"fmt"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/spf13/cobra"
)

func savedProjects() (application.SavedProjects, error) {
	return application.NewSavedProjects(application.NewProjectService())
}

func newProjectListCommand() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List locally saved projects", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		saved, err := savedProjects()
		if err != nil {
			return err
		}
		views, err := saved.List(cmd.Context())
		if err != nil {
			return err
		}
		for _, view := range views {
			name, state := view.Manifest.Metadata.Name, "available"
			if view.ManifestError != nil {
				name, state = "(unavailable)", view.ManifestError.Error()
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", view.Reference.ID, name, view.Reference.ManifestPath, state); err != nil {
				return err
			}
		}
		return nil
	}}
}

func newProjectRegisterCommand() *cobra.Command {
	var file string
	command := &cobra.Command{Use: "register", Short: "Register an existing local project", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		saved, err := savedProjects()
		if err != nil {
			return err
		}
		ref, err := saved.Register(cmd.Context(), file)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Registered %s (%s)\n", ref.ManifestPath, ref.ID)
		return err
	}}
	command.Flags().StringVar(&file, "file", "", "Manifest path or project directory")
	_ = command.MarkFlagRequired("file")
	return command
}

func newProjectForgetCommand() *cobra.Command {
	var file, id string
	command := &cobra.Command{Use: "forget", Short: "Forget a local project reference without deleting files", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if (file == "") == (id == "") {
			return fmt.Errorf("provide exactly one of --file or --id")
		}
		saved, err := savedProjects()
		if err != nil {
			return err
		}
		if file != "" {
			ref, err := saved.FindByPath(cmd.Context(), file)
			if err != nil {
				return err
			}
			id = ref.ID
		}
		if err := saved.Forget(cmd.Context(), id); err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Forgot project %s (files retained)\n", id)
		return err
	}}
	command.Flags().StringVar(&file, "file", "", "Manifest path or project directory")
	command.Flags().StringVar(&id, "id", "", "Saved project ID (useful if the project moved)")
	return command
}
