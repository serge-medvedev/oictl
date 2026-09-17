## Why

Open WebUI exposes runtime valve configuration for tools and functions, but oictl does not provide resource-oriented commands to inspect schemas, read current values, or update those settings. Adding these commands makes plugin configuration scriptable while preserving Open WebUI's existing authorization and validation behavior.

## What Changes

- Add valve operations under the `tools` command family for global valves, global valve specs, user valves, user valve specs, and updates.
- Add valve operations under the `functions` command family for global valves, global valve specs, user valves, user valve specs, and updates.
- Preserve server-side validation by submitting valve payloads to Open WebUI rather than interpreting valve models locally.
- Surface missing-valve, missing-resource, validation, and authorization responses consistently with other resource-control commands.
- Update command taxonomy expectations so `tools` and `functions` are treated as implemented resource families with nested valve operations.

## Capabilities

### New Capabilities
- `valves-control`: Commands for inspecting and updating global and user-scoped valve values and valve specs for Open WebUI tools and functions.

### Modified Capabilities
- `cli-command-taxonomy`: Classify `tools` and `functions` as implemented resource families and require valve operations to remain nested under those families.

## Impact

- Affected CLI command families: `oictl tools` and `oictl functions`.
- Affected Open WebUI API surfaces: `/api/v1/tools/id/{id}/valves`, `/api/v1/tools/id/{id}/valves/spec`, `/api/v1/tools/id/{id}/valves/update`, `/api/v1/tools/id/{id}/valves/user`, `/api/v1/tools/id/{id}/valves/user/spec`, `/api/v1/tools/id/{id}/valves/user/update`, `/api/v1/functions/id/{id}/valves`, `/api/v1/functions/id/{id}/valves/spec`, `/api/v1/functions/id/{id}/valves/update`, `/api/v1/functions/id/{id}/valves/user`, `/api/v1/functions/id/{id}/valves/user/spec`, and `/api/v1/functions/id/{id}/valves/user/update`.
- No new external dependencies are expected.
