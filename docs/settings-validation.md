# Application Settings and Git Authentication validation

Validated on November 2, 2026, locally on Windows with Go 1.27.0. Runtime changes
were integrated incrementally through PRs #12–#16, each using a normal pull
request/rebase merge without a branch-protection bypass or force push.

## Architecture and implementation

1. `internal/settings` defines validated optional workspace/provider preferences.
   Application services share those use cases between CLI and TUI.
2. `internal/localdb` uses `modernc.org/sqlite v1.61.1` through `database/sql`, with
   matching `modernc.org/libc v1.79.0`, no ORM/CGO, one pooled connection, and a
   five-second busy timeout. See [platform paths](local-persistence.md#location-and-ownership).
3. Transactional ordered migrations use `PRAGMA user_version`: version 1 creates
   preferences; version 2 creates account metadata with a unique provider/instance/
   subject identity. There are no speculative domain tables or destructive migrations.
4. Empty preferences are safe defaults. Saves validate before writing; empty
   values delete preference rows. The workspace pre-fills a new wizard destination.
5. `credentials.Native` adapts Windows Credential Manager, macOS Keychain, and
   Linux Secret Service via `go-keyring v0.2.8`; credentials and refresh material
   never enter SQLite. Errors and secret formatting are sanitized.
6. The provider contract identifies provider/instance/method and supports account
   identification, cancellable authorization, refresh, and local disconnection.
   Multiple account records are supported. Only GitHub.com is implemented.
7. GitHub uses real device authorization with a configured public client ID,
   no client secret, no requested OAuth scopes, interval/slow-down handling,
   account retrieval, optional expiring-token refresh, and cancellation.
8. Settings has General and Git Providers tabs, asynchronous I/O, device
   instructions/browser opening, status validation, disconnect/retry, and stale
   result protection. GitLab and Bitbucket are explicitly Coming soon.

## Passed validation

- `go fmt ./...`, `go vet ./...`, `go test -count=1 ./...`, and
  `go build ./cmd/provisionhub` completed successfully on final runtime code.
- CGO-free builds for Windows, Linux, and macOS amd64 completed successfully.
- GitHub Actions for all five runtime commits passed the existing Windows,
  Linux, and macOS validation matrix using the repository's Go version.
- SQLite tests exercised first run, missing directories, restart persistence,
  defaults without rows, invalid values, serialized concurrent reads, upgrade
  from version 1, deterministic ordering, transaction rollback, future versions,
  corruption/access errors, platform paths, and byte-identical no-op reopen.
- Vault-adapter tests exercised store/get/replace/delete, missing credentials,
  backend failures, and redacted errors/formatting. The opt-in Windows native
  vault test also stored, retrieved, and deleted a disposable non-sensitive value.
- Mock HTTP tests covered device success, pending/slow-down intervals, expiry,
  denial, cancellation, unsafe verification URLs, network/401 errors, user
  identity, and token refresh. No live account or token was required.
- Application tests covered SQL failure after vault storage, reconnect cleanup,
  missing/invalid/expired credentials, unavailable providers/vaults, disconnection
  metadata/deletion failures, and absence of token test sentinels in SQLite bytes.
- TUI model tests covered editing, asynchronous save/error feedback, future
  provider labels, device instructions, browser-result state, cancellation,
  connection/disconnection, partial cleanup errors, narrow layouts, and default
  workspace use. Existing wizard/navigation/generation coverage still passed.
- Separate compiled CLI processes saved/reloaded preferences and exercised
  settings show/status, version, help, and technology listing.
- With unreachable HTTP/HTTPS proxies, no GitHub client configured, and a
  deliberately unavailable application database path, the compiled CLI still
  created, planned, and approved generation of a NestJS + Next.js project in a
  system temporary directory. Planning was read-only; apply created 20 files.
- The official plugin repository was not modified. Local root and portable
  Windows executables were rebuilt; binaries remain ignored rather than committed.

## Security and limits

No tokens are persisted in SQLite, project manifests, generated files, or logs.
Provider bodies and transport errors are not echoed. SQL values are parameterized;
browser opening is restricted to GitHub's verification URL. SQLite/vault failures
never silently fall back to plaintext. General preference reads do not access the
vault. Project CLI generation never initializes settings or authenticates.

**Live GitHub browser authentication was not performed.** A registered OAuth app
with device flow enabled and `PROVISIONHUB_GITHUB_CLIENT_ID` is required; see
[setup instructions](git-providers.md#register-a-github-oauth-app). No client ID
or secret is fabricated or embedded in the binary.

The opt-in native-vault test is skipped in normal CI. Some ordinary symlink tests
can skip locally on Windows when the required privilege is unavailable; the
existing Linux/macOS CI matrix passed. Native vault operation was tested on
Windows only; macOS/Linux native unlock/session behavior was not exercised locally.
Real GUI/browser interaction was not automated; TUI tests drive the model directly.

Disconnection removes local credentials, not the remote OAuth grant. Native vault
calls may wait on OS prompts and cannot be forcibly cancelled by a context.
Cross-store crash atomicity is not guaranteed; interrupted credential/SQL updates
can leave orphan vault entries. Cleanup failures are explicit and retryable where
metadata remains. See [recovery details](local-persistence.md#failures-and-recovery).

## Integrated commits and next work

| Commit | Change | Pull request |
| --- | --- | --- |
| `eabc78d` | SQLite settings and migrations | #12 |
| `9bfadd5` | GitHub device flow and native credentials | #13 |
| `aae4876` | Settings TUI, CLI, and workspace defaults | #14 |
| `e57d9d3` | Credential recovery/security regression coverage | #15 |
| `9785009` | Vault-independent preference reads | #16 |

Each integrated commit was verified against `origin/main`. The final documentation
commit and final remote/working-tree verification are reported at task completion.
Recommended next work is a configured live GitHub device-flow acceptance check
and native-vault validation on macOS/Linux before adding a second provider adapter.
Repository publishing and component connection execution remain separate milestones.
