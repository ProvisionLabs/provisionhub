package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const connectionContract = `configuration:
  properties:
    port: {type: integer, default: 3000}
connections:
  provides:
    - id: endpoint
      type: transport
      metadata:
        port: {configuration: port}
        protocol: {value: plain}
  consumes:
    - id: remote
      type: transport
      requires:
        properties:
          port: {type: integer, required: true}
      configuration:
        properties:
          label: {type: string, default: Remote}
      files:
        - {type: template, source: integration.tmpl, destination: integration.json}
`

func TestConnectionManifestValidation(t *testing.T) {
	m, err := parse([]byte(testPlugin + connectionContract))
	if err != nil || len(m.AllFiles()) != 2 {
		t.Fatal(m, err)
	}
	for _, bad := range []string{
		strings.Replace(connectionContract, "type: transport", "type: 123", 1),
		strings.Replace(connectionContract, "configuration: port", "configuration: missing", 1),
		strings.Replace(connectionContract, "configuration: port", "configuration: port, value: 1", 1),
		strings.Replace(connectionContract, "value: plain", "value: null", 1),
		strings.Replace(connectionContract, "value: plain", "value: []", 1),
		strings.Replace(connectionContract, "type: string, default: Remote", "type: boolean, default: 'true'", 1),
		strings.Replace(connectionContract, "type: integer, required: true", "type: object", 1),
		strings.Replace(connectionContract, "type: integer, required: true", "type: integer, minimum: 1.5", 1),
		strings.Replace(connectionContract, "destination: integration.json", "destination: ../escape", 1),
		strings.Replace(connectionContract, "destination: integration.json", "destination: package.json", 1),
	} {
		if _, err := parse([]byte(testPlugin + bad)); err == nil {
			t.Fatal("accepted", bad)
		}
	}
}

func TestConnectionFilesPackagedAndIntegrityChecked(t *testing.T) {
	root := t.TempDir()
	folder := writeTestPlugin(t, root, "fixture", testPlugin+connectionContract)
	if err := os.WriteFile(filepath.Join(folder, "integration.tmpl"), []byte(`{"source":"{{source.name}}","target":"{{target.name}}","port":{{target.port}},"label":{{connection.label|json}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	data, m, err := Package(folder)
	if err != nil {
		t.Fatal(err)
	}
	extracted, err := Extract(data, t.TempDir())
	if err != nil || ContentChecksum(extracted) != ContentChecksum(m) {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "integration.tmpl"), []byte(`changed`), 0644); err != nil {
		t.Fatal(err)
	}
	changed, err := LoadFolder(folder)
	if err != nil || ContentChecksum(changed) == ContentChecksum(m) {
		t.Fatal("integration input missing from integrity checks", err)
	}
	if err := os.WriteFile(filepath.Join(folder, "integration.tmpl"), []byte(`{{target.unknown}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFolder(folder); err == nil {
		t.Fatal("accepted undeclared metadata reference")
	}
}
