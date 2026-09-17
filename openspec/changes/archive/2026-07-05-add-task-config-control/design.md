## Context

Open WebUI exposes task configuration through the tasks router rather than the general configs router. The task configuration payload includes task model fields `TASK_MODEL` and `TASK_MODEL_EXTERNAL`, backed by config keys `task.model.default` and `task.model.external`, plus other task-generation settings.

`oictl` already follows a resource-first command taxonomy. This change introduces the `tasks` resource family for the task API's configuration surface and classifies it as Stage 1 Foundation Controls because it provides safe, scriptable instance configuration.

## Goals / Non-Goals

**Goals:**

- Provide read and update commands for Open WebUI task configuration at `oictl tasks config get` and `oictl tasks config set --file <path>`.
- Preserve Open WebUI's payload shape, including `TASK_MODEL` and `TASK_MODEL_EXTERNAL`, instead of inventing a new local schema.
- Make clear that task model selection is task configuration, not a separate task-model attachment resource.

**Non-Goals:**

- No `oictl task-model`, `oictl task-models`, or model attachment resource commands.
- No partial field-specific setters for `task.model.default` or `task.model.external` in this change.
- No client-side validation of model identifiers beyond existing request and response handling.

## Decisions

- Use `oictl tasks config get/set` rather than `oictl config tasks get/set` because the upstream endpoints live under `/api/v1/tasks` and the taxonomy prefers Open WebUI domain nouns for resource families. The alternative would hide a tasks API surface under the existing config family, but that would make endpoint ownership less clear.
- Use the full upstream task config payload for `set` because Open WebUI's update form includes multiple task-generation fields. The CLI will pass the provided JSON body through rather than synthesizing missing fields or exposing narrowly scoped task-model-only mutations.
- Keep task model names as payload fields `TASK_MODEL` and `TASK_MODEL_EXTERNAL` while documenting their backing config keys. This avoids translating between two schemas and keeps examples aligned with Open WebUI API responses.

## Risks / Trade-offs

- Full payload updates can overwrite unrelated task configuration if a user submits stale JSON -> document and test the pass-through behavior so scripts can fetch, edit, then set the complete payload.
- The `tasks` family initially contains only configuration commands -> keep the command scope narrow and do not add unrelated task execution or generation endpoints in this change.
- Open WebUI may alter the task config payload over time -> pass-through JSON handling limits CLI churn and avoids hard-coded field validation.
