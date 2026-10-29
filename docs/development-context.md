# ProvisionHub development context

This is an onboarding snapshot checked against the source through the frontend generator extension of October 29, 2026. The repository source, [README](../README.md), specifications, and accepted ADRs take precedence over this summary. Check Git state and tests before relying on a remote branch or release claim.

## Product and history

ProvisionHub is a TUI-first, local-first developer provisioning tool for individuals and small teams. Its goal is to compose technologies into coherent application configurations, then eventually generate and provision them through plugins. The original IDP architecture assumed a control plane, queue, workers, and mandatory gRPC plugin boundary. [ADR-0001](adr/0001-go-monorepo.md) and [ADR-0002](adr/0002-grpc-plugin-contract.md) preserve that history. [ADR-0003](adr/0003-local-first-provisioning.md), [ADR-0004](adr/0004-tui-and-cli.md), and [ADR-0005](adr/0005-plugin-composition.md) establish the current local process, shared application services, and plugin composition direction. [ADR-0006](adr/0006-declarative-local-generation.md) records the first limited file-generation contract.

The ProvisionLabs ecosystem assigns the Go tool, CLI/TUI, catalog, planning, and execution to `ProvisionLabs/provisionhub`; official templates belong in the separate `provisionhub-official-plugins` repository. A ProvisionLabs website marketplace remains planned. That repository contains official declarative NestJS and Next.js generators under `plugins/nestjs` and `plugins/nextjs`, alongside the preserved experimental `keycloak-go` plugin. The Next.js integration uses the same local file contract with a matching frontend capability.

## Implemented structure

| Path | Current responsibility |
| --- | --- |
| [cmd/provisionhub](../cmd/provisionhub/) | Executable entry point and process errors |
| [internal/cli](../internal/cli/) | Cobra commands, flags, text/JSON output, interactive or `--yes` approval |
| [internal/tui](../internal/tui/) | Bubble Tea navigation, wizard, preview, confirmation, result display |
| [internal/application](../internal/application/) | Shared project creation, validation, planning, freshness checks, and apply coordination |
| [internal/catalog](../internal/catalog/) | Stable built-in IDs and category/explicit-conflict validation |
| [internal/project](../internal/project/) | Versioned desired-state YAML and no-overwrite manifest persistence |
| [internal/plugins](../internal/plugins/) | Versioned local YAML discovery and validation; no code execution |
| [internal/planning](../internal/planning/) | Pure conceptual/concrete operations, ordering, support and diagnostics |
| [internal/execution](../internal/execution/) | Root-confined staged file generation and no-overwrite publication |

The interfaces share `application.ProjectService`. The planner never writes or runs commands. Plugin files are read and validated at the application/plugin boundary; the executor is a separate, approved path. [Project structure](architecture/project-structure.md) and [architecture](architecture/architecture.md) describe package boundaries. Exact Go and dependency versions are in [go.mod](../go.mod); CI is [.github/workflows/ci.yml](../.github/workflows/ci.yml).

## User-visible behavior

| Entry point | Behavior |
| --- | --- |
| `provisionhub` | TUI home and six-step Create Project wizard |
| `provisionhub project init --name demo --path ./demo --backend nestjs` | Validate and save `provisionhub.yaml`; no source generation |
| `provisionhub technologies list` | Print stable built-in catalog entries |
| `provisionhub project plan --file ./demo/provisionhub.yaml [--plugins ./plugins] [--output json]` | Read-only plan preview; concrete operations if a matching local plugin exists |
| `provisionhub project apply --file ./demo/provisionhub.yaml --plugins ./plugins [--yes]` | Preview, preflight, require explicit approval, then generate declared files without overwrites |
| `provisionhub version`, `provisionhub --help` | Metadata and command help without a TUI |

The TUI's Review `p` previews unsaved configuration without writing; Enter on Review saves only the manifest. On the saved result screen, `p` previews current plugin-backed operations. If executable, `g` opens confirmation, `y` applies, and `n`/Esc cancels. `PROVISIONHUB_PLUGINS` supplies the local plugin directory. When unset or empty, the TUI uses a `plugins` directory beside its executable if present, enabling portable builds without environment setup. CLI commands still require `--plugins`. Text fields treat `p`, `q`, `j`, and `k` as ordinary characters. Open Existing Project, Plugin Marketplace, and Settings remain explanatory screens.

The [project manifest](project-manifest.md) is `provisionhub.dev/v1alpha1`, `kind: Project`, with a portable lowercase name, required backend, and optional frontend/database. Built-in backend IDs are `go`, `nestjs`, `spring-boot`; frontend IDs are `nextjs`, `react`; database IDs are `mysql`, `postgresql`. Catalog membership means a valid choice, not generation support. The manifest stores desired configuration and no credentials or destination path. Saving a manifest creates a complete temporary file and publishes it with a no-overwrite hard link.

The [plugin contract](architecture/plugins.md) is `provisionhub.dev/plugin/v1alpha1`, supporting local declarative backend and frontend file templates and copies. Only literal `{{projectName}}` substitution is supported. Missing generators remain warnings in a planned, non-executable plan. Mixed selections with unsupported capabilities cannot be applied. Planning rejects conflicting output paths across selected plugins. The [planning/execution guide](generation-planning.md) defines operation IDs, JSON fields, diagnostics, CLI/TUI approval, freshness revalidation, and filesystem guarantees.

## Limits and next work

The official NestJS plugin generates a NestJS 12 / TypeScript 6 backend under `backend/`, requiring Node.js 24.15+. The official Next.js plugin generates a Next.js 16 / React 19 / Tailwind CSS 4 frontend under `frontend/`. Local Windows integration tests used the actual CLI in temporary directories and passed combined generation, npm install/build, and HTTP endpoint checks for both applications. Core TUI model tests cover combined preview, cancellation, and explicit confirmation. Cross-OS plugin runtime results are not claimed. A broader technology resolver, database generators, plugin installation or marketplace, arbitrary plugin code execution, cloud/infrastructure provisioning, settings persistence, and remote execution are absent. The executor performs no commands and does not install dependencies. Multi-file publication is not a transaction: a crash or concurrent filesystem change can leave partial output despite best-effort cleanup. See [ADR-0006](adr/0006-declarative-local-generation.md).

Run the plugin repository's `node --test tests/nestjs.test.mjs tests/nextjs.test.mjs` with `PROVISIONHUB_BIN` set to an absolute core executable path to repeat integration validation. Next, add cross-OS plugin runtime validation and pursue database composition only from a concrete integration need, keeping optional remote distribution separate from local use.

## Working conventions

Read [AGENTS.md](../AGENTS.md) and the [development guide](development.md). Before changes, inspect status, branch, recent commits, source, tests, and relevant ADRs. Use small Conventional Commits with explicit staging, validate Go changes with `go fmt ./...`, `go vet ./...`, `go test ./...`, and `go build ./cmd/provisionhub`, and do not claim tests or cross-OS runtime validation that was not performed. Preserve unrelated work and do not push without explicit authorization.
