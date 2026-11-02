package localdb

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestPlatformApplicationDataPath(t *testing.T) {
	base := t.TempDir()
	var expected string
	switch runtime.GOOS {
	case "linux":
		t.Setenv("XDG_DATA_HOME", base)
		expected = filepath.Join(base, "provisionhub", "provisionhub.db")
	case "darwin":
		t.Setenv("HOME", base)
		expected = filepath.Join(base, "Library", "Application Support", "ProvisionHub", "provisionhub.db")
	case "windows":
		t.Setenv("APPDATA", base)
		expected = filepath.Join(base, "ProvisionHub", "provisionhub.db")
	default:
		t.Skip("unsupported test platform")
	}
	got, err := DefaultPath()
	if err != nil || got != expected {
		t.Fatalf("platform path: %q, %v", got, err)
	}
	if runtime.GOOS == "linux" {
		t.Setenv("XDG_DATA_HOME", "relative")
		if _, err := DefaultPath(); err == nil {
			t.Fatal("accepted relative XDG data directory")
		}
	}
}
