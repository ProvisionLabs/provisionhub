# Architecture

## Status

This describes the **target architecture** and identifies what exists today. The repository has a CLI, TUI project configuration wizard, built-in catalog, versioned project manifest, shared validation/persistence services, and a deterministic conceptual Generation Planning Engine with CLI/TUI previews. Execution, plugin loading, config, and registry behavior remain future work. The earlier control-plane, queue, and worker diagram was retired with the local-first decision in [ADR-0003](../adr/0003-local-first-provisioning.md).

## Runtime shape

```text
              TUI           non-interactive CLI
                \             /
                 application use cases
                 /       |       \
          project data  core planning  plugin management
                             |                |
                      local execution ---- plugin operations
                             |                |
                      files and external tools/providers
```

One local process is the default. TUI and CLI are adapters into the same application use cases. Application services coordinate project state, composition, planning, approval, and execution; they do not depend on terminal rendering or command parsing. The core owns deterministic composition rules and planned operation semantics. The plugin system discovers and invokes integrations under an explicit contract. Provider-specific behavior belongs in plugins or adapters at the external boundary.

## Responsibilities

| Boundary | Responsibility | Must not own |
| --- | --- | --- |
| TUI | Screens, navigation, input, plan and progress display | Composition rules or provider calls |
| CLI | Commands, flags, non-interactive input and output | A separate provisioning workflow |
| Application | Coordinate use cases and project lifecycle; apply approval policy | Terminal widgets or provider details |
| Planning (implemented) | Resolve current catalog selections, describe unavailable capabilities, validate/order operation dependencies, return structured diagnostics | Filesystem/network effects, provider execution, terminal rendering |
| Core | Resolve capabilities and constraints, form plans, execute predictable operations | Plugin distribution or UI behavior |
| Plugins | Discovery, metadata, lifecycle, and integration operations | Global workflow policy |
| Project | Local manifest and existing-project operations | Remote service state |
| Config | Tool-level settings and loading | Project technology decisions |
| Registry (future) | Catalog metadata retrieval and local catalog management | A required network dependency |
| External boundary | Filesystem, subprocesses, source-control services, cloud APIs | Application-level orchestration |

CLI, TUI, application, catalog, project, and planning packages now exist. The catalog holds technology definitions independently of presentation. The project package owns YAML serialization and local file operations; the application service coordinates saving and read-only planning. `internal/planning` reuses manifest/catalog validators, resolves selected entries, constructs conceptual operations, and applies deterministic dependency ordering without I/O. Broader capability resolution, execution, plugins, config, and registry remain conceptual boundaries. Add packages only when real code needs them. See [project structure](project-structure.md), the [manifest specification](../project-manifest.md), and [generation planning](../generation-planning.md).

## Planned flow

1. Read a local project manifest or collect new-project intent.
2. Gather plugin metadata and configuration requirements without executing side effects.
3. Resolve dependencies, capabilities, and compatibility constraints; report conflicts clearly.
4. Produce a reviewable operation plan with order, inputs, expected outputs, and risk information.
5. Require explicit confirmation before destructive operations. For automation, define a deliberate approval mechanism before implementing execution.
6. Execute operations locally, report progress and errors, and persist enough state for later inspection.

Built-in configuration, manifest loading, ID/category/explicit-conflict validation, and conceptual planning are implemented. `ProjectService.PlanProject` loads a file and delegates to `PlanManifest`; the TUI uses `PlanManifest` for unsaved Review selections. All capabilities are marked unavailable, target paths remain unknown, and every plan has `executable: false`. A valid graph has `status: planned` even with missing-provider warnings. Operations use stable IDs and lexical dependency-first ordering; graph defects produce structured errors, not a partial schedule. The broader plugin capability resolver and executor do not exist yet.

The TUI's optional preview is before saving: `p` from Review opens the plan, Enter/Esc returns without writes, and Enter on Review still saves only the manifest. This keeps configuration persistence separate from plan construction and future application. The existing ADRs already establish these boundaries; no plugin/runtime protocol decision is introduced. Cancellation, retries, rollback semantics, idempotency, risk/approval contracts, and credential handling still need design before execution is built; no guarantee is claimed yet.

## Deployment and extension

No centralized control plane, queue, distributed worker, mandatory gRPC service, or Kubernetes deployment is part of the default runtime. Optional remote or distributed execution can be designed later behind the application/core boundaries without changing the TUI and CLI use cases. Plugin interoperability and distribution formats remain open decisions; [ADR-0005](../adr/0005-plugin-composition.md) supersedes mandatory gRPC from [ADR-0002](../adr/0002-grpc-plugin-contract.md).

## Technology candidates

The first implementation milestone adopted all four libraries. Their tradeoffs remain relevant as the interfaces grow.

| Candidate | Why it fits | Tradeoff and decision gate |
| --- | --- | --- |
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Its model/update/view flow drives screen state and terminal events. | The event model is a framework dependency; keep provisioning behavior outside it. |
| [Bubbles](https://github.com/charmbracelet/bubbles) | Its help component renders context-specific shortcuts. | Add other components only where they reduce code; keep application services independent of component state. |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | It styles and lays out the framed screens. | Test narrow terminal dimensions and limited-color readability. |
| [Cobra](https://github.com/spf13/cobra) | It parses commands and provides help for the CLI. | Keep command handlers thin and share application use cases with the TUI. |

The concrete versions are pinned in `go.mod`. See the [development guide](../development.md) for current commands and shortcuts.
