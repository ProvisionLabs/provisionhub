package application

import (
	"context"
	"database/sql"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
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
	installed := map[string]plugins.Installation{}
	t.Setenv("PROVISIONHUB_DATA_DIR", manager.DataDirectory)
	t.Setenv("PROVISIONHUB_PLUGINS", "")
	for _, plugin := range index.Plugins {
		item, err := manager.InstallFrom(context.Background(), index, plugin.ID, "")
		if err != nil || item.State != "ready" {
			t.Fatal(item, err)
		}
		installed[plugin.ID] = item
	}
	restarted := NewProjectService()
	restarted.pluginManager = &manager
	destination := filepath.Join(t.TempDir(), "multi component café")
	components := []project.Component{{Name: "api", Type: "backend", Technology: "nestjs"}, {Name: "billing", Type: "backend", Technology: "nestjs"}, {Name: "web", Type: "frontend", Technology: "nextjs"}, {Name: "admin", Type: "frontend", Technology: "nextjs"}}
	configured := len(manifests[0].Configuration.Properties) > 0 && len(manifests[1].Configuration.Properties) > 0
	if configured {
		components[0].Configuration = configuration.Values{"greeting": "Welcome", "port": 4001}
		components[3].Configuration = configuration.Values{"welcomeMessage": "Administration", "showCredits": false}
	}
	manifest, err := restarted.ValidateComponentsInput("demo", destination, components)
	if err != nil {
		t.Fatal(err)
	}
	connected := len(manifests[0].Connections.Provides) > 0 && len(manifests[1].Connections.Consumes) > 0
	expectedFiles := 40
	if connected {
		manifest.Spec.Connections = []project.Connection{
			{From: "web", To: "api", Type: "http", Configuration: configuration.Values{"serverHost": "127.0.0.1"}},
			{From: "admin", To: "billing", Type: "http", Configuration: configuration.Values{"serverHost": "api.example.test"}},
		}
		expectedFiles = 44
	}
	assertNoSavedProjects(t, manager.DataDirectory)
	plan, err := restarted.PlanSpecification(manifest)
	if err != nil || !plan.Executable {
		t.Fatal(plan, err)
	}
	applied, err := restarted.ApplySpecification(manifest, destination, plan)
	if err != nil || len(applied.Created) != expectedFiles {
		t.Fatal(len(applied.Created), err)
	}
	for _, name := range []string{"provisionhub.yaml", filepath.Join(".provisionhub", "plugins.lock.yaml")} {
		if _, err := os.Stat(filepath.Join(destination, name)); !os.IsNotExist(err) {
			t.Fatalf("official generation wrote project metadata %s: %v", name, err)
		}
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
	for _, plugin := range index.Plugins {
		if _, err := secondManager.InstallFrom(context.Background(), index, plugin.ID, ""); err != nil {
			t.Fatal(err)
		}
	}
	second := NewProjectService()
	second.pluginManager = &secondManager
	secondDestination := filepath.Join(t.TempDir(), "second project")
	secondSpec, err := second.ValidateComponentsInput("demo", secondDestination, components)
	if err != nil {
		t.Fatal(err)
	}
	secondSpec.Spec.Connections = manifest.CanonicalConnections()
	secondManager.Source = fixedRegistry{registry.Index{}}
	server.Close() // Generation on both machines must now work offline.
	secondPlan, err := second.PlanSpecification(secondSpec)
	if err != nil || !secondPlan.Executable {
		t.Fatal(secondPlan, err)
	}
	secondApplied, err := second.ApplySpecification(secondSpec, secondDestination, secondPlan)
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
	if configured {
		for name, want := range map[string]string{"api/src/main.ts": "process.env.PORT ?? 4001", "billing/src/main.ts": "process.env.PORT ?? 3000", "api/src/app.service.ts": `"Welcome" + ' from api!'`, "billing/src/app.service.ts": `"Hello" + ' from billing!'`, "admin/src/app/page.tsx": "false &&", "web/src/app/page.tsx": "true &&"} {
			data, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(name)))
			if err != nil || !strings.Contains(string(data), want) {
				t.Fatal(name, string(data), err)
			}
		}
	}
	if connected {
		for name, want := range map[string]string{"web/.env.local": "API_ORIGIN=http://127.0.0.1:4001", "admin/.env.local": "API_ORIGIN=http://api.example.test:3000"} {
			data, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(name)))
			if err != nil || !strings.Contains(string(data), want) {
				t.Fatal(name, string(data), err)
			}
		}
		// Even unselected connection template inputs belong to managed integrity.
		item := installed["official/nextjs"]
		integration := filepath.Join(item.Path, filepath.FromSlash(manifests[1].Connections.Consumes[0].Files[0].Source))
		original, err := os.ReadFile(integration)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(integration, []byte("tampered"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := manager.check(item); err == nil {
			t.Fatal("managed integrity ignored connection input")
		}
		if err := os.WriteFile(integration, original, 0600); err != nil {
			t.Fatal(err)
		}
	}
	assertNoSavedProjects(t, manager.DataDirectory)
	assertNoSavedProjects(t, secondManager.DataDirectory)
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

func assertNoSavedProjects(t *testing.T, directory string) {
	t.Helper()
	name := filepath.ToSlash(filepath.Join(directory, "provisionhub.db"))
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	uri := url.URL{Scheme: "file", Path: name, RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT count(*) FROM saved_projects").Scan(&count); err != nil || count != 0 {
		t.Fatal("generation registered a project", count, err)
	}
}
