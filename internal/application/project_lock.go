package application

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
	"github.com/ProvisionLabs/provisionhub/internal/projectlock"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

// LockProject checks the current lock or explicitly writes a new resolution.
// Updates use the same installed selection rules as normal unlocked planning.
func (s ProjectService) LockProject(file string, check, update bool) (projectlock.Lock, error) {
	manifest, err := s.LoadProject(file)
	if err != nil {
		return projectlock.Lock{}, err
	}
	existing, found, err := projectlock.Load(file)
	if err != nil {
		return existing, err
	}
	if check {
		if !found {
			return existing, fmt.Errorf("project has no lockfile; run project lock --file %s", file)
		}
		_, err = s.loadLockedPlugins(manifest, existing)
		return existing, err
	}
	if found && !update {
		return existing, fmt.Errorf("lockfile exists; use --check or --update")
	}
	lock, err := s.resolveProjectLock(manifest)
	if err != nil {
		return lock, err
	}
	if err := projectlock.Save(file, lock, update); err != nil {
		return lock, err
	}
	return lock, nil
}

type ProjectLockPreview struct {
	Existing projectlock.Lock
	Found    bool
	Proposed projectlock.Lock
}

// PreviewProjectLock resolves installed plugins without changing project files.
func (s ProjectService) PreviewProjectLock(file string) (ProjectLockPreview, error) {
	manifest, err := s.LoadProject(file)
	if err != nil {
		return ProjectLockPreview{}, err
	}
	existing, found, err := projectlock.Load(file)
	if err != nil {
		return ProjectLockPreview{}, err
	}
	proposed, err := s.resolveProjectLock(manifest)
	return ProjectLockPreview{Existing: existing, Found: found, Proposed: proposed}, err
}

// ApplyProjectLockPreview checks both old and proposed states again after approval.
func (s ProjectService) ApplyProjectLockPreview(file string, preview ProjectLockPreview) error {
	current, found, err := projectlock.Load(file)
	if err != nil {
		return err
	}
	if found != preview.Found || !reflect.DeepEqual(current, preview.Existing) {
		return fmt.Errorf("project lock changed since preview; review it again")
	}
	manifest, err := s.LoadProject(file)
	if err != nil {
		return err
	}
	proposed, err := s.resolveProjectLock(manifest)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(proposed, preview.Proposed) {
		return fmt.Errorf("proposed plugin resolution changed since preview; review it again")
	}
	return projectlock.Save(file, proposed, found)
}

func (s ProjectService) resolveProjectLock(manifest project.Manifest) (projectlock.Lock, error) {
	loaded, err := s.loadPlugins()
	if err != nil {
		return projectlock.Lock{}, err
	}
	for _, c := range manifest.CanonicalComponents() {
		if c.Type == "backend" || c.Type == "frontend" {
			if _, ok := loaded.ForTechnology(c.Technology); !ok {
				return projectlock.Lock{}, fmt.Errorf("no installed generator for %s; install a plugin first", c.Technology)
			}
		}
	}
	manager := s.manager()
	items, err := manager.List(context.Background())
	if err != nil {
		return projectlock.Lock{}, err
	}
	lock := projectlock.Lock{LockVersion: 1, Plugins: []projectlock.Plugin{}}
	for _, m := range loaded.Manifests() {
		if !requiredTechnology(manifest, m.TechnologyID) {
			continue
		}
		entry := projectlock.Plugin{ID: "local/" + m.ID, Version: m.Version, Source: "local", ContentSHA256: plugins.ContentChecksum(m)}
		for _, item := range items {
			if item.Status == "healthy" && item.Version == m.Version && item.ContentChecksum == entry.ContentSHA256 {
				entry.ID = item.PluginID
				entry.Source = item.Source
				entry.SHA256 = item.Checksum
				break
			}
		}
		if entry.Source != "local" {
			if entry.Source != "official" {
				return lock, fmt.Errorf("plugin %s uses unsupported source %q for portable locking", entry.ID, entry.Source)
			}
			snapshot, err := manager.Cached()
			if err != nil {
				return lock, fmt.Errorf("resolve artifact for %s: cached registry unavailable: %w", entry.ID, err)
			}
			_, release, err := snapshot.Index.Resolve(entry.ID, entry.Version, registry.CoreVersion)
			if err != nil || release.SHA256 != entry.SHA256 {
				return lock, fmt.Errorf("cached registry does not match installed artifact for %s@%s", entry.ID, entry.Version)
			}
			entry.Artifact = release.Artifact
		}
		lock.Plugins = append(lock.Plugins, entry)
	}
	return lock, nil
}

