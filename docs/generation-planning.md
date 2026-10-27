# Generation planning

## Configuration, planning, and execution

ProvisionHub can validate a desired project configuration and construct a deterministic, reviewable generation plan. It cannot generate or execute the described capabilities yet.

- **Valid configuration:** the manifest schema, technology IDs/categories, and explicitly declared compatibility constraints pass existing validation.
- **Constructed plan:** `status: planned` means a structurally valid conceptual operation graph was built.
- **Executable plan:** `executable` is always `false` in this milestone. No generator, database integration provider, or execution engine is implemented. Catalog membership is not generation support.

Planning never creates a destination, modifies a manifest/project file, downloads templates, contacts providers, or starts commands. File-based planning only reads the supplied manifest. Plans are returned in memory or written to command output, not persisted as execution state. Existing explicit manifest-saving actions remain separate.

## Shared flow and boundaries

```text
CLI file -> project.Load (strict YAML/schema validation) --+
                                                         |
TUI Review -> ProjectService.ValidateInput (no writes) ----+-> ProjectService.PlanManifest
                                                              |
                                                planning.Build(manifest, catalog)
                                                              |
                                                structured Plan + diagnostics
                                                              |
                                                       CLI / TUI preview
```

`ProjectService.PlanProject(path)` coordinates file loading and calls the same `PlanManifest` used by the TUI. `internal/planning` reuses `Manifest.Validate`, `Catalog.ValidateSelection`, and `Catalog.ValidateCombination`; it does not import CLI/TUI libraries, access the filesystem, or execute operations. File/schema errors and planning errors return both structured error diagnostics and a Go error at the application boundary. Missing providers produce warnings and a successfully constructed, non-executable plan.

There is no provider-registration API or plugin SDK. The plain operation/diagnostic model and dependency-ordering helper leave room for future contributed definitions, but loading, runtime protocols, trust, permissions, and execution contracts remain deferred under [ADR-0005](adr/0005-plugin-composition.md).

## Domain and JSON model

The Go types in [internal/planning/plan.go](../internal/planning/plan.go) are also the current JSON representation. The JSON shape is an initial internal automation interface, not a stable plugin protocol.

| Plan field | Meaning |
| --- | --- |
| `projectName` | Manifest project name |
| `manifestVersion` | Input `apiVersion`, currently `provisionhub.dev/v1alpha1` |
| `technologies` | Resolved ID, display name, and category for selected entries; no records for omitted optional selections |
| `operations` | Conceptual capabilities in dependency order |
| `diagnostics` | Structured severity, code, message, and optional operation/technology/manifest-field context |
| `status` | `planned` for a constructed plan; `invalid` when loading, validation, or graph checks fail |
| `executable` | Always `false`; not inferred from valid configuration or graph structure |

Each operation has `id`, `description`, `type`, optional `technology`, `targetPaths`, `dependencies`, and `status`. Collection fields are JSON arrays, including when empty. Target paths are currently empty because no supported generation definition establishes them. Descriptions contain no shell commands or presentation markup.

The current manifest categories imply only these high-level capabilities:

| Selection | Stable operation ID | Type | Dependencies |
| --- | --- | --- | --- |
| Backend | `backend:<technology-id>` | `generate_backend` | None |
| Optional frontend | `frontend:<technology-id>` | `generate_frontend` | None |
| Optional database | `database:<technology-id>` | `integrate_database` | Selected backend operation |

All operation statuses are `unavailable`. The database dependency expresses that backend integration requires the backend capability; it specifies no framework integration steps. No other ordering constraint or precise output structure is invented. Paths, commands, input/output contracts, destructive-operation risks, and approval/execution semantics need concrete definitions before any capability can become supported.

## Determinism and graph validation

- Equivalent manifests, independent of YAML field order/comments or file location, produce equivalent logical plans and identical successful JSON output for the same catalog.
- Selections are resolved in manifest category order: backend, frontend, database. Omitted optional selections produce no capability or missing-provider warning.
- IDs derive only from category and technology ID. There are no timestamps, random IDs, or destination paths in a logical plan.
- Operations are topologically ordered, choosing the lexically smallest available ID at each step. For a full current selection this yields backend, database, frontend. Dependencies and target-path lists are copied, sorted, and deduplicated; input definitions are not mutated.
- Diagnostics are sorted lexically by severity, code, operation ID, technology, field, and message. Map iteration does not define output order.
- Empty/duplicate operation IDs, missing dependency references, and dependency cycles produce error diagnostics and no partial operation schedule. A cycle message lists operations that cannot be ordered, which can include downstream blocked operations, not only cycle members.

