package application

import "testing"

func TestInfo(t *testing.T) {
	info := Info()
	if info.Name != "ProvisionHub" || info.Version == "" || info.Tagline == "" {
		t.Fatalf("unexpected application metadata: %+v", info)
	}
	if again := Info(); again != info {
		t.Fatalf("metadata changed between calls: %+v and %+v", info, again)
	}
}
