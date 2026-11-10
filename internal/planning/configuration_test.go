package planning

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestConfigurationDiagnosticsBeforeOperations(t *testing.T) {
	manifest := testManifest()
	manifest.Spec = project.Spec{Components: []project.Component{{Name: "billing", Type: "backend", Technology: "nestjs"}, {Name: "api", Type: "backend", Technology: "nestjs"}}}
	plugin := plugins.Manifest{ID: "fixture", TechnologyID: "nestjs", Category: catalog.Backend, Configuration: configuration.Schema{Properties: map[string]configuration.Property{"title": {Type: "string"}}}, Files: []plugins.File{{Type: "template", Source: "title.tmpl", Destination: "title.json", Content: []byte(`{{config.title|json}}`)}}}
	invalid := BuildWithPlugins(manifest, catalog.Builtin(), plugins.FromManifest(plugin))
	if invalid.Status != StatusInvalid || invalid.Executable || len(invalid.Operations) != 0 || len(invalid.Diagnostics) != 2 {
		t.Fatal(invalid)
	}
	for _, d := range invalid.Diagnostics {
		if d.Code != InvalidConfiguration || d.ComponentID == "" || !strings.Contains(d.Message, "title.tmpl") || !strings.Contains(d.Message, "has no value") {
			t.Fatal(d)
		}
	}
	manifest.Spec.Components[0].Configuration = configuration.Values{"title": "Billing"}
	manifest.Spec.Components[1].Configuration = configuration.Values{"title": "API"}
	plan := BuildWithPlugins(manifest, catalog.Builtin(), plugins.FromManifest(plugin))
	if !plan.Executable {
		t.Fatal(plan)
	}
	manifest.Spec.Components[0], manifest.Spec.Components[1] = manifest.Spec.Components[1], manifest.Spec.Components[0]
	if reordered := BuildWithPlugins(manifest, catalog.Builtin(), plugins.FromManifest(plugin)); !reflect.DeepEqual(plan, reordered) {
		t.Fatal("component ordering changed configured plan")
	}
	plan.Components[0].Configuration["title"] = "mutated"
	if manifest.Spec.Components[0].Configuration["title"] != "API" {
		t.Fatal("plan shares configuration with input")
	}
	plugin.Configuration.Properties["title"] = configuration.Property{Type: "invalid"}
	invalid = BuildWithPlugins(manifest, catalog.Builtin(), plugins.FromManifest(plugin))
	if invalid.Executable || len(invalid.Operations) != 0 {
		t.Fatal("unchecked contributed schema", invalid)
	}
}
