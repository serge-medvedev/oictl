## Why

Administrators can list and search native Open WebUI users with `oictl`, but must leave the CLI to inspect, create, update, or delete individual users. Adding the corresponding native API adapters completes the small user-directory command surface without changing SCIM provisioning or existing settings behavior.

## What Changes

- Add authenticated `oictl users get <user-id>` support.
- Add authenticated `oictl users create`, `update`, and confirmed `delete` commands using pass-through JSON payloads and existing output/error handling.
- Require exact user-id arguments, required create/update payloads, escaped path segments, and explicit deletion confirmation before any request.
- Document the native administration commands and their distinction from `oictl scim users`.
- Preserve existing `users list`, `search`, `settings`, and `ui-settings` behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `users-directory-control`: Extend the native user directory with get, create, update, and safely confirmed delete operations, including request, output, help, validation, and failure contracts.

## Impact

- `internal/cli/surfaces.go`: Extend `runUsers` dispatch and help.
- `internal/cli/app_test.go`: Add focused `httptest` request-contract and validation coverage.
- `README.md`: Add native user administration examples and sensitive-input guidance.
- Open WebUI endpoints: `/api/v1/users/{id}`, `/api/v1/auths/add`, and `/api/v1/users/{id}/update`.
- No new dependencies, manifest kinds, transport abstractions, compatibility layers, or live user mutations during verification.
