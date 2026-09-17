## Why

Channel incoming webhooks are currently manageable only when callers already know a webhook ID, which makes repeatable automation awkward and pushes users toward manual lookup steps. A first-class ensure-by-name workflow lets scripts create a channel webhook only when absent, detect ambiguous names safely, and optionally verify or reveal the resulting URL without leaking secrets by default.

## What Changes

- Add a channel webhook ensure-by-name workflow under `oictl webhooks channels`.
- Resolve existing channel webhooks by display name before creating a new webhook.
- Create the webhook when no existing webhook with the requested name is found.
- Fail safely when multiple existing channel webhooks match the requested name.
- Preserve the existing default secret redaction behavior while supporting explicit URL reveal, file output, or verification for the ensured webhook.
- No breaking changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `webhooks-control`: Adds first-class channel webhook ensure-by-name behavior with create-if-absent, duplicate detection, and safe URL output or verification.

## Impact

- Affected CLI: `oictl webhooks channels` help and dispatch.
- Affected implementation: channel webhook list/create lookup flow, URL construction, redaction, and output handling in `internal/cli/surfaces.go`.
- Affected tests: channel webhook command tests in `internal/cli/webhooks_test.go`.
- External systems: Open WebUI channel incoming webhook endpoints already used by the existing webhook commands.
