# Application Settings

Open **Settings** from Home. Persistence is initialized asynchronously on demand;
a storage failure is displayed without preventing other local workflows.

## General

F1 selects General. Tab/Up/Down changes fields; Enter or Ctrl+S saves.
Letters such as `q`, `j`, and `k` are normal text while editing.

| Preference | Default | Validation and behavior |
| --- | --- | --- |
| Default workspace | Empty | Absolute, accessible existing directory; control characters rejected. Used to suggest `<workspace>/<project-name>` in a new TUI wizard. Manual destinations remain supported. |
| Preferred Git provider | Empty | `github`, `gitlab`, `bitbucket`, or empty. Records a preference; does not connect an account or enable future provider implementations. |

Empty values mean no preference and do not require stored default rows. Settings
persist across restarts and are separate from project manifests. A deleted or
inaccessible saved workspace is reported as invalid; clear or correct it before
saving. Initial database failures keep edits from being falsely reported saved.

CLI equivalents use the same services:

```sh
provisionhub settings show
provisionhub settings set --workspace /absolute/existing/workspace --provider github
provisionhub settings set --workspace "" --provider ""
```

On Windows, pass an absolute Windows directory for `--workspace`. Only specified
flags change a preference. Existing invalid values can be corrected with these
flags; no token input or raw database-key editor is exposed.

## Git Providers

F2 selects Git Providers. GitHub implements real device authorization. GitLab and
Bitbucket are explicitly **Coming soon**; there are no simulated connections.

| Key | Action |
| --- | --- |
| `c` | Connect/reconnect a GitHub account |
| `o` during authorization | Open the displayed trusted GitHub verification URL |
| `r` | Validate account credentials using the network; retry an initial load failure |
| Up/Down or `j`/`k` | Select an account |
| `d` | Disconnect the selected account locally and delete its credential |
| Esc during device authorization | Cancel polling; another Esc returns Home |
| Esc otherwise | Return Home, cancelling an outstanding operation |
| PgUp/PgDn | Scroll long settings content |
| Ctrl+C | Cancel settings work and quit |

Normal screen loads read local metadata without vault access or network
requests. Stored account metadata is shown as **not validated**, not connected.
Explicit validation or a completed connection verifies the provider subject ID.
States include not configured, not connected, authentication in progress,
connected, invalid/expired credentials, provider unavailable, and secure storage
unavailable. State changes happen on asynchronous results, not redraws.

```sh
provisionhub settings git-status
provisionhub settings git-status --validate
provisionhub settings git-disconnect <account-id>
```

The default status command is offline. `--validate` can contact GitHub and refresh
an expired token when refresh material is available. JSON contains no secrets;
the credential-reference field is omitted. Account metadata alone never proves authentication.

See [GitHub setup and security](git-providers.md) and
[database paths/migrations](local-persistence.md).
