# ProvisionHub development context

Checked against the source on November 5, 2026. Repository code and tests take precedence over this summary. Read [AGENTS.md](../AGENTS.md) and inspect current Git state before work.

Plugin Registry v1 and Plugin Manager are implemented. [ADR-0009](adr/0009-static-plugin-registry.md)
records namespaced distribution identity, immutable artifacts and trust boundaries.
The [registry guide](plugin-registry.md) documents schema, packaging, lifecycle,
SQLite, CLI/TUI, recovery, and offline use.
Official NestJS and Next.js 1.1.0 packages are published through the official
repository's manually dispatched release workflow.

## Product and decisions

ProvisionHub is a local-first TUI and CLI for composing and generating applications. [ADR-0003](adr/0003-local-first-provisioning.md) retired the original required control plane and queue; [ADR-0005](adr/0005-plugin-composition.md) established technology/plugin composition; [ADR-0006](adr/0006-declarative-local-generation.md) introduced safe local file generation. [ADR-0007](adr/0007-named-project-components.md) introduces named component instances and preserves legacy manifests. Historical ADRs remain unchanged.

The Go repository owns catalog definitions, manifest parsing, application services, planning, execution, CLI, and TUI. The separate [official plugins repository](https://github.com/ProvisionLabs/provisionhub-official-plugins) owns the NestJS and Next.js declarative generators. No plugin recompilation, downloaded runtime, or arbitrary execution hook is required.

## Implemented boundaries

| Path | Responsibility |
| --- | --- |
| [internal/project](../internal/project/) | Strict `v1alpha1` YAML, canonical named/legacy components, connection metadata, safe manifest persistence |
| [internal/catalog](../internal/catalog/) | Stable technology IDs and category/conflict validation |
| [internal/plugins](../internal/plugins/) | Local declarative plugin discovery, template/path validation, component output rebasing |
| [internal/registry](../internal/registry/) | Strict static index, stable release resolution, HTTPS source, explicit stale cache and immutable history |
| [internal/planning](../internal/planning/) | Pure deterministic component-owned operations, graph ordering, diagnostics, executability |
| [internal/execution](../internal/execution/) | Reviewed, root-confined staging and no-overwrite file publication |
| [internal/application](../internal/application/) | Shared validation, saved-project discovery, project creation, lock preview, plan freshness, and apply coordination |
| [internal/cli](../internal/cli/) | Cobra commands, text/JSON plans, terminal or `--yes` approval |
| [internal/tui](../internal/tui/) | Bubble Tea navigation, named-component wizard, My Projects, preview, explicit generation |
| [internal/settings](../internal/settings/) | Typed workspace/provider preferences and validation |
| [internal/localdb](../internal/localdb/) | Lazy database/sql SQLite repositories and transactional versioned migrations |
| [internal/credentials](../internal/credentials/) | Native OS credential-vault interface and secret redaction |
| [internal/gitprovider](../internal/gitprovider/) | Provider-neutral identity/authentication contract and GitHub device flow/refresh |

## User behavior

`provisionhub project init --name demo --path ./demo --backend nestjs` remains the simple legacy CLI flow. A complex project can use `project init --from ./source.yaml --path ./demo`. Both formats use `project plan --file ... --plugins ...` and `project apply --file ... --plugins ...`; apply requires explicit approval. The TUI creates named-component manifests directly: after project information, `a`, `e`, and `d` add, edit, and remove instances. Review `p` shows the read-only plan; Enter on **Generate Project** creates the manifest and generated files. The result screen offers a fresh preview and confirmed retry if generation fails after the manifest is saved.

Initialization registers the created manifest in SQLite after it is saved. `project list/register/forget` and Home → **My Projects** manage local references. SQLite migration 6 stores only IDs, canonical paths, and timestamps; manifest names/components/connections and lock versions are always read from disk. Moved or invalid projects remain listed for relocation or forgetting. Project details offer read-only plan review, confirmed apply, and plugin lock validation, creation, update preview, and explicit official artifact restoration. Local discovery and opening work offline. See [saved project management](project-management.md), [TUI controls](tui.md), and [CLI reference](cli-reference.md).

CLI and TUI tests now inject temporary saved-project databases, with package-level temporary data directories as a fallback. A real CLI subprocess regression test supplies isolated paths for all supported OS conventions and verifies a separate sentinel SQLite file remains unchanged. My Projects shows compact availability rows and offers a selective, confirmed cleanup preview for unavailable references; available references are never cleanup candidates. Unavailable rows remain until relocation, repair, or explicit forgetting.

`plugin list/search/info/install/update/updates/remove` and Home → Plugins share
`application.PluginManager`. List and generation remain offline. Discovery is
explicit, with clearly stale cache fallback. Installation verifies bounded HTTPS
downloads and manifests, stages files, activates by rename and transitions SQLite
metadata to ready. Migrations 3–5 add installation rows, content integrity and
minimum-core compatibility. Broken/interrupted packages never load as healthy.
Removal touches only managed version containers; unmanaged directories remain
supported via explicit paths, environment override and portable bundles.

Updates retain old versions. Multiple candidates require `--plugin-version
namespace/name@x.y.z` on CLI plan/apply, or Installed → `s` for a TUI session.
Projects can explicitly create `.provisionhub/plugins.lock.yaml` through
`project lock` or the Create Project result screen. Locked plan/apply resolution
uses exact installed versions and content digests. `--check`, `--update`, and
`--restore` provide validation, intentional updates, and explicit official
artifact restoration. See [ADR-0010](adr/0010-project-plugin-locking.md).

The manifest remains at the destination root, anchoring plan/apply. Named components generate below their names (`api/`, `billing/`, `web/`); legacy manifests retain `backend/` and `frontend/` outputs. `{{componentName}}` is the instance name for named projects and the project name for legacy projects. The same plugin definition serves repeated instances. Optional `http` connections are validated and serialized but do not wire applications together. A catalog technology without a generator makes the plan non-executable.

Planning does not render, write, install dependencies, or execute commands. Execution rebuilds and compares the reviewed plan, checks the destination and target paths, stages complete files, and uses no-overwrite hard links. It rejects path traversal, symlinked output parents, and collisions. Multi-file publication is not fully transactional after a process crash or hostile concurrent filesystem change.

## Validation and limits

Core validation: `go fmt ./...`, `go vet ./...`, `go test -count=1 ./...`, and `go build ./cmd/provisionhub`. Core CI runs on Windows, Linux, and macOS. With `PROVISIONHUB_BIN` set to an absolute core executable, the official plugin repository runs `node --test tests/nestjs.test.mjs tests/nextjs.test.mjs tests/components.test.mjs`. Local Windows validation generated and built two NestJS and two Next.js components; the legacy tests also started each app and verified HTTP responses. Cross-OS Node runtime validation is not claimed.

Registry milestone validation used mocked TLS servers, temporary storage and both
real official definitions. Official CI validates deterministic pinned-toolchain
packages, checksums, immutable history and offline four-component generation.
Live Windows CLI checks fetched the public registry, installed both published
packages, restarted with registry access blocked, generated 40 files, checked
same-version update behavior, removed packages and verified local discovery.
Updating to a newer version was tested with a mocked second release; there is
only one published version per official plugin. No new npm runtime/build checks
are claimed for this distribution milestone.

Application Settings is implemented: General persists an optional workspace and
preferred provider; Git Providers connects GitHub with a registered public client
ID via device authorization. SQLite stores preferences/account metadata, while
tokens and refresh material stay in the native vault. Reads are offline by default;
explicit validation determines connected state. Device polling and persistence
run outside the TUI event loop with cancellation and stale-result protection.
CLI `settings show/set/git-status/git-disconnect` shares the same services.
GitLab and Bitbucket are visible as Coming soon. Self-hosted adapters are future
work; metadata already supports distinct instances and multiple accounts.

No live GitHub browser authorization is claimed. Mocked HTTP lifecycle tests,
temporary SQLite restart/migration/error tests, and an opt-in Windows Credential
Manager round trip were performed. See [settings validation](settings-validation.md),
[OAuth setup](git-providers.md), and [ADR-0008](adr/0008-local-settings-and-credentials.md).

Database generation, component connection execution, authentication inside
generated applications, infrastructure/cloud provisioning, a marketplace website
or backend, existing-project TUI editing, and shell-command plugins remain out
of scope. See the [manifest specification](project-manifest.md),
[component architecture](architecture/components.md), [plugin contract](architecture/plugins.md),
and [planning/execution guide](generation-planning.md).
