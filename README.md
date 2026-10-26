# ProvisionHub

ProvisionHub is an open-source, TUI-first developer provisioning tool for individual developers and small teams. Its goal is to help developers compose, configure, initialize, and provision applications from frameworks, services, and infrastructure choices that work together.

The terminal interface now guides project configuration and manifest creation. Future milestones will add source generation, plan review, execution, and work on existing projects. Non-interactive CLI commands use the same application service as the TUI. The core runs locally without a control plane, database server, message broker, distributed workers, Kubernetes cluster, or persistent backend.

Plugins are intended to provide templates, technology integrations, and provisioning operations. Official plugins will live in the separate `provisionhub-official-plugins` repository. A searchable marketplace on the ProvisionLabs website is a future distribution channel; it is not part of this repository today.

## Current status

Running `provisionhub` opens a TUI with a functional **Create Project** wizard. It collects a name and destination, lets you select a backend and optional frontend/database, reviews the choices, and saves `provisionhub.yaml`. The CLI can create the same manifest without a terminal and list the built-in technologies. `version` and `--help` remain available.

Open Existing Project, Plugin Marketplace, and Settings remain explanatory screens. The built-in catalog defines valid **configuration choices**, not working generators. No application source code, infrastructure, or CI/CD files are generated. Plugin discovery, provisioning, and settings persistence are not implemented.

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

### Create a project configuration

From Home, choose **Create Project**. Enter a lowercase project name and destination, choose a backend, optionally choose a frontend and database, then review and confirm. The result screen appears only after `provisionhub.yaml` has been saved. An existing manifest is never overwritten.

The equivalent non-interactive command is:

```sh
provisionhub project init --name my-application --path ./my-application --backend nestjs --frontend nextjs --database postgresql
provisionhub technologies list
```

`--name`, `--path`, and `--backend` are required. Omit `--frontend` or `--database`, or pass `none`, to leave them out of the manifest. On Home and technology screens, use Up/Down or `j`/`k` to navigate and Enter to select. On the information screen, `j`, `k`, and `q` are normal text input; Tab or Up/Down changes fields. Esc goes back and Ctrl+C quits. `q` quits outside text input. The TUI has a compact layout for small terminals.

See the [manifest specification](docs/project-manifest.md) for the schema, built-in IDs, and validation rules. This command **saves desired configuration only**; it does not generate application code.

For development and testing commands, see the [development guide](docs/development.md).

## Direction

The current milestone adds the first project configuration workflow and a built-in catalog. The next milestone should turn saved manifests into reviewable generation plans. Plugin loading, integration-specific generation, provisioning execution, and opening existing projects through the TUI remain future work.

ProvisionHub aims to solve repeated project setup, incompatible technology choices, and fragmented integration work. For example, a future Spring Boot + PostgreSQL + Docker + GitHub Actions selection should produce one coherent project configuration, rather than four unrelated templates.

## Documentation

- [Product overview](docs/overview.md)
- [Architecture and boundaries](docs/architecture/architecture.md)
- [Planned project structure](docs/architecture/project-structure.md)
- [Plugin architecture](docs/architecture/plugins.md)
- [Decision records](docs/adr/)
- [Contributing](CONTRIBUTING.md)
- [Running and developing](docs/development.md)
- [Project manifest specification](docs/project-manifest.md)

The original Go monorepo and gRPC decisions remain in [ADR-0001](docs/adr/0001-go-monorepo.md) and [ADR-0002](docs/adr/0002-grpc-plugin-contract.md). [ADR-0003](docs/adr/0003-local-first-provisioning.md), [ADR-0004](docs/adr/0004-tui-and-cli.md), and [ADR-0005](docs/adr/0005-plugin-composition.md) describe the current direction and which older assumptions they supersede.