This is small operation-graph ordering, not a general dependency/capability resolver or workflow engine. Built-in catalog entries still declare no technology incompatibilities. Failure messages involving file I/O can include the supplied path and OS-specific error wording; byte-identical failures across operating systems are not promised.

## Diagnostics

| Code | Severity | Meaning |
| --- | --- | --- |
| `manifest_load_failed` | Error | The requested file could not be read |
| `invalid_manifest` | Error | Invalid YAML, schema, version/kind, name, or required fields |
| `invalid_technology` | Error | Unknown ID or incorrect catalog category; identifies the manifest field |
| `incompatible_technologies` | Error | Selected entries explicitly declare a conflict |
| `missing_operation_provider` | Warning | No generator/integration provider implements the selected capability |
| `invalid_operation` | Error | An operation ID is empty or duplicated |
| `invalid_dependency` | Error | A dependency references an unknown operation |
| `dependency_cycle` | Error | Dependencies cannot be ordered because of a cycle |

The model also supports informational severity, though no informational diagnostic is currently emitted. Fixed built-in conceptual operations have valid graphs; graph errors are guarded and tested for future definitions. Invalid configurations are rejected before capabilities are constructed; unknown selections may leave a partial list of successfully resolved technologies, but never an executable schedule.

## CLI and exit codes

```sh
provisionhub project plan --file ./my-application/provisionhub.yaml
provisionhub project plan --file ./my-application/provisionhub.yaml --output json
```

`--file` is required. `--output` is `text` by default; only `text` and `json` are accepted. The command does not require a terminal, prompts, or stdin. Text output includes selections, capabilities, dependencies, diagnostics, planning status, and execution unavailability.

| Process exit code | Meaning |
| --- | --- |
| `0` | A plan was constructed successfully, even when all providers and execution are unavailable; help also succeeds |
| `1` | Invalid flags/arguments/output format, unreadable or invalid manifest, invalid technology/conflict, graph failure, or output write failure |

A valid unsupported selection is not an invalid manifest. JSON planning/loading failures emit an `invalid` plan with error diagnostics to stdout and return a failure; the executable also reports the error on stderr. Argument/flag/format failures can occur before a plan exists and do not promise JSON output. A downstream output failure may leave partial output. Use the executable, not the Go tool's `go run` wrapper, when checking process exit semantics.

## TUI preview and save policy

From Create Project's **Review configuration** screen, press `p` to preview the unsaved configuration. Up/Down or `j`/`k` scrolls through the plan, including diagnostics. Enter, Esc, or `p` returns to Review. From Review, Esc returns to technology configuration with selections preserved; opening a preview again rebuilds it from the current choices.

The preview uses terminal-cell-aware wrapping, clamps scrolling after resizing, and keeps navigation available in small terminals. At extremely small sizes only part of a line/control hint fits; scrolling and Esc still work. Ctrl+C and `q` keep their existing quit behavior outside text fields. `p`, `q`, `j`, and `k` remain ordinary text while editing project information.

Preview is optional and before saving because this preserves the six-step wizard and existing explicit action: **Enter on Review saves only `provisionhub.yaml`; Enter inside preview never saves.** The preview makes no files, creates no destination, and never applies operations. Save results still appear only after persistence succeeds, with the existing no-overwrite guarantee. The Open Existing Project screen remains a placeholder; use the CLI to preview a saved manifest.

## Next milestone

The next recommended increment is one supported local project-generation vertical slice, starting with concrete operation definitions and safe, separately authorized application of reviewed output. It must establish truthful support/target/risk metadata, no-overwrite and path-safety behavior, approval, and useful failure handling before writes become available. Plugin runtime, marketplace access, infrastructure provisioning, and distributed execution are separate future work.

See the [architecture](architecture/architecture.md), [manifest specification](project-manifest.md), [development guide](development.md), and [development context](development-context.md).
