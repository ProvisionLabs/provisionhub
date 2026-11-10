# Project CLI reference

| Command | Effect |
| --- | --- |
| `project plan --name demo --path ./demo --component api:backend:nestjs` | Read-only plan for an in-memory specification |
| `project plan ... --output json` | Structured plan with operations and diagnostics |
| `project apply --name demo --path ./demo --component api:backend:nestjs` | Print the plan, preflight output, and ask for literal terminal approval |
| `project apply ... --yes` | Approve generation in noninteractive automation |

Repeat `--component name:type:technology` for named instances. Valid types are `backend`, `frontend`, and `database`. Components may repeat a technology but must have unique names. Both commands accept `--plugins <directory>` for local definitions and repeatable `--plugin-version namespace/name@x.y.z` for exact managed versions.

Repeat `--set component.key=value` to configure a component using its plugin's
schema. For example, with the official 1.3.0 source plugins:

```sh
provisionhub project plan --name commerce --path ./commerce --plugins ../provisionhub-official-plugins/plugins --component api:backend:nestjs --component billing:backend:nestjs --set api.port=4001 --set 'billing.greeting=Welcome'
```

Use the same flags with `project apply`, adding `--yes` only after review.
Strings and enums are literal text, booleans are exactly `true` or `false`, and
integers are signed 64-bit decimal values. Duplicate settings, unknown
components or keys, invalid values, and missing required properties fail.
Omitted values receive defaults. Text and JSON plans include resolved component
configuration. See the [configuration contract](plugin-configuration.md).

Planning does not write files or create the output directory. A valid but unsupported plan can be printed with exit code 0 and `executable: false`; apply refuses it. Generation uses the reviewed plan, reloads plugin content, and rejects changed inputs or existing targets. The commands do not create a project manifest, lockfile, or saved registration.

`project init`, `list`, `register`, `relocate`, `forget`, and `lock` are retired. Existing project files and SQLite rows are not deleted. See [generation planning](generation-planning.md) and [local persistence](local-persistence.md).

Both commands also accept repeated `--connect source:target:type` and
`--connection-set source:target:type.key=value`. For example:

```sh
provisionhub project plan --name demo --path ./demo --plugins ../provisionhub-official-plugins/plugins --component api:backend:nestjs --component web:frontend:nextjs --set api.port=4101 --connect web:api:http --connection-set web:api:http.serverHost=127.0.0.1
```

The host is explicit because the consumer server's network perspective matters.
Do not include a scheme, port, or path in `serverHost`; the selected provider
supplies its configured port. Repeat declarations for additional frontends.
Omit a declaration to remove it from a subsequent one-time invocation; existing
projects are never patched. Unknown references, duplicate settings/declarations,
incompatible or ambiguous capabilities, slot conflicts, missing required values,
and dependency cycles fail before approval. Text/JSON plans show resolved
connection data and outputs. See the [single/shared/distinct target examples](component-connections.md).
