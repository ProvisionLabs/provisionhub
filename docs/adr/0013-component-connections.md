# ADR-0013: Plugin-defined component connections

- Status: Accepted
- Date: 2026-11-10
- Owners: ProvisionHub Core Team
- Extends: [ADR-0012](0012-declarative-component-configuration.md) and [ADR-0011](0011-one-time-project-generation.md)
- Supersedes: The metadata-only connection behavior in [ADR-0007](0007-named-project-components.md)

## Context

The in-memory model already has named connection records, but only accepts
hardcoded `http` metadata and does not generate integration output. Plugins
need independent provider/consumer contracts. Existing literal templates are
always generated and cannot reference an optional connection while preserving
unconnected output. The one-time model and execution safeguards must remain.

## Decision

Plugins optionally declare provider slots with typed scalar metadata bindings
to literals or resolved component options, and consumer slots with a shared
type, required metadata schema, connection configuration schema, and bounded
file outputs. Reuse the scalar resolver and literal renderer. A pure connection
resolver requires exactly one compatible slot pair on the explicitly named
instances, rejects duplicates and cycles, and permits one target per consumer
slot. Multiple consumer instances can share a provider.

Connection-owned files are contributed only for selected valid connections.
They use existing path validation, packaging/integrity, planning, rendering,
staging, approval, and no-overwrite publication. No general conditional template
language is added. Planning resolves the complete connection set before
contributing operations and records configuration, metadata, outputs, and hashes.
Review freshness includes all connection and plugin inputs.

Add a bounded string `format: host` and `|host` rendering mode for validated DNS
or IP hosts, including IPv6 URI brackets. This avoids unsafe concatenation in
endpoint templates while keeping framework logic in plugins. No DNS lookups or
runtime network probes occur during planning.

The official consumer generates a server-only API_ORIGIN with an explicitly
supplied consumer-reachable host and the selected provider's configured port.
It does not infer browser reachability, publish NEXT_PUBLIC values, configure
CORS, install dependencies, or start services. Core compatibility becomes 0.3.0.

## Rationale

Capabilities and metadata schemas allow integration without framework pairs in
Go. Connection-owned file sets are the smallest extension that preserves
unconnected generation and avoids patching/overwriting base files. Explicit
hosts make network perspective a user decision. Shared validation and execution
avoid parallel configuration models and filesystem implementations.

## Trade-offs

- Consumer slots have one target, literal nonoverlapping output paths, and no
  arbitrary expressions or aggregate fan-out templates.
- Ambiguous slots require corrected plugin definitions rather than guessing a
  provider. No slot-selection preference or dependency solver is introduced.
- Generation dependency cycles and self-connections are unsupported in this
  first version, even if some runtime protocols could allow cyclic traffic.
- Generated endpoints are initial configuration. Later runtime address/port
  changes require application configuration changes, not reconciliation.

## Consequences

CLI/TUI can collect optional typed connections between named instances and show
their resolved plan. Existing plugins and projects without connections retain
their behavior. Metadata-only declarations now require compatible capabilities
to be executable. No project YAML, lockfile, registration, scripts, cloud
service, database dependency, or deployment infrastructure is added.

## Rejected Alternatives

1. Framework pairing rules in Go: violates plugin independence.
2. Global technology-level endpoints: selects the wrong repeated instance.
3. Mutating base files or unconditional empty integration files: changes unconnected behavior and complicates conflict protection.
4. General conditional templates or deployment discovery: expands the contract beyond local source generation.

## References

- [Connection contract and authoring](../component-connections.md)
- [Generation planning](../generation-planning.md)
- [Architecture](../architecture/architecture.md)
