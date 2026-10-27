# ProvisionHub development context

## 1. Overview and audience

ProvisionHub is an open-source developer provisioning tool maintained by ProvisionLabs, primarily for individual developers and small teams. It is TUI-first, local-first, and intended to be plugin-driven. It should help users compose compatible technologies into one application rather than concatenate unrelated templates.

This context was checked against the implementation through commit `93d9202`, including the Generation Planning Engine. It is an onboarding map, not a replacement for the linked specifications or a claim about a release or remote CI result. Recheck the source, Git state, and accepted ADRs before continuing development, and update this snapshot when behavior changes. Repository content takes precedence over previous conversations and agent memory.

Start with the [README](../README.md), [product overview](overview.md), [architecture](architecture/architecture.md), and [development guide](development.md).

## 2. Long-term product vision

The intended journey is to create or open a local project, choose frameworks, databases, services, and infrastructure providers, resolve composition constraints, review a generation plan, and explicitly approve execution. Plugins should supply integration-specific templates and operations; the core should keep the resulting application coherent.

New-project configuration, manifest persistence, and read-only conceptual plan review are exposed end to end. The CLI previews saved manifests as text/JSON, and the TUI previews unsaved Review selections. Source generation, execution, infrastructure provisioning, and interactive management of existing projects remain future work. ProvisionHub is not a hosted control plane, CI/CD service, secrets manager, or Kubernetes replacement. Optional external integrations must not turn local use into a mandatory network or server dependency. See the [planned user journey](overview.md#planned-user-journey).

## 3. ProvisionLabs repository ecosystem

| Repository or surface | Documented responsibility | Status established here |
| --- | --- | --- |
| `ProvisionLabs/provisionhub` | This Go module, CLI/TUI, application services, project data, and core architecture documentation | Present in this checkout; module identity is declared in [go.mod](../go.mod) |
| `provisionhub-official-plugins` | Separate home for official templates and technology/provider integrations | Planned by the [plugin design](architecture/plugins.md); no official plugins or loader exist in this checkout |
| ProvisionLabs website | Future searchable plugin marketplace and discovery surface | Planned; no marketplace client or registry protocol exists here |

These are documented ecosystem roles, not verification that the external repositories or services are deployed. The current documents do not establish a website repository name or an SDK repository; do not invent them or add those dependencies to the local runtime.

## 4. Original enterprise-oriented architecture

The original documentation targeted internal developer platforms and platform teams. Its flow was a replaceable browser UI or CLI, a Control Plane, a queue, distributed Workers, and provider plugins. It described centralized orchestration and system state, REST/gRPC APIs, worker scaling, and a mandatory gRPC/Protocol Buffers plugin boundary.

[ADR-0001](adr/0001-go-monorepo.md) preserves the original Go monorepo decision and service assumptions. [ADR-0002](adr/0002-grpc-plugin-contract.md) preserves the former mandatory plugin transport decision. Their original text is historical, not a list of implemented capabilities or a requirement to restore old services. Earlier experiments in Git history must not be mistaken for the current implementation.

## 5. Migration to TUI-first, local-first development

[ADR-0003](adr/0003-local-first-provisioning.md), [ADR-0004](adr/0004-tui-and-cli.md), and [ADR-0005](adr/0005-plugin-composition.md), accepted on October 26, 2026, establish the current direction:

- One local Go process by default; separate responsibilities do not imply separate services.
- A primary guided TUI and automation-friendly CLI using shared application services.
- No required control plane, server database, broker, distributed workers, Kubernetes deployment, or gRPC service.
- Plugins as the future integration boundary, with composition metadata before operations; runtime protocol, process model, packaging, and implementation languages are deferred.
- Optional remote execution or distribution only when justified by a concrete use case.

Commit `54b79d8` updated the architecture guides, marked the older ADRs amended/superseded, and removed the obsolete server diagram and browser UI rebuild guide. Do not recreate those retired requirements from historical text.

## 6. Current architecture and package responsibilities

The current path is `cmd/provisionhub` -> CLI or TUI -> application services -> planning, catalog, and project packages. The interfaces share `application.ProjectService`; application and domain code do not depend on terminal widgets or Cobra. Planning is pure: file loading is coordinated by the application service and no provider or execution API exists.

| Existing path | Responsibility |
| --- | --- |
| [cmd/provisionhub/main.go](../cmd/provisionhub/main.go) | Wire the root command, print errors to stderr, and return a failing process status |
| [internal/cli](../internal/cli/) | Cobra commands, flags, and non-interactive output; no separate provisioning workflow |
| [internal/tui](../internal/tui/) | Bubble Tea model/update/view, navigation, wizard input, rendering, and save-result presentation |
| [internal/application/info.go](../internal/application/info.go) | Product metadata shared by both interfaces |
| [internal/application/project.go](../internal/application/project.go) | Validate project input and selections, expose catalog choices, coordinate creation, and load/validate manifests |
| [internal/application/planning.go](../internal/application/planning.go) | Read a saved manifest or accept one in memory, coordinate shared planning, and propagate diagnostics/errors |
| [internal/catalog/catalog.go](../internal/catalog/catalog.go) | Built-in technology definitions, deterministic listing, ID/category checks, and explicit incompatibility checks |
| [internal/planning](../internal/planning/) | Structured conceptual plans, existing validator reuse, technology resolution, deterministic operation dependency ordering, and diagnostics without I/O |
| [internal/project/manifest.go](../internal/project/manifest.go) | Versioned desired-configuration schema, portable name validation, and strict YAML serialization/parsing |
| [internal/project/storage.go](../internal/project/storage.go) | Destination checks, no-overwrite local persistence, and manifest file loading |
| [.github/workflows/ci.yml](../.github/workflows/ci.yml) | Formatting, vet, tests, and build on Windows, Linux, and macOS |

The stack is Go 1.26.1 as declared in [go.mod](../go.mod), Cobra, Bubble Tea v2, Bubbles v2, Lip Gloss v2, and YAML v3. Resolve exact dependency versions from the module rather than this summary.

Conceptual planning now exists in `internal/planning`; broader plugin capability resolution, execution, plugin management, tool configuration, and registry remain future boundaries. The [project structure guide](architecture/project-structure.md) distinguishes implemented locations from planned ones. Add packages only with real implementation; keep provider-specific integrations out of the application/core and avoid speculative public APIs or empty scaffolding. The [planning specification](generation-planning.md) owns the model, status/diagnostic semantics, dependency rules, and JSON shape.

## 7. Implemented features and commands

| Entry point | Verified behavior |
| --- | --- |
| `provisionhub` | Open the home TUI with navigation and compact rendering for small terminals |
| Create Project in the TUI | Six steps: project information, backend, optional frontend, optional database, review, and save result; optional `p` plan preview from Review before saving |
| `provisionhub project init --name my-app --path ./my-app --backend go` | Validate configuration, create the destination if necessary, save `provisionhub.yaml`, and print its path; does not generate application source |
| `provisionhub project plan --file ./my-app/provisionhub.yaml [--output json]` | Read/validate a saved manifest and preview conceptual capabilities, support, dependencies, diagnostics, and execution availability without writes or commands |
| `provisionhub technologies list` | Print built-in technology category, ID, and display name |
| `provisionhub version` | Print shared application metadata; the inspected source reports `ProvisionHub 0.1.0-dev` |
| `provisionhub --help` | Show command help without opening a terminal session |

`project init` requires `--name`, `--path`, and `--backend`. `--frontend` and `--database` may be omitted or set to `none`. Cobra also exposes standard help and shell-completion commands; these are not additional project workflows.

The wizard preserves selections when going back, displays validation/persistence errors, and shows save success only after persistence. From Review, `p` opens a scrollable plan; Enter/Esc/`p` returns without saving, while Enter on Review still saves only the manifest. The optional pre-save preview preserves the existing six-step save workflow. In text fields, `p`, `q`, `j`, and `k` are text rather than global shortcuts. Tests drive the model without an interactive terminal. See the [development guide](development.md#current-interface) for shortcuts and build/run commands.

`ProjectService.LoadProject` loads and validates an existing manifest programmatically. There is no CLI project-open command, and Open Existing Project, Plugin Marketplace, and Settings in the TUI remain explanatory placeholders. No settings are persisted.

`PlanProject(path)` and `PlanManifest(manifest)` share the planning domain for CLI and TUI. Plan construction succeeds with `status: planned` even though every operation is `unavailable` and `executable` is always `false`. Valid unsupported previews exit `0`; argument, file, schema, technology/conflict, graph, or output errors exit `1`. JSON loading/validation failures contain an `invalid` plan and error diagnostics on stdout, with an executable error on stderr; argument/format failures can precede JSON output. See [planning diagnostics and exit codes](generation-planning.md).

## 8. Project manifest format and persistence

The authoritative schema and example are in the [manifest specification](project-manifest.md); the implementation is [manifest.go](../internal/project/manifest.go).

- File name: `provisionhub.yaml`; `apiVersion: provisionhub.dev/v1alpha1`; `kind: Project`.
- `metadata.name`: 1–63 lowercase ASCII letters, digits, or hyphens, starting and ending with a letter or digit.
- `spec.backend.technology` is required. `spec.frontend.technology` and `spec.database.technology` are optional; omit their sections for no selection. `none` is an input convenience, not a manifest technology ID.
- The file records desired configuration, not generated/provisioned state. The selected destination is not serialized. There are no credential fields; never put secrets in the manifest.
- The YAML reader rejects unknown fields, malformed or multiple documents, unsupported versions/kinds, and missing required structural fields. The application service adds catalog ID/category and explicit combination checks on create and load.
- Future schema evolution requires an explicit version and documented migration path, not silently reinterpreting an existing file.

Storage resolves the destination, rejects invalid paths and file ancestors, checks for an existing manifest with `Lstat`, and checks again when saving. It creates a temporary file in the destination, writes/syncs/closes it, then publishes with a same-directory hard link that fails if the target already exists. The temporary file is removed afterward. This avoids the platform-dependent overwrite semantics of rename.

The no-overwrite guarantee must be preserved. Hard-link support is required by the current implementation; unsupported filesystems produce an error, not an overwrite fallback. This is not a filesystem sandbox, symlink-containment guarantee, or general transactional generation engine. Directory creation can remain after a later save failure. See [storage.go](../internal/project/storage.go) and [storage tests](../internal/project/storage_test.go).

## 9. Built-in technology catalog

| Category | IDs | Selection rule |
| --- | --- | --- |
| Backend | `go`, `nestjs`, `spring-boot` | Required |
| Frontend | `nextjs`, `react` | Optional |
| Database | `mysql`, `postgresql` | Optional |

Entries carry an ID, name, category, description, and optional `IncompatibleWith` IDs. The catalog copies entry data and orders listings deterministically. It rejects unknown IDs, category mismatches, and explicitly declared conflicting selections.

No built-in entry currently declares an incompatibility. Conflict checks are exercised using custom catalog entries in tests. Therefore, historical reports of "technology compatibility validation" mean these limited checks, not dependency/capability resolution, version compatibility, or a guarantee that a combination can be generated. The built-in catalog is metadata, not a plugin runtime or collection of working generators. See [catalog.go](../internal/catalog/catalog.go), [catalog tests](../internal/catalog/catalog_test.go), and the [plugin design](architecture/plugins.md).

## 10. Completed development milestones

These milestones are verified in the current tree and local Git history; they do not imply released versions or successful remote CI runs.

| Completed increment | Traceable commits |
| --- | --- |
| Go module, executable wiring, CLI metadata, and navigable TUI shell | `fc0ef78`, `abb0ef1` |
| Metadata/navigation tests, bounded terminal layout, and three-OS CI configuration | `b492667`, `fed654d`, `cef18b5` |
| Local-first architecture migration and current ADRs | `54b79d8` |
| Versioned YAML manifest and deterministic built-in catalog | `84eb5c7`, `67f7407` |
| Shared project validation, safe persistence, and application-level loading | `2277bb2` |
| Non-interactive initialization and technology listing | `1873134` |
| Interactive configuration wizard | `362bc66` |
| Explicit incompatibility validation and manifest/workflow documentation | `ea03ccf`, `85b46af` |
| Text-input shortcut regression coverage and portable file-ancestor errors | `ceb5576`, `78da52d` |
| Validated conceptual plan domain, stable IDs, missing-provider diagnostics, and deterministic graph ordering/error handling | `3cfb4e7`, `4d2235a` |
| Shared read-only application planning, non-interactive text/JSON CLI preview, and optional pre-save TUI preview | `fdac8f6`, `f96461b`, `93d9202` |

Unit tests cover manifest round-trips and invalid inputs, catalog behavior, application services, CLI creation/listing/planning, persistence conflicts and Unicode paths, and TUI navigation, resizing, save failures/retries, preview scrolling, and input handling. Planning tests also cover deterministic plans/JSON, graph ordering/errors/cycles, explicit conflicts, structured diagnostics, and unchanged filesystem state. The CI workflow defines native Windows/Linux/macOS validation; inspect actual CI results separately when making claims about a branch or PR.

## 11. Relevant ADRs and architectural decisions

| ADR | Current meaning |
| --- | --- |
| [0001: Go monorepo](adr/0001-go-monorepo.md) | Amended by 0003 and 0004; Go and one core repository remain, but service topology and reference browser UI are historical |
| [0002: gRPC plugin boundary](adr/0002-grpc-plugin-contract.md) | Superseded by 0005 as a mandatory runtime protocol |
| [0003: Local-first provisioning](adr/0003-local-first-provisioning.md) | Accepted; local execution is the default and server infrastructure is not a prerequisite |
| [0004: TUI and CLI](adr/0004-tui-and-cli.md) | Accepted; guided TUI and automation CLI share use cases, with no required backend API |
| [0005: Plugin composition](adr/0005-plugin-composition.md) | Accepted; integration metadata should drive composition, while transport and runtime choices remain deferred |

ADR-0004 records the UI libraries as candidates at decision time; [go.mod](../go.mod) and the implementation establish that they were subsequently adopted. Keep historical ADR context intact; record significant new decisions using the [ADR template](adr/template.md) and explicitly mark supersession/amendment.

## 12. Current limitations and historical-report discrepancies

- A project can be configured and conceptually planned, not generated or provisioned. The Create Project wizard and `project init` persist desired configuration only; preview actions never save or apply a plan.
- Compatibility validation is limited to IDs/categories and declared conflicts. There is no general resolver, and the built-in entries declare no conflicts.
- Existing manifests can be loaded through the application service, but no existing-project management UI/CLI is implemented.
- Plugin contract/runtime, discovery, installation, trust model, official integrations, registry, and marketplace access are absent. Runtime language, packaging, and protocol are undecided; gRPC is not required.
- Docker, CI/CD configuration generation, authentication integrations, and infrastructure provisioning are not available. The repository's own CI workflow is not a user-project CI generator.
- The planner has no supported generators/providers, concrete target paths, or execution engine. It reports missing-provider warnings and never marks a plan executable. Operation dependency ordering is not a general plugin capability resolver. There is no progress engine, persisted execution state, settings persistence, or remote execution. Cancellation, retries, rollback, idempotency, approval, and credential contracts for future execution still need design.
- Target-architecture diagrams and historical ADR text are not evidence that conceptual packages or services exist. Cross-platform CI configuration and successful compilation alone are not proof of runtime behavior on every OS/filesystem.

The historical feature report is otherwise supported by the implementation: CLI/TUI navigation, configuration, catalog selection, versioned manifests, local persistence, non-interactive initialization, listing, and focused unit tests exist. Preserve the qualifications above in future status reports.

## 13. Planned roadmap

This is a recommended sequence consistent with the [README direction](../README.md#direction) and [target architecture](architecture/architecture.md), not an implemented or scheduled release plan:

The Generation Planning Engine milestone is implemented for conceptual built-in capabilities; it deliberately leaves all operations unavailable. Recommended follow-up work:

1. Project generation/execution vertical slice: support one concrete local application with explicit operation definitions, safe filesystem/approval behavior, and useful failure reporting.
2. Plugin contract and runtime: extract proven integration needs into metadata, lifecycle, versioning, security, and operation contracts; choose transport only after a concrete composition scenario.
3. Official plugin integrations and distribution: develop the separate official-plugins ecosystem and later optional registry/website marketplace access without requiring a network for local use.
4. Broader integrations: Docker and CI/CD configuration, services, and infrastructure provisioning through the integration boundary.
5. Existing-project management: expose loading/editing/replanning through TUI and CLI, with an explicit migration and state model.

The later items may be refined or interleaved by future ADRs; no plugin protocol, package format, marketplace API, or execution reliability guarantee is settled here.

## 14. Next recommended milestone: first supported local generation slice

Validated manifests now feed a deterministic, reviewable conceptual plan in both interfaces. The next increment should define and implement one real local generation capability, not jump directly into a plugin marketplace or distributed runtime. Planning must remain pure even when a separately authorized apply path is introduced.

Recommended acceptance criteria for that future work:

- Establish concrete operation definitions with truthful support, known targets, inputs/outputs, and risk information for one small backend scenario before advertising generation support.
- Keep CLI/TUI coordinated through application use cases, reuse validation, and preserve deterministic planning without write/execute effects.
- Separate preview from apply; define explicit approval, target/path containment, conflict/no-overwrite behavior, and actionable failure reporting before writing generated files.
- Retain warnings for unsupported technology combinations and test the supported output plus rejection of unsafe destinations/conflicts. Do not infer generation support from catalog membership.
- Document lifecycle/state implications and add packages only when the implementation needs them. Plugins, remote APIs, and infrastructure changes are not prerequisites for proving the first local slice.

The exact apply commands, operation/provider APIs, and reliability guarantees still require design. Broader execution, infrastructure operations, plugin loading, and marketplace work must remain independently scoped rather than incidental additions. No such functionality was implemented by the planning milestone.

## 15. Git workflow and validation requirements

Before making changes, run `git status`, `git branch --show-current`, and `git log -5 --oneline`. Confirm the intended repository/worktree, read relevant code/specifications/ADRs and any existing agent instructions, and preserve unrelated changes. Do not switch branches or restore files merely to obtain a clean tree.

Use gradual, small, logically separated Conventional Commits. Keep intermediate commits consistent, validate before each commit, review the staged diff, and stage explicit relevant paths rather than `git add .`. Do not amend, squash, rewrite existing history, or force-push. Push only with explicit authorization; never assume a locally present milestone is already on the upstream branch.

For Go changes, use the commands in the [development guide](development.md#validation):

```sh
go fmt ./...
go vet ./...
go test ./...
go build ./cmd/provisionhub
```

Keep tests focused and close to the changed package. Follow the existing use of temporary directories and direct TUI-model tests; cover errors and no-overwrite behavior. Preserve Windows, Linux, and macOS support, including portable paths, spaces/Unicode, filesystem errors, terminal dimensions, and non-interactive operation. CI checks formatting, vet, tests, and build on all three operating systems; do not claim those remote checks passed without reading their results.

For documentation-only work, Go tests are optional. Check referenced paths and relative Markdown links, compare architecture statements against accepted ADRs and actual source, clearly separate implemented behavior from plans, run `git diff --check` (and `git diff --cached --check` after staging), and review final Git status. Keep schema rules in the manifest specification, design in the architecture guides/ADRs, and this context as a linked summary rather than a competing specification. Never claim a validation command or result that was not actually run.
