# ADR-0007: Named project components

- Status: Accepted
- Date: 2026-10-30
- Owners: ProvisionHub Core Team
- Extends: [ADR-0005](0005-plugin-composition.md) and [ADR-0006](0006-declarative-local-generation.md)

## Context

`spec.backend`, `spec.frontend`, and `spec.database` each allowed one selection. That prevented projects with two NestJS services or two Next.js frontends, even though one declarative plugin can generate either instance. Array position is unsuitable as identity because reordering a manifest would change operation names and output locations.

## Decision

Add `spec.components` to `provisionhub.dev/v1alpha1`. Each entry has a unique, portable name, a catalog category, and a technology ID. Require at least one component in the new format. Normalize legacy singleton fields into the same internal component model while preserving their plan IDs, output paths, and template naming. Reject mixed formats. Keep `v1alpha1`: the new fields are additive, strict validation disambiguates the formats, and existing files remain valid. A new API version would force migration without changing the meaning of an existing field.

Identify a named component as `component/<name>`. Rebase each validated plugin output under `<name>/`, removing an existing leading category directory for official templates. Resolve one plugin definition by technology and reuse it for all matching instances. Give every planned file component ownership and a stable operation ID. Sort components and connection declarations for deterministic serialization and planning. Preserve the existing planner, graph checks, executor, approval, and no-overwrite rules.

Allow optional `http` connections between distinct existing component names as declarative metadata. They do not generate configuration, clients, ports, or network behavior in this milestone.

## Migration

Legacy:

```yaml
spec:
  backend: {technology: nestjs}
  frontend: {technology: nextjs}
```

Named:

```yaml
spec:
  components:
    - {name: api, type: backend, technology: nestjs}
    - {name: web, type: frontend, technology: nextjs}
```

The legacy project continues to generate `backend/` and `frontend/`. The named project generates `api/` and `web/`. Migrate into a fresh directory; application files are not moved or overwritten automatically.

## Consequences

- A component name is part of its stable identity; renaming it changes output location and operation IDs.
- A catalog technology can appear more than once without duplicating a plugin definition or template files.
- Unknown technologies and unsupported generators still produce explicit diagnostics and non-executable plans.
- Connections are metadata until a later, separately reviewed integration milestone.
- The new TUI mode uses F2 from project information; the legacy simple wizard and CLI flags remain available. Complex CLI projects can use `project init --from <manifest> --path <destination>`.

See the [component architecture](../architecture/components.md) and [manifest specification](../project-manifest.md).
