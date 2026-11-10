# Named components

A technology is a reusable catalog entry such as `nestjs`. A plugin is the declarative generator for that technology. A component is one named use of it, such as `api` or `billing`. The CLI and TUI build an in-memory `project.Specification` with a unique name, category, and technology for each component. Several components may use one plugin definition.

Each component also owns an optional `Configuration` scalar map. The shared
[configuration engine](../plugin-configuration.md) validates values against the
selected plugin schema and resolves defaults independently for every instance.
Canonical component copies clone these maps. Configuration changes affect
rendered content and review freshness while component names continue to define
operation identity and ordering. Configuration is never written as project state.

The planner sorts components by name and gives each a stable `component/<name>` capability. Generated file operations use `file:<plugin-id>:<component-path>` IDs. A validated plugin destination is rebased under the component directory; a leading `backend/` or `frontend/` is removed for official templates. `{{componentName}}` is the instance name. Component names reject separators, traversal, and platform device names.

Planning records ownership and output paths, checks conflicts, and orders dependencies deterministically. Execution reloads plugin content, compares the reviewed plan, and publishes without overwrite. Optional connections select provider/consumer capabilities on explicit component names and contribute consumer-owned integration files. Repeated technologies never choose a target implicitly. See the [connection contract](../component-connections.md).

See [generation planning](../generation-planning.md) and [ADR-0011](../adr/0011-one-time-project-generation.md). The historical [project manifest guide](../project-manifest.md) is no longer the input contract.
