# ADR-0010: Project plugin locks

- Status: Superseded by [ADR-0011](0011-one-time-project-generation.md)
- Date: 2026-11-04
- Owners: ProvisionHub Core Team

## Context

Installed plugins are global state. Adding another installed version makes an unlocked project ambiguous, and changing a local generator can change its output. SQLite records installations on one machine, so it cannot carry a project's resolution to another machine.

## Decision

The root manifest remains the logical architecture. An explicitly created `.provisionhub/plugins.lock.yaml` records one entry per required generator. `lockVersion: 1` entries contain canonical distribution ID, exact version, source, content SHA-256, and, for official releases, the immutable versioned HTTPS artifact URL and archive SHA-256. The content digest binds the plugin manifest and every declared source file. `official/nestjs` maps to the existing plugin manifest ID `official-nestjs` through installed metadata; local definitions use `local/<manifest ID>`. No installation path or credential is serialized.

An existing lock is authoritative for plan and apply. A conflicting `--plugin-version` requires `project lock --update`. Without a lock, existing installed selection and ambiguity rules apply. `project lock` creates, `--check` validates, and `--update` intentionally regenerates from installed plugins. Global updates do not touch project locks. The same locked plugin can serve repeated component instances; two versions of one generator technology cannot coexist in a plan.

Plan reads only local files and never downloads or writes. Apply reloads and checks the same lock and plugin content before the existing execution preflight; it rejects changes since review. `project lock --restore` explicitly fetches missing official releases through the existing installer after checking the current registry entry against the locked URL and SHA-256. If registry access or that exact release is unavailable, restoration fails; an already verified installation remains usable offline.

Local plugins use a content digest but have no invented remote URL. Transfer the matching plugin directory or portable bundle separately and select it with `--plugins`; a lock alone cannot restore an unmanaged definition. The lock belongs to the logical project beside its manifest, independent of whether generated components later live in one or multiple repositories.

## Consequences

Locking stays local-first and adds no project database, dependency solver, executable plugin hook, or background sync. Official release hashes rely on the registry's existing HTTPS and repository ownership trust boundary; they are not publisher signatures. A registry that removes or alters a locked release prevents restoration but cannot silently change an already installed locked generator.
