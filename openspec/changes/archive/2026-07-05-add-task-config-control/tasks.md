## 1. CLI Routing

- [x] 1.1 Add `tasks` to the top-level command dispatch and help output.
- [x] 1.2 Implement `oictl tasks config get` as an authenticated `GET` to `/api/v1/tasks/config`.
- [x] 1.3 Implement `oictl tasks config set --file <path>` as an authenticated `POST` to `/api/v1/tasks/config/update` with the file JSON as the request body.
- [x] 1.4 Ensure unknown `tasks` subcommands and missing `config` operations return non-zero usage errors consistent with existing command families.

## 2. Documentation And Examples

- [x] 2.1 Document that `TASK_MODEL` maps to Open WebUI config key `task.model.default`.
- [x] 2.2 Document that `TASK_MODEL_EXTERNAL` maps to Open WebUI config key `task.model.external`.
- [x] 2.3 Document that task model configuration is managed through `oictl tasks config` and that no separate task-model attachment resource exists.

## 3. Tests

- [x] 3.1 Add a CLI test for `oictl tasks config get` verifying method, path, authentication, and output.
- [x] 3.2 Add a CLI test for `oictl tasks config set --file <path>` verifying method, path, authentication, request body forwarding, and output.
- [x] 3.3 Add a CLI test for invalid `oictl tasks` usage or unknown task config operations.
- [x] 3.4 Run the relevant Go test suite and record the result.
