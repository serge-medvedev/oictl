## Context

`oictl tasks` currently manages Open WebUI task configuration through `tasks config get` and `tasks config set`. Task model settings are stored in the task config payload as `TASK_MODEL` and `TASK_MODEL_EXTERNAL`, while Open WebUI loads default skills from the selected model metadata at `model.info.meta.skillIds` during chat completion processing.

Models and skills are already exposed by `oictl`, and model manifests already preserve `meta`, but there is no task-focused command that resolves the configured task model and appends skill IDs safely.

## Goals / Non-Goals

**Goals:**
- Add a task-scoped workflow for attaching one or more existing skills to the configured task model.
- Resolve skills by exact ID or unique exact name.
- Preserve existing model fields and metadata while adding missing skill IDs to `meta.skillIds` without duplicates.
- Keep error behavior deterministic for unresolved or ambiguous resources.

**Non-Goals:**
- Creating, updating, activating, or granting access to skills.
- Changing Open WebUI task model selection semantics.
- Adding a separate task-model resource or manifest kind.
- Removing skills from task models in this change.

## Decisions

1. Add `oictl tasks skills attach [--external] <skill-id-or-name>...`.

   Rationale: The workflow belongs under `tasks` because the target model comes from task config, but the operation is specifically skill attachment. `--external` maps to `TASK_MODEL_EXTERNAL`; without it the command resolves `TASK_MODEL`.

   Alternative considered: add `oictl task-model ...`. This conflicts with existing documentation that task model settings are config fields, not a separate resource.

2. Resolve skills using the skill list endpoint before updating the model.

   Rationale: The skill list endpoint provides IDs and names in one call, allowing exact ID matches and unique exact name matches. This avoids probing individual skill endpoints for names and keeps ambiguity checks local.

   Alternative considered: treat every argument as an ID first and fall back to `skills get`. This cannot resolve by name without extra search semantics and gives weaker ambiguity detection.

3. Fetch the configured model before posting the update.

   Rationale: The model update endpoint expects a full model payload. Fetching the model first lets the command preserve existing fields and nested `meta` values while only changing `meta.skillIds`.

   Alternative considered: send a minimal update payload with only `id`, `name`, and `meta.skillIds`. This risks clearing fields if the server treats missing fields as replacement values.

4. Treat attachment as idempotent append.

   Rationale: Re-running the command should not create duplicate IDs or report a spurious change. The command will keep existing `skillIds` order and append newly resolved IDs in argument order.

   Alternative considered: sort all skill IDs. Stable append is less surprising because it preserves existing server order.

## Risks / Trade-offs

- Task config may omit the selected task model key -> Return a clear non-zero error before model or skill mutation.
- Skill names may not be unique -> Require exact unique name matches; ambiguous names fail without mutation.
- Existing `meta.skillIds` may contain non-string values -> Preserve string values and reject malformed arrays rather than silently deleting unknown data.
- Model update payload shape may differ between Open WebUI versions -> Reuse existing model payload handling and test against the current API shape.
