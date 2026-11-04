# ProvisionHub

ProvisionHub is an open-source, TUI-first developer provisioning tool for individual developers and small teams. Its goal is to help developers compose, configure, initialize, and provision applications from frameworks, services, and infrastructure choices that work together.

The terminal interface guides project configuration, manifest creation, and generation-plan review. Local declarative plugins can contribute concrete file operations; execution requires a separate approval. Non-interactive CLI commands use the same application service as the TUI. The core runs locally without a control plane, database server, message broker, distributed workers, Kubernetes cluster, or persistent backend.

Plugins are intended to provide templates, technology integrations, and provisioning operations. The local contract supports declarative backend and frontend file generation. Official plugins belong in the separate `provisionhub-official-plugins` repository. A searchable marketplace on the ProvisionLabs website remains future work.

## Current status

Running `provisionhub` opens a TUI with a **Create Project** wizard for multiple named components. It collects a name and destination, lets you add components, previews a generation plan, and saves `provisionhub.yaml` only on explicit confirmation. The CLI can still create either manifest format without a terminal, list built-in technologies, and preview a saved manifest as text or JSON. `version` and `--help` remain available.

Settings persists local workspace/provider preferences in SQLite and manages optional GitHub account connections using device authorization and the OS credential vault. **Plugins** discovers, installs, lists, updates and removes official declarative packages through a static registry and versioned GitHub Release assets. Open Existing Project remains explanatory. The built-in catalog defines valid **configuration choices**; a matching healthy plugin is required for generation. Official NestJS and Next.js generators live in the separate [official plugins repository](https://github.com/ProvisionLabs/provisionhub-official-plugins). Infrastructure provisioning and CI/CD generation remain future work.

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
CLI plan/apply commands share managed discovery and also accept `--plugins`.

### Install official plugins

```sh
provisionhub plugin search
provisionhub plugin install official/nestjs
provisionhub plugin install official/nextjs
provisionhub plugin list
provisionhub plugin updates
```

Installed generators persist in application data and work offline. Home →
**Plugins** offers Available, Installed and Updates. See the
[Plugin Registry and Manager guide](docs/plugin-registry.md) for commands,
checksums, recovery, local plugin compatibility and explicit version selection.

### Application settings and GitHub

Open **Settings** from Home. F1 edits the default workspace and preferred provider;
Enter saves. F2 manages Git provider accounts: `c` connects GitHub, `r` validates
stored credentials, and `d` disconnects the selected account. GitLab and Bitbucket
are visibly marked Coming soon. Esc cancels device authorization.

```sh
provisionhub settings show
provisionhub settings set --workspace /absolute/existing/workspace --provider github
provisionhub settings git-status
provisionhub settings git-status --validate
```

GitHub connection requires your registered OAuth app's public client ID in
`PROVISIONHUB_GITHUB_CLIENT_ID`, with device flow enabled. No client secret or
repository permissions are requested. Tokens stay in Windows Credential Manager,
macOS Keychain, or Linux Secret Service, never SQLite or project files. Missing
network, authentication, vault, or settings storage does not block local generation.
See [Settings](docs/settings.md), [GitHub setup](docs/git-providers.md), and
[database locations/migrations](docs/local-persistence.md).

### Create a project configuration

From Home, choose **Create Project**. Enter a lowercase project name and destination, add at least one named component, then review and confirm. The result screen appears only after `provisionhub.yaml` has been saved. An existing manifest is never overwritten.

On the component list, `a` adds a component, `e` edits the selected component, and `d` removes it. Multiple components may use the same technology. The manifest stays at the project root as the `plan`/`apply` anchor; generated applications live in isolated component directories.

The equivalent non-interactive command is:

```sh
provisionhub project init --name my-application --path ./my-application --backend nestjs --frontend nextjs --database postgresql
provisionhub technologies list
```

For legacy flags, `--name`, `--path`, and `--backend` are required. Omit `--frontend` or `--database`, or pass `none`, to leave them out of the manifest. For complex projects, author a [named-component manifest](docs/project-manifest.md), then run `provisionhub project init --from ./source.yaml --path ./my-application`. `--from` cannot be combined with legacy technology flags. On Home and technology screens, use Up/Down or `j`/`k` to navigate and Enter to select. On text fields, `j`, `k`, and `q` are normal input. Esc goes back and Ctrl+C quits. The TUI has a compact layout for small terminals.

See the [manifest specification](docs/project-manifest.md) for the schema, built-in IDs, and validation rules. This command **saves desired configuration only**; it does not generate application code.

For development and testing commands, see the [development guide](docs/development.md).

### Preview a generation plan

```sh
provisionhub project plan --file ./my-application/provisionhub.yaml
provisionhub project plan --file ./my-application/provisionhub.yaml --output json
```

The shared Planning Engine validates the manifest and catalog selections, resolves technologies, and produces deterministic operations with stable IDs, dependencies, support status, and diagnostics. It does not create directories, change project files, or execute commands. `status: planned` means the plan was constructed; `executable` is true only if every selected capability has a local generator. A valid unsupported plan exits `0`; argument, file, validation, planning, or output errors exit `1`.

In the TUI, press `p` on **Review configuration** to inspect the read-only plan. Up/Down or `j`/`k` scrolls; Enter or Esc returns to Review without writing. When every selected technology has a generator, Review shows **Generate Project** and Enter explicitly creates the manifest and declared files in one action. Otherwise Review shows **Save Manifest Only**, preserving valid configurations without generators. If generation fails after the manifest is saved, the result screen retains the manifest and offers `p` to review and `g` then `y` to retry after resolving the problem. Set `PROVISIONHUB_PLUGINS` to a local plugin directory before launching the TUI, or place `plugins` beside the executable.

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

For two backends and two frontends, save this configuration as `source.yaml`:

```yaml
apiVersion: provisionhub.dev/v1alpha1
kind: Project
metadata: {name: commerce-platform}
spec:
  components:
    - {name: api, type: backend, technology: nestjs}
    - {name: billing, type: backend, technology: nestjs}
    - {name: web, type: frontend, technology: nextjs}
    - {name: admin, type: frontend, technology: nextjs}
```

```sh
provisionhub project init --from ./source.yaml --path ../commerce-platform
provisionhub project plan --file ../commerce-platform/provisionhub.yaml --plugins ../provisionhub-official-plugins/plugins
provisionhub project apply --file ../commerce-platform/provisionhub.yaml --plugins ../provisionhub-official-plugins/plugins
```

The plan lists `component/api`, `component/billing`, `component/web`, and `component/admin`; approved generation writes each application to its own directory.

## Direction

Named-component composition, local declarative plugin loading, concrete file plans, and separately approved local file generation are implemented. Official NestJS and Next.js templates in `provisionhub-official-plugins` have been validated locally through four-component generation and npm builds, plus legacy HTTP requests. Connection execution, infrastructure provisioning, and opening existing projects through the TUI remain future work.

ProvisionHub aims to solve repeated project setup, incompatible technology choices, and fragmented integration work. For example, a future Spring Boot + PostgreSQL + Docker + GitHub Actions selection should produce one coherent project configuration, rather than four unrelated templates.

## Documentation

- [Application settings](docs/settings.md)
- [Git provider authentication and OAuth setup](docs/git-providers.md)
- [Local SQLite persistence](docs/local-persistence.md)
- [Settings/authentication validation](docs/settings-validation.md)

- [Product overview](docs/overview.md)
- [Architecture and boundaries](docs/architecture/architecture.md)
- [Named component model](docs/architecture/components.md)
- [Planned project structure](docs/architecture/project-structure.md)
- [Plugin architecture](docs/architecture/plugins.md)
- [Plugin Registry v1 and Plugin Manager](docs/plugin-registry.md)
- [Decision records](docs/adr/)
- [Contributing](CONTRIBUTING.md)
- [Running and developing](docs/development.md)
- [Project manifest specification](docs/project-manifest.md)
- [Generation planning](docs/generation-planning.md)
- [Development context](docs/development-context.md)

The original Go monorepo and gRPC decisions remain in [ADR-0001](docs/adr/0001-go-monorepo.md) and [ADR-0002](docs/adr/0002-grpc-plugin-contract.md). [ADR-0003](docs/adr/0003-local-first-provisioning.md), [ADR-0004](docs/adr/0004-tui-and-cli.md), and [ADR-0005](docs/adr/0005-plugin-composition.md) describe the current direction and which older assumptions they supersede.
