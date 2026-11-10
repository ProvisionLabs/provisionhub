package connections

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func fixture() (plugins.Manifest, plugins.Manifest, []project.Component) {
	provider := plugins.Manifest{ID: "provider", TechnologyID: "service", Configuration: configuration.Schema{Properties: map[string]configuration.Property{"port": {Type: "integer", Default: 3000}}}, Connections: plugins.ConnectionContract{Provides: []plugins.Provider{{ID: "endpoint", Type: "transport", Metadata: map[string]plugins.MetadataValue{"port": {Configuration: "port"}, "extra": {Value: "ignored"}}}}}}
	consumer := plugins.Manifest{ID: "consumer", TechnologyID: "client", Connections: plugins.ConnectionContract{Consumes: []plugins.Consumer{{ID: "remote", Type: "transport", Requires: configuration.Schema{Properties: map[string]configuration.Property{"port": {Type: "integer", Required: true}}}, Configuration: configuration.Schema{Properties: map[string]configuration.Property{"host": {Type: "string", Required: true}, "enabled": {Type: "boolean", Default: true}, "size": {Type: "integer", Default: 1}, "mode": {Type: "enum", Options: []string{"small", "large"}, Default: "small"}}}, Files: []plugins.File{{Type: "template", Source: "connection.tmpl", Destination: "endpoint.json"}}}}}}
	components := []project.Component{{Name: "api-a", Type: "backend", Technology: "service", Configuration: configuration.Values{"port": 4001}}, {Name: "api-b", Type: "backend", Technology: "service", Configuration: configuration.Values{"port": 4002}}, {Name: "web", Type: "frontend", Technology: "client"}, {Name: "admin", Type: "frontend", Technology: "client"}}
	return provider, consumer, components
}

func set(t *testing.T, provider, consumer plugins.Manifest) plugins.Set {
	t.Helper()
	loaded, err := plugins.Merge(plugins.FromManifest(provider), plugins.FromManifest(consumer))
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func declaration(from, to string) project.Connection {
	return project.Connection{From: from, To: to, Type: "transport", Configuration: configuration.Values{"host": "reachable.example"}}
}

func TestResolveNamedInstancesDefaultsAndIsolation(t *testing.T) {
	provider, consumer, components := fixture()
	loaded := set(t, provider, consumer)
	declarations := []project.Connection{declaration("web", "api-b"), declaration("admin", "api-a")}
	resolved, err := Resolve(components, declarations, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if resolved[0].From != "admin" || resolved[0].Metadata["port"] != int64(4001) || resolved[1].Metadata["port"] != int64(4002) || resolved[0].Configuration["enabled"] != true {
		t.Fatal(resolved)
	}
	if !reflect.DeepEqual(resolved[0].Outputs, []string{"admin/endpoint.json"}) {
		t.Fatal(resolved[0])
	}
	if len(declarations[0].Configuration) != 1 || components[0].Configuration["port"] != 4001 {
		t.Fatal("mutated caller values")
	}
	for range 20 {
		again, err := Resolve([]project.Component{components[3], components[2], components[1], components[0]}, []project.Connection{declarations[1], declarations[0]}, loaded)
		if err != nil || !reflect.DeepEqual(again, resolved) {
			t.Fatal(again, err)
		}
	}
	resolved[0].Configuration["host"] = "changed"
	if resolved[1].Configuration["host"] != "reachable.example" || declarations[1].Configuration["host"] != "reachable.example" {
		t.Fatal("connection values shared")
	}
	declarations[0].To = "api-a"
	resolved, err = Resolve(components, declarations, loaded)
	if err != nil || resolved[1].Metadata["port"] != int64(4001) {
		t.Fatal("two consumers cannot target same provider", err)
	}
	if result, err := Resolve(components, nil, plugins.Set{}); result != nil || err != nil {
		t.Fatal("unconnected compatibility", result, err)
	}
}

func TestInvalidConnections(t *testing.T) {
	provider, consumer, components := fixture()
	loaded := set(t, provider, consumer)
	for _, tc := range []struct {
		name         string
		declarations []project.Connection
		want         string
	}{
		{"source", []project.Connection{declaration("absent", "api-a")}, "unknown component"},
		{"target", []project.Connection{declaration("web", "absent")}, "unknown component"},
		{"self", []project.Connection{declaration("web", "web")}, "itself"},
		{"duplicate", []project.Connection{declaration("web", "api-a"), declaration("web", "api-a")}, "duplicate connection"},
		{"slot", []project.Connection{declaration("web", "api-a"), declaration("web", "api-b")}, "supports one target"},
		{"direction", []project.Connection{declaration("api-a", "web")}, "no compatible"},
		{"required", []project.Connection{{From: "web", To: "api-a", Type: "transport"}}, "required value"},
		{"unknown option", []project.Connection{{From: "web", To: "api-a", Type: "transport", Configuration: configuration.Values{"unknown": true}}}, "unknown key"},
		{"invalid type", []project.Connection{{From: "web", To: "api-a", Type: "transport", Configuration: configuration.Values{"host": true}}}, "expected string"},
		{"invalid enum", []project.Connection{{From: "web", To: "api-a", Type: "transport", Configuration: configuration.Values{"host": "remote", "mode": "other"}}}, "one of"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Resolve(components, tc.declarations, loaded)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal(err)
			}
		})
	}
	consumer.Connections.Consumes[0].Requires.Properties["port"] = configuration.Property{Type: "boolean", Required: true}
	if _, err := Resolve(components, []project.Connection{declaration("web", "api-a")}, set(t, provider, consumer)); err == nil || !strings.Contains(err.Error(), "expected boolean") {
		t.Fatal(err)
	}
}

