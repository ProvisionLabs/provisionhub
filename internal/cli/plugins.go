package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
	"github.com/spf13/cobra"
)

func newPluginCommand() *cobra.Command { return pluginCommand(application.PluginManager{}) }
func pluginCommand(manager application.PluginManager) *cobra.Command {
	root := &cobra.Command{Use: "plugin", Short: "Discover and manage declarative plugin packages"}
	var localDirectory string
	list := &cobra.Command{Use: "list", Short: "List installed packages and local generators without network access", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		items, err := manager.List(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "ID\tVERSION\tSOURCE\tSTATUS")
		for _, item := range items {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", item.PluginID, item.Version, item.Source, item.Status)
		}
		local, err := manager.Local(localDirectory)
		if err != nil {
			return err
		}
		for _, p := range local {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\tlocal (unmanaged)\tvalidated\n", p.ID, p.Version)
		}
		return nil
	}}
	list.Flags().StringVar(&localDirectory, "plugins", "", "Also inspect this local plugin directory")
	var cached bool
	search := &cobra.Command{Use: "search [query]", Short: "Search official plugins by identity, name, technology or category", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		var snapshot registry.Snapshot
		var err error
		if cached {
			snapshot, err = manager.Cached()
		} else {
			snapshot, err = manager.Discover(cmd.Context())
		}
		if err != nil {
			return err
		}
		printRegistryStatus(cmd, snapshot)
		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		items, err := manager.List(cmd.Context())
		if err != nil {
			return err
		}
		for _, p := range snapshot.Index.Search(query) {
			_, release, err := snapshot.Index.Resolve(p.ID, "", registry.CoreVersion)
			version := "no compatible version"
			if err == nil {
				version = release.Version
			}
			status := "not installed"
			for _, item := range items {
				if item.PluginID == p.ID {
					status = "installed " + item.Version + " (" + item.Status + ")"
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n  %s\n", p.ID, p.Name, p.Category, version, status, p.Description)
		}
		return nil
	}}
	search.Flags().BoolVar(&cached, "cached", false, "Use cached metadata only (always marked stale)")
	var infoCached bool
	info := &cobra.Command{Use: "info <plugin-id>", Short: "Inspect plugin metadata and all release versions", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		var snapshot registry.Snapshot
		var err error
		if infoCached {
			snapshot, err = manager.Cached()
		} else {
			snapshot, err = manager.Discover(cmd.Context())
		}
		if err != nil {
			return err
		}
		printRegistryStatus(cmd, snapshot)
		for _, p := range snapshot.Index.Plugins {
			if p.ID == args[0] {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(p)
			}
		}
		return fmt.Errorf("plugin %q not found", args[0])
	}}
	info.Flags().BoolVar(&infoCached, "cached", false, "Use cached registry without a network request")
	install := &cobra.Command{Use: "install <plugin-id[@version]>", Short: "Verify and install a compatible package; explicit versions permit downgrades", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, version := application.PluginReference(args[0])
		if !registry.ValidID(id) || (strings.Contains(args[0], "@") && !registry.ValidVersion(version)) {
			return errors.New("expected namespace/name or namespace/name@x.y.z")
		}
		item, err := manager.Install(cmd.Context(), id, version)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Installed %s@%s (%s)\n", item.PluginID, item.Version, item.Status)
		return err
	}}
	update := &cobra.Command{Use: "update <plugin-id>", Short: "Install the latest compatible version, retaining older versions", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		item, err := manager.Update(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Installed %s@%s; older versions retained. Select a version for generation when multiple are installed.\n", item.PluginID, item.Version)
		return err
	}}
	updates := &cobra.Command{Use: "updates", Short: "Check the official registry for compatible updates", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		items, snapshot, err := manager.Updates(cmd.Context())
		if err != nil {
			return err
		}
		printRegistryStatus(cmd, snapshot)
		for _, item := range items {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s -> %s\n", item.PluginID, item.InstalledVersion, item.AvailableVersion)
		}
		if len(items) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No compatible updates found")
		}
		return nil
	}}
	var yes bool
	remove := &cobra.Command{Use: "remove <plugin-id[@version]>", Short: "Remove a managed package; projects using it may require reinstalling it", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if !yes {
			input, ok := cmd.InOrStdin().(*os.File)
			if !ok {
				return errors.New("removal requires terminal confirmation or --yes for automation")
			}
			stat, err := input.Stat()
			if err != nil || stat.Mode()&os.ModeCharDevice == 0 {
				return errors.New("removal requires terminal confirmation or --yes for automation")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Existing projects may need %s. Type yes to remove it: ", args[0])
			line, err := bufio.NewReader(input).ReadString('\n')
			if err != nil {
				return err
			}
			if strings.TrimSpace(line) != "yes" {
				return errors.New("removal cancelled")
			}
		}
		id, version := application.PluginReference(args[0])
		if strings.Contains(args[0], "@") && version == "" {
			return errors.New("explicit version must not be empty")
		}
		if err := manager.Remove(cmd.Context(), id, version); err != nil {
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", args[0])
		return err
	}}
	remove.Flags().BoolVar(&yes, "yes", false, "Approve removal without a terminal prompt")
	root.AddCommand(list, search, info, install, update, updates, remove)
	return root
}
func printRegistryStatus(cmd *cobra.Command, snapshot registry.Snapshot) {
	if snapshot.Stale {
		fmt.Fprintf(cmd.ErrOrStderr(), "STALE registry fetched %s: %s\n", snapshot.FetchedAt.Format("2006-01-02T15:04:05Z"), snapshot.Warning)
	} else if snapshot.Warning != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), snapshot.Warning)
	}
}
