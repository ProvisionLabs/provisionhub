# Named project components

A **technology** is a reusable catalog definition such as `nestjs`. A **plugin** is the declarative generator for that technology. A **component** is one named use of it, such as `api` or `billing`. A **connection** is a declared relationship between two component names; it currently has no generation effect.

Both manifest formats normalize to `Manifest.CanonicalComponents()` in `internal/project`. New-format components are sorted by name. Legacy selections normalize to `backend`, `frontend`, and `database` with compatibility metadata. The planner uses one plugin definition per technology and creates a separate capability and file operations per component. The plugin manifest is not duplicated.

| Item | Named format | Legacy format |
| --- | --- | --- |
| Component identity | `component/<name>` | Internal normalized component; old plan IDs retained |
| Capability operation ID | `component/api` | `backend:nestjs` |
| File operation ID | `file:<plugin-id>:api/package.json` | Existing plugin destination ID |
| Output path | `<name>/<plugin-relative-path>` | Original plugin destination |
| `{{componentName}}` | Component name | Project name |

For named components, the loader's validated plugin destination is rebased under the component name. A leading category directory (`backend/` or `frontend/`) is removed so the official templates generate `api/package.json`, `billing/package.json`, and `web/package.json`. Other relative plugin destinations are simply placed below the component directory. The same plugin sources and content digests are used for every instance. Component names cannot contain separators, traversal, absolute prefixes, or platform device names.

Planning records component ownership, technology, plugin, and output paths. Lexical operation ordering and stable names make plans independent of the YAML array order. The existing graph validator rejects duplicate/overlapping paths and dependency defects. Execution rechecks the reviewed plan, resolves each file through its component and plugin definition, renders only validated variables, and uses the existing root-confined, no-overwrite publication path. Planning remains read only, and applying still requires explicit approval.

Connections are sorted and included in the plan as metadata. The initial `http` declaration validates names and duplicate/self relationships. It does not imply a working API connection. Future composition can use these identities to generate integration operations after a separate design decision.

See the [manifest schema](../project-manifest.md), [planning/execution guide](../generation-planning.md), and [ADR-0007](../adr/0007-named-project-components.md).
