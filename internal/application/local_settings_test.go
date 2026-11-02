package application

import (
	"context"
	"github.com/ProvisionLabs/provisionhub/internal/credentials"
	"github.com/ProvisionLabs/provisionhub/internal/gitprovider"
	"github.com/ProvisionLabs/provisionhub/internal/localdb"
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

type forbiddenVault struct{}

func (forbiddenVault) Get(string) (credentials.Secret, error) {
	panic("general preferences must not read credentials")
}
func (forbiddenVault) Set(string, credentials.Secret) error {
	panic("general preferences must not write credentials")
}
func (forbiddenVault) Delete(string) error { panic("general preferences must not delete credentials") }
func TestGeneralPreferencesDoNotRequireCredentialVault(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "app.db")
	db, err := localdb.Open(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.PutAccount(ctx, gitprovider.Account{ID: "metadata-only", Provider: "github", InstanceURL: "https://github.com", SubjectID: "1", Username: "sample", CredentialRef: "missing"}); err != nil {
		t.Fatal(err)
	}
	db.Close()
	backend := &LocalSettings{Path: filename, Vault: forbiddenVault{}}
	values, statuses, err := backend.Load(ctx)
	if err != nil || len(statuses) != 1 || statuses[0].State != gitprovider.Unverified {
		t.Fatal(values, statuses, err)
	}
	if err := backend.Save(ctx, settings.Values{PreferredProvider: "github"}); err != nil {
		t.Fatal(err)
	}
}
