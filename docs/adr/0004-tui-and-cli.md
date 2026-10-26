# ADR-0004: TUI-first and CLI-compatible interfaces

- Status: Accepted
- Date: 2026-10-26
- Owners: ProvisionHub Core Team

## Context

Developers need guided discovery and configuration of multiple technologies, while scripts and CI/CD need repeatable non-interactive commands. The replaceable browser UI described in [ADR-0001](0001-go-monorepo.md) is no longer the primary interface.

## Decision

The terminal UI is the primary guided interface. A non-interactive CLI exposes the same application use cases. Both interfaces are adapters: they collect input and present validation, plans, progress, and errors; application and core code own provisioning behavior. No permanently running backend or API is required between an interface and the application layer.

## Rationale and tradeoffs

A TUI fits local developer workflows and can support navigation across selections and plan review. The CLI makes the same capabilities available to automation. Maintaining two presentations adds design and test work; shared use cases and common project data prevent behavior drift. Terminal accessibility, small screens, and non-interactive approval semantics require attention during implementation.

## Consequences

The first implementation milestone should establish a minimal TUI shell and navigation and a CLI entry point, then connect them to a small shared use case. [Bubble Tea, Bubbles, Lip Gloss, and Cobra](../architecture/architecture.md#technology-candidates) are candidates to evaluate during that milestone, not dependencies adopted by this ADR. Browser interfaces remain possible later through the same application boundary.
