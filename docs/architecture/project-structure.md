# Project structure

## Current implementation

```text
README.md
AGENTS.md
CONTRIBUTING.md
go.mod
go.sum
cmd/provisionhub/main.go
internal/
  application/info.go, project.go, planning.go
  catalog/catalog.go
  planning/plan.go, build.go, dependencies.go
  plugins/local.go
  execution/apply.go
  project/manifest.go, storage.go
  cli/root.go, version.go, project.go, technologies.go, plan.go, apply.go
  tui/app.go, navigation.go, wizard.go, plan.go
.github/workflows/ci.yml
docs/
  overview.md
  development.md
  development-context.md
  generation-planning.md
  project-manifest.md
  architecture/
    architecture.md
    project-structure.md
    plugins.md
  adr/
    0001-go-monorepo.md ... 0006-declarative-local-generation.md
    template.md
LICENSE
```

Each listed Go package has focused tests. The obsolete server architecture image and UI rebuild guide were removed because they described a required control plane and browser UI.

## Incremental Go layout

The module, entry point, CLI, TUI, catalog, project manifest, shared configuration use case, planning engine, local plugin loader, and file executor now exist. Introduce other packages only with their first real implementation:

| Location | Status | Role |
| --- | --- | --- |
| `cmd/provisionhub` | Implemented | Executable wiring and process exit behavior |
| `internal/cli` | Implemented | Command parsing and non-interactive presentation |
| `internal/tui` | Implemented | Terminal screens, navigation, and rendering |
| `internal/application` | Implemented | Metadata, shared project configuration, and file/in-memory planning use cases |
| `internal/catalog` | Implemented | Built-in technology definitions and category validation |
| `internal/planning` | Implemented | Pure conceptual/concrete plans, catalog resolution, operation graph ordering, and diagnostics |
| `internal/core` | Planned | Broader capability composition/resolution, only if real code needs a separate boundary |
| `internal/plugins` | Implemented narrowly | Local declarative discovery, manifest/path/template validation; no runtime or installation |
| `internal/execution` | Implemented narrowly | Reviewed local file generation with no-overwrite publication |
| `internal/project` | Implemented | Versioned manifest serialization, loading, and no-overwrite persistence |
| `internal/config` | Planned | Tool settings and loading |
| `internal/registry` | Planned | Optional catalog metadata when distribution is implemented |

Use `internal` until a concrete external Go API needs `pkg`; do not create `pkg`, `examples`, or empty screen/component packages preemptively. Package boundaries should follow dependencies: interfaces depend on application use cases; application depends on planning/project/catalog and future core/plugin abstractions; provider adapters depend on external tools. Application and domain packages must not import terminal adapters. The planning package has no I/O or execution dependencies; see [generation planning](../generation-planning.md).

See [architecture](architecture.md) and [plugin design](plugins.md) for responsibilities. [ADR-0001](../adr/0001-go-monorepo.md) records the original Go repository choice; [ADR-0003](../adr/0003-local-first-provisioning.md) changes its runtime topology.
