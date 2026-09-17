## Why

Task automation in Open WebUI uses the configured task model, but `oictl` currently only gets or replaces task config JSON. Administrators need a workflow that attaches existing skills to that task model without manually fetching models, resolving skills, and editing `meta.skillIds` by hand.

## What Changes

- Add a task workflow that resolves the active configured task model from `TASK_MODEL` or `TASK_MODEL_EXTERNAL` according to Open WebUI task model selection rules.
- Allow users to attach skills to the resolved task model by skill ID or unique skill name.
- Update the resolved model's `meta.skillIds` idempotently, preserving existing model metadata and avoiding duplicate skill IDs.
- Surface clear errors for missing task model config, unresolved models, unknown skills, ambiguous skill names, and failed remote updates.

## Capabilities

### New Capabilities
- `task-model-skill-attachment`: Attaching Open WebUI skills to the configured task model through `oictl tasks`.

### Modified Capabilities

## Impact

- Affected CLI areas: `internal/cli/app.go`, task command help, model and skill API request helpers, and tests.
- Affected Open WebUI APIs: `/api/v1/tasks/config`, `/api/v1/models/list`, `/api/v1/models/model`, `/api/v1/models/model/update`, and `/api/v1/skills/list`.
- No new external dependencies are expected.
