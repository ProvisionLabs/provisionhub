# Project structure

## Current implementation

```text
README.md
CONTRIBUTING.md
go.mod
go.sum
cmd/provisionhub/main.go
internal/
  application/info.go, project.go
  catalog/catalog.go
  project/manifest.go, storage.go
  cli/root.go, version.go, project.go, technologies.go
  tui/app.go, navigation.go, wizard.go
.github/workflows/ci.yml
docs/
  overview.md
  development.md
  project-manifest.md
  architecture/
    architecture.md
    project-structure.md
    plugins.md
  adr/
    0001-go-monorepo.md ... 0005-plugin-composition.md
    template.md
LICENSE
```

Each listed Go package has focused tests. The obsolete server architecture image and UI rebuild guide were removed because they described a required control plane and browser UI.

## Incremental Go layout

The module, entry point, CLI, TUI, catalog, project manifest, and shared configuration use case now exist. Introduce other packages only with their first real implementation:

| Location | Status | Role |
| --- | --- | --- |
| `cmd/provisionhub` | Implemented | Executable wiring and process exit behavior |
| `internal/cli` | Implemented | Command parsing and non-interactive presentation |
| `internal/tui` | Implemented | Terminal screens, navigation, and rendering |
| `internal/application` | Implemented | Metadata and shared project configuration use case |
| `internal/catalog` | Implemented | Built-in technology definitions and category validation |
| `internal/core` | Planned | Composition, resolution, planning, and execution as those capabilities arrive |
| `internal/plugins` | Planned | Discovery, contracts, and lifecycle |
| `internal/project` | Implemented | Versioned manifest serialization, loading, and no-overwrite persistence |
| `internal/config` | Planned | Tool settings and loading |
| `internal/registry` | Planned | Optional catalog metadata when distribution is implemented |

Use `internal` until a concrete external Go API needs `pkg`; do not create `pkg`, `examples`, or empty screen/component packages preemptively. Package boundaries should follow dependencies: interfaces depend on application use cases; application depends on core/project/plugin abstractions; provider adapters depend on external tools. Avoid imports from core or application into terminal packages.

See [architecture](architecture.md) and [plugin design](plugins.md) for responsibilities. [ADR-0001](../adr/0001-go-monorepo.md) records the original Go repository choice; [ADR-0003](../adr/0003-local-first-provisioning.md) changes its runtime topology.
