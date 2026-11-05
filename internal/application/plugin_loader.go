package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

func pluginDirectory(directory string) (string, error) {
	if directory != "" {
		return directory, nil
	}
	if value := os.Getenv("PROVISIONHUB_PLUGINS"); value != "" {
		return value, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	adjacent := filepath.Join(filepath.Dir(executable), "plugins")
	if _, err := os.Stat(adjacent); errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return adjacent, nil
}

// WithPluginVersions selects installed versions for one session. It does not
// mutate global defaults or write nonportable paths into a project manifest.
func (s ProjectService) WithPluginVersions(selections []string) (ProjectService, error) {
	s.pluginVersions = map[string]string{}
	for _, selection := range selections {
		id, version, found := strings.Cut(selection, "@")
		if !found || !registry.ValidID(id) || !registry.ValidVersion(version) {
			return s, fmt.Errorf("invalid plugin selection %q; expected namespace/name@x.y.z", selection)
		}
		if _, ok := s.pluginVersions[id]; ok {
			return s, fmt.Errorf("duplicate version selection for %q", id)
		}
		s.pluginVersions[id] = version
	}
	if len(selections) > 0 {
		s.localOnly = false
	}
	return s, nil
}
func (s ProjectService) loadPlugins() (plugins.Set, error) {
	directory, err := pluginDirectory(s.pluginDirectory)
	if err != nil {
		return plugins.Set{}, err
	}
	local, err := plugins.LoadDirectory(directory, s.catalog)
	if err != nil {
		return plugins.Set{}, err
	}
	if s.localOnly {
		return local, nil
	}
	manager := s.manager()
	items, err := manager.List(context.Background())
	if err != nil {
		return plugins.Set{}, fmt.Errorf("read managed plugins: %w", err)
	}
	sets := []plugins.Set{local}
	selected := map[string]bool{}
	for _, item := range items {
		version, explicit := s.pluginVersions[item.PluginID]
		if explicit && version != item.Version {
			continue
		}
		if item.Status != "healthy" {
			if explicit {
				return plugins.Set{}, fmt.Errorf("selected plugin %s@%s: %s", item.PluginID, item.Version, item.Status)
			}
			continue
		}
		manifest, err := manager.check(item)
		if err != nil {
			return plugins.Set{}, err
		}
		sets = append(sets, plugins.FromManifest(manifest))
		selected[item.PluginID] = true
	}
	for id, version := range s.pluginVersions {
		if !selected[id] {
			return plugins.Set{}, fmt.Errorf("selected plugin %s@%s is not installed", id, version)
		}
	}
	return plugins.Merge(sets...)
}
