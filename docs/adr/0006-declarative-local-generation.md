# ADR-0006: Declarative local file generation

- Status: Accepted
- Date: 2026-10-28
- Owners: ProvisionHub Core Team
- Extends: [ADR-0005](0005-plugin-composition.md) for the first concrete generator

## Context

The conceptual planner could not generate a project. The first vertical slice needs an independently maintained technology template and a reviewable operation plan, while keeping local use simple and refusing unintended writes. The official plugin repository was not available in this workspace, so the initial contract is exercised with test fixtures and awaits its first official plugin.

## Decision

Use a versioned, local YAML plugin manifest describing backend file templates and static copies. The loader validates the manifest, technology identity, paths, symlinks, and template variables before supplying bytes to the pure planner. Planning records concrete file operations and content digests without writing output. Application services reload manifest and plugin inputs before execution and require the reviewed plan to match exactly. CLI and TUI require a separate, explicit approval after preview.

The executor confines file operations with Go `os.Root`, renders all output in memory, stages complete files under the project directory, checks conflicts, and publishes with no-overwrite hard links. No plugin command or npm process runs during generation.

## Rationale

Declarative file operations are enough to test one framework end to end. The same local contract can be interpreted by all supported operating systems without a plugin process or required network service. Content-based comparison makes stale reviews visible. Hard-link publication preserves the existing no-overwrite policy across platforms.

## Trade-offs

- Generation requires hard-link support on the destination filesystem; unsupported filesystems return an error rather than falling back to overwriting rename.
- Multi-file publication is not an atomic transaction. On an error, the executor attempts to remove files it published and empty directories it created; a crash, permission change, or concurrent modification can leave partial output. The result/error reports cleanup failures. Users should inspect the destination before retrying.
- `os.Root` confines relative operations on Windows, Linux, and macOS, but this is not a sandbox against bind mounts, devices, or hostile changes outside the approved root. Plugin manifests and templates are untrusted data, not executable code.
- The contract supports only one backend generator and one literal variable today. Wider composition and plugin execution need separate decisions.

## Rejected alternatives

1. Shell commands such as `npx nest new`: they require network/tool state and give plugins arbitrary execution power.
2. Go native plugins, WASM, or RPC: unnecessary process/runtime complexity for static generation.
3. Direct writes or platform-dependent overwriting rename: they can replace user files or leave incomplete file contents.

## References

- [Plugin architecture](../architecture/plugins.md)
- [Generation planning and execution](../generation-planning.md)
