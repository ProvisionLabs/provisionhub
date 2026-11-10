# Component connections and plugin authoring

Connections are optional in-memory declarations between named component
instances. `web -> api-a` selects `api-a`, even when `api-b` uses the same plugin.
The core matches data capabilities, validates values and dependencies, and
plans plugin-owned files. It contains no framework pairing rules.

Connection declarations extend the existing plugin manifest in core
compatibility 0.3.0. Publish a new plugin version with that minimum core version;
never replace existing release artifacts. Old plugins and projects without
connections retain their generation behavior. Project YAML, project lockfiles,
saved registrations, scripts, network provisioning, and deployment orchestration
are not part of this model. See [ADR-0013](adr/0013-component-connections.md).

## User input

CLI `project plan` and `project apply` accept repeatable
`--connect source:target:type` and
`--connection-set source:target:type.key=value`. Names identify instances;
`type` identifies a shared plugin capability. Connection settings use the
existing scalar schema parser, defaults, and constraints. Duplicate declarations
or settings are errors. To remove a connection in a later one-time invocation,
omit its flags; generation still requires a fresh output directory.

With the local official 1.3.0 source plugins, two frontends can share one backend:

```sh
provisionhub project plan --name example --path ./example \
  --plugins ../provisionhub-official-plugins/plugins \
  --component api-a:backend:nestjs --component api-b:backend:nestjs \
  --component web:frontend:nextjs --component admin:frontend:nextjs \
  --set api-a.port=4101 --set api-b.port=4102 \
  --connect web:api-a:http --connect admin:api-a:http \
  --connection-set web:api-a:http.serverHost=127.0.0.1 \
  --connection-set admin:api-a:http.serverHost=127.0.0.1
```

This example explicitly chooses same-host loopback. For a distinct target,
replace the admin declaration and its setting with `admin:api-b:http` and
`admin:api-b:http.serverHost=<reachable-host>`. Its endpoint will use port 4102;
web continues to use api-a on 4101. Use the same flags with `project apply`,
reviewing before approval. The public registry still contains published 1.1.0
packages; these connection examples require local source until a separate release.

In the TUI, press `c` from Components. Add/edit/remove connections with `a`/`e`/`d`.
The async target list offers compatible source/target/type pairs and excludes
ambiguous pairs, occupied consumer slots, duplicates, and cycles. Enter opens the
existing dynamic configuration form. Esc preserves the current draft when
returning to target selection; selecting a different target starts its settings
afresh. Enter on the last field saves. Enter/Esc from Connections returns to
Components, then Enter opens Review. Rename updates references; deleting a
component removes its connections with feedback. Review and `p` preview show
resolved configuration, target metadata, and output files.

## Provider contract

Add optional `connections.provides` entries to a plugin. For example, a generator
with a declared integer component option named `port` can expose:

```yaml
connections:
  provides:
    - id: http-api
      type: http
      description: Plain HTTP endpoint
      metadata:
        port: {configuration: port}
        protocol: {value: http}
```

Each slot has a unique `id` within its role, a shared `type`, and optional
description. IDs use lowercase ASCII letters, digits, and hyphens, at most 63
characters, starting/ending with a letter or digit. Declaration types also obey
the portable identifier validation used by the specification. Metadata keys
use configuration key syntax. Every binding declares exactly one scalar
`value` or a `configuration` key from that provider's component schema. A
referenced optional value must exist when a connection selects the provider.
Bindings cannot reference other connections, evaluate expressions, or execute
code. Values come from the selected instance's resolved configuration.

## Consumer contract

Add optional `connections.consumes` entries. These are supported integration
slots; they do not require a connection in an otherwise unconnected project.

```yaml
connections:
  consumes:
    - id: server-api
      type: http
      description: Server-side API configuration
      requires:
        properties:
          port: {type: integer, required: true, minimum: 1, maximum: 65535}
          protocol: {type: enum, options: [http], required: true}
      configuration:
        properties:
          serverHost:
            type: string
            format: host
            required: true
            label: Host reachable from the consumer server
      files:
        - {type: template, source: templates/api.env.tmpl, destination: .env.local}
```

`requires` is a scalar schema for the consumed metadata. A provider can expose
extra fields; the consumer validates and receives only declared fields. Types,
defaults, required values, enum choices, and constraints use the shared
[configuration engine](plugin-configuration.md). `configuration` is the separate
schema for values on this particular connection, not component-global values.
It can be omitted when no connection options are needed.

