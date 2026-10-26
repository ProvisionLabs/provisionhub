# ADR-0002: gRPC as Plugin Contract Boundary

- Status: Superseded by [ADR-0005](0005-plugin-composition.md)
- Date: 2026-04-03
- Owners: ProvisionHub Core Team

> Historical decision. gRPC is no longer mandatory for local plugins. The original rationale and tradeoffs remain below; ADR-0005 defines the current direction and defers the runtime protocol choice.

## Context

ProvisionHub's differentiation is a plugin-first model where plugins may be implemented in different languages. The platform needs a strict interoperability boundary between core runtime and plugins with:

- Language neutrality
- Strongly typed contracts
- Explicit versioning
- Performance suitable for high-frequency plugin calls
- Tooling support for code generation and compatibility checks

Candidate contract models:

1. gRPC + Protocol Buffers
2. HTTP/JSON contracts
3. In-process SDK-only contracts

## Decision

Use gRPC with Protocol Buffers as the primary plugin contract boundary.

- Define plugin interfaces in .proto files
- Generate stubs for supported languages
- Version contracts explicitly (for example, package.v1, package.v2)
- Keep contract compatibility rules enforced in CI

## Rationale

- Multi-language plugins become first-class with stable generated clients/servers
- Typed schemas reduce integration drift versus ad-hoc JSON models
- Better performance characteristics for internal service/plugin communication
- Mature ecosystem for code generation, validation, and backward compatibility checks

## Trade-offs

Pros:

- Strong contracts and clear evolution path
- Cross-language support with shared schema source
- Good performance and operational predictability

Cons:

- Requires proto governance discipline
- Binary protocol is less human-readable than JSON for ad-hoc inspection
- Versioning mistakes can create compatibility friction

## Consequences

- Contract changes require ADR review for breaking changes
- CI must include proto linting and compatibility checks
- Plugin docs must always include required fields and semantic behavior, not only schema
- Core and plugin authors need explicit backward compatibility policy

## Rejected Alternatives

1. HTTP/JSON contracts

- Rejected due to weaker typing and higher schema drift risk
- Could still be used at public API edges, but not as plugin runtime boundary

2. In-process SDK-only contracts

- Rejected because it tightly couples plugin language/runtime to core implementation
- Conflicts with language-agnostic plugin goal

## Follow-up Work

- Publish base plugin contract documentation
- Define mandatory RPCs (for example, Validate, Provision) and error semantics
- Add contract conformance tests for plugin certification

## References

- docs/architecture/architecture.md
- docs/overview.md
