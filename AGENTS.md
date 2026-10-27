# Agent instructions

## Project and source of truth

ProvisionHub, maintained by ProvisionLabs, is an open-source, TUI-first, local-first developer provisioning tool for individuals and small teams. Its goal is coherent application composition through technology choices and future plugin integrations.

This file is the canonical, agent-agnostic source of permanent coding instructions. Use repository code, documentation, and accepted ADRs rather than agent memory or conversation history. If agent-specific instruction files become necessary, reference this file instead of copying its rules. Read [development context](docs/development-context.md) for current capabilities, limitations, milestones, and recommended next work; verify that snapshot against the source before relying on it.

## Architecture and navigation

- Use Go and the toolchain/dependency versions in [go.mod](go.mod): Cobra for CLI, Bubble Tea/Bubbles/Lip Gloss for TUI, and YAML for project manifests.
- Keep CLI/TUI as thin presentation adapters into shared application services. Keep validation, composition, and future planning/execution independent of command parsing and terminal rendering. Provider-specific operations belong at plugin/adapter boundaries.
- Preserve one local process as the default. Do not introduce a required central server, database, broker, distributed workers, Kubernetes, marketplace connection, or gRPC service. Plugin runtime/transport remain deferred decisions, not an invitation to restore the retired server architecture.
- Navigate via [README](README.md), [project structure](docs/architecture/project-structure.md), and [architecture](docs/architecture/architecture.md). Existing code lives in `cmd/provisionhub`, `internal/cli`, `internal/tui`, `internal/application`, `internal/catalog`, and `internal/project`; tests sit beside it. CI is in [.github/workflows/ci.yml](.github/workflows/ci.yml).
- Read [ADR-0003](docs/adr/0003-local-first-provisioning.md), [ADR-0004](docs/adr/0004-tui-and-cli.md), and [ADR-0005](docs/adr/0005-plugin-composition.md). ADR-0001 is amended and ADR-0002 is superseded; their original service and mandatory-gRPC assumptions are historical.
- Add packages and dependencies only for concrete implementation needs. Do not create empty planned packages, speculative public APIs, or unrelated features/refactors.

## Quality and portability

- Trace definitions and usages, inspect neighboring code/tests, and read relevant specifications before editing. Preserve existing conventions and keep changes scoped.
- For Go changes, add focused regression/behavior tests, run `go fmt ./...`, `go vet ./...`, `go test ./...`, and `go build ./cmd/provisionhub`. Follow the [development guide](docs/development.md); keep interface behavior aligned through shared services.
- Preserve Windows, Linux, and macOS support. Use portable filesystem APIs and paths; test spaces/Unicode, conflicts/errors, narrow terminal layouts, and non-interactive CLI behavior. Prefer temporary directories and direct TUI-model tests over real infrastructure or interactive terminals.
- For documentation-only changes, Go tests are optional. Check referenced paths and relative Markdown links, architectural consistency, and `git diff --check`. Report only validation actually performed; do not imply cross-OS runtime or remote CI success from local compilation alone.

## Filesystem and infrastructure safety

- Preserve existing files and unrelated work. Maintain the manifest no-overwrite guarantee; do not replace its publication method with an overwriting fallback. Follow the [manifest specification](docs/project-manifest.md), including explicit schema evolution and strict validation.
- Do not put credentials in manifests, source, examples, logs, or commits. Do not read or expose credential files unless explicitly required and authorized.
- Keep planning free of write/execute side effects. Do not run generated commands, load untrusted plugins, delete user data, or create/change infrastructure without explicit authorization and a clear target/scope. Future destructive execution requires a reviewable plan and deliberate approval; do not invent rollback or sandbox guarantees.

## Documentation and Git workflow

- Distinguish implemented behavior from target design. Update affected specifications/guides and [development context](docs/development-context.md) when behavior or architectural status changes; link to authoritative documents instead of maintaining competing copies.
- Record significant decisions with the [ADR template](docs/adr/template.md). Preserve historical context and explicitly mark superseded/amended decisions rather than silently rewriting them.
- Before work, run `git status`, `git branch --show-current`, and `git log -5 --oneline`; confirm the intended repository/worktree and preserve unrelated changes.
- Use gradual, small, atomic Conventional Commits with consistent intermediate states. Validate before committing, stage only explicit relevant paths, review the staged diff, and run `git diff --cached --check`. Never blindly use `git add .`.
- Do not amend/squash existing commits, rewrite history, force-push, or discard user changes. Do not push without explicit authorization. Finish by reviewing Git status and reporting changed files, actual validation, remaining limitations, and any commits created.
