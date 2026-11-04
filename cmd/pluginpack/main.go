// pluginpack builds the official registry and reproducible release assets locally.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: pluginpack <official-plugins-directory> <empty-output-directory>")
	}
	if err := os.Mkdir(args[1], 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(args[0])
	if err != nil {
		return err
	}
	index := registry.Index{SchemaVersion: 1, Plugins: []registry.Plugin{}}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, m, err := plugins.Package(filepath.Join(args[0], entry.Name()))
		if err != nil {
			return err
		}
		if !strings.HasPrefix(m.ID, "official-") {
			return fmt.Errorf("unexpected official manifest ID %q", m.ID)
		}
		id := "official/" + strings.TrimPrefix(m.ID, "official-")
		tag := m.ID + "-v" + m.Version
		filename := tag + ".tar.gz"
		digest := plugins.Checksum(data)
		if err := os.WriteFile(filepath.Join(args[1], filename), data, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(args[1], filename+".sha256"), []byte(digest+"  "+filename+"\n"), 0o644); err != nil {
			return err
		}
		index.Plugins = append(index.Plugins, registry.Plugin{ID: id, ManifestID: m.ID, Publisher: "ProvisionLabs", Name: m.Name, Description: m.Description, TechnologyID: m.TechnologyID, Category: string(m.Category), Source: "https://github.com/ProvisionLabs/provisionhub-official-plugins", Versions: []registry.Release{{Version: m.Version, MinimumCoreVersion: registry.CoreVersion, Artifact: "https://github.com/ProvisionLabs/provisionhub-official-plugins/releases/download/" + tag + "/" + filename, SHA256: digest}}})
	}
	data, err := index.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(args[1], "registry.json"), data, 0o644)
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
