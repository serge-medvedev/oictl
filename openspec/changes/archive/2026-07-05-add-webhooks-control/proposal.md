## Why

Open WebUI now exposes both incoming channel webhooks and global event webhooks, but `oictl` has no typed way to inspect or manage them from automation. This change adds CLI control over those webhook surfaces while treating generated webhook tokens and outbound webhook URLs as sensitive values.

## What Changes

- Add a `oictl webhooks` command family for channel incoming webhook management and admin event webhook management.
- Support listing, creating, updating, deleting, and retrieving/copying generated channel webhook URLs for channel managers and admins.
- Support listing the event catalog and managing global event webhook definitions, including enabled state, event filters, target filters, and destination URL.
- Redact webhook tokens and webhook URLs in human-readable output by default, with explicit flags required to reveal or write full secret-bearing values.
- Preserve Open WebUI server authorization and validation behavior rather than attempting local permission bypasses or client-only schema enforcement.

## Capabilities

### New Capabilities
- `webhooks-control`: CLI commands for managing Open WebUI channel incoming webhooks and global event webhooks with safe output handling for webhook tokens and URLs.

### Modified Capabilities

None.

## Impact

- Affected CLI surfaces: new `oictl webhooks` command family, help text, output rendering, and command tests.
- Affected Open WebUI APIs: channel webhook endpoints under `/api/v1/channels/{channel_id}/webhooks...`, public channel webhook URL construction, event catalog at `/api/events`, and event webhook endpoints under `/api/events/webhooks...`.
- Security impact: secret-bearing webhook tokens and destination URLs must be redacted unless the user explicitly requests full output.
