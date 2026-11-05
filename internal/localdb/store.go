// Package localdb implements application persistence with database/sql and SQLite.
package localdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ProvisionLabs/provisionhub/internal/settings"
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

func DefaultPath() (string, error) {
	if directory := os.Getenv("PROVISIONHUB_DATA_DIR"); directory != "" {
		if !filepath.IsAbs(directory) {
			return "", errors.New("PROVISIONHUB_DATA_DIR must be absolute")
		}
		return filepath.Join(directory, "provisionhub.db"), nil
	}
	if runtime.GOOS == "linux" {
		data := os.Getenv("XDG_DATA_HOME")
		if data == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			data = filepath.Join(home, ".local", "share")
		}
		if !filepath.IsAbs(data) {
			return "", errors.New("XDG_DATA_HOME must be absolute")
		}
		return filepath.Join(data, "provisionhub", "provisionhub.db"), nil
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(config, "ProvisionHub", "provisionhub.db"), nil
}

func Open(ctx context.Context, filename string) (*Store, error) {
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
		return nil, fmt.Errorf("create application data directory: %w", err)
	}
	file, err := os.OpenFile(absolute, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err == nil {
		err = file.Close()
	} else if errors.Is(err, os.ErrExist) {
		info, statErr := os.Lstat(absolute)
		if statErr != nil || !info.Mode().IsRegular() {
			return nil, errors.New("database must be a regular file, not a symlink")
		}
		err = nil
	}
	if err != nil {
		return nil, fmt.Errorf("open local database file: %w", err)
	}
	uriPath := filepath.ToSlash(absolute)
	if runtime.GOOS == "windows" {
		uriPath = "/" + uriPath
	}
	uri := url.URL{Scheme: "file", Path: uriPath}
	query := uri.Query()
	query.Add("_pragma", "busy_timeout(5000)")
	uri.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, fmt.Errorf("open local database: %w", err)
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := migrate(ctx, db, migrations); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize local database: %w", err)
	}
	return store, nil
}

// Rollback journal is intentional: WAL is not assumed safe on network filesystems.
var migrations = []string{
	`CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);`,
	`CREATE TABLE git_accounts (
 id TEXT PRIMARY KEY, provider TEXT NOT NULL, instance_url TEXT NOT NULL,
 subject_id TEXT NOT NULL, username TEXT NOT NULL, credential_ref TEXT NOT NULL,
 disconnected INTEGER NOT NULL DEFAULT 0 CHECK(disconnected IN (0,1)),
 created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL,
 UNIQUE(provider,instance_url,subject_id));`,
	`CREATE TABLE installed_plugins (
 plugin_id TEXT NOT NULL, version TEXT NOT NULL, source TEXT NOT NULL,
 installation_path TEXT NOT NULL UNIQUE, checksum TEXT NOT NULL,
 installed_at TEXT NOT NULL, state TEXT NOT NULL CHECK(state IN ('installing','ready','removing')),
 PRIMARY KEY(plugin_id,version));`,
	`ALTER TABLE installed_plugins ADD COLUMN content_checksum TEXT NOT NULL DEFAULT '';`,
	`ALTER TABLE installed_plugins ADD COLUMN minimum_core_version TEXT NOT NULL DEFAULT '';`,
	`CREATE TABLE saved_projects (
 id TEXT PRIMARY KEY, manifest_path TEXT NOT NULL UNIQUE,
 registered_at TEXT NOT NULL, last_opened_at TEXT);`,
}

func migrate(ctx context.Context, db *sql.DB, steps []string) error {
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > len(steps) {
		return errors.New("database schema is newer than this application; upgrade ProvisionHub")
	}
	if version == len(steps) {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	for index := version; index < len(steps); index++ {
		if _, err := tx.ExecContext(ctx, steps[index]); err != nil {
			return fmt.Errorf("migration %d failed: %w", index+1, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", index+1)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) LoadSettings(ctx context.Context) (settings.Values, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT key,value FROM settings ORDER BY key")
	if err != nil {
		return settings.Values{}, err
	}
	defer rows.Close()
	var values settings.Values
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return values, err
		}
		switch key {
		case "workspace":
			values.Workspace = value
		case "preferred_provider":
			values.PreferredProvider = value
		}
	}
	if err := rows.Err(); err != nil {
		return values, err
	}
	if err := values.Validate(); err != nil {
		return values, fmt.Errorf("%w: %v", settings.ErrInvalidStored, err)
	}
	return values, nil
}

func (s *Store) SaveSettings(ctx context.Context, values settings.Values) error {
	if err := values.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, pair := range [][2]string{{"workspace", values.Workspace}, {"preferred_provider", values.PreferredProvider}} {
		if pair[1] == "" {
			_, err = tx.ExecContext(ctx, "DELETE FROM settings WHERE key=?", pair[0])
		} else {
			_, err = tx.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value WHERE value<>excluded.value", pair[0], pair[1])
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
