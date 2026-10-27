# Running and developing ProvisionHub

## Prerequisites

- Go 1.26.1 or later, as declared in `go.mod`.
- An ANSI-compatible terminal for the interactive TUI. `version`, `--help`, `project init`, `project plan`, and `technologies list` do not require a terminal.

## Build and run

From the repository root:

```sh
go mod download
go run ./cmd/provisionhub
go run ./cmd/provisionhub version
go run ./cmd/provisionhub --help
go run ./cmd/provisionhub technologies list
go run ./cmd/provisionhub project init --name my-app --path ./my-app --backend go
go run ./cmd/provisionhub project plan --file ./my-app/provisionhub.yaml
go run ./cmd/provisionhub project plan --file ./my-app/provisionhub.yaml --output json
go build ./cmd/provisionhub
```

`go build` writes `provisionhub` (or `provisionhub.exe` on Windows) to the repository root. These generated binaries are ignored by Git. To install into your Go binary directory, run `go install ./cmd/provisionhub` and ensure that directory is on your `PATH`.

## Current interface

The home screen lists Create Project, Open Existing Project, Plugin Marketplace, and Settings. Create Project starts a six-step wizard: project information, backend, frontend, database, review, and result. Review offers an optional, scrollable generation-plan preview before saving. The other three screens remain explanatory placeholders.

| Key | Action |
| --- | --- |
| Up/Down or `k`/`j` | Move through Home and technology choices |
| Tab or Up/Down | Move between name and destination fields |
| Enter | Select or advance; create the manifest on Review; return to Review from plan preview without saving |
| `p` on Review | Open plan preview; `p` in preview also returns to Review |
| Up/Down or `k`/`j` in preview | Scroll through operations and diagnostics |
| Esc | Return to the previous step or Home |
| Ctrl+C | Quit from any screen |
| `q` | Quit outside the text input step; type `q` normally in a field |

The TUI switches to a compact layout in small terminals and uses the alternate screen buffer, which Bubble Tea restores on exit. `provisionhub version` prints the version from the same application metadata used by the TUI.

## Generation planning

`project plan --file <manifest>` reads and validates an existing manifest, resolves catalog selections, and renders a conceptual generation plan. `--output json` returns the structured plan for automation; `text` is the default. Neither form writes project files, creates destinations, or executes operations. The TUI uses the same application-level planning API for unsaved Review selections. Preview is optional and before saving so the existing explicit Enter-to-save action is unchanged.

Successful plan construction returns `status: planned` and exit code `0`, even though every current operation is `unavailable` and `executable` is `false`. Invalid arguments, unreadable/invalid manifests, unknown/conflicting selections, graph errors, and output failures return exit code `1`. Loading/validation failures in JSON mode include an `invalid` plan with diagnostics on stdout; the executable prints its error on stderr. Argument/format failures may happen before JSON exists. Check exit codes with the built executable rather than the `go run` wrapper.

See the [planning specification](generation-planning.md) for domain fields, deterministic ordering, diagnostic codes, dependencies, and limitations. Planning tests and direct TUI-model interaction tests require no network or external generators.

## Validation

```sh
go fmt ./...
go vet ./...
go test ./...
go build ./cmd/provisionhub
```

CI checks formatting, vet, tests, and build on Windows, Linux, and macOS. Tests drive the Bubble Tea model directly, so they do not require an interactive terminal.

The application layer now validates and saves project manifests through a built-in technology catalog. See the [manifest specification](project-manifest.md) for the schema, IDs, and safety rules. Application source generation, plugin discovery, marketplace access, provisioning, settings persistence, and remote execution are future work. See the [architecture](architecture/architecture.md) and [ADRs](adr/) for the intended boundaries.
