## 1. Command Surface

- [x] 1.1 Extend `oictl tasks --help` to document `tasks skills attach [--external] <skill-id-or-name>...`.
- [x] 1.2 Update `runTasks` parsing to route `tasks skills attach` separately from existing `tasks config` operations.
- [x] 1.3 Validate required skill arguments and reject unknown task skill subcommands with deterministic usage errors.

## 2. Resolution And Update Logic

- [x] 2.1 Fetch task config and resolve `TASK_MODEL` by default or `TASK_MODEL_EXTERNAL` when `--external` is supplied.
- [x] 2.2 Fetch the resolved model and preserve all update-required model fields and existing `meta` values.
- [x] 2.3 Fetch skills, resolve each argument by exact ID or unique exact name, and fail before mutation for unknown or ambiguous references.
- [x] 2.4 Merge resolved skill IDs into `meta.skillIds` idempotently, preserving existing order and appending new IDs in argument order.
- [x] 2.5 Post the updated model payload to `/api/v1/models/model/update` only after all validation succeeds.

## 3. Tests And Documentation

- [x] 3.1 Add tests for default and `--external` task model resolution, skill ID and name resolution, and successful model update payloads.
- [x] 3.2 Add tests for idempotent existing skill IDs, ambiguous skill names, missing task model config, unknown skills, missing models, and malformed `meta.skillIds`.
- [x] 3.3 Update README task examples and task model note to include the new attach workflow.
- [x] 3.4 Run `go test ./...` and address any failures.
