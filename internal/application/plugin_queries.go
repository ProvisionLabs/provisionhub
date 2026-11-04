package application

import (
	"context"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

type PluginUpdate struct{ PluginID, InstalledVersion, AvailableVersion string }

func PluginUpdates(index registry.Index, installed []plugins.Installation) []PluginUpdate {
	highest := map[string]string{}
	for _, item := range installed {
		if item.State != "ready" || !registry.ValidVersion(item.Version) {
			continue
		}
		if highest[item.PluginID] == "" || registry.Compare(item.Version, highest[item.PluginID]) > 0 {
			highest[item.PluginID] = item.Version
		}
	}
	updates := []PluginUpdate{}
	for _, p := range index.Search("") {
		current := highest[p.ID]
		if current == "" {
			continue
		}
		_, release, err := index.Resolve(p.ID, "", registry.CoreVersion)
		if err == nil && registry.Compare(release.Version, current) > 0 {
			updates = append(updates, PluginUpdate{p.ID, current, release.Version})
		}
	}
	return updates
}
func (m PluginManager) Updates(ctx context.Context) ([]PluginUpdate, registry.Snapshot, error) {
	items, err := m.List(ctx)
	if err != nil {
		return nil, registry.Snapshot{}, err
	}
	snapshot, err := m.Discover(ctx)
	if err != nil {
		return nil, snapshot, err
	}
	return PluginUpdates(snapshot.Index, items), snapshot, nil
}
func (m PluginManager) Local(directory string) ([]plugins.Manifest, error) {
	directory, err := pluginDirectory(directory)
	if err != nil {
		return nil, err
	}
	set, err := plugins.LoadDirectory(directory, catalog.Builtin())
	if err != nil {
		return nil, err
	}
	return set.Manifests(), nil
}
func PluginReference(value string) (string, string) {
	id, version, _ := strings.Cut(value, "@")
	return id, version
}
