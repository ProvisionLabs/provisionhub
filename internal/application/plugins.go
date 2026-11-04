package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProvisionLabs/provisionhub/internal/localdb"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

// PluginManager coordinates local metadata and declarative package distribution.
// Its zero value uses only the official registry. Constructing it performs no I/O.
type PluginManager struct {
	DataDirectory string
	Source        registry.Source
	HTTP          *http.Client
}

func (m PluginManager) directory() (string, error) {
	if m.DataDirectory != "" {
		return canonicalDataDirectory(m.DataDirectory)
	}
	if value := os.Getenv("PROVISIONHUB_DATA_DIR"); value != "" {
		if !filepath.IsAbs(value) {
			return "", errors.New("PROVISIONHUB_DATA_DIR must be absolute")
		}
		return canonicalDataDirectory(value)
	}
	db, err := localdb.DefaultPath()
	if err != nil {
		return "", err
	}
	return canonicalDataDirectory(filepath.Dir(db))
}

// Resolve configured application-data ancestors once. macOS /var and /tmp are
// system symlinks; symlinks inside the resulting managed tree remain forbidden.
func canonicalDataDirectory(directory string) (string, error) {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	current := absolute
	tail := []string{}
	for {
		_, err := os.Lstat(current)
		if err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		tail = append(tail, filepath.Base(current))
		current = parent
	}
}
func (m PluginManager) source() (registry.Source, error) {
	if m.Source != nil {
		return m.Source, nil
	}
	directory, err := m.directory()
	if err != nil {
		return nil, err
	}
	return registry.Client{CachePath: filepath.Join(directory, "registry-cache.json"), HTTP: m.HTTP}, nil
}
func (m PluginManager) Discover(ctx context.Context) (registry.Snapshot, error) {
	source, err := m.source()
	if err != nil {
		return registry.Snapshot{}, err
	}
	return source.Fetch(ctx)
}
func (m PluginManager) Cached() (registry.Snapshot, error) {
	directory, err := m.directory()
	if err != nil {
		return registry.Snapshot{}, err
	}
	return (registry.Client{CachePath: filepath.Join(directory, "registry-cache.json")}).Cached()
}
func (m PluginManager) open(ctx context.Context) (*localdb.Store, error) {
	directory, err := m.directory()
	if err != nil {
		return nil, err
	}
	return localdb.Open(ctx, filepath.Join(directory, "provisionhub.db"))
}
func installationPath(directory, id, version string) string {
	return filepath.Join(directory, "plugins", filepath.FromSlash(id), version, "package")
}

