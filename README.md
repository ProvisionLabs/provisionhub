# ProvisionHub

ProvisionHub is an open-source, TUI-first developer provisioning tool for individual developers and small teams. Its goal is to help developers compose, configure, initialize, and provision applications from frameworks, services, and infrastructure choices that work together.

The planned terminal interface will guide project creation, technology selection, configuration, plan review, execution, and work on existing projects. Non-interactive CLI commands will support scripts and CI/CD through the same application services. The core will run locally without a control plane, database server, message broker, distributed workers, Kubernetes cluster, or persistent backend.

Plugins are intended to provide templates, technology integrations, and provisioning operations. Official plugins will live in the separate `provisionhub-official-plugins` repository. A searchable marketplace on the ProvisionLabs website is a future distribution channel; it is not part of this repository today.

## Current status

The repository now has a Go executable. Running `provisionhub` opens a navigable TUI with Home, Create Project, Open Existing Project, Plugin Marketplace, and Settings screens. The four destination screens explain that their features are not implemented. `provisionhub version` and `provisionhub --help` work without an interactive terminal. CLI and TUI read the same application metadata. Unit tests and a cross-platform CI workflow cover this foundation.

There is still no project creation, manifest, technology selection, plugin discovery or installation, provisioning engine, registry, template, or settings persistence. The application layer currently provides metadata only.

## Build and run

Requires Go 1.26.1 or later. Use an ANSI-compatible terminal for the TUI.

```sh
go mod download
go run ./cmd/provisionhub
go run ./cmd/provisionhub version
go run ./cmd/provisionhub --help
go build ./cmd/provisionhub
```

After `go build`, run `./provisionhub` on Linux/macOS or `.\provisionhub.exe` on Windows. `go install ./cmd/provisionhub` installs to your Go binary directory.

On Home, use Up/Down or `j`/`k` to navigate and Enter to select. Esc returns to the previous screen. `q` or Ctrl+C quits from any screen. The TUI uses a compact layout in small terminals.

For development and testing commands, see the [development guide](docs/development.md).

## Direction

The first implementation milestone established the TUI shell, navigation, a non-interactive CLI entry point, and a shared metadata use case. Later milestones will add interactive project creation, a project manifest, local plugin discovery, technology composition, plan review, and execution. The interfaces will remain separate from provisioning decisions and provider calls.

ProvisionHub aims to solve repeated project setup, incompatible technology choices, and fragmented integration work. For example, a future Spring Boot + PostgreSQL + Docker + GitHub Actions selection should produce one coherent project configuration, rather than four unrelated templates.

## Documentation

- [Product overview](docs/overview.md)
- [Architecture and boundaries](docs/architecture/architecture.md)
- [Planned project structure](docs/architecture/project-structure.md)
- [Plugin architecture](docs/architecture/plugins.md)
- [Decision records](docs/adr/)
- [Contributing](CONTRIBUTING.md)
- [Running and developing](docs/development.md)

The original Go monorepo and gRPC decisions remain in [ADR-0001](docs/adr/0001-go-monorepo.md) and [ADR-0002](docs/adr/0002-grpc-plugin-contract.md). [ADR-0003](docs/adr/0003-local-first-provisioning.md), [ADR-0004](docs/adr/0004-tui-and-cli.md), and [ADR-0005](docs/adr/0005-plugin-composition.md) describe the current direction and which older assumptions they supersede.
