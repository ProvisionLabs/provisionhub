# ProvisionHub overview

ProvisionHub is a local-first, one-time project initializer for individuals and small teams. The TUI or CLI collects a project name, output path, and named component instances. It validates the selections, previews a deterministic generation plan, and generates source code only after explicit approval. It does not track or reconcile the project afterward.

The built-in catalog offers NestJS, Spring Boot, or Go backends; Next.js or React frontends; and PostgreSQL or MySQL databases. A project can contain several components using the same technology. The official NestJS and Next.js plugins provide generators; other catalog choices may yield a valid but non-executable plan until a generator is installed. Optional component connections remain metadata and do not generate integrations.

Plugins are declarative and use their own YAML manifests and templates. The Plugin Manager discovers and installs official packages, verifies artifact integrity, and supports explicit installed-version selection. Settings and Git provider account metadata remain local application state. Project generation itself uses an in-memory specification and does not create project YAML, project lockfiles, or saved registrations. SQLite remains for Settings, accounts, and managed plugin metadata; local plugin directories can generate without it.

ProvisionHub is not a control plane, CI/CD service, secrets manager, or infrastructure provisioner. It does not install generated app dependencies or start apps. See [architecture](architecture/architecture.md), [generation planning](generation-planning.md), [plugin management](plugin-registry.md), and [ADR-0011](adr/0011-one-time-project-generation.md).
