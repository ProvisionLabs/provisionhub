# Plugin architecture

## Initial local contract

ProvisionHub discovers immediate subdirectories of an explicitly supplied local plugin directory. Each plugin has a `plugin.yaml` and local source files. The loader reads only YAML and file bytes; no plugin code, shell commands, subprocesses, downloads, or scripts are run. The built-in catalog retains stable technology IDs. A plugin can attach a generator to an existing ID but cannot redefine its category or display name. A selected technology without a generator remains a valid configuration with an unavailable planning capability.

The current contract is `provisionhub.dev/plugin/v1alpha1`. It supports `generate_backend` for backend technologies and `generate_frontend` for frontend technologies, with two file operations: `template` and `copy`. One plugin definition can serve multiple named components using its technology. For example:

```yaml
apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nestjs
version: 1.0.0
technologyId: nestjs
category: backend
name: NestJS
description: Minimal NestJS backend
capability: generate_backend
files:
  - type: template
    source: templates/package.json.tmpl
    destination: package.json
  - type: copy
    source: assets/.gitignore
    destination: .gitignore
```

`id` and `technologyId` use 1–63 lowercase ASCII letters, digits, and hyphens, starting and ending with a letter or digit. `version` is a nonempty, at most 64-character version label using lowercase letters, digits, dots, and hyphens. The loader rejects missing or unknown fields, multiple YAML documents, duplicate plugin IDs, duplicate generators for a technology, duplicate or overlapping output paths within a plugin (including case-only conflicts), unsupported category/capability combinations, and unsupported versions. Planning also rejects duplicate or overlapping output paths across selected plugins. Every source must be a regular file of at most 1 MiB; manifests have the same size limit.

Source and destination paths use `/`, are relative and clean, and cannot contain `..`, absolute prefixes, backslashes, colons, NUL, or empty components. `provisionhub.yaml` remains reserved for compatibility with existing output safety rules. Symlinked plugin entries, manifests, template paths, and output parents are rejected. Supported literal template variables are `{{projectName}}` (project name) and `{{componentName}}` (instance name). Configuration references support explicit escaping; see the [configuration schema and authoring guide](../plugin-configuration.md). Other or malformed template delimiters are rejected. `copy` files are copied byte-for-byte. There is no general template interpreter or command action.

Official plugins live in the separate `provisionhub-official-plugins` repository. Install official packages with the [Plugin Manager](../plugin-registry.md), or supply that repository's `plugins` directory to local discovery. The NestJS plugin declares files under `backend/`; the Next.js plugin declares files under `frontend/`. Named components rebase them below each name, such as `api/package.json` or `web/package.json`. Both official plugins use `{{componentName}}` for application identity. Database selections still lack providers. The plugin repository documents Node/npm requirements and repeatable integration checks. A website marketplace remains future work.

## Discovery and planning

CLI commands accept `--plugins <directory>`. Both interfaces also honor `PROVISIONHUB_PLUGINS`, then an adjacent portable `plugins` directory. An explicitly supplied but missing or invalid directory reports an error. Healthy managed installations are loaded from SQLite metadata without network access when managed discovery is used. Identical definitions deduplicate; different versions or implementations for a technology report conflicts. Multiple managed versions require explicit session selection. Planning reloads and integrity-checks plugin files for each preview. The pure planner receives validated data and creates stable conceptual and concrete file operations after resolving per-component configuration and rendering in memory, without writing files. The plan records resolved configuration per component. Each concrete operation records component ownership, plugin identity and digest, source path/type/content digest, rendered content digest, target, status, and dependency on its component capability. The plan is executable only when every selected capability is supported. See [named components](components.md).

See [generation planning](../generation-planning.md) for plan semantics and [ADR-0006](../adr/0006-declarative-local-generation.md) for the initial execution decision.

## Future composition

[ADR-0005](../adr/0005-plugin-composition.md) still governs the broader direction: configuration inputs are implemented as flat plugin-defined schemas; future plugins may express requirements, dependencies, compatibility ranges, and provider operations. Static declarative distribution is defined by [ADR-0009](../adr/0009-static-plugin-registry.md); runtime transport, cross-language execution, a general resolver, third-party publishing and sandboxing remain separate decisions. Mandatory gRPC from [ADR-0002](../adr/0002-grpc-plugin-contract.md) remains superseded. A registry or website connection is never required for local project generation.
