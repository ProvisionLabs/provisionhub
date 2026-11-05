# ADR-0011: One-time project generation

- Status: Accepted
- Date: 2026-11-05
- Owners: ProvisionHub Core Team
- Supersedes: [ADR-0010](0010-project-plugin-locking.md) and the project persistence portion of [ADR-0007](0007-named-project-components.md)

## Context

ProvisionHub previously saved project manifests, optional plugin locks, and local SQLite references to generated projects. The product now needs to initialize source once and leave the result independent of ProvisionHub. Planning, execution, and plugin management already have separate boundaries, so the project lifecycle can be simplified without replacing those engines or the plugin contract.

## Decision

CLI and TUI construct a validated in-memory project specification with named components. The Planning Engine uses that value and locally validated plugin definitions. After explicit review and approval, the Execution Engine reloads plugin content, checks freshness and output safety, and generates source files. Neither workflow writes a project YAML file, project plugin lockfile, or saved-project registration.

Explicitly supplied local plugin definitions can generate without SQLite. Managed plugins still use SQLite installation metadata and content integrity checks. Explicit version selection resolves ambiguity. Plugin manifests and package formats stay unchanged. The versioned SQLite migration that created `saved_projects` remains in the migration history; existing rows are left untouched and no startup migration drops them.

## Rationale

The generated project is ordinary source code and no longer needs ProvisionHub-specific metadata. A single in-memory specification avoids a write-then-read step before generation. Keeping managed plugin metadata preserves installation and integrity guarantees, while leaving old SQLite rows and files avoids destructive upgrades.

## Trade-offs

- Generated projects cannot be reopened or reconciled through ProvisionHub.
- Repeating CLI generation requires repeating its component flags. Existing targets remain protected by the executor's no-overwrite rules.
- Local plugins do not have a portable project lock; users must select an exact managed version when multiple versions are installed.
- Historical `saved_projects` tables remain in databases until a separately designed migration or manual archival policy.

## Consequences

Project YAML loading/saving, lockfile services, saved-project services, and My Projects navigation are retired. Settings, Git provider metadata, Plugin Manager, plugin manifests, registry installation security, deterministic planning, and guarded execution remain. Historical project-management documentation is marked as such pending cleanup.

## Rejected alternatives

1. Keep writing a manifest only to feed the existing planner: this would retain metadata the generated project does not need.
2. Drop the saved-project table on startup: this would delete user data for no generation benefit.
3. Replace the Planning or Execution Engine: the existing engines already provide ordering, freshness checks, and filesystem safeguards.
