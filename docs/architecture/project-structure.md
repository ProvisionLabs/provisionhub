# Project structure

The Go module has one CLI/TUI executable in `cmd/provisionhub`. Core packages are under `internal`:

| Package | Role |
| --- | --- |
| `application` | Shared validation, plugin selection, planning, and apply coordination |
| `project` | In-memory specification and destination validation |
| `catalog` | Technology IDs, categories, and compatibility |
| `configuration` | Pure schema validation, typed parsing, and per-component default resolution |
| `planning` | Deterministic operation graph and diagnostics |
| `execution` | Reviewed, no-overwrite local file publication |
| `plugins` | Declarative plugin manifests, templates, and local discovery |
| `registry` | Official plugin metadata and package resolution |
| `localdb` | Settings, Git account, and managed plugin SQLite storage with versioned migrations |
| `settings`, `credentials`, `gitprovider` | Preferences and optional provider authentication |
| `cli`, `tui` | Thin presentation adapters |

The project package no longer loads or saves a project YAML file. Plugin YAML remains in `plugins`. The historical migration for `saved_projects` remains in `localdb`, but its repository and application service are retired. See [architecture](architecture.md), [generation planning](../generation-planning.md), and [development context](../development-context.md).
