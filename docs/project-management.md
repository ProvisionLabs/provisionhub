# Saved project management

A saved project is a local SQLite reference to an existing `provisionhub.yaml`. The manifest remains the source of truth for the project name, components, technologies, and connections. `.provisionhub/plugins.lock.yaml` remains the source of truth for exact plugin versions and artifact checksums. The database stores only an ID, canonical manifest path, registration time, and last opened time. You can copy or use the project without its database registration.

## Register and reopen

Creating a project through `project init` or the TUI registers it after the manifest is written. In the TUI, open **My Projects** and press `r` to register an existing manifest or project directory. From the CLI, use `provisionhub project register --file ./demo/provisionhub.yaml`. Registration validates the manifest and current catalog IDs, resolves filesystem aliases, and rejects duplicate paths. Projects with the same manifest name are allowed. Neither a lockfile nor network access is required.

`provisionhub project list` and **My Projects** read registered paths and then inspect each manifest independently. A missing or invalid manifest is shown as unavailable; other projects remain accessible. Opening a project reads its current manifest and lockfile from disk and records a local last opened time. No project files are changed by listing or opening.

## Relocate and forget

If the project moves, select it in **My Projects**, press `m`, and enter the new manifest path or directory. The CLI equivalent is `provisionhub project relocate --id <saved-id> --file ./moved/provisionhub.yaml`. Relocation validates the replacement manifest, rejects a path already registered to another project, and keeps the original local ID. It changes only the SQLite path. A missing old path does not prevent relocation.

Press `f` and confirm with `y` to **Forget Project**. This removes only the local reference; the project directory, generated files, manifest, lockfile, and Git repositories remain untouched. The CLI equivalent is `provisionhub project forget --file ./demo/provisionhub.yaml`, or `--id <saved-id>` if the old path no longer exists.

## Plan, apply, and plugin locks

From project details, `p` reads the current manifest and local plugins and displays the Planning Engine's operations and diagnostics. `g` performs a read-only preflight; `y` then authorizes the Execution Engine. Esc or `n` cancels. Apply reloads the manifest, lock, and plugin contents and refuses a stale plan or existing targets. Opening a project never regenerates components.

Press `l` for the plugin lock screen. It shows exact versions and content checksums. `v` validates the lock, `c` previews creation when missing, `u` previews an update, and `r` explicitly restores missing official artifacts through the existing installer. Preview shows current and proposed identities and artifact data without writing. Enter requests confirmation; only `y` writes. A changed lock or plugin resolution requires a fresh preview. Restoration needs registry access only for a missing official artifact; listing, opening, planning, and validation are offline.

If registration fails after initialization, the manifest and any generated files remain in place and the error names the saved path. Fix the local database problem, then register the manifest. For database location and recovery, see [local persistence](local-persistence.md). For lock semantics, see [project plugin locking](project-plugin-locking.md).
