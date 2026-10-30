# Generation planning and local execution

## Configuration, planning, and execution

A valid project manifest records desired technologies. It does not imply source generation support. The built-in catalog validates IDs/categories and explicit conflicts; a local plugin supplies a generator for an existing technology ID. Missing generators produce `missing_operation_provider` warnings. `status: planned` means a valid operation graph was constructed. `executable: true` requires every selected capability to be supported and at least one concrete file operation.

`project plan` and the TUI preview read manifests/plugins and construct a deterministic, in-memory plan. They do not create a destination, render templates, install packages, run commands, or write files. `project apply` and the TUI's explicit Review action request local file generation. See the [plugin contract](architecture/plugins.md), [component model](architecture/components.md), and [ADR-0007](adr/0007-named-project-components.md).

## Plan model

The Go types in [internal/planning/plan.go](../internal/planning/plan.go) define the current text/JSON representation. JSON is an initial automation interface, not a stable external plugin protocol.

| Field | Meaning |
| --- | --- |
| `projectName`, `manifestVersion` | Validated desired project identity and schema version |
| `components`, `connections` | Named instances and optional declarative relationships in new-format manifests |
| `technologies` | Selected built-in catalog IDs, names, and categories |
| `operations` | Dependency-ordered conceptual capabilities and concrete files |
| `diagnostics` | Structured severity, code, message, and optional operation/technology/field context |
| `status` | `planned` for a valid graph, `invalid` for validation/planning errors |
| `executable` | True only if every selected capability has a local generator |

Legacy capability IDs remain `backend:<id>`, `frontend:<id>`, and `database:<id>`. A named component uses `component/<name>` for its capability and `file:<plugin-id>:<rebased-destination>` for each file. Operations carry `componentId` ownership. Types remain `generate_backend`, `generate_frontend`, and `integrate_database`. Legacy database integration depends on the selected backend; new-format database components have no implied backend dependency because connections are not operational. Without plugins these operations remain `unavailable`, with empty targets. A loaded backend or frontend generator marks its capability `supported` and adds one `generate_file` operation per declared file, with plugin and source digests, target path, and dependency on its component capability. An unsupported selection still makes the whole plan non-executable.

Equivalent manifests and plugin content yield the same logical plan regardless of file location, YAML field or component array order, or discovery directory order. Operations are topologically ordered with lexical IDs as a tie breaker; dependency and target lists are copied, sorted, and deduplicated. Diagnostics are deterministically sorted. Duplicate/empty operation IDs, missing dependencies, cycles, and conflicting output paths across components or plugins produce an invalid plan without an executable schedule.

Diagnostic codes include `manifest_load_failed`, `invalid_manifest`, `invalid_technology`, `incompatible_technologies`, `invalid_plugin`, `missing_operation_provider`, `invalid_operation`, `invalid_dependency`, and `dependency_cycle`. Missing providers are warnings; invalid inputs/graphs are errors. A valid unsupported plan exits `0` from `project plan`; invalid arguments, file/plugin/manifest problems, or output failures exit `1`. JSON file/schema/planning failures return an invalid plan on stdout and an error on stderr; argument or plugin-directory errors may occur before JSON is available.

## CLI workflow

Use the legacy `project init --name demo --path ./demo --backend nestjs` for a simple project, or author a named-component manifest and run `project init --from ./source.yaml --path ./demo`. The source is validated and copied without overwriting an existing manifest. Both formats use the same plan/apply commands:

```sh
provisionhub project plan --file ./my-app/provisionhub.yaml --plugins ./path/to/plugins
provisionhub project plan --file ./my-app/provisionhub.yaml --plugins ./path/to/plugins --output json
provisionhub project apply --file ./my-app/provisionhub.yaml --plugins ./path/to/plugins
provisionhub project apply --file ./my-app/provisionhub.yaml --plugins ./path/to/plugins --yes
```

`--plugins` points to the directory containing plugin subdirectories, each with `plugin.yaml`. Without it, the existing conceptual plan is returned and apply refuses unavailable capabilities. `project apply` prints the reviewed plan, performs a read-only destination preflight, and requires literal `yes` on an interactive terminal. In a non-interactive environment it fails promptly unless `--yes` is supplied. `--yes` authorizes only the validated generation plan; it does not bypass checks.

At apply time the application reloads the project manifest and plugin files, rebuilds the plan, and compares the complete plan (including plugin manifest and file content digests) with the reviewed plan. It then revalidates the destination and refuses conflicts. Changed inputs require a new review. The executor renders `{{projectName}}` and `{{componentName}}` or copies bytes; it never invokes npm or other commands. The destination is the directory containing the manifest. Each named component writes below its own validated directory.

## TUI workflow

After project information, the TUI opens the component list. `a` adds a component, `e` edits the selected component, `d` removes it, and Enter reviews the list. Names must be unique. The TUI uses the same read-only plan preview and explicit generation action for all named components; legacy manifests remain supported by CLI plan/apply.

On Create Project's Review step, `p` previews the current unsaved configuration without writing. Enter/Esc leaves the preview. When the plan is executable, Review shows **Generate Project** and Enter explicitly authorizes manifest creation followed by file generation. The application checks that the plan is still current before saving and repeats checks before publishing. Otherwise Review shows **Save Manifest Only** and Enter saves just `provisionhub.yaml`. If generation fails after the manifest is saved, the result screen keeps it and reports the error; `p` opens a fresh plan, `g` performs read-only preflight, and `y` confirms a retry. Generated files appear only after successful application. Set `PROVISIONHUB_PLUGINS` to a local plugin directory before launching the TUI or place `plugins` beside the executable. Up/Down or `j`/`k` scrolls the preview; `q` quits outside text input. Keyboard quit is ignored while a generation command is in progress to avoid interrupting publication.

## Filesystem guarantees and limits

- Plugin paths are relative, clean slash paths; traversal, absolute paths, symlinked sources/outputs, duplicate or overlapping outputs, and writes to `provisionhub.yaml` are rejected.
- The executor resolves symlinked ancestors (including standard macOS paths such as `/var`), opens the chosen directory with `os.Root`, checks that it contains a regular manifest, and rejects a symlink at the destination itself or in generated output parents. Root-relative operations cannot escape the opened root on Windows, Linux, and macOS.
- All content is rendered and staged in a private directory under the destination before publication. Each file is published with a no-overwrite hard link. Pre-existing targets stop application before staging; conflicts during publication also fail without replacing them.
- On a later failure, the executor attempts to remove published files that still refer to its staged content and to remove empty directories it created. Multi-file generation is not fully atomic. A crash or concurrent filesystem modification may leave partial files or directories; cleanup failures are returned. Check the destination before retrying.
- Hard links must be supported by the destination filesystem. There is no overwriting fallback, shell execution, dependency installation, network call, or infrastructure operation. This is not a general sandbox or rollback engine.

## Official plugin integration

The NestJS and Next.js official plugins use the same declarative contract. Their test suite generated two NestJS and two Next.js components in separate directories and ran `npm install` and `npm run build` for all four on Windows. The legacy tests also built and served each application over HTTP. Repeatable integration tests live in the [official plugin repository](https://github.com/ProvisionLabs/provisionhub-official-plugins). Core Go CI validates on Windows, Linux, and macOS; cross-OS Node runtime results are not claimed.