func TestAmbiguityAndMetadataBindings(t *testing.T) {
	provider, consumer, components := fixture()
	second := provider.Connections.Provides[0]
	second.ID = "second"
	provider.Connections.Provides = append(provider.Connections.Provides, second)
	if _, err := Resolve(components, []project.Connection{declaration("web", "api-a")}, set(t, provider, consumer)); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatal(err)
	}
	provider, consumer, components = fixture()
	extra := consumer.Connections.Consumes[0]
	extra.ID = "second"
	consumer.Connections.Consumes = append(consumer.Connections.Consumes, extra)
	if _, err := Resolve(components, []project.Connection{declaration("web", "api-a")}, set(t, provider, consumer)); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatal(err)
	}
	provider, consumer, components = fixture()
	provider.Configuration.Properties["port"] = configuration.Property{Type: "integer"}
	components[0].Configuration = nil
	if _, err := Resolve(components, []project.Connection{declaration("web", "api-a")}, set(t, provider, consumer)); err == nil || !strings.Contains(err.Error(), "has no value") {
		t.Fatal(err)
	}
}

func TestCyclesAndCompatibleOptions(t *testing.T) {
	provider, consumer, components := fixture()
	loaded := set(t, provider, consumer)
	options := Options(components, nil, loaded)
	if len(options) != 4 {
		t.Fatal(options)
	}
	options = Options(components, []project.Connection{declaration("web", "api-a")}, loaded)
	if len(options) != 2 || options[0].Connection.From != "admin" {
		t.Fatal(options)
	}
	provider.Connections.Consumes = consumer.Connections.Consumes
	consumer.Configuration = provider.Configuration
	consumer.Connections.Provides = provider.Connections.Provides
	loaded = set(t, provider, consumer)
	cycle := []project.Connection{declaration("web", "api-a"), declaration("api-a", "web")}
	_, err := Resolve(components, cycle, loaded)
	if !errors.Is(err, ErrCycle) || !strings.Contains(err.Error(), "api-a -> web -> api-a") {
		t.Fatal(err)
	}
	for _, option := range Options(components, cycle[:1], loaded) {
		if option.Connection.From == "api-a" && option.Connection.To == "web" {
			t.Fatal("cycle offered as compatible", option)
		}
	}
}
