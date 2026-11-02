package application

import (
	"context"
	"github.com/ProvisionLabs/provisionhub/internal/credentials"
	"github.com/ProvisionLabs/provisionhub/internal/settings"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalSettingsLazinessAndRestart(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "missing", "app.db")
	backend := &LocalSettings{Path: filename, Vault: &memoryVault{values: map[string]credentials.Secret{}}}
	if directory, err := backend.DefaultWorkspace(ctx); err != nil || directory != "" {
		t.Fatal(directory, err)
	}
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Fatal("default workspace created database")
	}
	values := settings.Values{Workspace: t.TempDir(), PreferredProvider: "github"}
	if err := backend.Save(ctx, values); err != nil {
		t.Fatal(err)
	}
	restarted := &LocalSettings{Path: filename, Vault: backend.Vault}
	got, statuses, err := restarted.Load(ctx)
	if err != nil || got != values || len(statuses) != 0 {
		t.Fatal(got, statuses, err)
	}
	if directory, err := restarted.DefaultWorkspace(ctx); err != nil || directory != values.Workspace {
		t.Fatal(directory, err)
	}
}
