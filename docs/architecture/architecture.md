# Architecture

## Status

This describes the **target architecture** and identifies what exists today. The repository has a CLI, TUI project configuration wizard, built-in catalog, versioned project manifest, shared services, deterministic planning, local declarative plugin loading, and separately approved file generation. Config and registry behavior remain future work. The official NestJS plugin repository was unavailable during this implementation. The earlier control-plane, queue, and worker diagram was retired with the local-first decision in [ADR-0003](../adr/0003-local-first-provisioning.md).

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

One local process is the default. TUI and CLI are adapters into the same application use cases. Application services coordinate project state, composition, planning, approval, and execution; they do not depend on terminal rendering or command parsing. The core owns deterministic composition rules and planned operation semantics. The current plugin system discovers declarative local file generators under an explicit contract; it does not invoke plugin code. Provider-specific behavior belongs in plugins or adapters at the external boundary.

## Responsibilities

| Boundary | Responsibility | Must not own |
| --- | --- | --- |
| TUI | Screens, navigation, input, plan and progress display | Composition rules or provider calls |
| CLI | Commands, flags, non-interactive input and output | A separate provisioning workflow |
| Application | Coordinate use cases and project lifecycle; apply approval policy | Terminal widgets or provider details |
| Planning (implemented) | Resolve selected technologies and plugin capabilities, validate/order conceptual and concrete operations, return structured diagnostics | Filesystem/network effects, provider execution, terminal rendering |
| Core | Resolve capabilities and constraints, form plans, execute predictable operations | Plugin distribution or UI behavior |
| Plugins | Discovery, metadata, lifecycle, and integration operations | Global workflow policy |
| Project | Local manifest and existing-project operations | Remote service state |
| Config | Tool-level settings and loading | Project technology decisions |
| Registry (future) | Catalog metadata retrieval and local catalog management | A required network dependency |
| External boundary | Filesystem, subprocesses, source-control services, cloud APIs | Application-level orchestration |

CLI, TUI, application, catalog, project, planning, plugins, and execution packages now exist. The catalog holds stable technology definitions. The project package owns manifest YAML and persistence; application services coordinate saving, planning, preflight, and explicit apply. `internal/planning` remains pure, `internal/plugins` validates local declarative generators, and `internal/execution` publishes reviewed files without overwrite. Broader composition, plugin installation, config, and registry remain future boundaries. See [project structure](project-structure.md), the [manifest specification](../project-manifest.md), and [generation planning](../generation-planning.md).

## Planned flow

1. Read a local project manifest or collect new-project intent.
2. Gather plugin metadata and configuration requirements without executing side effects.
3. Resolve dependencies, capabilities, and compatibility constraints; report conflicts clearly.
4. Produce a reviewable operation plan with order, inputs, expected outputs, and risk information.
5. Require explicit approval before file generation. The TUI's labeled **Generate Project** action is the approval for a new project; CLI apply requires literal terminal approval or `--yes` for automation.
6. Execute operations locally, report progress and errors, and persist enough state for later inspection.

Built-in configuration, manifest loading, ID/category/explicit-conflict validation, conceptual and concrete planning, and local file execution are implemented. `ProjectService.PlanProject` loads a file and delegates to `PlanManifest`; the TUI uses `PlanManifest` for unsaved Review selections. A local plugin can support a backend or frontend capability with concrete target paths. Unprovided capabilities remain unavailable and make a plan non-executable. A valid graph has `status: planned` even with missing-provider warnings. Operations use stable IDs and lexical dependency-first ordering; graph defects produce structured errors, not a partial schedule.

The TUI's optional `p` preview opens the read-only plan from Review; Enter/Esc returns without writes. Review labels its action **Generate Project** when all selected capabilities are supported. Enter explicitly authorizes saving the manifest and applying that reviewed plan in one command. Unsupported selections show **Save Manifest Only**. If application fails after manifest publication, the result screen keeps the manifest and offers a fresh preview and confirmed retry. The CLI still separates plan from apply and requires `yes` or `--yes`. [ADR-0006](../adr/0006-declarative-local-generation.md) records the initial file-generation contract and its limited cleanup guarantees. Broader operation risk, cancellation, idempotency, credential handling, and infrastructure approval need separate designs.

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
