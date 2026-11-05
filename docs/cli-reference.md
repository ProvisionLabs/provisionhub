# Project CLI reference

| Command | Effect |
| --- | --- |
| `project init --name demo --path ./demo --backend nestjs` | Create a legacy manifest and register it locally |
| `project init --from ./source.yaml --path ./demo` | Copy a validated manifest and register it |
| `project list` | List local references with manifest-derived names and availability |
| `project register --file ./demo/provisionhub.yaml` | Register an existing manifest or project directory |
| `project forget --file ./demo/provisionhub.yaml` | Remove only its local reference |
| `project forget --id <saved-id>` | Forget a moved project using the ID from `project list` |
| `project plan --file ./demo/provisionhub.yaml` | Read-only generation plan; `--output json` is available |
| `project apply --file ./demo/provisionhub.yaml` | Reviewed generation; `--yes` authorizes noninteractive use |
| `project lock --file ./demo/provisionhub.yaml` | Create a plugin lock; `--check`, `--update`, and `--restore` are explicit alternatives |

`project init` leaves the created manifest intact if SQLite registration fails and reports the saved path. Register it later after resolving the database error. `project list` and `register` work offline. Plan and apply select installed plugins from the existing lock workflow; neither command requires registration. See [project management](project-management.md) and [project plugin locking](project-plugin-locking.md).
