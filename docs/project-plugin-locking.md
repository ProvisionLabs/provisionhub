# Project plugin locking

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

The TUI Create Project result screen shows lock status and offers `l` to create the lock after the manifest is saved. The CLI supplies lock checks, updates, and restoration for saved projects.
