# ProvisionHub development context

Checked against the source on October 30, 2026. Repository code and tests take precedence over this summary. Read [AGENTS.md](../AGENTS.md) and inspect current Git state before work.

## Product and decisions

ProvisionHub is a local-first TUI and CLI for composing and generating applications. [ADR-0003](adr/0003-local-first-provisioning.md) retired the original required control plane and queue; [ADR-0005](adr/0005-plugin-composition.md) established technology/plugin composition; [ADR-0006](adr/0006-declarative-local-generation.md) introduced safe local file generation. [ADR-0007](adr/0007-named-project-components.md) introduces named component instances and preserves legacy manifests. Historical ADRs remain unchanged.

The Go repository owns catalog definitions, manifest parsing, application services, planning, execution, CLI, and TUI. The separate [official plugins repository](https://github.com/ProvisionLabs/provisionhub-official-plugins) owns the NestJS and Next.js declarative generators. No plugin recompilation, downloaded runtime, or arbitrary execution hook is required.

## Implemented boundaries

| Path | Responsibility |
| --- | --- |
| [internal/project](../internal/project/) | Strict `v1alpha1` YAML, canonical named/legacy components, connection metadata, safe manifest persistence |
| [internal/catalog](../internal/catalog/) | Stable technology IDs and category/conflict validation |
| [internal/plugins](../internal/plugins/) | Local declarative plugin discovery, template/path validation, component output rebasing |
| [internal/planning](../internal/planning/) | Pure deterministic component-owned operations, graph ordering, diagnostics, executability |
| [internal/execution](../internal/execution/) | Reviewed, root-confined staging and no-overwrite file publication |
| [internal/application](../internal/application/) | Shared validation, project creation, plan freshness, and apply coordination |
| [internal/cli](../internal/cli/) | Cobra commands, text/JSON plans, terminal or `--yes` approval |
| [internal/tui](../internal/tui/) | Bubble Tea navigation, named-component wizard, preview, explicit generation |

## User behavior

`provisionhub project init --name demo --path ./demo --backend nestjs` remains the simple legacy CLI flow. A complex project can use `project init --from ./source.yaml --path ./demo`. Both formats use `project plan --file ... --plugins ...` and `project apply --file ... --plugins ...`; apply requires explicit approval. The TUI creates named-component manifests directly: after project information, `a`, `e`, and `d` add, edit, and remove instances. Review `p` shows the read-only plan; Enter on **Generate Project** creates the manifest and generated files. The result screen offers a fresh preview and confirmed retry if generation fails after the manifest is saved.

The manifest remains at the destination root, anchoring plan/apply. Named components generate below their names (`api/`, `billing/`, `web/`); legacy manifests retain `backend/` and `frontend/` outputs. `{{componentName}}` is the instance name for named projects and the project name for legacy projects. The same plugin definition serves repeated instances. Optional `http` connections are validated and serialized but do not wire applications together. A catalog technology without a generator makes the plan non-executable.

Planning does not render, write, install dependencies, or execute commands. Execution rebuilds and compares the reviewed plan, checks the destination and target paths, stages complete files, and uses no-overwrite hard links. It rejects path traversal, symlinked output parents, and collisions. Multi-file publication is not fully transactional after a process crash or hostile concurrent filesystem change.

## Validation and limits

Core validation: `go fmt ./...`, `go vet ./...`, `go test -count=1 ./...`, and `go build ./cmd/provisionhub`. Core CI runs on Windows, Linux, and macOS. With `PROVISIONHUB_BIN` set to an absolute core executable, the official plugin repository runs `node --test tests/nestjs.test.mjs tests/nextjs.test.mjs tests/components.test.mjs`. Local Windows validation generated and built two NestJS and two Next.js components; the legacy tests also started each app and verified HTTP responses. Cross-OS Node runtime validation is not claimed.

Database generation, connection execution, auth, infrastructure/cloud provisioning, plugin marketplace/installation, existing-project TUI editing, and shell-command plugins remain out of scope. See the [manifest specification](project-manifest.md), [component architecture](architecture/components.md), [plugin contract](architecture/plugins.md), and [planning/execution guide](generation-planning.md).
