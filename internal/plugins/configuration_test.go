package plugins

import (
	"strings"
	"testing"
)

func TestConfigurationManifest(t *testing.T) {
	valid := `configuration:
  properties:
    title: {type: string, label: Title, description: Display title, default: Hello, minLength: 1, maxLength: 30}
    enabled: {type: boolean, default: true}
    port: {type: integer, default: 3000, minimum: 1, maximum: 65535}
    mode: {type: enum, options: [small, large], default: small, required: true}
`
	m, err := parse([]byte(testPlugin + valid))
	if err != nil || len(m.Configuration.Properties) != 4 {
		t.Fatal(m, err)
	}
	m, err = parse([]byte(testPlugin))
	if err != nil || len(m.Configuration.Properties) != 0 {
		t.Fatal(m, err)
	}
	for _, invalid := range []string{
		strings.Replace(valid, "type: string", "type: object", 1),
		strings.Replace(valid, "default: true", "default: 'true'", 1),
		strings.Replace(valid, "default: true", "default: null", 1),
		strings.Replace(valid, "minimum: 1", "minimum: 1.5", 1),
		strings.Replace(valid, "[small, large]", "[1, true]", 1),
		strings.Replace(valid, "[small, large]", "[small, small]", 1),
		strings.Replace(valid, "default: small", "default: other", 1),
		strings.Replace(valid, "default: 3000", "default: 70000", 1),
		strings.Replace(valid, "type: boolean", "type: boolean, surprise: true", 1),
		valid + "    title: {type: string}\n",
		"configuration: null\n", "configuration: {properties: []}\n",
	} {
		if _, err := parse([]byte(testPlugin + invalid)); err == nil {
			t.Fatalf("accepted %s", invalid)
		}
	}
}
