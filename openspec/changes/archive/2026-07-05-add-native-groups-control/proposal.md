## Why

Open WebUI has native groups used by its own access-control model, but `oictl` currently only exposes SCIM groups. Operators need first-class control of native groups so they can inspect, manage, export, preview, and adjust memberships without confusing those resources with SCIM provisioning groups.

## What Changes

- Add a native `oictl groups` command family distinct from `oictl scim groups`.
- Support native group list, create, get, info, export, update, delete, and preview operations.
- Support native group user membership add, remove, and list operations.
- Preserve SCIM group commands under `oictl scim groups` with SCIM authentication and SCIM payloads.
- Use normal Open WebUI API target resolution, authentication, input, output, and error behavior for native group commands.

## Capabilities

### New Capabilities

- `native-groups-control`: Native Open WebUI group management, export/preview workflows, and user membership operations.

### Modified Capabilities

- None.

## Impact

- CLI command tree: adds `groups` as a top-level native Open WebUI resource family.
- Open WebUI API client layer: adds native group CRUD, export/preview, and membership request handling.
- Output and input handling: reuses existing structured JSON/file output conventions and confirmation behavior for destructive deletes.
- Tests and documentation: adds coverage for native group command dispatch, request construction, output behavior, error handling, and separation from SCIM groups.
