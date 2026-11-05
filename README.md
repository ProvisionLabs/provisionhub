# ProvisionHub

ProvisionHub is a local-first, one-time project initializer. Choose named components and technologies, review a deterministic plan, and approve generation of source files. ProvisionHub does not track the generated project afterward.

The core uses an in-memory Go project specification. Generation does not create `provisionhub.yaml`, `.provisionhub/plugins.lock.yaml`, or a saved-project SQLite registration. Each named component writes under its own directory, so several components can use the same technology. The Planning and Execution Engines remain separate: planning writes nothing, and execution checks the reviewed plan and refuses to overwrite existing files.

## Build and run

Go 1.26.1 or later is required. An ANSI-compatible terminal is needed for the TUI.

```sh
go mod download
go run ./cmd/provisionhub
go run ./cmd/provisionhub --help
go build ./cmd/provisionhub
```

The TUI offers **Create Project**, **Plugins**, and **Settings**. Create Project collects a name, destination, and named components. Press `p` on Review for a read-only plan; Enter approves generation when every selected capability has a generator. If generation fails, the result screen can preview a fresh plan and request explicit approval for a retry. It does not save unsupported configurations.

## CLI generation

Use `--component name:type:technology` once per component. Types are `backend`, `frontend`, or `database`. A generator must be available for every selected capability before applying.

```sh
provisionhub technologies list
provisionhub project plan --name commerce --path ./commerce \
  --component api:backend:nestjs --component billing:backend:nestjs \
  --component web:frontend:nextjs --component admin:frontend:nextjs \
  --plugins ./path/to/plugins
provisionhub project apply --name commerce --path ./commerce \
  --component api:backend:nestjs --component billing:backend:nestjs \
  --component web:frontend:nextjs --component admin:frontend:nextjs \
  --plugins ./path/to/plugins
```

`project plan --output json` emits the same plan as structured JSON. `project apply` prints the plan and asks for literal `yes` in a terminal. Automation may pass `--yes` after reviewing the plan. The CLI does not install dependencies, start generated applications, or create Git repositories.

Install managed official generators with `provisionhub plugin install official/nestjs` and `provisionhub plugin install official/nextjs`. `--plugin-version official/nestjs@1.1.0` selects a specific installed version; omit the flag only when selection is unambiguous. An explicit `--plugins` directory, or `PROVISIONHUB_PLUGINS`, loads local definitions without opening SQLite for generation. Managed installations use SQLite metadata for discovery and revalidate their artifacts before use. Plugin manifests and packaging formats have not changed. See the [plugin guide](docs/plugin-registry.md).

## Persistent application state

SQLite remains for Settings, Git provider account metadata, and managed plugin installations. It is not a project database. Historical `saved_projects` tables are left untouched for migration compatibility; the application no longer reads or writes them. Existing generated directories and metadata files are not migrated or removed. See [local persistence](docs/local-persistence.md).

## Documentation

- [CLI reference](docs/cli-reference.md)
- [TUI controls](docs/tui.md)
- [Generation planning and execution](docs/generation-planning.md)
- [Plugin Registry and Manager](docs/plugin-registry.md)
- [Development guide](docs/development.md)
- [Development context](docs/development-context.md)

Connection execution, database generation, infrastructure provisioning, and automatic project reconciliation remain outside the current scope. Historical [project management](docs/project-management.md), [project manifest](docs/project-manifest.md), and [project lock](docs/project-plugin-locking.md) guides describe retired workflows and are retained pending documentation cleanup.
