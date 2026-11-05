# Running and developing ProvisionHub

Use Go 1.26.1 or later. The TUI needs an ANSI-compatible terminal; version, help, catalog, and project plan commands do not.

```sh
go mod download
go run ./cmd/provisionhub
go run ./cmd/provisionhub technologies list
go run ./cmd/provisionhub project plan --name demo --path ./demo --component api:backend:nestjs --plugins ./path/to/plugins
go run ./cmd/provisionhub project apply --name demo --path ./demo --component api:backend:nestjs --plugins ./path/to/plugins --yes
go build ./cmd/provisionhub
```

The TUI opens Create Project, Plugins, or Settings. Create Project collects named components, offers a read-only plan with `p`, and requires Enter on an executable Review to generate. CLI apply prints and checks a fresh plan before asking for literal `yes`, unless `--yes` explicitly approves automation. Neither path writes a project manifest, lockfile, or saved registration.

## Validation

```sh
go fmt ./...
go vet ./...
go test -count=1 ./...
go build ./cmd/provisionhub
```

CI runs these checks on Windows, Linux, and macOS. Tests drive Bubble Tea models directly and use local plugin fixtures and temporary output directories. Application and plugin tests use temporary SQLite databases. CLI/TUI/application package test setup sets `PROVISIONHUB_DATA_DIR` to a disposable directory for default Plugin Manager calls. Explicit `--plugins` or `PROVISIONHUB_PLUGINS` definitions can plan and generate with an unusable SQLite path; managed plugin installation and discovery still use the SQLite metadata store. Tests do not need network access, live GitHub authentication, or the developer's home database.

To exercise the official NestJS and Next.js definitions from a sibling checkout, set `PROVISIONHUB_OFFICIAL_PLUGINS` to its `plugins` directory and run the official package lifecycle test in `internal/application`. Core CI does not require that checkout. See [generation planning](generation-planning.md), [CLI reference](cli-reference.md), and [plugin registry](plugin-registry.md).

## Settings and authentication development

SQLite uses `modernc.org/sqlite` without CGO. The native credential vault uses `github.com/zalando/go-keyring`: macOS needs `security`, Linux needs a session D-Bus/Secret Service, and Windows uses Credential Manager. Tests mock HTTP and vault interfaces. Do not put live tokens in fixtures.

An opt-in native-vault integration test stores and deletes a disposable non-sensitive value:

```powershell
$env:PROVISIONHUB_TEST_NATIVE_VAULT = "1"
go test -count=1 -run TestNativeVaultIntegration ./internal/credentials
Remove-Item Env:PROVISIONHUB_TEST_NATIVE_VAULT
```

On Unix, prefix the command with `PROVISIONHUB_TEST_NATIVE_VAULT=1`. This may trigger a native unlock prompt; it is not a GitHub OAuth test. See [Settings](settings.md), [Git providers](git-providers.md), and [local persistence](local-persistence.md).
