package application

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProvisionLabs/provisionhub/internal/localdb"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

func TestInstallationIntegrityDoesNotDependOnGzipEncoding(t *testing.T) {
	manager, index, original := managerFixture(t)
	reader, err := gzip.NewReader(bytes.NewReader(original))
	if err != nil {
		t.Fatal(err)
	}
	tarBytes, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	writer, _ := gzip.NewWriterLevel(&encoded, gzip.NoCompression)
	writer.Write(tarBytes)
	writer.Close()
	data := encoded.Bytes()
	if bytes.Equal(data, original) {
		t.Fatal("test needs a different archive encoding")
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer server.Close()
	index.Plugins[0].Versions[0].Artifact = server.URL + "/releases/download/official-nestjs-v1.1.0/plugin.tar.gz"
	index.Plugins[0].Versions[0].SHA256 = plugins.Checksum(data)
	manager.HTTP = server.Client()
	if _, err := manager.InstallFrom(context.Background(), index, "official/nestjs", ""); err != nil {
		t.Fatal(err)
	}
	items, err := manager.List(context.Background())
	if err != nil || len(items) != 1 || items[0].Status != "healthy" {
		t.Fatal(items, err)
	}
}
func TestConfiguredSymlinkAncestorAndManagedSymlinkRejection(t *testing.T) {
	manager, index, _ := managerFixture(t)
	parent := t.TempDir()
	alias := filepath.Join(parent, "alias")
	if err := os.Symlink(manager.DataDirectory, alias); err != nil {
		t.Skip("symlink creation unavailable", err)
	}
	manager.DataDirectory = filepath.Join(alias, "user-data")
	item, err := manager.InstallFrom(context.Background(), index, "official/nestjs", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(item.Path, item.Path+"-real"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(item.Path+"-real", item.Path); err != nil {
		t.Fatal(err)
	}
	items, err := manager.List(context.Background())
	if err != nil || items[0].Status == "healthy" {
		t.Fatal(items, err)
	}
	if err := manager.Remove(context.Background(), item.PluginID, item.Version); err == nil {
		t.Fatal("followed managed symlink")
	}
}

type fixedRegistry struct{ index registry.Index }

func (s fixedRegistry) Fetch(context.Context) (registry.Snapshot, error) {
	return registry.Snapshot{Index: s.index}, nil
}
func managerFixture(t *testing.T) (PluginManager, registry.Index, []byte) {
	t.Helper()
	folder := t.TempDir()
	os.Mkdir(filepath.Join(folder, "templates"), 0o700)
	os.WriteFile(filepath.Join(folder, "templates", "package.tmpl"), []byte(`{"name":"{{componentName}}"}`), 0o600)
	manifest := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nestjs
version: 1.1.0
technologyId: nestjs
category: backend
name: NestJS
description: test generator
capability: generate_backend
files:
  - type: template
    source: templates/package.tmpl
    destination: backend/package.json
`
	os.WriteFile(filepath.Join(folder, "plugin.yaml"), []byte(manifest), 0o600)
	data, _, err := plugins.Package(folder)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	t.Cleanup(server.Close)
	index := registry.Index{SchemaVersion: 1, Plugins: []registry.Plugin{{ID: "official/nestjs", ManifestID: "official-nestjs", Publisher: "ProvisionLabs", Name: "NestJS", Description: "test generator", TechnologyID: "nestjs", Category: "backend", Versions: []registry.Release{{Version: "1.1.0", MinimumCoreVersion: "0.1.0", Artifact: server.URL + "/releases/download/official-nestjs-v1.1.0/plugin.tar.gz", SHA256: plugins.Checksum(data)}}}}}
	return PluginManager{DataDirectory: t.TempDir(), HTTP: server.Client(), Source: fixedRegistry{index}}, index, data
}
func TestManagedInstallRestartOfflineRemove(t *testing.T) {
	manager, index, _ := managerFixture(t)
	ctx := context.Background()
	first, err := manager.InstallFrom(ctx, index, "official/nestjs", "")
	if err != nil {
		t.Fatal(err)
	}
	again, err := manager.InstallFrom(ctx, index, "official/nestjs", "")
	if err != nil || !first.InstalledAt.Equal(again.InstalledAt) {
		t.Fatal("idempotence", again, err)
	}
	restarted := PluginManager{DataDirectory: manager.DataDirectory}
	installed, err := restarted.List(ctx)
	if err != nil || len(installed) != 1 || installed[0].Status != "healthy" {
		t.Fatal(installed, err)
	}
	if _, err := manager.Update(ctx, "official/nestjs"); err != nil {
		t.Fatal("same update", err)
	}
	if err := restarted.Remove(ctx, "official/nestjs", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first.Path); !os.IsNotExist(err) {
		t.Fatal("files retained", err)
	}
	if err := restarted.Remove(ctx, "official/nestjs", ""); err == nil {
		t.Fatal("missing removal")
	}
}
func TestManagedChecksumAndIdentity(t *testing.T) {
	for _, kind := range []string{"checksum", "identity", "version", "technology"} {
		t.Run(kind, func(t *testing.T) {
			manager, index, _ := managerFixture(t)
			switch kind {
			case "checksum":
				index.Plugins[0].Versions[0].SHA256 = strings.Repeat("a", 64)
			case "identity":
				index.Plugins[0].ManifestID = "someone-else"
			case "version":
				index.Plugins[0].Versions[0].Version = "1.2.0"
				index.Plugins[0].Versions[0].Artifact = strings.ReplaceAll(index.Plugins[0].Versions[0].Artifact, "1.1.0", "1.2.0")
			case "technology":
				index.Plugins[0].TechnologyID = "nextjs"
			}
			if _, err := manager.InstallFrom(context.Background(), index, "official/nestjs", ""); err == nil {
				t.Fatal("invalid release installed")
			}
			items, err := manager.List(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for _, item := range items {
				if item.Status == "healthy" {
					t.Fatal("failure marked healthy")
				}
				if err := manager.Remove(context.Background(), item.PluginID, item.Version); err != nil {
					t.Fatal("failure recovery", err)
				}
			}
		})
	}
}
func TestManagedMissingCorruptAndInterrupted(t *testing.T) {
	for _, kind := range []string{"missing", "corrupt", "installing", "removing"} {
		t.Run(kind, func(t *testing.T) {
			manager, index, _ := managerFixture(t)
			ctx := context.Background()
			item, err := manager.InstallFrom(ctx, index, "official/nestjs", "")
			if err != nil {
				t.Fatal(err)
			}
			if kind == "missing" {
				os.Remove(filepath.Join(item.Path, "plugin.yaml"))
			} else if kind == "corrupt" {
				os.WriteFile(filepath.Join(item.Path, "templates", "package.tmpl"), []byte("changed"), 0o600)
			} else {
				db, err := manager.open(ctx)
				if err != nil {
					t.Fatal(err)
				}
				err = db.TransitionInstallation(ctx, item.PluginID, item.Version, "ready", kind)
				db.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			items, err := manager.List(ctx)
			if err != nil || len(items) != 1 || items[0].Status == "healthy" {
				t.Fatal(items, err)
			}
			if _, err := manager.InstallFrom(ctx, index, "official/nestjs", ""); err == nil {
				t.Fatal("broken reinstall overwrote")
			}
			if err := manager.Remove(ctx, item.PluginID, item.Version); err != nil {
				t.Fatal("recovery", err)
			}
		})
	}
}
func TestManagedUnownedDirectoryAndPermissionFailure(t *testing.T) {
	manager, index, _ := managerFixture(t)
	container := filepath.Dir(installationPath(manager.DataDirectory, "official/nestjs", "1.1.0"))
	os.MkdirAll(container, 0o700)
	sentinel := filepath.Join(container, "user.txt")
	os.WriteFile(sentinel, []byte("preserve"), 0o600)
	if _, err := manager.InstallFrom(context.Background(), index, "official/nestjs", ""); err == nil {
		t.Fatal("claimed user files")
	}
	if err := manager.Remove(context.Background(), "official/nestjs", ""); err == nil {
		t.Fatal("removed unowned directory")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatal(err)
	}
	// A regular file as a required parent deterministically exercises filesystem
	// failures on Windows too, without relying on POSIX permission bits.
	manager.DataDirectory = filepath.Join(t.TempDir(), "not-a-directory")
	os.WriteFile(manager.DataDirectory, []byte("file"), 0o600)
	if _, err := manager.InstallFrom(context.Background(), index, "official/nestjs", ""); err == nil {
		t.Fatal("filesystem failure ignored")
	}
}
func TestInstallationPersistenceAndUniqueReservation(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "state.db")
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		db, err := localdb.Open(ctx, filename)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			item := plugins.Installation{PluginID: "official/nestjs", Version: "1.1.0", Source: "official", Path: filepath.Join(t.TempDir(), "package"), Checksum: strings.Repeat("a", 64), State: "installing", InstalledAt: time.Now().UTC(), MinimumCoreVersion: registry.CoreVersion}
			if err := db.ReserveInstallation(ctx, item); err != nil {
				t.Fatal(err)
			}
			if err := db.ReserveInstallation(ctx, item); err == nil {
				t.Fatal("duplicate reservation")
			}
			if err := db.TransitionInstallation(ctx, item.PluginID, item.Version, "ready", "removing"); err == nil {
				t.Fatal("invalid transition")
			}
		}
		items, err := db.Installations(ctx)
		if err != nil || len(items) != 1 {
			t.Fatal(fmt.Sprint(items), err)
		}
		db.Close()
	}
}

func TestInstalledCompatibilitySurvivesOfflineRestart(t *testing.T) {
	manager, index, _ := managerFixture(t)
	item, err := manager.InstallFrom(context.Background(), index, "official/nestjs", "")
	if err != nil {
		t.Fatal(err)
	}
	if item.MinimumCoreVersion != index.Plugins[0].Versions[0].MinimumCoreVersion {
		t.Fatal("compatibility not persisted", item)
	}
	item.MinimumCoreVersion = "9.0.0"
	if _, err := manager.check(item); err == nil {
		t.Fatal("core downgrade compatibility ignored")
	}
	index.Plugins[0].Versions[0].MinimumCoreVersion = "9.0.0"
	other := manager
	other.DataDirectory = t.TempDir()
	if _, err := other.InstallFrom(context.Background(), index, "official/nestjs", ""); err == nil {
		t.Fatal("incompatible installation")
	}
}

func TestPartialDownloadAndCancellationNeverActivate(t *testing.T) {
	for _, kind := range []string{"partial", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			manager, index, _ := managerFixture(t)
			started := make(chan struct{})
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if kind == "partial" {
					w.Header().Set("Content-Length", "1000")
					w.Write([]byte("short"))
					return
				}
				w.WriteHeader(200)
				w.(http.Flusher).Flush()
				close(started)
				<-r.Context().Done()
			}))
			defer server.Close()
			manager.HTTP = server.Client()
			index.Plugins[0].Versions[0].Artifact = server.URL + "/releases/download/official-nestjs-v1.1.0/plugin.tar.gz"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := manager.InstallFrom(ctx, index, "official/nestjs", ""); done <- err }()
			if kind == "cancel" {
				select {
				case <-started:
					cancel()
				case <-time.After(5 * time.Second):
					t.Fatal("download did not start")
				}
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("invalid download installed")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("download did not stop")
			}
			items, err := manager.List(context.Background())
			if err != nil || len(items) != 0 {
				t.Fatal(items, err)
			}
		})
	}
}
