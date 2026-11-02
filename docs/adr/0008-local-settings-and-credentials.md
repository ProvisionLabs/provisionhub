# ADR-0008: SQLite application settings and native credential storage

- Status: Accepted
- Date: 2026-11-02
- Extends: ADR-0003 for optional local application state

## Context

Workspace preferences and Git account metadata must persist independently of
portable project manifests. Authentication must remain optional for generation.

## Decision

Use database/sql with the pure-Go modernc.org/sqlite driver for application
preferences and non-sensitive account metadata. Resolve platform application-data
directories; migrate with ordered transactional SQL and SQLite user_version.
Use a bounded busy timeout, a single pooled connection, and the default rollback
journal rather than assuming WAL is appropriate on every filesystem. Store no
default rows until the user saves a preference. Open storage lazily for settings
operations, not project commands or every startup.

Credentials belong in the OS credential vault behind a small mockable interface,
never in SQLite or manifests. Account rows reference vault entries. Providers
own network protocols; application services coordinate persistence and partial
failure recovery. TUI adapters perform blocking work in Bubble Tea commands.

## Consequences

There is no required database server, network, login, or vault for local project
generation. Settings errors are explicit and do not silently drop writes.
Future domains can add migrations without replacing project manifests. OS vault
availability varies; no plaintext fallback is permitted. GitHub device flow
requires an explicitly configured registered public client ID, not a secret.
