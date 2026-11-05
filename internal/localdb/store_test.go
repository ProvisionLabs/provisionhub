package localdb

import (
	"bytes"
	"context"
	"github.com/ProvisionLabs/provisionhub/internal/settings"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestSettingsRestartDefaultsAndConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "missing", "app.db")
	s, err := Open(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	values, err := s.LoadSettings(ctx)
	if err != nil || values != (settings.Values{}) {
		t.Fatalf("defaults: %+v %v", values, err)
	}
	var count int
	s.db.QueryRow("SELECT count(*) FROM settings").Scan(&count)
	if count != 0 {
		t.Fatal("defaults persisted")
	}
	values = settings.Values{Workspace: t.TempDir(), PreferredProvider: "github"}
	if err := s.SaveSettings(ctx, values); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			if _, err := s.LoadSettings(ctx); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	s.Close()
	s, err = Open(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.LoadSettings(ctx)
	if err != nil || got != values {
		t.Fatalf("restart: %+v %v", got, err)
	}
	if err := s.SaveSettings(ctx, settings.Values{Workspace: "relative"}); err == nil {
		t.Fatal("invalid workspace accepted")
	}
	if err := s.SaveSettings(ctx, settings.Values{PreferredProvider: "unknown"}); err == nil {
		t.Fatal("invalid provider accepted")
	}
	if err := s.SaveSettings(ctx, settings.Values{}); err != nil {
		t.Fatal(err)
	}
	s.db.QueryRow("SELECT count(*) FROM settings").Scan(&count)
	if count != 0 {
		t.Fatal("empty preferences persisted")
	}
}

func TestMigrationsRollbackAndFutureSchema(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	steps := append(append([]string{}, migrations...), "CREATE TABLE upgrade_check (id TEXT);", "INVALID SQL")
	if err := migrate(ctx, s.db, steps); err == nil {
		t.Fatal("accepted broken migration")
	}
	var version int
	s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if version != len(migrations) {
		t.Fatal("partial migration committed")
	}
	if _, err := s.db.Exec("SELECT * FROM upgrade_check"); err == nil {
		t.Fatal("failed transaction left table")
	}
	if err := migrate(ctx, s.db, steps[:len(steps)-1]); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, s.db, migrations); err == nil {
		t.Fatal("accepted future database")
	}
}

func TestCorruptAndInaccessibleDatabase(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "bad.db")
	os.WriteFile(filename, []byte("not a database"), 0o600)
	if s, err := Open(context.Background(), filename); err == nil {
		s.Close()
		t.Fatal("accepted corrupt database")
	}
	if s, err := Open(context.Background(), filepath.Join(filename, "child.db")); err == nil {
		s.Close()
		t.Fatal("accepted inaccessible directory")
	}
}

func TestReopenDoesNotWriteAndInvalidValuesCanBeRepaired(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "app.db")
	s, err := Open(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	before, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	after, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("reopening database changed bytes")
	}
	s, err = Open(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.db.Exec("INSERT INTO settings(key,value) VALUES(?,?)", "preferred_provider", "invalid-provider"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadSettings(ctx); err == nil {
		t.Fatal("accepted invalid stored preferences")
	}
	if err := s.SaveSettings(ctx, settings.Values{}); err != nil {
		t.Fatal(err)
	}
}

func TestSavedProjectMigrationFromVersionFive(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec("DROP TABLE saved_projects"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec("PRAGMA user_version=5"); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var count int
	if err := store.db.QueryRow("SELECT count(*) FROM saved_projects").Scan(&count); err != nil {
		t.Fatalf("version five migration: %v", err)
	}
	var version int
	if err := store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != len(migrations) {
		t.Fatalf("schema version %d: %v", version, err)
	}
}

func TestLegacySavedProjectRowsRemainAfterReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec("INSERT INTO saved_projects(id,manifest_path,registered_at) VALUES(?,?,?)", "legacy-id", filepath.Join(t.TempDir(), "provisionhub.yaml"), "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var id string
	if err := store.db.QueryRow("SELECT id FROM saved_projects WHERE id=?", "legacy-id").Scan(&id); err != nil || id != "legacy-id" {
		t.Fatalf("legacy row changed: %q, %v", id, err)
	}
}