// safePath refuses symlinked ancestors before touching managed content.
func safePath(filename string) error {
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	current := absolute
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("managed path contains symlink %q", current)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return nil
}
func (m PluginManager) check(item plugins.Installation) (plugins.Manifest, error) {
	directory, err := m.directory()
	if err != nil {
		return plugins.Manifest{}, err
	}
	if !registry.ValidID(item.PluginID) || !registry.ValidVersion(item.Version) || filepath.Clean(item.Path) != installationPath(directory, item.PluginID, item.Version) {
		return plugins.Manifest{}, errors.New("installation metadata has an unsafe path or identity")
	}
	if err := safePath(item.Path); err != nil {
		return plugins.Manifest{}, err
	}
	if item.State != "ready" {
		return plugins.Manifest{}, fmt.Errorf("interrupted %s operation; remove this exact version and reinstall", item.State)
	}
	manifest, err := plugins.LoadFolder(item.Path)
	if err != nil {
		return manifest, err
	}
	if manifest.Version != item.Version || plugins.ContentChecksum(manifest) != item.ContentChecksum {
		return manifest, errors.New("installed package integrity check failed; remove and reinstall")
	}
	return manifest, nil
}
func (m PluginManager) List(ctx context.Context) ([]plugins.Installation, error) {
	directory, err := m.directory()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(directory, "provisionhub.db")); errors.Is(err, os.ErrNotExist) {
		return []plugins.Installation{}, nil
	} else if err != nil {
		return nil, err
	}
	db, err := m.open(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	items, err := db.Installations(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		manifest, err := m.check(items[i])
		items[i].Name = manifest.Name
		if err != nil {
			items[i].Status = err.Error()
		} else {
			items[i].Status = "healthy"
		}
	}
	return items, nil
}
func (m PluginManager) download(ctx context.Context, artifact string) ([]byte, error) {
	parsed, err := url.Parse(artifact)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil {
		return nil, errors.New("artifact requires HTTPS")
	}
	client := m.HTTP
	if client == nil {
		client = registry.HTTPClient()
	}
	secured := *client
	secured.CheckRedirect = registry.HTTPClient().CheckRedirect
	if secured.Timeout == 0 {
		secured.Timeout = 60 * time.Second
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifact, nil)
	if err != nil {
		return nil, err
	}
	response, err := secured.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("artifact returned HTTP %d; verify that the release has been published", response.StatusCode)
	}
	directory, err := m.directory()
	if err != nil {
		return nil, err
	}
	if err := safePath(directory); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(directory, ".plugin-download-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	n, copyErr := io.Copy(file, io.LimitReader(response.Body, plugins.MaxArchiveSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		return nil, fmt.Errorf("download artifact: %w", copyErr)
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if n > plugins.MaxArchiveSize {
		return nil, errors.New("artifact exceeds compressed size limit")
	}
	data, err := os.ReadFile(file.Name())
	if err != nil {
		return nil, fmt.Errorf("download artifact: %w", err)
	}
	if int64(len(data)) > plugins.MaxArchiveSize {
		return nil, errors.New("artifact exceeds compressed size limit")
	}
	return data, nil
}
func (m PluginManager) Install(ctx context.Context, id, version string) (plugins.Installation, error) {
	snapshot, err := m.Discover(ctx)
	if err != nil {
		return plugins.Installation{}, err
	}
	return m.InstallFrom(ctx, snapshot.Index, id, version)
}

