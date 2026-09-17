## 1. Command Surface

- [x] 1.1 Update `oictl webhooks` help and channel command usage to include `channels ensure <channel-id> --name name` and URL handling options.
- [x] 1.2 Add `ensure` dispatch under `runWebhookChannels` with argument validation for `<channel-id>` and a non-empty `--name`.

## 2. Ensure Workflow

- [x] 2.1 Implement channel webhook lookup by exact trimmed `name` using the existing channel webhook list endpoint.
- [x] 2.2 Return the single matching webhook without creating a new one when a match exists.
- [x] 2.3 Create a new channel webhook with the supplied mutation fields when no matching webhook exists.
- [x] 2.4 Detect duplicate name matches, exit non-zero, and avoid create/update/delete side effects.

## 3. URL Handling And Redaction

- [x] 3.1 Reuse or refactor URL construction so ensure can build the incoming webhook URL from base URL, webhook ID, and token.
- [x] 3.2 Implement mutually exclusive `--show-url`, `--out`, and `--verify-url` modes for ensured webhooks.
- [x] 3.3 Preserve default metadata output redaction so ensure does not print full tokens or full incoming webhook URLs unless explicitly requested.
- [x] 3.4 Ensure duplicate, authorization, missing-token, and validation errors are redacted consistently with existing webhook commands.

## 4. Verification

- [x] 4.1 Add tests for reusing an existing named webhook, creating when absent, and failing on duplicate names.
- [x] 4.2 Add tests for `--show-url`, `--out`, `--verify-url`, URL mode exclusivity, and default redaction.
- [x] 4.3 Run the Go test suite for the CLI package.
