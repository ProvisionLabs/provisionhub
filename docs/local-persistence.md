# Local application persistence

ProvisionHub stores application preferences, Git account metadata, and managed
plugin installations in SQLite. Project generation uses an in-memory
specification and does not store project references, YAML files, or lockfiles.
Explicitly supplied local plugins can generate without SQLite. Managed plugins
use SQLite metadata for discovery and artifact verification.
The TUI optionally reads a saved workspace preference when creating a project;
read failures leave manual destination entry available.

## Location and ownership

| Platform | Database |
| --- | --- |
| Linux | `$XDG_DATA_HOME/provisionhub/provisionhub.db`, defaulting to `~/.local/share/provisionhub/provisionhub.db` |
| macOS | `~/Library/Application Support/ProvisionHub/provisionhub.db` |
| Windows | `%AppData%\ProvisionHub\provisionhub.db` |

Linux requires an absolute XDG data directory. macOS/Windows use `os.UserConfigDir`.
No home directory is hardcoded. Persistent data never goes in a cache directory.
New directories request mode `0700` and database files `0600`; Windows access is
governed by the user's application-data ACL. The database must be a regular file,
not a symlink. It is not encrypted and contains non-secret paths and account names.

## Boundary and lifecycle

`internal/settings` defines typed validated preferences. `internal/localdb` uses
`database/sql` with `modernc.org/sqlite`, without CGO or an ORM. Application services
own validation and account/credential coordination; CLI and TUI do not issue SQL.

Storage opens for settings/account/plugin operations or an optional existing-workspace
preference read. Each facade operation closes its connection on return. There is
one pooled connection per store and a 5-second busy timeout on each connection.
SQLite's default rollback journal is retained: WAL is not automatically suitable
for every filesystem. Prefer a local filesystem with SQLite locking support.

Opening an existing current database does not write default rows or rerun
migrations. Saving an empty preference removes its row. All values use parameterized
queries; schema SQL and numeric migration versions are application-controlled.

## Schema and upgrades

`PRAGMA user_version` records the schema version. Ordered deterministic migrations
run in a transaction; failed pending migrations roll back together. A newer
unknown database is refused rather than downgraded or erased. The current schema:

```sql
-- Version 1
CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);

-- Version 2
CREATE TABLE git_accounts (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  instance_url TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  username TEXT NOT NULL,
  credential_ref TEXT NOT NULL,
  disconnected INTEGER NOT NULL DEFAULT 0 CHECK(disconnected IN (0,1)),
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  UNIQUE(provider, instance_url, subject_id)
);

-- Historical version 6 (versions 3-5 add installed plugin metadata)
CREATE TABLE saved_projects (
  id TEXT PRIMARY KEY,
  manifest_path TEXT NOT NULL UNIQUE,
  registered_at TEXT NOT NULL,
  last_opened_at TEXT
);
```

Stable provider subject IDs and instance URLs distinguish multiple accounts,
including identities whose display names change. The credential reference is a
random native-vault entry name, not a token. Migration 6 remains in the
versioned history so existing databases open safely. The `saved_projects`
table and any existing rows are left untouched, but no current application
service reads or writes them. New databases still run this historical
migration. ProvisionHub does not delete old project files or records on startup.

## Failures and recovery

Initialization, corruption, incompatible versions, locking, and write failures
are reported explicitly. ProvisionHub does not erase or rebuild a corrupt
database automatically. Close ProvisionHub before taking a backup or performing
manual recovery, including any active SQLite journal. Invalid stored preferences
are reported and can be corrected in Settings or with `settings set`.

Secrets live only in the OS vault. A successful vault write followed by a failed
metadata write triggers deletion of the new entry; replacement retains the old
credential until new metadata is saved. Cleanup failures are reported. Disconnect
marks the metadata disconnected before deleting its credential, so a later SQL
deletion failure cannot make it look authenticated. Retry disconnect to clean up.
No distributed transaction spans SQLite and the vault: a process crash between
steps can leave an orphan vault entry. ProvisionHub does not claim crash-atomic
cross-store rollback or automatic orphan removal.

See [ADR-0008](adr/0008-local-settings-and-credentials.md),
[Settings](settings.md), and [Git providers](git-providers.md).
