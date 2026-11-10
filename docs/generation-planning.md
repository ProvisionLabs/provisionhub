# Generation planning and local execution

The CLI and TUI build an in-memory `project.Specification` from the selected name, destination, and named components. The application validates component names, technologies, categories, duplicates, and catalog conflicts. The Planning Engine then combines that specification with validated local plugin definitions. It produces dependency-ordered capabilities and concrete file operations with diagnostics. Planning resolves plugin configuration and renders templates in memory to reject missing references and compute rendered content digests. It does not create directories, install packages, run commands, or write files.

A plan is `executable` only when every selected capability has a generator and there is at least one concrete file operation. A valid plan can still be `planned` but non-executable when a plugin is missing. Named component operations use `component/<name>` and files are rebased under `<name>/`, allowing repeated technologies. `project plan --output json` exposes the same plan with `projectName`, `specificationVersion`, `components`, `connections`, `technologies`, `operations`, `diagnostics`, `status`, and `executable` fields. Optional connections are resolved against plugin provider/consumer capabilities before operations are contributed. Plans show source/target names, connection configuration, consumed metadata, integration outputs, and dependencies.

Generation requires explicit approval in the TUI or CLI. The application reloads plugin content and compares the fresh plan with the reviewed plan. The Execution Engine repeats this check, verifies the destination and target paths, renders files, stages them under the output root, and publishes through no-overwrite hard links. It rejects path traversal, symlinked output parents, conflicting targets, and a symlink at the destination itself. It never creates `provisionhub.yaml`, a project lockfile, or a saved-project registration.

Managed plugins are selected from installed SQLite metadata and their content is revalidated before planning and apply. Multiple installed versions require explicit `--plugin-version` selection or a TUI session selection. Explicit local plugin directories can generate without a SQLite database when they cover every requested generator; otherwise the service also checks managed installations. Optional plugin configuration declarations extend the manifest; package integrity checks and registry installation security remain intact. No registry request is needed for planning or generation.

Multi-file publication is not fully atomic after a process crash or hostile concurrent filesystem change. On an ordinary error, the executor attempts to remove files it published and empty directories it created; cleanup failures are reported. Hard-link support is required. Generated files are never overwritten. No dependency install, application startup, shell plugin, network provisioning, or infrastructure action runs during generation.

See the [plugin contract](architecture/plugins.md), [component model](architecture/components.md), and [ADR-0011](adr/0011-one-time-project-generation.md).

Connections add `connect_components` operations with dependencies on the source
and target capabilities and base files; integration file operations depend on
their connection. Source capabilities depend on their selected targets.
Unsupported/ambiguous pairs, occupied consumer slots, missing values, and cycles
make the plan invalid before any operations are contributed. The complete
connection data and rendered hashes participate in review freshness. Integration
files use the existing executor and its no-overwrite/cleanup semantics.

Generation does not probe or establish network connectivity. For the official
HTTP integration, the backend port comes from resolved configuration, while the
consumer-reachable host is explicit. `API_ORIGIN` is private server-side runtime
configuration. Later deployment or `PORT` overrides require updating application
configuration. See [connections](component-connections.md) for examples and the
build-time, generation-time, and runtime distinction.
