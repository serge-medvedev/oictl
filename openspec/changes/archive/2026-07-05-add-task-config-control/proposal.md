## Why

Open WebUI stores task model selection as task configuration values under the tasks API, but `oictl` does not yet expose a focused way to read or update those settings. Operators need a small CLI surface for controlling the default and external task models without treating them as standalone model attachment resources.

## What Changes

- Add `oictl tasks config get` to read the Open WebUI task configuration payload from `/api/v1/tasks/config`.
- Add `oictl tasks config set --file <path>` to update task configuration through `/api/v1/tasks/config/update`.
- Document that task model settings are configuration keys, including `task.model.default` and `task.model.external`, represented by Open WebUI as `TASK_MODEL` and `TASK_MODEL_EXTERNAL` in the task config API payload.
- Clarify that this change does not add a separate `task-model`, `task-models`, or task-model attachment resource family.

## Capabilities

### New Capabilities

- `tasks-control`: Task configuration commands for Open WebUI task model configuration.

### Modified Capabilities

- None.

## Impact

- CLI command surface: adds the `oictl tasks` resource family with `config get` and `config set` operations.
- Roadmap stage: Stage 1 Foundation Controls, because this is safe, scriptable instance configuration.
- Open WebUI APIs: uses `/api/v1/tasks/config` for reads and `/api/v1/tasks/config/update` for updates.
- Tests: add CLI routing tests proving the expected methods, paths, auth behavior, and request body handling.
- No new dependencies, persistence format changes, or standalone task-model resource APIs are introduced.
