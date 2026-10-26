# Running and developing ProvisionHub

## Prerequisites

- Go 1.26.1 or later, as declared in `go.mod`.
- An ANSI-compatible terminal for the interactive TUI. The `version` and `--help` commands do not require a terminal.

## Build and run

From the repository root:

```sh
go mod download
go run ./cmd/provisionhub
go run ./cmd/provisionhub version
go run ./cmd/provisionhub --help
go build ./cmd/provisionhub
```

`go build` writes `provisionhub` (or `provisionhub.exe` on Windows) to the repository root. These generated binaries are ignored by Git. To install into your Go binary directory, run `go install ./cmd/provisionhub` and ensure that directory is on your `PATH`.

## Current interface

The home screen lists Create Project, Open Existing Project, Plugin Marketplace, and Settings. Each selection opens an explanatory screen; none performs its named operation yet.

| Key | Action |
| --- | --- |
| Up/Down or `k`/`j` | Move through the home menu, wrapping at the ends |
| Enter | Open the selected screen |
| Esc | Return to the previous screen |
| `q` or Ctrl+C | Quit from any screen |

The TUI switches to a compact layout in small terminals and uses the alternate screen buffer, which Bubble Tea restores on exit. `provisionhub version` prints the version from the same application metadata used by the TUI.

## Validation

```sh
go fmt ./...
go vet ./...
go test ./...
go build ./cmd/provisionhub
```

CI checks formatting, vet, tests, and build on Windows, Linux, and macOS. Tests drive the Bubble Tea model directly, so they do not require an interactive terminal.

The current application layer provides metadata only. Project creation, manifests, technology composition, plugin discovery, marketplace access, provisioning, settings persistence, and remote execution are future work. See the [architecture](architecture/architecture.md) and [ADRs](adr/) for the intended boundaries.
