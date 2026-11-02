package localdb

import (
	"context"
	"database/sql"
	"github.com/ProvisionLabs/provisionhub/internal/gitprovider"
	"path/filepath"
	"testing"
	"time"
)

func TestUpgradeFromSettingsSchemaAndMultipleAccounts(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "app.db")
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, db, migrations[:1]); err != nil {
		t.Fatal(err)
	}
	db.Exec("INSERT INTO settings VALUES('preferred_provider','github')")
	db.Close()
	s, err := Open(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	values, err := s.LoadSettings(ctx)
	if err != nil || values.PreferredProvider != "github" {
		t.Fatal(values, err)
	}
	for _, id := range []string{"one", "two"} {
		if err := s.PutAccount(ctx, gitprovider.Account{ID: id, Provider: "github", InstanceURL: "https://github.com", SubjectID: id, Username: id, CredentialRef: "ref-" + id, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
	}
	accounts, err := s.Accounts(ctx)
	if err != nil || len(accounts) != 2 {
		t.Fatal(accounts, err)
	}
}
