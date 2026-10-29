# Generation planning and local execution

## Configuration, planning, and execution

A valid project manifest records desired technologies. It does not imply source generation support. The built-in catalog validates IDs/categories and explicit conflicts; a local plugin supplies a generator for an existing technology ID. Missing generators produce `missing_operation_provider` warnings. `status: planned` means a valid operation graph was constructed. `executable: true` requires every selected capability to be supported and at least one concrete file operation.

`project plan` and the TUI preview read manifests/plugins and construct a deterministic, in-memory plan. They do not create a destination, render templates, install packages, run commands, or write files. `project apply` and the TUI `g` → `y` path separately request local file generation. See the [plugin contract](architecture/plugins.md) and [ADR-0006](adr/0006-declarative-local-generation.md).

## Plan model

The Go types in [internal/planning/plan.go](../internal/planning/plan.go) define the current text/JSON representation. JSON is an initial automation interface, not a stable external plugin protocol.

| Field | Meaning |
| --- | --- |
| `projectName`, `manifestVersion` | Validated desired project identity and schema version |
| `technologies` | Selected built-in catalog IDs, names, and categories |
| `operations` | Dependency-ordered conceptual capabilities and concrete files |
| `diagnostics` | Structured severity, code, message, and optional operation/technology/field context |
| `status` | `planned` for a valid graph, `invalid` for validation/planning errors |
| `executable` | True only if every selected capability has a local generator |

Conceptual capability IDs remain `backend:<id>`, `frontend:<id>`, and `database:<id>`. Their types are `generate_backend`, `generate_frontend`, and `integrate_database`. Database integration depends on the selected backend capability. Without plugins these operations remain `unavailable`, with empty targets. A loaded backend or frontend generator marks its capability `supported` and adds one `generate_file` operation per declared file, with a stable `file:<plugin-id>:<destination>` ID, plugin ID/manifest digest, source/type/content digest, target path, `supported` status, and dependency on its technology capability. The capability lists all its target paths. An unsupported selection still makes the whole plan non-executable.

Equivalent manifests and plugin content yield the same logical plan regardless of file location, YAML field order in the project manifest, or discovery directory order. Operations are topologically ordered with lexical IDs as a tie breaker; dependency and target lists are copied, sorted, and deduplicated. Diagnostics are deterministically sorted. Duplicate/empty operation IDs, missing dependencies, cycles, and conflicting output paths across plugins produce an invalid plan without an executable schedule.

Diagnostic codes include `manifest_load_failed`, `invalid_manifest`, `invalid_technology`, `incompatible_technologies`, `invalid_plugin`, `missing_operation_provider`, `invalid_operation`, `invalid_dependency`, and `dependency_cycle`. Missing providers are warnings; invalid inputs/graphs are errors. A valid unsupported plan exits `0` from `project plan`; invalid arguments, file/plugin/manifest problems, or output failures exit `1`. JSON file/schema/planning failures return an invalid plan on stdout and an error on stderr; argument or plugin-directory errors may occur before JSON is available.

## CLI workflow

```sh
provisionhub project plan --file ./my-app/provisionhub.yaml --plugins ./path/to/plugins
provisionhub project plan --file ./my-app/provisionhub.yaml --plugins ./path/to/plugins --output json
provisionhub project apply --file ./my-app/provisionhub.yaml --plugins ./path/to/plugins
provisionhub project apply --file ./my-app/provisionhub.yaml --plugins ./path/to/plugins --yes
```

`--plugins` points to the directory containing plugin subdirectories, each with `plugin.yaml`. Without it, the existing conceptual plan is returned and apply refuses unavailable capabilities. `project apply` prints the reviewed plan, performs a read-only destination preflight, and requires literal `yes` on an interactive terminal. In a non-interactive environment it fails promptly unless `--yes` is supplied. `--yes` authorizes only the validated generation plan; it does not bypass checks.

At apply time the application reloads the project manifest and plugin files, rebuilds the plan, and compares the complete plan (including plugin manifest and file content digests) with the reviewed plan. It then revalidates the destination and refuses conflicts. Changed inputs require a new review. The executor renders `{{projectName}}` or copies bytes; it never invokes npm or other commands. The destination is the directory containing the manifest.

## TUI workflow

On Create Project's Review step, `p` previews the current unsaved configuration. Enter/Esc returns without writing; Enter on Review saves only `provisionhub.yaml`. After save, `p` on the result screen previews the saved project. If all capabilities are available, `g` performs a read-only preflight and opens a separate confirmation. `y` applies; `n` or Esc cancels. Errors appear on the result screen, which can reopen the preview. Generated files appear only after successful application. Set `PROVISIONHUB_PLUGINS` to a local plugin directory before launching the TUI. Up/Down or `j`/`k` scrolls the preview; `q` quits outside text input. Keyboard quit is ignored while a generation command is in progress to avoid interrupting publication.

## Filesystem guarantees and limits

- Plugin paths are relative, clean slash paths; traversal, absolute paths, symlinked sources/outputs, duplicate or overlapping outputs, and writes to `provisionhub.yaml` are rejected.
- The executor resolves symlinked ancestors (including standard macOS paths such as `/var`), opens the chosen directory with `os.Root`, checks that it contains a regular manifest, and rejects a symlink at the destination itself or in generated output parents. Root-relative operations cannot escape the opened root on Windows, Linux, and macOS.
- All content is rendered and staged in a private directory under the destination before publication. Each file is published with a no-overwrite hard link. Pre-existing targets stop application before staging; conflicts during publication also fail without replacing them.
- On a later failure, the executor attempts to remove published files that still refer to its staged content and to remove empty directories it created. Multi-file generation is not fully atomic. A crash or concurrent filesystem modification may leave partial files or directories; cleanup failures are returned. Check the destination before retrying.
- Hard links must be supported by the destination filesystem. There is no overwriting fallback, shell execution, dependency installation, network call, or infrastructure operation. This is not a general sandbox or rollback engine.

## Official NestJS integration

The official plugin at `provisionhub-official-plugins/plugins/nestjs` uses this existing contract to generate ten files under `backend/`. Point `--plugins` to that repository's `plugins` directory and select only `nestjs`. Local Windows validation covered discovery, deterministic supported operations, non-interactive approval refusal, approved generation, no-overwrite conflicts, npm installation/build, and HTTP 200 from the started application. Repeatable integration tests and detailed results live in the plugin repository. No core runtime changes were required. Linux/macOS runtime validation is not claimed. Broader composition should follow concrete integration needs.
