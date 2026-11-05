# Running and developing ProvisionHub

## Prerequisites

- Go 1.26.1 or later, as declared in `go.mod`.
- An ANSI-compatible terminal for the interactive TUI. `version`, `--help`, `project init`, `project plan`, and `technologies list` do not require a terminal. `project apply` runs non-interactively with `--yes`.

## Build and run

From the repository root:

```sh
go mod download
go run ./cmd/provisionhub
go run ./cmd/provisionhub version
go run ./cmd/provisionhub --help
go run ./cmd/provisionhub technologies list
go run ./cmd/provisionhub project init --name my-app --path ./my-app --backend go
go run ./cmd/provisionhub project init --from ./source.yaml --path ./multi-app
go run ./cmd/provisionhub project plan --file ./my-app/provisionhub.yaml
go run ./cmd/provisionhub project plan --file ./my-app/provisionhub.yaml --output json
go run ./cmd/provisionhub project plan --file ./my-app/provisionhub.yaml --plugins ./path/to/plugins
go run ./cmd/provisionhub project apply --file ./my-app/provisionhub.yaml --plugins ./path/to/plugins --yes
go build ./cmd/provisionhub
```

`go build` writes `provisionhub` (or `provisionhub.exe` on Windows) to the repository root. These generated binaries are ignored by Git. To install into your Go binary directory, run `go install ./cmd/provisionhub` and ensure that directory is on your `PATH`.

## Current interface

The home screen lists Create Project, My Projects, Plugins, and Settings. Create Project collects project information, then opens the named-component list. Add, edit, or remove components before review. Review offers an optional, scrollable generation-plan preview before saving. My Projects registers and reopens manifests, shows current component and lock details, and offers confirmed apply, relocation, and forgetting. Settings provides persistent General preferences and Git Providers; Plugins manages local installations. Legacy manifests and CLI flags remain supported outside the TUI. See the [TUI guide](tui.md) for saved-project controls.

| Key | Action |
| --- | --- |
| Up/Down or `k`/`j` | Move through Home and technology choices |
| Tab or Up/Down | Move between name and destination fields |
| `a`, `e`, `d` on component list | Add, edit, or remove the selected component |
| Enter | Select or advance; create the manifest on Review; return to Review from plan preview without saving |
| `p` on Review | Open unsaved configuration preview; `p` in preview returns to Review |
| `p` on saved result | Preview the saved project's current plan |
| `g` in executable saved-project preview | Open generation confirmation after read-only preflight |
| `y` / `n` or Esc in confirmation | Apply the plan / cancel without writing |
| Up/Down or `k`/`j` in preview | Scroll through operations and diagnostics |
| Esc | Return to the previous step or Home |
| Ctrl+C | Quit from any screen |
| `q` | Quit outside the text input step; type `q` normally in a field |

The TUI switches to a compact layout in small terminals and uses the alternate screen buffer, which Bubble Tea restores on exit. `provisionhub version` prints the version from the same application metadata used by the TUI.

## Generation planning

`project plan --file <manifest>` reads and validates an existing manifest, resolves catalog selections and optional local plugins, and renders a conceptual or concrete generation plan. `--output json` returns the structured plan for automation; `text` is the default. Neither form writes project files, creates destinations, or executes operations. The TUI uses the same application-level planning API for unsaved Review selections. Set `PROVISIONHUB_PLUGINS` to the plugin directory before launching the TUI.

Successful plan construction returns `status: planned` and exit code `0`, even when a missing generator makes the plan non-executable. A matching local plugin contributes concrete supported file operations; every selected capability must be supported for `executable: true`. Invalid arguments, unreadable/invalid manifests, unknown/conflicting selections, graph errors, and output failures return exit code `1`. Loading/validation failures in JSON mode include an `invalid` plan with diagnostics on stdout; the executable prints its error on stderr. Argument/plugin-directory errors may happen before JSON exists. Check exit codes with the built executable rather than the `go run` wrapper.

See the [planning specification](generation-planning.md) for domain fields, deterministic ordering, diagnostic codes, dependencies, and limitations. Planning tests and direct TUI-model interaction tests require no network or external generators.

## Validation

```sh
go fmt ./...
go vet ./...
go test -count=1 ./...
go build ./cmd/provisionhub
```

CI checks formatting, vet, tests, and build on Windows, Linux, and macOS. Tests drive the Bubble Tea model directly, so they do not require an interactive terminal.

CLI and TUI test fixtures pass explicit temporary SQLite paths to their command and model constructors. CLI, TUI, and application package `TestMain` setup also sets `PROVISIONHUB_DATA_DIR` to a disposable directory as a safeguard for legacy default Plugin Manager calls. The real CLI subprocess isolation test overrides `PROVISIONHUB_DATA_DIR`, XDG paths, Windows application-data paths, and home variables in the child process; it checks that a separate sentinel database does not change. Application persistence tests use `t.TempDir()` databases. Production path selection is documented in [local persistence](local-persistence.md). Tests must not register temporary projects through the default user database.

The application layer validates and saves project manifests through the built-in catalog, loads local declarative plugins, and coordinates reviewed file generation. See the [manifest specification](project-manifest.md), [component model](architecture/components.md), [plugin contract](architecture/plugins.md), and [generation planning/execution guide](generation-planning.md). Official NestJS and Next.js plugins are in the separate [official plugins repository](https://github.com/ProvisionLabs/provisionhub-official-plugins); marketplace access, infrastructure provisioning, and remote execution remain future work.

## Settings and authentication development

SQLite uses `modernc.org/sqlite` without CGO; the pinned `modernc.org/libc` must
match the driver's own module dependency. The native vault uses
`github.com/zalando/go-keyring`: macOS needs the system `security` utility, Linux
needs a session D-Bus/Secret Service, and Windows uses Credential Manager. Browser
opening is optional (`open`, `xdg-open`, or the Windows protocol handler).

No account or network is needed to run normal Go tests. HTTP servers and vaults
are mocked; SQL tests use temporary databases. Do not put a real token in test
fixtures. A deliberately opt-in native-vault integration test stores and deletes
a disposable non-sensitive value:

```powershell
$env:PROVISIONHUB_TEST_NATIVE_VAULT = "1"
go test -count=1 -run TestNativeVaultIntegration ./internal/credentials
Remove-Item Env:PROVISIONHUB_TEST_NATIVE_VAULT
```

On Unix, prefix the command with `PROVISIONHUB_TEST_NATIVE_VAULT=1` instead.
This may trigger a native unlock prompt; it is not a GitHub OAuth test. For live
device-flow verification, register an app and configure its public client ID as
documented in [Git providers](git-providers.md). Never embed client secrets or
live tokens in source. Disconnect test accounts when finished; remote grant
revocation is a separate action in GitHub settings.

See [Settings controls](settings.md), [SQLite paths and migrations](local-persistence.md),
and [validation results](settings-validation.md).
