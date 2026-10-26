# ADR-0003: Local-first developer provisioning

- Status: Accepted
- Date: 2026-10-26
- Owners: ProvisionHub Core Team

## Context

[ADR-0001](0001-go-monorepo.md) assumed a Go repository of control-plane and worker services. That model requires a deployment topology and operating components disproportionate to a developer starting or modifying a local project. ProvisionHub now primarily serves individuals and small teams.

## Decision

The default ProvisionHub runtime is a local Go process. Application orchestration, composition, planning, and execution may be separate code boundaries, but do not require separate services. Local project data is the starting point. A control plane, server database, message queue, distributed workers, mandatory gRPC communication, and Kubernetes deployment are not core prerequisites. Go and a single core repository remain appropriate.

Remote or distributed execution may later be an optional adapter when a real use case justifies it.

## Rationale and tradeoffs

Local operation makes installation, offline use, debugging, and single-user workflows simpler. It avoids requiring teams to operate infrastructure merely to create an application. The tradeoff is that long-running or shared execution needs later design for state, coordination, credentials, and failure recovery. The local engine must still provide observable progress, predictable operation ordering, explicit confirmation for destructive actions, and useful errors.

## Consequences

The old server topology in ADR-0001 is historical, not a roadmap. The active [architecture](../architecture/architecture.md) describes local boundaries. No Go packages are created by this decision alone.
