package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsCommandsAndOfflineGeneration(t *testing.T) {
	base := t.TempDir()
	t.Setenv("APPDATA", base)
	t.Setenv("HOME", base)
	t.Setenv("XDG_DATA_HOME", base)
	t.Setenv("PROVISIONHUB_GITHUB_CLIENT_ID", "")
	workspace := t.TempDir()
	if _, err := executeCLI("settings", "set", "--workspace", workspace, "--provider", "github"); err != nil {
		t.Fatal(err)
	}
	output, err := executeCLI("settings", "show")
	if err != nil || !strings.Contains(output, "github") {
		t.Fatal(output, err)
	}
	if _, err := executeCLI("settings", "set", "--provider", "unknown"); err == nil {
		t.Fatal("accepted invalid settings")
	}
	if output, err := executeCLI("settings", "git-status"); err != nil || strings.TrimSpace(output) != "[]" {
		t.Fatal(output, err)
	}
	if _, err := executeCLI("project", "init", "--name", "offline", "--path", filepath.Join(base, "offline"), "--backend", "nestjs"); err != nil {
		t.Fatal("generation now requires login", err)
	}
}
