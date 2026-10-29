# ProvisionHub

ProvisionHub is an open-source, TUI-first developer provisioning tool for individual developers and small teams. Its goal is to help developers compose, configure, initialize, and provision applications from frameworks, services, and infrastructure choices that work together.

The terminal interface guides project configuration, manifest creation, and generation-plan review. Local declarative plugins can contribute concrete file operations; execution requires a separate approval. Non-interactive CLI commands use the same application service as the TUI. The core runs locally without a control plane, database server, message broker, distributed workers, Kubernetes cluster, or persistent backend.

Plugins are intended to provide templates, technology integrations, and provisioning operations. The initial local contract supports declarative backend file generation only. Official plugins belong in the separate `provisionhub-official-plugins` repository. A searchable marketplace on the ProvisionLabs website remains future work.

## Current status

Running `provisionhub` opens a TUI with a functional **Create Project** wizard. It collects a name and destination, lets you select a backend and optional frontend/database, reviews the choices, optionally previews a generation plan, and saves `provisionhub.yaml` only on explicit confirmation. The CLI can create the same manifest without a terminal, list built-in technologies, and preview a saved manifest as text or JSON. `version` and `--help` remain available.

Open Existing Project, Plugin Marketplace, and Settings remain explanatory screens. The built-in catalog defines valid **configuration choices**; a matching local plugin is required for generation. Without plugins, plans retain unavailable-capability diagnostics. The first official NestJS generator lives in the separate [official plugins repository](https://github.com/ProvisionLabs/provisionhub-official-plugins/tree/main/plugins/nestjs). Infrastructure provisioning, CI/CD generation, plugin installation, and settings persistence are not implemented.

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

Portable builds can include a `plugins` directory beside the executable. The TUI
loads it automatically unless `PROVISIONHUB_PLUGINS` supplies an explicit override.
Keep the executable and that directory together when moving a portable build.
CLI plan/apply commands continue to use the explicit `--plugins` flag.

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

### Preview a generation plan

```sh
provisionhub project plan --file ./my-application/provisionhub.yaml
provisionhub project plan --file ./my-application/provisionhub.yaml --output json
```

The shared Planning Engine validates the manifest and catalog selections, resolves technologies, and produces deterministic operations with stable IDs, dependencies, support status, and diagnostics. It does not create directories, change project files, or execute commands. `status: planned` means the plan was constructed; `executable` is true only if every selected capability has a local generator. A valid unsupported plan exits `0`; argument, file, validation, planning, or output errors exit `1`.

In the TUI, press `p` on **Review configuration** to preview before saving. Up/Down or `j`/`k` scrolls; Enter or Esc returns to Review without saving. Enter on Review saves only the manifest. After saving, press `p` on the result screen to review the saved plan. For an executable plan, press `g`, then `y` to confirm generation; `n` or Esc cancels. Set `PROVISIONHUB_PLUGINS` to a local plugin directory before launching the TUI.

For CLI generation, review and approve the same local plugin plan:

```sh
provisionhub project plan --file ./my-application/provisionhub.yaml --plugins ./path/to/plugins
provisionhub project apply --file ./my-application/provisionhub.yaml --plugins ./path/to/plugins
# Non-interactive automation must pass --yes after reviewing the plan.
```

`project apply` never installs dependencies or runs generated commands. See the [plugin contract](docs/architecture/plugins.md) and [generation planning](docs/generation-planning.md) for safety guarantees and limitations.

With the official plugins repository checked out alongside this one, a NestJS
and Next.js configuration can generate both applications:

```sh
provisionhub project init --name sample-web --path ../sample-web --backend nestjs --frontend nextjs
provisionhub project plan --file ../sample-web/provisionhub.yaml --plugins ../provisionhub-official-plugins/plugins
provisionhub project apply --file ../sample-web/provisionhub.yaml --plugins ../provisionhub-official-plugins/plugins
cd ../sample-web/frontend
npm install
npm run build
npm run start
```

Review and approve generation at the prompt. The Next.js frontend serves a
starter page at <http://localhost:3000/>. The generated NestJS backend runs
separately from `../sample-web/backend` after its own `npm install` and
`npm run build`. NestJS requires Node.js 24.15+. Choose another `PORT` if running
both applications at once. Database generation remains unavailable.

## Direction

Configuration, local declarative plugin loading, concrete file plans, and separately approved local file generation are implemented. Official NestJS and Next.js templates in `provisionhub-official-plugins` have been validated locally through combined generation, npm installation, builds, and HTTP requests. Broader composition, infrastructure provisioning, and opening existing projects through the TUI remain future work.

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
- [Generation planning](docs/generation-planning.md)
- [Development context](docs/development-context.md)

The original Go monorepo and gRPC decisions remain in [ADR-0001](docs/adr/0001-go-monorepo.md) and [ADR-0002](docs/adr/0002-grpc-plugin-contract.md). [ADR-0003](docs/adr/0003-local-first-provisioning.md), [ADR-0004](docs/adr/0004-tui-and-cli.md), and [ADR-0005](docs/adr/0005-plugin-composition.md) describe the current direction and which older assumptions they supersede.
