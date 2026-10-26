# ADR-0001: Go Monorepo for Core Platform

- Status: Amended by [ADR-0003](0003-local-first-provisioning.md) and [ADR-0004](0004-tui-and-cli.md)
- Date: 2026-04-03
- Owners: ProvisionHub Core Team

> Historical decision. Go and a single repository remain the direction, but the Control Plane, Worker, queue, and reference browser UI assumptions below are no longer requirements. See ADR-0003 and ADR-0004. The original context and decision are retained for traceability.

## Context

ProvisionHub core includes Control Plane, Worker runtime primitives, queue adapters, contract tooling, and shared observability/security libraries. The repository also includes a standard reference UI used as a default interface, but the product must remain operable via API/CLI even if that UI is replaced. We need high consistency across build/test/release workflows and fast propagation of cross-cutting changes.

Alternative repository models considered:

1. Polyrepo (one repo per service)
2. Monorepo (single repo for core services)

Language options for core runtime were also considered, but Go emerged as strongest fit for service binaries, concurrency, static distribution, and operational simplicity.

## Decision

Use a Go monorepo for ProvisionHub core components.

- Single repository for Control Plane, Worker runtime, shared packages, contract helpers, and integration test harnesses
- Go as implementation language for core runtime services and shared SDK components
- Module boundaries maintained via directory structure and package ownership
- Include a standard reference UI in the same monorepo as an independently replaceable component
- Document a deterministic internal path for rebuilding/replacing the UI without changing core runtime components

## Rationale

- Consistent toolchain and CI behavior across all core services
- Easier refactors when contracts or shared runtime behavior change
- Faster onboarding with one dependency graph and one contribution model
- Strong Go performance profile for network-bound orchestration and worker execution
- Simple deployment artifacts (static binaries, container-friendly)
- Default UI can evolve quickly while core remains stable due to explicit API boundary

## Trade-offs

Pros:

- Better cross-service consistency
- Atomic changes across multiple components
- Shared quality gates and standards

Cons:

- Larger repository scale over time
- CI must be optimized to avoid full rebuilds on every change
- Requires clear ownership boundaries to avoid coupling
- Requires strict contract governance so UI does not leak business logic from backend

## Consequences

- We must invest in selective CI (path-based test/build execution)
- We need explicit code ownership to prevent monolith-by-accident
- Shared packages must remain minimal and stable to avoid tight coupling
- Release process should support component-level versioning where needed
- API-first documentation becomes mandatory to support internal UI rebuilds

## Rejected Alternatives

1. Polyrepo

- Rejected due to higher coordination overhead for contract and runtime changes
- Increases friction for atomic platform-wide updates

2. Mixed-language core

- Rejected for now to avoid operational and build complexity in early platform phase
- Multi-language support remains focused on plugin boundary via gRPC

## References

- docs/architecture/architecture.md
- docs/overview.md
- docs/adr/0002-grpc-plugin-contract.md
- docs/adr/0004-tui-and-cli.md (current interface decision; the former UI rebuild guide was retired)
