# ProvisionHub development context

Checked against the source on November 10, 2026. Repository code and tests take precedence. Read [AGENTS.md](../AGENTS.md) before work.

ProvisionHub is a local-first, one-time source initializer. CLI and TUI collect a name, destination, and named component instances; `internal/project` validates the in-memory specification; `internal/planning` builds a deterministic, read-only plan; `internal/execution` applies an approved plan with root-confined, no-overwrite publication. Multiple instances can use the same technology. Optional connections are declarative only. Generation does not write a project YAML file, lockfile, or saved-project registration. The result is ordinary generated source code.

`internal/catalog` owns technology IDs and conflicts. `internal/plugins` loads declarative plugin manifests and templates, validates paths and sources, and rebases outputs by component name. `internal/registry` and `application.PluginManager` retain official package discovery, installation, version selection, metadata, and artifact integrity checks. Explicit local plugin directories can generate without SQLite when they cover every requested generator; managed plugin discovery uses installed metadata in SQLite. Planning and generation do not need GitHub authentication or a registry request. The official NestJS and Next.js packages live in the [official plugins repository](https://github.com/ProvisionLabs/provisionhub-official-plugins).

`internal/localdb` still stores Settings, Git account metadata, and managed plugin installations. Migration 6 creates the historical `saved_projects` table; its schema and existing rows are preserved for safe compatibility, but project registration services and UI are retired. No startup migration deletes user records. Existing project files and lockfiles on disk are untouched. See [local persistence](local-persistence.md) and [ADR-0011](adr/0011-one-time-project-generation.md).

CLI: `project plan` and `project apply` take `--name`, `--path`, and repeatable `--component name:type:technology` and `--set component.key=value`; `--plugins`, `--plugin-version`, `--output json` for plan, and `--yes` for apply remain available. TUI Home has Create Project, Plugins, and Settings. Review `p` previews; Enter approves executable generation. Failed generation can be re-reviewed and retried with confirmation. `project init`, project list/register/relocate/forget/lock, and My Projects are retired. The Plugin Manager, Settings, and Git provider flows remain.

Run `go fmt ./...`, `go vet ./...`, `go test -count=1 ./...`, and `go build ./cmd/provisionhub`. Tests use temporary directories and databases. CLI and TUI tests exercise commands and Bubble Tea models without live network or credentials. The official plugin lifecycle test is enabled by `PROVISIONHUB_OFFICIAL_PLUGINS`; core CI does not require a sibling repository. CI runs on Windows, Linux, and macOS.

Generation cannot wire component connections, generate a database, install dependencies, start apps, create repositories, or provision infrastructure. Hard links are required and multi-file publication cannot be crash-atomic. Read [generation planning](generation-planning.md), [plugin registry](plugin-registry.md), and [TUI controls](tui.md) for current behavior. Older [project management](project-management.md), [project manifest](project-manifest.md), and [project lock](project-plugin-locking.md) guides are historical and await a documentation cleanup milestone.

Declarative component configuration is implemented. `internal/configuration`
validates flat string, boolean, signed 64-bit integer, and string enum schemas,
parses interface text, and resolves defaults without I/O. Optional plugin
`configuration.properties` definitions drive CLI settings and asynchronous TUI
fields. Values are isolated in each in-memory component and reviewed plan.
Planning validates all components before operations and renders in memory;
rendered hashes and resolved values participate in approval freshness. Literal
`config` references require JSON or HTML escaping for strings; there are no
scripts, arbitrary templates, nested schemas, or conditional file operations.
See [configuration contract](plugin-configuration.md) and
[ADR-0012](adr/0012-declarative-component-configuration.md).

Core compatibility is 0.2.0. Official 1.2.0 source examples implement NestJS
`greeting`/`port` and Next.js `welcomeMessage`/`showCredits`; the published 1.1.0
registry packages remain unchanged until a separate release. Tests cover schema
rejection, deterministic resolution, rendering, CLI flags, dynamic TUI editing,
managed integrity, isolated generation and existing execution safeguards.
Recommended next work: a separately scoped Component Connection Engine design;
no connection behavior or persistent project management is part of this change.

Validation on Windows passed `go fmt ./...`, `go vet ./...`,
`go test -count=1 ./...`, and `go build ./cmd/provisionhub`, plus cross-compilation
for Windows/Linux/macOS amd64. Official package lifecycle and offline configured
CLI checks passed. Four generated applications completed real npm installations
and builds; the default NestJS and Next.js HTTP checks and Next.js lint also
passed after correcting Windows test cleanup ordering. Runtime checks are
explicit integration work, never generation actions.