// RestoreLockedProject downloads missing exact official releases only when
// explicitly requested. The registry must still publish matching immutable data.
func (s ProjectService) RestoreLockedProject(ctx context.Context, file string) error {
	lock, found, err := projectlock.Load(file)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("project has no lockfile")
	}
	manager := s.manager()
	items, err := manager.List(ctx)
	if err != nil {
		return err
	}
	for _, entry := range lock.Plugins {
		if entry.Source != "official" {
			continue
		}
		installed := false
		for _, item := range items {
			if item.PluginID == entry.ID && item.Version == entry.Version {
				installed = true
				if item.Status != "healthy" || item.Checksum != entry.SHA256 || item.ContentChecksum != entry.ContentSHA256 {
					return fmt.Errorf("locked plugin %s@%s differs from installed artifact; remove that exact version before restoring", entry.ID, entry.Version)
				}
			}
		}
		if installed {
			continue
		}
		snapshot, err := manager.Discover(ctx)
		if err != nil {
			return fmt.Errorf("restore %s@%s: %w", entry.ID, entry.Version, err)
		}
		_, release, err := snapshot.Index.Resolve(entry.ID, entry.Version, registry.CoreVersion)
		if err != nil || release.SHA256 != entry.SHA256 || release.Artifact != entry.Artifact {
			return fmt.Errorf("registry no longer matches locked artifact %s@%s", entry.ID, entry.Version)
		}
		item, err := manager.InstallFrom(ctx, snapshot.Index, entry.ID, entry.Version)
		if err != nil {
			return err
		}
		if item.ContentChecksum != entry.ContentSHA256 {
			return fmt.Errorf("restored %s@%s content checksum differs from lock", entry.ID, entry.Version)
		}
	}
	return nil
}

func requiredTechnology(manifest project.Manifest, technology string) bool {
	for _, c := range manifest.CanonicalComponents() {
		if c.Technology == technology {
			return true
		}
	}
	return false
}

func (s ProjectService) loadLockedPlugins(manifest project.Manifest, lock projectlock.Lock) (plugins.Set, error) {
	if err := lock.Validate(); err != nil {
		return plugins.Set{}, err
	}
	directory, err := pluginDirectory(s.pluginDirectory)
	if err != nil {
		return plugins.Set{}, err
	}
	var local plugins.Set
	needsLocal := false
	for _, entry := range lock.Plugins {
		if entry.Source == "local" {
			needsLocal = true
			break
		}
	}
	if needsLocal {
		local, err = plugins.LoadDirectory(directory, s.catalog)
		if err != nil {
			return plugins.Set{}, err
		}
	}
	manager := s.manager()
	items, err := manager.List(context.Background())
	if err != nil {
		return plugins.Set{}, err
	}
	sets := []plugins.Set{}
	selected := map[string]bool{}
	for _, entry := range lock.Plugins {
		if version, ok := s.pluginVersions[entry.ID]; ok && version != entry.Version {
			return plugins.Set{}, fmt.Errorf("selection %s@%s conflicts with lock at %s; run project lock --update", entry.ID, version, entry.Version)
		}
		var m plugins.Manifest
		found := false
		if entry.Source == "local" {
			for _, candidate := range local.Manifests() {
				if "local/"+candidate.ID == entry.ID && candidate.Version == entry.Version {
					m = candidate
					found = true
					break
				}
			}
			if !found {
				return plugins.Set{}, fmt.Errorf("locked local plugin %s@%s is missing; supply its directory with --plugins", entry.ID, entry.Version)
			}
		} else {
			for _, item := range items {
				if item.PluginID != entry.ID || item.Version != entry.Version {
					continue
				}
				if item.Status != "healthy" {
					return plugins.Set{}, fmt.Errorf("locked plugin %s@%s is unhealthy: %s", entry.ID, entry.Version, item.Status)
				}
				if item.Checksum != entry.SHA256 || item.ContentChecksum != entry.ContentSHA256 {
					return plugins.Set{}, fmt.Errorf("locked plugin %s@%s checksum mismatch; restore exact artifact", entry.ID, entry.Version)
				}
				m, err = manager.check(item)
				if err != nil {
					return plugins.Set{}, err
				}
				found = true
				break
			}
			if !found {
				return plugins.Set{}, fmt.Errorf("locked plugin %s@%s is missing; run project lock --restore", entry.ID, entry.Version)
			}
		}
		if plugins.ContentChecksum(m) != entry.ContentSHA256 {
			return plugins.Set{}, fmt.Errorf("locked plugin %s@%s content checksum mismatch", entry.ID, entry.Version)
		}
		if !requiredTechnology(manifest, m.TechnologyID) {
			return plugins.Set{}, fmt.Errorf("locked plugin %s is no longer required by manifest; run project lock --update", entry.ID)
		}
		if selected[m.TechnologyID] {
			return plugins.Set{}, fmt.Errorf("multiple locked plugins supply %s", m.TechnologyID)
		}
		selected[m.TechnologyID] = true
		sets = append(sets, plugins.FromManifest(m))
	}
	for _, c := range manifest.CanonicalComponents() {
		if (c.Type == "backend" || c.Type == "frontend") && !selected[c.Technology] {
			return plugins.Set{}, fmt.Errorf("manifest technology %s has no locked plugin; run project lock --update", c.Technology)
		}
	}
	for id := range s.pluginVersions {
		if !strings.HasPrefix(id, "local/") {
			found := false
			for _, p := range lock.Plugins {
				if p.ID == id {
					found = true
				}
			}
			if !found {
				return plugins.Set{}, fmt.Errorf("selection %s is absent from lock; run project lock --update", id)
			}
		}
	}
	return plugins.Merge(sets...)
}
