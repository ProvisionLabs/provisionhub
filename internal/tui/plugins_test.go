package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
)

func TestAdjacentPluginDirectory(t *testing.T) {
	base := filepath.Join(t.TempDir(), "portable app café")
	if err := os.Mkdir(base, 0o755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(base, "provisionhub.exe")
	if directory, err := adjacentPluginDirectory(executable); err != nil || directory != "" {
		t.Fatalf("without bundled plugins: %q, %v", directory, err)
	}
	plugins := filepath.Join(base, "plugins")
	if err := os.Mkdir(plugins, 0o755); err != nil {
		t.Fatal(err)
	}
	if directory, err := adjacentPluginDirectory(executable); err != nil || directory != plugins {
		t.Fatalf("bundled plugins: %q, %v", directory, err)
	}
}

func TestPluginEnvironmentOverridesPortableBundle(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "explicit plugins")
	t.Setenv("PROVISIONHUB_PLUGINS", explicit)
	if directory, err := localPluginDirectory(); err != nil || directory != explicit {
		t.Fatalf("explicit plugin override: %q, %v", directory, err)
	}
}

func TestPortablePluginDirectoryEnablesGeneration(t *testing.T) {
	_, destination := generatorModel(t)
	directory, err := adjacentPluginDirectory(filepath.Join(filepath.Dir(destination), "provisionhub.exe"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewModelWithPlugins(application.Info(), directory)
	if err != nil {
		t.Fatal(err)
	}
	reviewGeneratorWizard(t, m, destination)
	if !m.wizard.plan.Executable {
		t.Fatal("portable plugin did not produce an executable plan")
	}
	cmd := press(m, tea.KeyEnter, "")
	if cmd == nil || !m.wizard.applying {
		t.Fatal("portable generation did not start from review")
	}
	m.Update(cmd())
	if len(m.wizard.applyResult.Created) != 1 {
		t.Fatal("portable generation did not create the declared file")
	}
}
