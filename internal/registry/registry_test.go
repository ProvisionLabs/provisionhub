package registry

import (
	"strings"
	"testing"
)

func fixture() Index {
	return Index{SchemaVersion: 1, Plugins: []Plugin{{ID: "official/nestjs", ManifestID: "official-nestjs", Publisher: "ProvisionLabs", Name: "NestJS", Description: "Backend", TechnologyID: "nestjs", Category: "backend", Versions: []Release{{Version: "1.1.0", MinimumCoreVersion: "0.1.0", Artifact: "https://github.com/ProvisionLabs/provisionhub-official-plugins/releases/download/official-nestjs-v1.1.0/plugin.tar.gz", SHA256: strings.Repeat("a", 64)}}}}}
}
func TestRegistry(t *testing.T) {
	index := fixture()
	data, err := index.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Search("NEST")) != 1 || len(parsed.Search("frontend")) != 0 {
		t.Fatal("search")
	}
	_, v, err := parsed.Resolve("official/nestjs", "", "0.1.0")
	if err != nil || v.Version != "1.1.0" {
		t.Fatal(v, err)
	}
	if _, _, err := parsed.Resolve("official/nestjs", "", "0.0.9"); err == nil {
		t.Fatal("incompatible")
	}
	if _, _, err := parsed.Resolve("official/nestjs", "1.0.0", "0.1.0"); err == nil {
		t.Fatal("missing version")
	}
	if Compare("1.10.0", "1.2.0") <= 0 {
		t.Fatal("numeric ordering")
	}
}
func TestInvalidRegistry(t *testing.T) {
	for _, data := range []string{`{"schemaVersion":1,"schemaVersion":2}`, `{"schemaVersion":1,"plugins":[{"id":"official/a","id":"official/b"}]}`} {
		if _, err := Parse([]byte(data)); err == nil {
			t.Fatal("duplicate object key accepted")
		}
	}
	for _, id := range []string{"official/con", "aux/example", "official/com1"} {
		if ValidID(id) {
			t.Fatal("nonportable identity", id)
		}
	}
	for _, change := range []func(*Index){
		func(i *Index) { i.SchemaVersion = 2 },
		func(i *Index) { i.Plugins = append(i.Plugins, i.Plugins[0]) },
		func(i *Index) { i.Plugins[0].Versions = append(i.Plugins[0].Versions, i.Plugins[0].Versions[0]) },
		func(i *Index) { i.Plugins[0].ID = "../nestjs" },
		func(i *Index) { i.Plugins[0].Versions[0].Version = "01.0.0" },
		func(i *Index) { i.Plugins[0].Versions[0].Artifact = "https://github.com/repo/archive/main.tar.gz" },
		func(i *Index) { i.Plugins[0].Versions[0].SHA256 = "bad" },
	} {
		index := fixture()
		change(&index)
		if index.Validate() == nil {
			t.Fatal("accepted invalid registry", index)
		}
	}
	data, _ := fixture().Marshal()
	if _, err := Parse(append(data, []byte(`{}`)...)); err == nil {
		t.Fatal("trailing document")
	}
	if _, err := Parse([]byte(`{"schemaVersion":1,"unknown":true}`)); err == nil {
		t.Fatal("unknown field")
	}
}
