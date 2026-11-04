# ADR-0009: Static plugin registry and immutable declarative packages

- Status: Accepted
- Date: 2026-11-03
- Owners: ProvisionHub Core Team

## Context

Official generators need installation without Git, accounts, or a required server.
The existing declarative contract and local loader already validate generation
inputs. Distribution must not introduce a scripting runtime.

## Decision

Use a schema-versioned JSON index hosted in the official plugins repository and
HTTPS GitHub Release artifacts. Distribution IDs are namespace/name; manifest IDs
remain unchanged for compatibility (official/nestjs maps to official-nestjs).
Technology IDs identify capabilities, not packages. Release versions are stable
semantic versions; v1 supports minimumCoreVersion, with future range support
requiring an explicit schema evolution. No dependency solver is introduced.

Packages are deterministic gzip-compressed tar archives containing plugin.yaml
and only manifest-declared source files at their original relative paths.
Index checksums verify downloaded bytes. HTTPS and repository/release ownership
are the authenticity boundary; hashes are not publisher signatures. Release tags
and artifacts must never be replaced after publication.

Managed packages live beside the existing SQLite database, isolated by namespace,
name and version. Metadata activation follows complete filesystem staging. An
interrupted unreferenced directory is never automatically trusted or overwritten.
Local paths and environment overrides remain supported. Conflicts are errors,
not arbitrary precedence. The loader never fetches registry data.

Keep old versions on update. Existing project manifests have no package locks;
multiple installed versions must require explicit selection rather than silently
changing an existing project's generator. Future portable locks record package ID,
version, source and checksum, never installation paths.

## Rationale

Static hosting provides public, cacheable distribution with no proprietary service
and keeps generation offline. Additional sources can implement the same index
contract later without changing installation or presentation services.

## Trade-offs

- Repository and GitHub hosting availability control online discovery.
- Checksums do not establish independent publisher authenticity.
- SQLite and filesystem operations cannot commit atomically; incomplete and
  orphaned installations require explicit diagnostics and recovery.
- Project-specific locking remains a separate milestone.

## Consequences

No marketplace backend, accounts, publishing platform, downloaded executable,
script hooks or automatic background synchronization are required.

## Rejected Alternatives

1. Central marketplace API: operational cost without a current requirement.
2. Mutable branch archives: cannot identify immutable releases.
3. Technology IDs as package IDs: collide across independent publishers.

## References

- [Declarative generation](0006-declarative-local-generation.md)
- [Local SQLite](0008-local-settings-and-credentials.md)
