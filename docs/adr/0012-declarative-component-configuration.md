# ADR-0012: Declarative component configuration

- Status: Accepted
- Date: 2026-11-10
- Owners: ProvisionHub Core Team
- Extends: [ADR-0005](0005-plugin-composition.md) and [ADR-0011](0011-one-time-project-generation.md)

## Context

Named components can reuse one plugin but need independent settings. CLI and
TUI must present plugin-defined fields consistently without introducing
framework knowledge or restoring persistent project state.

## Decision

Add optional flat scalar configuration declarations to the existing plugin
manifest. A pure shared resolver validates schemas and user values and applies
defaults into private per-component maps. Support string, boolean, integer,
and string enum values with limited length and range constraints. Keep values
only in the in-memory specification and review plan.

Extend literal template references with configuration lookup and explicit JSON
literal or HTML text escaping for strings. Keep the renderer a fixed single-pass
substitution engine. Planning validates configuration and renders bytes in
memory before contributing operations, recording rendered content digests.
Execution repeats resolution and rendering and retains freshness, approval,
root confinement, and no-overwrite publication.

## Rationale

A flat schema meets the current generator needs and provides a small common
contract for both interfaces. Explicit escaping preserves scalar meaning across
JSON and source templates without a general template execution environment.
Rendered hashes make reviewed output sensitive to configuration changes.

## Trade-offs

- Optional fields without defaults cannot be referenced by a selected template
  unless the user supplies a value.
- Templates cannot conditionally include files. A literal boolean can control
  generated application code, while generation itself remains declarative.
- Authors must select escaping appropriate to each destination context.
- Old cores reject new manifest fields. Registry compatibility advances to
  0.2.0 and configured plugins need a new version and compatibility floor.

## Consequences

Plugins without configuration remain compatible. Definitions and package
integrity remain shared across component instances; settings do not mutate the
manifest or change operation identity. No project YAML, lockfile, registration,
dependency installation, or plugin scripts are introduced.

## Rejected Alternatives

1. Framework-specific flags and defaults in Go: prevents independent plugin evolution.
2. Full JSON Schema and general template programs: exceeds current needs and broadens execution semantics.
3. Global settings per technology or saved project files: breaks instance isolation or the one-time model.

## References

- [Configuration contract](../plugin-configuration.md)
- [Architecture](../architecture/architecture.md)
