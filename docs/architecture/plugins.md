# Plugin architecture

## Status and purpose

This document defines intended boundaries only. There is no plugin contract, loader, registry, or official plugin in this repository yet. Official plugins are planned for the separate `provisionhub-official-plugins` repository. The ProvisionLabs website marketplace is a future discovery surface, not a runtime requirement.

Plugins should add project templates, frameworks, databases, authentication, CI/CD, containerization, infrastructure providers, and other operations without putting provider-specific logic in the core. Independently developed plugins should be possible, subject to a versioned contract and a documented compatibility policy that remain to be designed.

## Metadata for composition

Before any operation runs, a plugin should be able to describe:

- Its identity and compatible contract version.
- Capabilities it provides and capabilities it requires.
- Dependencies on other capabilities or plugins.
- Compatibility constraints, including mutually exclusive or version-sensitive choices.
- Configuration inputs, validation rules, and sensitive input markers.
- Operations it can propose and any environmental prerequisites.

These are requirements for a future contract, not a schema. For example, a PostgreSQL integration may require a backend database capability; a Docker integration may consume the selected backend's build/run details. The resolver should report missing capabilities and conflicts before planning. Avoid hardcoding a fixed set of technology categories in the core.

## Responsibility boundary

The application layer coordinates the user's selected capabilities and project lifecycle. The core resolves constraints, produces an ordered plan, and controls execution semantics. Plugins validate their own inputs and propose or perform integration-specific operations. External APIs and command-line tools are accessed at the plugin/provider boundary. TUI and CLI display the same validation, plan, and execution results.

The plan should expose side effects and identify destructive operations so the application can request explicit approval before execution. Plugin errors should carry enough context to identify the failing operation and recovery path. The exact operation and error contracts are deferred until the first vertical slice.

## Distribution and trust

Local discovery should work without a network connection. Plugin installation, update, provenance, sandboxing, permissions, secrets handling, version pinning, and the runtime protocol need explicit designs before loading third-party code. The former mandatory gRPC boundary in [ADR-0002](../adr/0002-grpc-plugin-contract.md) is superseded by [ADR-0005](../adr/0005-plugin-composition.md); gRPC can still be evaluated as an optional transport later. A catalog or website must not be required to open an existing local project.
