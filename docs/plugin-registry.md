# Plugin Registry v1 and Plugin Manager

ProvisionHub discovers public official packages from a static
[JSON index](https://github.com/ProvisionLabs/provisionhub-official-plugins/blob/main/registry.json).
There is no marketplace backend, login requirement, publishing platform, runtime
process, dependency solver or startup network request. CLI and Bubble Tea call
the same `application.PluginManager`; the registry source is independent of UI.
See [ADR-0009](adr/0009-static-plugin-registry.md) for the distribution decision.

## Identities and compatibility

| Concept | Example | Meaning |
| --- | --- | --- |
| Distribution plugin ID | `official/nestjs` | Stable namespace/name, independent of registry |
| Declarative manifest ID | `official-nestjs` | Existing loader/plan identity, preserved by `manifestId` mapping |
| Publisher | `ProvisionLabs` | Distribution owner; distinct from technology |
| Technology ID | `nestjs` | Catalog capability; multiple publishers may provide it |
| Plugin version | `1.1.0` | Immutable stable semantic release, independent of framework version |
| Source | `official` | Official static registry and GitHub Release artifacts |
| Installation | ID + version + local metadata | One locally managed package instance |

Namespaces and names use portable lowercase slugs, each at most 63 characters;
reserved Windows device names are rejected. Distribution v1 accepts stable
`major.minor.patch` versions with numeric ordering and no leading zeroes. Legacy
local manifest version validation remains unchanged. `minimumCoreVersion` is an
inclusive lower bound, checked during install and offline loading. The current
core compatibility version is 0.1.0, separately from its development build label.
Future compatibility ranges require an explicit contract evolution, not a solver.

## Registry contract

The authoritative types and strict validator are in
[internal/registry](../internal/registry/registry.go). The index has
`schemaVersion: 1` and `plugins`. Each plugin contains `id`, `manifestId`,
`publisher`, `name`, `description`, `technologyId`, `category` and `versions`;
`documentation` and `source` HTTPS links are optional. Each version contains
`version`, `minimumCoreVersion`, `artifact` and lowercase hexadecimal `sha256`.
Categories currently attach generators to existing backend/frontend catalog IDs.

Unknown fields, duplicate JSON keys, duplicate plugin IDs or versions, unsafe
identities, invalid semantic versions/checksums, excessive nesting, and unsupported
schemas are errors. Canonical serialization sorts plugins by ID and releases
numerically. A release URL must use HTTPS and a versioned release tag, such as
`official-nestjs-v1.1.0`, never a mutable branch archive. The index may evolve;
published release metadata and assets must never be replaced. Historical releases
remain in the index when templates advance.

The public index is fetched over HTTPS from the official repository on main.
Requests have cancellation, timeouts, bounded metadata and HTTPS-only redirects.
The reusable `registry.Source` boundary allows future additional registry adapters;
only the official source is exposed now. It is not a third-party publishing API.

## Package and integrity

A package is a gzip-compressed USTAR archive containing `plugin.yaml` at its root
and exactly the source files declared by that manifest, at their original relative
paths. Templates, static assets and other generation inputs are declared through
the existing `files` contract. Development files, credentials and unrelated
repository contents are excluded. Official packaging fixes entry ordering, modes,
timestamps and gzip headers and pins Go 1.26.1 because encoders can change between
toolchains. See the official repository's
[distribution guide](https://github.com/ProvisionLabs/provisionhub-official-plugins/blob/main/docs/distribution.md).

The installer downloads to a temporary file, enforces a 16 MiB compressed limit,
and verifies SHA-256 before extraction. It allows at most 1,024 regular file
entries, 1 MiB per file and 32 MiB of payload, with a bounded allowance for tar
headers. It rejects absolute/traversal/nonportable paths, links, special files,
extended headers, case-insensitive duplicates, overlapping file paths, extra
streams, undeclared files, malformed manifests and missing inputs. Manifest ID,
version, name, technology and category must match registry metadata. No downloaded
code or package script executes.

SHA-256 proves that downloaded bytes match the index; it does not independently
prove publisher authenticity. HTTPS, official repository/release ownership and
local data-directory security form the current trust boundary. No signatures or
sandbox against hostile same-user filesystem modifications are claimed. A separate
content digest binds the validated manifest and every declared input for offline
integrity checks without depending on gzip re-encoding.

## Storage, lifecycle and recovery

Use the existing platform application-data directory: Linux
`$XDG_DATA_HOME/provisionhub` (default `~/.local/share/provisionhub`), Windows
`%APPDATA%/ProvisionHub`, and macOS `~/Library/Application Support/ProvisionHub`.
`PROVISIONHUB_DATA_DIR` can select an absolute isolated data directory for tests or
a separate profile; it applies to settings and plugin state together.

```text
<application-data>/
  provisionhub.db
  registry-cache.json
  plugins/official/nestjs/1.1.0/package/plugin.yaml
  plugins/official/nestjs/1.1.0/package/templates/...
  plugins/official/nextjs/1.1.0/package/...
```

No administrator privileges or writes beside the executable are needed. SQLite's
existing transactional migration mechanism adds `installed_plugins` with primary
key `(plugin_id, version)`, unique `installation_path`, `source`, archive `checksum`,
`content_checksum`, `minimum_core_version`, `installed_at` and checked `state`.
No templates or archives are stored as database blobs.

Installation claims a new version container without overwriting existing paths,
reserves `installing` metadata, extracts into an isolated stage, validates contents,
records the content digest, renames the complete stage to `package`, and changes
the row to `ready`. Only verified ready packages load. Updates install alongside
old versions. Removal first records `removing`, deletes only the validated managed
version container through `os.Root`, then deletes metadata. Each metadata mutation
is transactional; SQLite and filesystem changes do not share one transaction.

After a crash, `plugin list` diagnoses interrupted, missing or corrupted packages.
Remove the exact version and reinstall to recover. A failed removal can be retried.
Unreferenced stage/download files never load. A directory created before metadata
reservation may be orphaned: installation refuses to claim it. Inspect and move
that exact orphan aside before retrying; the manager never deletes an unrecorded
directory automatically. Old rows missing new integrity/compatibility metadata
require explicit removal and reinstallation rather than silently acquiring trust.

## CLI

```sh
provisionhub plugin search
provisionhub plugin search backend
provisionhub plugin info official/nestjs
provisionhub plugin install official/nestjs
provisionhub plugin install official/nextjs@1.1.0
provisionhub plugin list
provisionhub plugin updates
provisionhub plugin update official/nestjs
provisionhub plugin remove official/nestjs@1.1.0
provisionhub plugin remove official/nextjs --yes
```

`list` is offline and also reports unmanaged generators from existing local
discovery. `search --cached` and `info --cached` never request the network and mark
metadata stale. Online discovery may fall back to validated cache after a network
failure, preserving its original timestamp and showing a warning; malformed new
metadata remains an error. An already installed healthy exact release is
idempotent. Updating to the same version is harmless. Explicit `install id@version`
permits side-by-side older releases; implicit downgrades are refused. Missing
packages return errors on removal; multiple versions require an exact reference.
Removal requires terminal confirmation or explicit `--yes` for automation.

Commands return nonzero on errors and respect cancellation. Cancellation after
reservation can leave an interrupted installation requiring explicit cleanup.
There is no registry request during generation, local listing or app startup.
No artifact cache is promised for installing new packages offline.

## TUI

Home → **Plugins** starts on offline **Installed**. `a` opens **Available**, `l`
returns to Installed, `t` checks **Updates**, and `r` refreshes the current view.
Available shows descriptions, categories, compatible versions and install status;
`/` searches, Enter inspects, `v` cycles compatible versions, and `i` installs.
Installed shows name, version, source, health and known update; `u` updates,
`d` then `y` removes, and `s` selects a healthy version for generation this session.
Esc cancels a running operation; after it finishes, Esc returns. Ctrl+C quits and
cancels network work. Work executes outside the Bubble Tea event loop; messages
describe real operations without fabricated percentages. Errors remain visible.

## Discovery and version selection

CLI `--plugins` supplies an unmanaged directory; `PROVISIONHUB_PLUGINS` and then an adjacent portable `plugins` directory are fallbacks. Explicit local definitions can generate without SQLite when they cover every requested generator; otherwise managed installations are also checked. Managed packages are discovered offline from their SQLite installation metadata and revalidated against their content checksums. Conflicting generators or installed versions produce an explicit error.

The in-memory project specification declares technologies. Select an exact installed version when several candidates exist:

```sh
provisionhub project plan --name demo --path ./demo --component api:backend:nestjs --plugin-version official/nestjs@1.1.0
provisionhub project apply --name demo --path ./demo --component api:backend:nestjs --plugin-version official/nestjs@1.1.0 --yes
```

Repeat the flag for additional managed plugins. The TUI's Installed `s` action selects a version for that session. Project-level lockfiles are retired; selected installed versions are rechecked before generation. Local plugins must be supplied separately on another machine.
## Validation

Run the normal Go formatting, vet, test and build commands. Mock TLS servers and
temporary SQLite/filesystem fixtures cover distribution without credentials.
Set `PROVISIONHUB_OFFICIAL_PLUGINS` to the official repository's absolute `plugins`
directory to enable `TestOfficialPackageLifecycle`; its CI does this automatically.
That test installs both actual definitions, closes the server, restarts generation,
creates four named components and checks removal isolation. A second compatible
release is tested with a mock, separately from published official versions.
