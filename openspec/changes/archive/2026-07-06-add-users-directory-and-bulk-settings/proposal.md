## Why

Admins need a reliable CLI path to find target users before applying settings changes, and they need a safe way to roll out UI settings across many users without hand-running one request per account. The current user settings commands only cover the authenticated user's settings and one-off admin UI patches, which makes broad configuration changes error-prone.

## What Changes

- Add user enumeration and search commands under the existing `users` command family, including pagination/query filters passed through to Open WebUI.
- Add a bulk user UI settings update workflow that accepts target users from command arguments or a file, applies the existing sensitive UI key guardrails, and reports a result for each user.
- Add `--dry-run` support for the bulk workflow so admins can validate target selection and payload handling without sending mutation requests.
- Preserve existing current-user settings and single-user admin UI patch behavior.

## Capabilities

### New Capabilities
- `users-directory-control`: Enumerating and searching Open WebUI users from the CLI so admins can discover target user IDs and inspect user directory records.

### Modified Capabilities
- `user-settings-control`: Adds bulk admin UI settings patching with dry-run behavior, target input handling, sensitive key guardrails, and per-user result reporting.

## Impact

- Affects `internal/cli/surfaces.go` user command routing, help text, request construction, bulk execution, and reporting.
- Affects CLI tests in `internal/cli/app_test.go` or adjacent test files for user directory and bulk settings workflows.
- Uses existing Open WebUI user and settings APIs; no new external dependencies are expected.
