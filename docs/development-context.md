# ProvisionHub development context

Checked against source on November 10, 2026. Code/tests take precedence.
Read [AGENTS.md](../AGENTS.md) before work.

ProvisionHub is a local-first, one-time source initializer. CLI/TUI collect a
name, destination, named components, scalar options, and optional connections.
The project specification stays in memory. Generation writes ordinary source
and application configuration, never project YAML, a project lockfile, or a
saved-project registration. Repeated technologies are independent instances.

`internal/configuration` owns flat string/boolean/integer/enum schemas, typed
text parsing, strict validation, defaults, and bounded host formatting.
`internal/connections` matches plugin provider/consumer capabilities on named
instances, resolves metadata and connection options, computes compatible target
choices, and rejects ambiguity, slot conflicts, duplicates, and dependency cycles.
Both packages are pure and perform no filesystem, network, or execution work.

`internal/plugins` validates YAML, literal references, paths, and every declared
base/integration source. Optional connection providers bind metadata to literals
or component configuration; consumers declare metadata requirements, scalar
connection schemas, and connection-owned file sets. One consumer slot has one
target per component; multiple components may share a provider. There is no
framework pairing logic, dependency solver, arbitrary expression, or plugin script.

`internal/planning` resolves all component and connection values before
contributing operations, renders in memory, and records stable dependencies,
ownership, metadata, outputs, and digests. `internal/execution` repeats review
freshness checks and uses existing root confinement, staging, no-overwrite hard
links, and cleanup for both base and integration files. Hard-link support is
required; crash-atomic multi-file publication is not promised.

CLI `project plan`/`project apply` retain component flags and `--set`, and add
repeatable `--connect source:target:type` plus
`--connection-set source:target:type.key=value`. JSON and text plans include
resolved connections. Apply requires terminal approval or explicit `--yes`.
TUI Components opens Connections with `c`; `a`/`e`/`d` manage declarations.
Asynchronous selection filters incompatible targets and reuses the dynamic
configuration form. Review/preview show connections and generation outputs.
Draft editing, component rename/removal, narrow layouts, and stale reviews are tested.

`application.PluginManager`, `internal/registry`, and `internal/localdb` retain
managed package discovery, exact version selection, offline loading, and content
integrity. Integration templates are protected even when not selected for output.
Explicit local plugins can generate with an unusable SQLite path when they
cover all selected generators. SQLite remains for Settings, Git account
metadata, and plugin installations; historical saved-project tables/rows are
preserved, with no project lifecycle services or registrations restored.

Core compatibility is 0.3.0. Official source plugins are 1.3.0 (unreleased):
NestJS provides HTTP metadata using its resolved port; Next.js consumes it and
creates `.env.local`/`.env.example` with private server-side `API_ORIGIN`.
The consumer host is mandatory and explicit. Published registry packages remain
immutable 1.1.0; 1.2.0 was not published at investigation time. No releases are
published by this milestone. Use local sources or local managed lifecycle
fixtures until a separate authorized release.

Generation configures initial source; it does not establish networking, install
dependencies, start services, configure CORS/TLS/proxies, discover containers,
provision infrastructure, or reconcile later changes. No public browser endpoint
is inferred. Backend PORT/deployment changes require updating application
configuration. Unconnected projects retain their existing file output.

Validation passed on Windows: `go fmt ./...`, `go vet ./...`,
`go test -count=1 ./...`, and `go build ./cmd/provisionhub`. Cross-platform
compilation targets Windows/Linux/macOS amd64. Core and official CI validate
natively on all three OS families. Unit and offline tests use temporary local
fixtures without live service access. The opt-in official lifecycle installs
packages over local HTTPS, restarts offline, generates configured connections,
checks tamper detection/removal isolation, and verifies zero saved registrations.

Official offline Node tests verify shared/distinct targets, all base files
unchanged, deterministic plans/files, approval, unsafe paths, no overwrite, and
blocked persistence. The separate runtime test passed both scenarios on Windows:
eight actual dependency installations/builds, Next.js lint, and real HTTP
requests from two generated Next.js servers to selected NestJS instances. Probe
routes exist only in temporary tests; no client/page is shipped by the plugins.

See [connections and authoring](component-connections.md),
[configuration](plugin-configuration.md), [planning](generation-planning.md),
[CLI](cli-reference.md), [TUI](tui.md), and [ADR-0013](adr/0013-component-connections.md).
Recommended next milestone: plugin contract conformance and release validation,
with package publication requiring a separate explicit authorization.
