## Why

Operators can inspect and update current-user profile basics today, but user settings and UI preferences still require raw `oictl api` calls. Open WebUI exposes dedicated settings endpoints, including admin-only UI patching for another user, and the CLI needs typed commands that preserve server authorization while avoiding accidental exposure or unsafe mutation of sensitive UI setting keys.

## What Changes

- Add typed `users settings` commands for fetching and updating the authenticated user's full settings object.
- Add an admin-only `users ui-settings patch` command for patching another user's `settings.ui` map without replacing the rest of their settings.
- Accept JSON payloads from `--data`, `--file`, or stdin consistently with existing payload commands.
- Protect sensitive UI setting keys by default: current-user updates shall not include restricted UI keys unless explicitly allowed, and command output/errors shall not leak sensitive submitted values.
- Preserve Open WebUI server-side authorization, permission filtering, response shapes, and validation errors.
- No breaking changes.

## Capabilities

### New Capabilities
- `user-settings-control`: Typed CLI control of Open WebUI current-user settings and admin user UI settings patching, including safe handling of sensitive UI setting keys.

### Modified Capabilities

None.

## Impact

- Affected code: `cmd/oictl`, `internal/cli`, shared request body loading, output handling, help text, and command tests.
- Affected commands: new `users settings` and `users ui-settings` command family.
- Affected APIs: Open WebUI `/api/v1/users/user/settings`, `/api/v1/users/user/settings/update`, and `/api/v1/users/{user_id}/settings/ui`.
- Affected security posture: CLI must preserve token redaction, avoid logging or persisting submitted settings values, warn/block unsafe current-user UI key updates by default, and rely on Open WebUI for final authorization and permission enforcement.
- Dependencies: none expected; use existing standard-library CLI and HTTP helpers.
