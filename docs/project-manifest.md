# Project manifest: `provisionhub.yaml`

The manifest records desired project composition. Its directory is the generation destination; the destination path is not stored in YAML. The current API version remains `provisionhub.dev/v1alpha1` because named components are an additive format with explicit legacy compatibility. See [ADR-0007](adr/0007-named-project-components.md).

## Named components

```yaml
apiVersion: provisionhub.dev/v1alpha1
kind: Project
metadata:
  name: commerce-platform
spec:
  components:
    - name: api
      type: backend
      technology: nestjs
    - name: billing
      type: backend
      technology: nestjs
    - name: web
      type: frontend
      technology: nextjs
    - name: admin
      type: frontend
      technology: nextjs
  connections:
    - from: web
      to: api
      type: http
```

`components` requires at least one entry and permits any number of each category. Each `name` must be unique, 1–63 lowercase ASCII letters, digits, or hyphens, starting and ending with a letter or digit. Windows device names such as `con` and `lpt1` are rejected. `type` is `backend`, `frontend`, or `database`; `technology` must exist in that category in the built-in catalog. The current IDs are:

| Type | Technology IDs |
| --- | --- |
| `backend` | `go`, `nestjs`, `spring-boot` |
| `frontend` | `nextjs`, `react` |
| `database` | `mysql`, `postgresql` |

For example, the catalog ID is `spring-boot`, not `springboot`. A valid technology selection does not guarantee a generation plugin. Current official generators cover `nestjs` and `nextjs`; selecting another technology leaves the plan non-executable. The component name gives a stable planning identity (`component/api`) and output directory (`api/`). YAML list position does not affect either.

`connections` is optional declarative metadata. The only accepted type is `http`; endpoints must name distinct existing components, and duplicate declarations are rejected. Connections appear in the canonical plan but do not configure clients, ports, networking, or environment variables. Neither components nor connections currently accept extra configuration fields. Unknown YAML fields and multiple documents are rejected.

## Legacy manifests

Existing `v1alpha1` manifests remain valid:

```yaml
apiVersion: provisionhub.dev/v1alpha1
kind: Project
metadata:
  name: my-application
spec:
  backend:
    technology: nestjs
  frontend:
    technology: nextjs
```

Legacy `backend` is required; `frontend` and `database` are optional. The parser normalizes them internally to components named `backend`, `frontend`, and `database`. Legacy operation IDs, plugin output paths (`backend/` and `frontend/`), and project-name template rendering are preserved. `spec.components` or `spec.connections` cannot appear with legacy fields, even as empty or null values.

To migrate, replace the singleton fields with named entries. Moving `backend` to `api` changes its output directory from `backend/` to `api/`; generate into a fresh destination rather than applying over an existing project. A frontend-only *named-component* project is valid, although legacy format still requires a backend.

## Validation and persistence

The parser rejects malformed YAML, unknown fields, unsupported version/kind, unsafe or duplicate names, invalid connection references, and mixed formats. The application validates catalog IDs/categories and explicit catalog incompatibilities. Serialization sorts components and connections by stable names/fields. `project init` and the TUI publish a complete manifest without overwriting an existing one. No credentials or secrets belong in the current schema.

See [component architecture](architecture/components.md), [planning and execution](generation-planning.md), and [CLI/TUI usage](development.md).
