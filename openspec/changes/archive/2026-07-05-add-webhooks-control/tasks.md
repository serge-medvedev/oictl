## 1. Shared Webhook Foundations

- [x] 1.1 Review existing command registration, HTTP client, output formatting, JSON input, file output, and confirmation helpers to reuse for webhook commands.
- [x] 1.2 Add webhook-specific redaction helpers for channel webhook tokens, constructed channel webhook URLs, and event destination webhook URLs.
- [x] 1.3 Add tests proving default human-readable output and CLI-generated errors do not expose webhook tokens, full webhook URLs, or local credentials.

## 2. Channel Incoming Webhooks

- [x] 2.1 Register `oictl webhooks channels` commands with help text for list, get, create, update, delete, and URL reveal operations.
- [x] 2.2 Implement channel webhook list and get requests against the Open WebUI channel webhook endpoints with JSON output and redacted default output.
- [x] 2.3 Implement channel webhook create and update commands with name and profile image URL inputs, preserving server validation and authorization responses.
- [x] 2.4 Implement channel webhook delete with explicit channel ID, webhook ID, and `--yes` confirmation for non-interactive safety.
- [x] 2.5 Implement explicit channel webhook URL construction from the resolved Open WebUI base URL, webhook ID, and token, supporting reveal to stdout and write to `--out`.
- [x] 2.6 Add HTTP fixture tests for channel webhook paths, payloads, forbidden responses, deletion confirmation, redaction, and explicit URL reveal behavior.

## 3. Global Event Webhooks

- [x] 3.1 Register `oictl webhooks events` commands with help text for catalog, list, get, create, update, enable, disable, and delete operations.
- [x] 3.2 Implement event catalog retrieval from `/api/events` with JSON and human-readable output.
- [x] 3.3 Implement event webhook list and get requests against `/api/events/webhooks` with destination URLs redacted in default output.
- [x] 3.4 Implement event webhook create and update commands using JSON file/stdin payloads and any minimal scalar flags selected during implementation.
- [x] 3.5 Implement event webhook enable and disable commands as update operations that preserve other server-returned webhook fields.
- [x] 3.6 Implement event webhook delete with explicit webhook ID and `--yes` confirmation for non-interactive safety.
- [x] 3.7 Add HTTP fixture tests for event catalog access, webhook lifecycle requests, invalid payload responses, admin authorization failures, enable/disable behavior, and URL redaction.

## 4. Documentation and Verification

- [x] 4.1 Update command documentation or README examples for channel webhook management, event webhook management, and safe URL/token reveal workflows.
- [x] 4.2 Run Go formatting on changed Go files.
- [x] 4.3 Run the relevant Go test suite for webhook command coverage.
- [x] 4.4 Run OpenSpec validation/status checks for `add-webhooks-control`.