// InstallFrom shares a validated discovery snapshot with CLI/TUI and tests.
func (m PluginManager) InstallFrom(ctx context.Context, index registry.Index, id, version string) (plugins.Installation, error) {
	if err := index.Validate(); err != nil {
		return plugins.Installation{}, err
	}
	p, release, err := index.Resolve(id, version, registry.CoreVersion)
	if err != nil {
		return plugins.Installation{}, err
	}
	items, err := m.List(ctx)
	if err != nil {
		return plugins.Installation{}, err
	}
	for _, item := range items {
		if item.PluginID == id && version == "" && registry.ValidVersion(item.Version) && registry.Compare(item.Version, release.Version) > 0 {
			return item, errors.New("install would downgrade; specify an explicit id@version")
		}
		if item.PluginID == id && item.Version == release.Version {
			if item.Status != "healthy" {
				return item, errors.New(item.Status)
			}
			if item.Checksum != release.SHA256 {
				return item, errors.New("registry changed an installed immutable release")
			}
			return item, nil
		}
	}
	directory, err := m.directory()
	if err != nil {
		return plugins.Installation{}, err
	}
	target := installationPath(directory, id, release.Version)
	if err := safePath(target); err != nil {
		return plugins.Installation{}, err
	}
	data, err := m.download(ctx, release.Artifact)
	if err != nil {
		return plugins.Installation{}, err
	}
	if plugins.Checksum(data) != release.SHA256 {
		return plugins.Installation{}, errors.New("artifact SHA-256 mismatch; installation refused")
	}
	if err := ctx.Err(); err != nil {
		return plugins.Installation{}, err
	}
	db, err := m.open(ctx)
	if err != nil {
		return plugins.Installation{}, err
	}
	defer db.Close()
	item := plugins.Installation{PluginID: id, Version: release.Version, Source: "official", Path: target, Checksum: release.SHA256, InstalledAt: time.Now().UTC(), State: "installing"}
	if m.Source != nil {
		item.Source = "custom"
	}
	// The reservation serializes operations for this exact package; no existing
	// directory is claimed or overwritten, even if it lacks metadata.
	container := filepath.Dir(target)
	if err := os.MkdirAll(filepath.Dir(container), 0o700); err != nil {
		return item, err
	}
	if err := safePath(container); err != nil {
		return item, err
	}
	if err := os.Mkdir(container, 0o700); err != nil {
		return item, fmt.Errorf("refusing to claim an existing installation directory: %w", err)
	}
	// Claim the empty container before recording ownership. A crash here leaves
	// an unreferenced directory, never metadata that could authorize its deletion.
	if err := db.ReserveInstallation(ctx, item); err != nil {
		os.Remove(container)
		return item, fmt.Errorf("reserve installation: %w", err)
	}
	stage, err := os.MkdirTemp(container, ".stage-")
	if err != nil {
		return item, err
	}
	// Failed stages remain non-ready and can be removed explicitly; a process
	// crash can never turn them into a healthy installation.
	manifest, err := plugins.Extract(data, stage)
	if err != nil {
		return item, err
	}
	if manifest.ID != p.ManifestID || manifest.Version != release.Version || manifest.TechnologyID != p.TechnologyID || string(manifest.Category) != p.Category || manifest.Name != p.Name {
		return item, errors.New("packaged identity, version or technology does not match the registry")
	}
	item.ContentChecksum = plugins.ContentChecksum(manifest)
	if err := db.SetInstallationContent(ctx, id, release.Version, item.ContentChecksum); err != nil {
		return item, err
	}
	if err := ctx.Err(); err != nil {
		return item, err
	}
	if err := os.Rename(stage, target); err != nil {
		return item, fmt.Errorf("activate installation: %w", err)
	}
	if err := db.TransitionInstallation(ctx, id, release.Version, "installing", "ready"); err != nil {
		return item, err
	}
	item.State = "ready"
	item.Status = "healthy"
	item.Name = manifest.Name
	return item, nil
}
func (m PluginManager) Update(ctx context.Context, id string) (plugins.Installation, error) {
	items, err := m.List(ctx)
	if err != nil {
		return plugins.Installation{}, err
	}
	installed := false
	var highest string
	for _, item := range items {
		if item.PluginID == id {
			installed = true
			if highest == "" || registry.Compare(item.Version, highest) > 0 {
				highest = item.Version
			}
		}
	}
	if !installed {
		return plugins.Installation{}, fmt.Errorf("plugin %q is not installed", id)
	}
	snapshot, err := m.Discover(ctx)
	if err != nil {
		return plugins.Installation{}, err
	}
	_, release, err := snapshot.Index.Resolve(id, "", registry.CoreVersion)
	if err != nil {
		return plugins.Installation{}, err
	}
	if registry.Compare(release.Version, highest) < 0 {
		return plugins.Installation{}, errors.New("update would downgrade; use install with an explicit version")
	}
	return m.InstallFrom(ctx, snapshot.Index, id, release.Version)
}
func (m PluginManager) Remove(ctx context.Context, id, version string) error {
	if !registry.ValidID(id) || (version != "" && !registry.ValidVersion(version)) {
		return errors.New("invalid plugin identity or version")
	}
	items, err := m.List(ctx)
	if err != nil {
		return err
	}
	matches := []plugins.Installation{}
	for _, item := range items {
		if item.PluginID == id && (version == "" || version == item.Version) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return fmt.Errorf("managed plugin %s@%s is not installed; local directories are never removed", id, version)
	}
	if len(matches) > 1 {
		return errors.New("multiple installed versions; remove an explicit id@version")
	}
	item := matches[0]
	directory, err := m.directory()
	if err != nil {
		return err
	}
	expected := installationPath(directory, id, item.Version)
	if item.Path != expected {
		return errors.New("unsafe installation metadata; refusing removal")
	}
	if err := safePath(expected); err != nil {
		return err
	}
	db, err := m.open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	if item.State != "removing" {
		if err := db.TransitionInstallation(ctx, id, item.Version, item.State, "removing"); err != nil {
			return err
		}
	}
	// os.Root confines recursive deletion to the expected application-data tree.
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	relative, err := filepath.Rel(directory, filepath.Dir(expected))
	if err != nil || strings.HasPrefix(relative, "..") {
		return errors.New("unsafe removal path")
	}
	if err := root.RemoveAll(relative); err != nil {
		return fmt.Errorf("remove installation (retry to recover): %w", err)
	}
	return db.ForgetInstallation(ctx, id, item.Version)
}
