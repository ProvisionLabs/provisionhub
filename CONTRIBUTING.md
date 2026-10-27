# Contributing to ProvisionHub

ProvisionHub has CLI/TUI project configuration, a built-in catalog, versioned manifests, and read-only generation-plan previews. Source generation, execution, and plugin features are not implemented yet. Start with the [agent instructions](AGENTS.md), [development context](docs/development-context.md), [overview](docs/overview.md), [architecture](docs/architecture/architecture.md), [generation planning](docs/generation-planning.md), [plugin design](docs/architecture/plugins.md), [ADRs](docs/adr/), and [development guide](docs/development.md).

## Documentation changes

- Keep implemented behavior separate from planned behavior.
- Update the existing product and architecture documents when a design changes; do not add a parallel source of truth.
- Use [the ADR template](docs/adr/template.md) for significant decisions. Append a new ADR and mark an older one superseded or amended rather than rewriting its original context and decision.
- Use links relative to the current Markdown file. Check them when moving documents.

## Go changes

Add packages only with working code. Put terminal behavior in TUI/CLI adapters and shared use cases below them. Keep provider-specific integrations out of the core. Do not introduce a central service, queue, worker, or mandatory network dependency into the local workflow.

Format changed Go files with `go fmt ./...`, run `go test ./...` and `go vet ./...`, and build with `go build ./cmd/provisionhub`. The CI workflow runs these checks on Windows, Linux, and macOS. Add focused tests for behavior and package boundaries as features are implemented. Dependency choices should be justified by a concrete implementation need.
