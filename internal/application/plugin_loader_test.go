package application

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

func TestInstalledPluginGenerationAfterRestart(t *testing.T) {
	manager, index, _ := managerFixture(t)
	t.Setenv("PROVISIONHUB_DATA_DIR", manager.DataDirectory)
	t.Setenv("PROVISIONHUB_PLUGINS", "")
	item, err := manager.InstallFrom(context.Background(), index, "official/nestjs", "")
	if err != nil {
		t.Fatal(err)
	}
	service := NewProjectService()
	destination := filepath.Join(t.TempDir(), "project café")
	components := []project.Component{{Name: "api", Type: "backend", Technology: "nestjs"}, {Name: "billing", Type: "backend", Technology: "nestjs"}}
	manifest, err := service.ValidateComponentsInput("demo", destination, components)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := service.PlanSpecification(manifest)
	if err != nil || !plan.Executable {
		t.Fatal(plan, err)
	}
	result, err := service.ApplySpecification(manifest, destination, plan)
	if err != nil || len(result.Created) != 2 {
		t.Fatal(result, err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "billing", "package.json"))
	if err != nil || string(data) != `{"name":"billing"}` {
		t.Fatal(string(data), err)
	}
	if err := manager.Remove(context.Background(), item.PluginID, item.Version); err != nil {
		t.Fatal(err)
	}
	plan, err = NewProjectService().PlanSpecification(manifest)
	if err != nil || plan.Executable {
		t.Fatal("removed generator discovered", plan, err)
	}
}

func TestUpdateKeepsVersionsAndRequiresExplicitSelection(t *testing.T) {
	manager, index, original := managerFixture(t)
	t.Setenv("PROVISIONHUB_DATA_DIR", manager.DataDirectory)
	ctx := context.Background()
	if _, err := manager.InstallFrom(ctx, index, "official/nestjs", ""); err != nil {
		t.Fatal(err)
	}
	folder := t.TempDir()
	if _, err := plugins.Extract(original, folder); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(folder, "plugin.yaml")
	text, _ := os.ReadFile(file)
	os.WriteFile(file, []byte(strings.Replace(string(text), "version: 1.1.0", "version: 1.2.0", 1)), 0o600)
	updated, _, err := plugins.Package(folder)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "v1.2.0") {
			w.Write(updated)
		} else {
			w.Write(original)
		}
	}))
	defer server.Close()
	index.Plugins[0].Versions[0].Artifact = server.URL + "/releases/download/official-nestjs-v1.1.0/plugin.tar.gz"
	index.Plugins[0].Versions = append(index.Plugins[0].Versions, registry.Release{Version: "1.2.0", MinimumCoreVersion: "0.1.0", Artifact: server.URL + "/releases/download/official-nestjs-v1.2.0/plugin.tar.gz", SHA256: plugins.Checksum(updated)})
	manager.Source = fixedRegistry{index}
	manager.HTTP = server.Client()
	if _, err := manager.Update(ctx, "official/nestjs"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewProjectService().loadPlugins(); err == nil {
		t.Fatal("arbitrary version selected")
	}
	selected, err := NewProjectService().WithPluginVersions([]string{"official/nestjs@1.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := selected.loadPlugins()
	if err != nil {
		t.Fatal(err)
	}
	manifest, ok := loaded.ForTechnology("nestjs")
	if !ok || manifest.Version != "1.1.0" {
		t.Fatal(manifest)
	}
	if err := manager.Remove(ctx, "official/nestjs", ""); err == nil {
		t.Fatal("ambiguous removal accepted")
	}
	if err := manager.Remove(ctx, "official/nestjs", "1.2.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewProjectService().loadPlugins(); err != nil {
		t.Fatal(err)
	}
}
func TestManagedLocalConflictAndExplicitSelection(t *testing.T) {
	manager, index, _ := managerFixture(t)
	t.Setenv("PROVISIONHUB_DATA_DIR", manager.DataDirectory)
	item, err := manager.InstallFrom(context.Background(), index, "official/nestjs", "")
	if err != nil {
		t.Fatal(err)
	}
	// Identical manual and managed copies deduplicate without changing old IDs.
	service, err := NewProjectServiceWithPlugins(filepath.Dir(item.Path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.loadPlugins(); err != nil {
		t.Fatal("identical source conflict", err)
	}
	service, err = NewProjectService().WithPluginVersions([]string{"official/nestjs@1.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.loadPlugins(); err != nil {
		t.Fatal(err)
	}
	service, err = NewProjectService().WithPluginVersions([]string{"official/nestjs@9.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.loadPlugins(); err == nil {
		t.Fatal("missing version ignored")
	}
	if _, err := service.WithPluginVersions([]string{"bad"}); err == nil {
		t.Fatal("invalid selection")
	}
}
