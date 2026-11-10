package plugins

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/configuration"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestRenderConfiguration(t *testing.T) {
	s := configuration.Schema{Properties: map[string]configuration.Property{
		"text": {Type: "string"}, "enabled": {Type: "boolean"}, "port": {Type: "integer"}, "mode": {Type: "enum", Options: []string{"small", "large"}},
	}}
	values := configuration.Values{"text": "\"</script>\n{{config.secret}}", "enabled": false, "port": int64(42), "mode": "small"}
	f := File{Type: "template", Content: []byte(`{"text":{{config.text|json}},"enabled":{{config.enabled}},"port":{{config.port}},"mode":{{config.mode|json}}}`)}
	data, err := Render(f, "demo", project.Component{Name: "api"}, s, values)
	if err != nil || !json.Valid(data) {
		t.Fatal(string(data), err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["text"] != values["text"] || decoded["enabled"] != false || decoded["port"] != float64(42) {
		t.Fatal(decoded)
	}
	f.Content = []byte(`{{projectName}}/{{componentName}}/{{component.name}}/{{config.text|html}}`)
	data, err = Render(f, "demo", project.Component{Name: "api"}, s, values)
	if err != nil || !strings.HasPrefix(string(data), "demo/api/api/&#34;&lt;/script&gt;") {
		t.Fatal(string(data), err)
	}
	for _, ref := range []string{"config.text", "config.unknown|json", "config.text|shell", "if config.enabled", "config.port|json|html", "config.port", "projectName|json"} {
		f.Content = []byte("{{" + ref + "}}")
		_, err := Render(f, "demo", project.Component{Name: "api"}, s, nil)
		if err == nil {
			t.Fatalf("accepted missing/unsupported reference %q", ref)
		}
	}
	for _, content := range []string{"{{", "{{projectName", "}}", "}}{{projectName}}"} {
		if err := validateTemplate([]byte(content), s); err == nil {
			t.Fatal(content)
		}
	}
}
