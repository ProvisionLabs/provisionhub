package registry

import (
	"strings"
	"testing"
)

func TestRetainImmutableReleaseHistory(t *testing.T) {
	old := fixture()
	current := fixture()
	current.Plugins[0].Versions[0].Version = "1.2.0"
	current.Plugins[0].Versions[0].Artifact = strings.ReplaceAll(current.Plugins[0].Versions[0].Artifact, "1.1.0", "1.2.0")
	merged, err := RetainHistory(current, old)
	if err != nil || len(merged.Plugins[0].Versions) != 2 {
		t.Fatal(merged, err)
	}
	if _, _, err := merged.Resolve("official/nestjs", "1.1.0", CoreVersion); err != nil {
		t.Fatal(err)
	}
	current = fixture()
	current.Plugins[0].Versions[0].SHA256 = strings.Repeat("b", 64)
	if _, err := RetainHistory(current, old); err == nil {
		t.Fatal("release replaced")
	}
	current = fixture()
	current.Plugins[0].ManifestID = "different"
	if _, err := RetainHistory(current, old); err == nil {
		t.Fatal("identity replaced")
	}
	current = Index{SchemaVersion: 1}
	merged, err = RetainHistory(current, old)
	if err != nil || len(merged.Plugins) != 1 {
		t.Fatal("history removed", merged, err)
	}
}