Matching requires exactly one provider/consumer pair with the same type and
compatible resolved metadata. Zero pairs are unsupported; multiple pairs are
ambiguous and rejected with slot IDs. Each consumer slot supports one target
per component in this version. Multiple components may consume one provider.
For multiple distinct integrations on one source, declare distinct compatible
types/slots with nonoverlapping outputs. No dependency solver or arbitrary
provider preference is introduced.

`files` uses existing `template`/`copy` operations and path restrictions. It is
a bounded connection-owned file set, activated only by a valid selected
connection. Existing always-generated templates cannot do this without either
requiring references for unconnected components or changing their output.
There are no general `when` conditions, loops, or conditional expressions.
Paths remain literal; configuration never expands into a path. All base and
consumer output paths in a plugin must be nonoverlapping, even for different
slots. Every integration input is packaged and integrity-checked, including
when it is not selected for generation.

## Integration template context

Integration files retain existing project/component/config references and add:

| Reference | Meaning |
| --- | --- |
| `{{source.name}}` | Named consuming component |
| `{{target.name}}` | Named providing component |
| `{{config.key\|json}}` | Resolved consuming component option |
| `{{connection.key\|json}}` | Resolved connection-specific option |
| `{{target.port}}` | Consumed scalar metadata declared in `requires` |
| `{{connection.serverHost\|host}}` | Validated URI host, with IPv6 brackets |

String/enum values require explicit escaping. JSON literal and HTML text modes
remain supported. New `|host` is restricted to string properties declaring
`format: host`; it validates ASCII DNS names or IPv4/IPv6 addresses without DNS
lookups and adds IPv6 brackets. It rejects schemes, ports, paths, credentials,
scoped addresses, whitespace, and control characters. Other formats or arbitrary
filters are unsupported. Bare booleans/integers render as literals. Missing or
unsupported references fail before execution. Expansion is single-pass.

The official consumer's fixed template can therefore contain:

```text
API_ORIGIN=http://{{connection.serverHost|host}}:{{target.port}}
```

## Planning, approval, and networking

Resolution validates endpoints, rejects self-connections, duplicates,
unsupported/ambiguous capabilities, invalid configuration, and generation
dependency cycles. Components and declarations are sorted deterministically.
The planner contributes no operations from an invalid connection set. Valid
plans contain `connect_components` operations and connection-owned file
operations with stable identities, source/target information, resolved values,
output paths, and rendered content digests. Source component capabilities
depend on target capabilities; connection operations wait for both components'
base files. File operations depend on their connection operation.

Apply reloads plugin definitions and compares the complete fresh plan with the
review. Changed endpoints, configuration, metadata, templates, or plugin versions
invalidate approval. Integration files use the same root confinement, conflict
checks, staging, no-overwrite publication, and cleanup semantics as base files.

Generation records application configuration; it does not establish runtime
networking or probe services. Syntax-valid hosts are not proof of reachability.
The official Next.js consumer creates private `API_ORIGIN` in `.env.local`
and `.env.example`, for `process.env.API_ORIGIN` in server-side code. Next.js
loads `.env.local`; an existing runtime environment value takes precedence.
Changing environment files generally requires restarting the server. Generated
files are initial values, not ongoing synchronization with backend settings.
If backend `PORT` or deployment addresses change, update API_ORIGIN accordingly.

Environment configuration can be read during a Next.js build as well as when
server code runs. Data consumed by static generation can remain in build output;
use dynamic server execution when reads must reflect runtime settings. The
integration test deliberately uses a dynamic server route and starts real
servers to verify runtime consumption, rather than inferring it from compilation.

No `NEXT_PUBLIC_*` value is generated. Browser-visible addresses are a different
network perspective, and public Next.js values can be embedded at build time.
Do not reuse internal server addresses in browser code. Loopback works only
when the Next.js server and backend share the appropriate host/network context;
bind addresses such as 0.0.0.0 are not portable destination addresses. The
plugin supports direct plain HTTP, not TLS termination, CORS setup, proxying,
container discovery, or deployment. Those require separate decisions.

## Reproducible validation

Core tests use local fixtures; the official repository adds offline CLI tests
and a managed lifecycle with local HTTPS fixtures, offline restart, input
tamper detection, and no saved registrations. Native CI covers Windows, Linux,
and macOS. No live network is required for unit/offline generation tests.

For actual runtime validation, build the core, set `PROVISIONHUB_BIN` to its
absolute path, and run in the official repository:

```sh
node --test tests/connections-runtime.test.mjs
```

This explicitly installs/builds generated applications, starts two NestJS and
two Next.js servers for shared and distinct-target scenarios, and checks real
HTTP response bodies and selected origins. A small Next.js route is added only
to temporary test fixtures; the plugin ships no HTTP client, proxy, or extra
page. This test requires npm/tooling access and is separate from offline CI.
