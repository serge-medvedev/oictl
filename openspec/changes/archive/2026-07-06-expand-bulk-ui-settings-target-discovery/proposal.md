## Why

Bulk UI settings patching currently requires operators to enumerate target user IDs manually or maintain a separate newline-delimited file. For tenant-wide or query-scoped rollouts, that adds avoidable scripting around an existing user directory API and makes broad mutations harder to audit before execution.

## What Changes

- Extend `oictl users ui-settings bulk-patch` with target discovery from the user directory.
- Add `--all` to target all users returned by the admin users listing endpoint.
- Add `--query text` to discover targets using the same server-supported user query filter exposed by `oictl users list`.
- Require safe behavior for broad discovered mutations, including explicit operator acknowledgement before non-dry-run directory-derived patches are sent.
- Expand dry-run output so discovered-target runs show which users would be patched and how the target set was selected, without sending mutation requests.
- Keep existing positional, `--user-id`, and `--users-file` target modes working unchanged.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `user-settings-control`: bulk admin UI settings patching gains directory-derived target discovery with `--all` and `--query`, guardrails for broad mutations, and richer dry-run planning output.

## Impact

- Affects `internal/cli/surfaces.go` users UI settings bulk-patch parsing, target collection, directory calls, dry-run result shaping, and help text.
- Adds or updates CLI tests for `--all`, `--query`, mutual-exclusion/validation errors, broad-mutation acknowledgement, dry-run output, and pagination if needed by the directory endpoint shape.
- Uses existing Open WebUI admin user directory APIs; no Open WebUI server changes or new dependencies are expected.
