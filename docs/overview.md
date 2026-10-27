# ProvisionHub overview

## Purpose and audience

ProvisionHub is a local developer tool for individuals and small teams. It configures projects and previews conceptual generation plans through a guided terminal interface and an automation-friendly CLI; generation and provisioning remain future work. The intended output is a coherent project: selected technologies must be compatible and integrated, rather than emitted as isolated templates.

The current built-in catalog offers NestJS, Spring Boot, or Go backends; Next.js or React frontends; and PostgreSQL or MySQL databases. Frontend and database are optional. These are valid manifest choices, not implemented generators. Angular, MongoDB, authentication, CI/CD, Docker, and infrastructure integrations remain future possibilities.

## Product principles

- **TUI-first:** The terminal interface is the primary guided experience. It currently configures projects and previews conceptual plans; opening projects, configuring providers, browsing plugins, and executing workflows with progress remain future work.
- **CLI-compatible:** Non-interactive commands should expose the same use cases for scripts and CI/CD. UI state must not contain provisioning rules.
- **Local-first:** The default workflow runs on the developer's machine without an always-on service or cluster. Remote execution may be offered later as an optional capability.
- **Plugin-driven:** Independently developed plugins may supply templates, framework and service integrations, infrastructure providers, and operations. Official plugins are planned for the separate `provisionhub-official-plugins` repository.
- **Composition-aware:** Plugin metadata should eventually declare capabilities, dependencies, compatibility constraints, and configuration requirements so selected technologies can form one valid application.

## Planned user journey

1. Create a new project or open an existing project manifest.
2. Select technologies and enter required configuration using the TUI or CLI.
3. Resolve compatibility and required capabilities before generating a plan.
4. Preview ordered operations and request explicit confirmation before destructive actions.
5. Execute locally through plugins and external tools, with progress and actionable errors.
6. Save enough project state to inspect and modify the project later.

Configuration and conceptual plan review are implemented: TUI/CLI create a validated [project manifest](project-manifest.md); `project plan --file` previews a saved manifest, and the TUI offers an optional preview before saving. [Generation planning](generation-planning.md) resolves built-in IDs/categories and explicit conflicts, describes unavailable operations, and orders their dependencies without side effects. Opening existing projects in the TUI, resolving plugin capabilities, and execution remain future work. A constructed plan is not an executable plan.

## Scope

ProvisionHub is not intended to be a central control plane, a CI/CD service, a secrets manager, a Kubernetes replacement, or a permanently running developer portal. It may integrate with external systems when a selected plugin requires them. Local use must not require a database server, message broker, workers, or Kubernetes.

The ProvisionLabs website may eventually host a searchable plugin marketplace. Marketplace and registry protocols are deferred. See the [architecture](architecture/architecture.md), [plugin design](architecture/plugins.md), and [ADRs](adr/) for boundaries and decision history.
