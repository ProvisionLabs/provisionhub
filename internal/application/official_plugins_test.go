package application

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
	"github.com/ProvisionLabs/provisionhub/internal/projectlock"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

// The official repository enables this test in its distribution CI. Core CI
// remains self-contained; no sibling checkout or network credentials are needed.
func TestOfficialPackageLifecycle(t *testing.T) {
	directory := os.Getenv("PROVISIONHUB_OFFICIAL_PLUGINS")
	if directory == "" {
		t.Skip("set PROVISIONHUB_OFFICIAL_PLUGINS for official package integration")
	}
	packages := map[string][]byte{}
	manifests := []plugins.Manifest{}
	for _, name := range []string{"nestjs", "nextjs"} {
		data, manifest, err := plugins.Package(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		packages[manifest.ID] = data
		manifests = append(manifests, manifest)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for id, data := range packages {
			if strings.Contains(r.URL.Path, "/"+id+"-v") {
				w.Write(data)
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	index := registry.Index{SchemaVersion: 1, Plugins: []registry.Plugin{}}
	for _, manifest := range manifests {
		id := "official/" + strings.TrimPrefix(manifest.ID, "official-")
		index.Plugins = append(index.Plugins, registry.Plugin{ID: id, ManifestID: manifest.ID, Publisher: "ProvisionLabs", Name: manifest.Name, Description: manifest.Description, TechnologyID: manifest.TechnologyID, Category: string(manifest.Category), Versions: []registry.Release{{Version: manifest.Version, MinimumCoreVersion: registry.CoreVersion, Artifact: server.URL + "/releases/download/" + manifest.ID + "-v" + manifest.Version + "/plugin.tar.gz", SHA256: plugins.Checksum(packages[manifest.ID])}}})
	}
	manager := PluginManager{DataDirectory: t.TempDir(), HTTP: server.Client()}
	t.Setenv("PROVISIONHUB_DATA_DIR", manager.DataDirectory)
	t.Setenv("PROVISIONHUB_PLUGINS", "")
	for _, plugin := range index.Plugins {
		item, err := manager.InstallFrom(context.Background(), index, plugin.ID, "")
		if err != nil || item.State != "ready" {
			t.Fatal(item, err)
		}
	}
	cache, err := json.Marshal(registry.Snapshot{Index: index, FetchedAt: time.Now().UTC(), URL: registry.OfficialURL})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manager.DataDirectory, "registry-cache.json"), cache, 0600); err != nil {
		t.Fatal(err)
	}
	restarted := NewProjectService()
	restarted.pluginManager = &manager
	destination := filepath.Join(t.TempDir(), "multi component café")
	components := []project.Component{{Name: "api", Type: "backend", Technology: "nestjs"}, {Name: "billing", Type: "backend", Technology: "nestjs"}, {Name: "web", Type: "frontend", Technology: "nextjs"}, {Name: "admin", Type: "frontend", Technology: "nextjs"}}
	manifest, err := restarted.ValidateComponentsInput("demo", destination, components)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := restarted.PlanManifest(manifest)
	if err != nil || !plan.Executable {
		t.Fatal(plan, err)
	}
	created, applied, err := restarted.CreateAndApplyComponentProject("demo", destination, components, plan)
	if err != nil || len(applied.Created) != 40 {
		t.Fatal(len(applied.Created), err)
	}
	lock, err := restarted.LockProject(created.ManifestPath, false, false)
	if err != nil || len(lock.Plugins) != 2 {
		t.Fatal(lock, err)
	}
	hashes := map[string]string{}
	for _, name := range applied.Created {
		data, err := os.ReadFile(filepath.Join(destination, name))
		if err != nil {
			t.Fatal(err)
		}
		hashes[name] = plugins.Checksum(data)
	}
	secondManager := PluginManager{DataDirectory: t.TempDir(), Source: fixedRegistry{index}, HTTP: server.Client()}
	second := NewProjectService()
	second.pluginManager = &secondManager
	secondDestination := filepath.Join(t.TempDir(), "second project")
	secondCreated, err := second.CreateComponentProject("demo", secondDestination, components)
	if err != nil {
		t.Fatal(err)
	}
	firstLockPath, err := projectlock.Path(created.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	secondLockPath, err := projectlock.Path(secondCreated.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	lockBytes, err := os.ReadFile(firstLockPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(secondLockPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondLockPath, lockBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := second.RestoreLockedProject(context.Background(), secondCreated.ManifestPath); err != nil {
		t.Fatal(err)
	}
	secondManager.Source = fixedRegistry{registry.Index{}} // planning must ignore changed registry metadata
	server.Close()                                         // Generation on both machines must now work offline.
	secondPlan, err := second.PlanProject(secondCreated.ManifestPath)
	if err != nil || !secondPlan.Executable {
		t.Fatal(secondPlan, err)
	}
	secondApplied, err := second.ApplyProject(secondCreated.ManifestPath, secondPlan)
	if err != nil {
		t.Fatal(err)
	}
	secondHashes := map[string]string{}
	for _, name := range secondApplied.Created {
		data, err := os.ReadFile(filepath.Join(secondDestination, name))
		if err != nil {
			t.Fatal(err)
		}
		secondHashes[name] = plugins.Checksum(data)
	}
	if !maps.Equal(hashes, secondHashes) {
		t.Fatalf("generated output differs: first=%v second=%v", hashes, secondHashes)
	}
	for _, component := range components {
		data, err := os.ReadFile(filepath.Join(destination, component.Name, "package.json"))
		if err != nil || !strings.Contains(string(data), `"name": "`+component.Name+`"`) {
			t.Fatal(component.Name, string(data), err)
		}
	}
	for _, name := range []string{"api/src/main.ts", "billing/src/app.module.ts", "web/src/app/page.tsx", "admin/src/app/layout.tsx"} {
		if _, err := os.Stat(filepath.Join(destination, filepath.FromSlash(name))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := NewProjectService().PlanProject(created.ManifestPath); err != nil {
		t.Fatal("restart project", err)
	}
	if err := manager.Remove(context.Background(), "official/nestjs", manifests[0].Version); err != nil {
		t.Fatal(err)
	}
	remaining, err := NewProjectService().loadPlugins()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := remaining.ForTechnology("nestjs"); ok {
		t.Fatal("removed plugin loaded")
	}
	if _, ok := remaining.ForTechnology("nextjs"); !ok {
		t.Fatal("unrelated plugin lost")
	}
}
