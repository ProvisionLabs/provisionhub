# Project manifest: `provisionhub.yaml`

ProvisionHub saves the user's **desired project configuration** in a versioned YAML file. It does not record provisioned state or generate application code. The destination directory is selected by the user and is not stored in the manifest.

## Current schema

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
    database:
        technology: postgresql
```

| Field | Rule |
| --- | --- |
| `apiVersion` | Required; currently only `provisionhub.dev/v1alpha1` |
| `kind` | Required; must be `Project` |
| `metadata.name` | Required, 1–63 characters; lowercase ASCII letters, digits, or hyphens; must start and end with a letter or digit |
| `spec.backend.technology` | Required backend ID |
| `spec.frontend.technology` | Optional frontend ID; omit `frontend` for None |
| `spec.database.technology` | Optional database ID; omit `database` for None |

The current built-in catalog contains:

| Category | IDs |
| --- | --- |
| Backend | `go`, `nestjs`, `spring-boot` |
| Frontend | `nextjs`, `react` |
| Database | `mysql`, `postgresql` |

`None` is an absence of a selection, not a technology ID in the manifest. The CLI accepts omitted optional flags or the value `none`. Technology IDs must exist in the catalog and belong to the selected category. Catalog entries can name explicitly incompatible technology IDs, but no built-in entries currently do. A valid configuration is **not** a promise that source generation for the combination exists.

## Validation and evolution

The YAML reader rejects malformed YAML, unknown fields, multiple documents, missing required fields, unsupported API versions, and unsupported kinds. The application layer additionally validates technology IDs and category membership when creating or loading a manifest. It does not accept arbitrary unrecognized technologies yet.

The `v1alpha1` version marks the current schema as subject to change. Future schema changes should use an explicit new `apiVersion` and a documented migration path; they should not silently reinterpret existing files. Future plugins may contribute catalog entries; plugin loading and a generic resolver are not implemented.

Do not put credentials or secrets in the manifest. No credential fields exist in this schema.

## Local file behavior

`project init` and the TUI create the destination directory if needed, write a complete temporary file there, and publish it without replacing an existing `provisionhub.yaml`. A conflict or filesystem error is reported to the caller. Existing manifests can be parsed and validated by the Go application service; the **Open Existing Project** TUI screen remains a placeholder.

## Read-only planning

`provisionhub project plan --file <manifest> [--output json]` reads this same schema without changing the file or inspecting/creating a generation destination. The TUI can preview unsaved Review selections through the same application planning service. Planning reuses structural, ID/category, and explicit compatibility validation; it does not relax manifest rules or require a schema change.

A valid manifest can produce a `planned` conceptual plan while every operation remains `unavailable` and `executable` is `false`. There are no generators or integration providers yet. Unknown IDs, wrong categories, explicit conflicts, and invalid manifests produce error diagnostics rather than supported operations. A successfully constructed unsupported plan exits `0`; input/loading/validation/planning/output failures exit `1`. See [generation planning](generation-planning.md) for the domain and JSON fields, diagnostic codes, and TUI save policy.

See the [development guide](development.md) for commands and the [architecture](architecture/architecture.md) for package boundaries.
