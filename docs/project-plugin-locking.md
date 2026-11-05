# Project plugin locking

> Historical guide: Project-level lockfiles were retired by [ADR-0011](adr/0011-one-time-project-generation.md). Managed plugin version selection and artifact integrity checks remain. Existing lockfiles on disk are not removed.

Create a project manifest as usual. Install its generators, then create a lock explicitly:

```sh
provisionhub project lock --file ./demo/provisionhub.yaml
provisionhub project lock --file ./demo/provisionhub.yaml --check
provisionhub project plan --file ./demo/provisionhub.yaml
provisionhub project apply --file ./demo/provisionhub.yaml --yes
```

The lock is `.provisionhub/plugins.lock.yaml` beside the manifest. Commit it alongside the manifest. Its versioned YAML lists each required plugin once, even when several components use it. Official entries record ID, version, source, immutable release URL, archive SHA-256, and loaded content SHA-256. Local entries record ID, version, source, and content SHA-256. It contains no machine paths or secrets.

An existing lock controls normal plan and apply. Both fail if a required version is missing, content differs, a manifest technology changes, or a requested `--plugin-version` conflicts. Plan never changes the lock or contacts the registry. Apply checks the installed content again after review. An unlocked project follows existing installed version selection rules; multiple candidates require explicit `--plugin-version namespace/name@x.y.z`.

To change a lock intentionally, select the desired installed versions and run:

```sh
provisionhub project lock --file ./demo/provisionhub.yaml --plugin-version official/nestjs@1.1.0 --update
```

To restore missing official versions, run `provisionhub project lock --file ./demo/provisionhub.yaml --restore`. This uses the secure registry installer only if its exact release URL and checksum still match the lock. It never chooses the latest release. Already installed verified versions work offline; restoration of missing versions requires registry and artifact access.

Local plugins are content checked but cannot be downloaded from a lock. Move the matching plugin directory or portable bundle to the new machine and pass `--plugins` (or set `PROVISIONHUB_PLUGINS`). If its bytes differ, plan and apply fail. Distinct versions for the same technology cannot be loaded simultaneously; use one selected version per project.

The TUI Create Project result screen shows lock status and offers `l` to create the lock after the manifest is saved. In **My Projects**, open project details and press `l`: `v` validates, `c` previews creation, `u` previews an update, and `r` explicitly restores missing official artifacts. The preview shows old and proposed versions, content digests, and artifact identities without writing. Enter requests confirmation; `y` writes and `n` or Esc cancels. The application rechecks both the previous lock and proposed plugin resolution before writing, so a stale preview must be refreshed. The CLI retains its explicit `--update` workflow.
