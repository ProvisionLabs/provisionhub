# ADR-0005: Plugin-driven technology composition

- Status: Accepted
- Date: 2026-10-26
- Owners: ProvisionHub Core Team
- Supersedes: [ADR-0002](0002-grpc-plugin-contract.md) as a mandatory runtime protocol

## Context

The original [gRPC plugin decision](0002-grpc-plugin-contract.md) assumed server-side, multi-language plugin processes. The local-first product needs independently developed integrations that can describe how technologies fit together before any operation runs. A transport choice should follow a concrete contract and security model.

## Decision

Plugins are the extension boundary for templates, technology integrations, and provider operations. Future plugin metadata must express capabilities, requirements, dependencies, compatibility constraints, and configuration needs. The core will use that metadata to resolve a coherent selection and produce a reviewable plan; plugins will own integration-specific actions. A plugin's implementation language, process model, package format, and RPC protocol are deferred. gRPC remains an option, not a requirement. A marketplace or remote registry is not required for local use.

## Rationale and tradeoffs

Capabilities and constraints let the product compose technologies instead of concatenating templates. Deferring a runtime protocol avoids committing to generated stubs, subprocess overhead, or in-process trust before the first end-to-end integration exists. The tradeoff is that plugin distribution and cross-language support cannot be promised yet. Local third-party code also raises trust and credential questions that require design before loading plugins.

## Consequences

The [plugin architecture](../architecture/plugins.md) records contract requirements without inventing a schema or loader. Official plugins are planned in `provisionhub-official-plugins`; a ProvisionLabs website marketplace is future work. The first implementation should validate one small composition scenario before standardizing the plugin protocol.
