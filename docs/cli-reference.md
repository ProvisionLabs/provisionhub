# Project CLI reference

| Command | Effect |
| --- | --- |
| `project plan --name demo --path ./demo --component api:backend:nestjs` | Read-only plan for an in-memory specification |
| `project plan ... --output json` | Structured plan with operations and diagnostics |
| `project apply --name demo --path ./demo --component api:backend:nestjs` | Print the plan, preflight output, and ask for literal terminal approval |
| `project apply ... --yes` | Approve generation in noninteractive automation |

Repeat `--component name:type:technology` for named instances. Valid types are `backend`, `frontend`, and `database`. Components may repeat a technology but must have unique names. Both commands accept `--plugins <directory>` for local definitions and repeatable `--plugin-version namespace/name@x.y.z` for exact managed versions.

Planning does not write files or create the output directory. A valid but unsupported plan can be printed with exit code 0 and `executable: false`; apply refuses it. Generation uses the reviewed plan, reloads plugin content, and rejects changed inputs or existing targets. The commands do not create a project manifest, lockfile, or saved registration.

`project init`, `list`, `register`, `relocate`, `forget`, and `lock` are retired. Existing project files and SQLite rows are not deleted. See [generation planning](generation-planning.md) and [local persistence](local-persistence.md).
