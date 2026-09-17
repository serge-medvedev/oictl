## 1. Shared Foundations

- [x] 1.1 Review existing CLI command registration, profile/auth resolution, HTTP transport, output formatting, and body-loading helpers from earlier stages.
- [x] 1.2 Add or extend shared helpers for query parameters, repeated flags, JSON file/stdin payload loading, raw body output, output files, and status-oriented HTTP errors.
- [x] 1.3 Add or extend streaming response support for provider passthrough commands without buffering full response bodies.
- [x] 1.4 Add confirmation handling for destructive commands in non-interactive contexts.
- [x] 1.5 Add credential redaction tests covering normal Open WebUI tokens, SCIM tokens, and provider passthrough diagnostics.

## 2. Chats Control

- [x] 2.1 Register the `chats` command family with help text for list, search, get, export, import, share, tags, archive, delete, and compact operations.
- [x] 2.2 Implement chat list and search commands with pagination, filters, JSON output, and table output for list responses.
- [x] 2.3 Implement chat get, export, and import commands with file/stdin input and output-file support where applicable.
- [x] 2.4 Implement chat workflow commands for share, tag updates, archive/pin state where supported, and compaction payloads.
- [x] 2.5 Implement chat delete with explicit ID handling and `--yes` confirmation.
- [x] 2.6 Add HTTP server tests for chat command request paths, query parameters, payload handling, output formats, and forbidden responses.

## 3. Analytics Control

- [x] 3.1 Register the `analytics` command family with help text for models, users, messages, summary, daily, tokens, and model-detail reports.
- [x] 3.2 Implement common analytics filters for start date, end date, group ID, user ID, model ID, chat ID, skip, and limit where supported by each report.
- [x] 3.3 Implement model, user, summary, daily, and token report commands with complete JSON output and table output for list-style reports.
- [x] 3.4 Implement message query and model-detail report commands, including model chat and model overview endpoints.
- [x] 3.5 Add tests for admin success responses, non-admin/invalid-token failures, filter serialization, and output formatting.

## 4. Automations Control

- [x] 4.1 Register the `automations` command family with help text for list, create, get, update, toggle, run, delete, and runs list operations.
- [x] 4.2 Implement automation list and get commands with pagination/status filters and JSON/table output.
- [x] 4.3 Implement automation create and update commands using JSON file/stdin payloads without replacing server-side RRULE validation.
- [x] 4.4 Implement automation toggle and run-now commands and surface server feature-flag, quota, and permission errors.
- [x] 4.5 Implement automation delete with explicit ID handling and `--yes` confirmation.
- [x] 4.6 Implement automation run-history listing and add tests for lifecycle requests, invalid schedules, run history, and disabled-feature responses.

## 5. SCIM Control

- [x] 5.1 Register the `scim` command family with help text for service-provider-config, resource-types, schemas, users, and groups.
- [x] 5.2 Implement SCIM token resolution from explicit flag, `OPEN_WEBUI_SCIM_TOKEN`, and any supported SCIM-specific profile field without falling back to normal API tokens.
- [x] 5.3 Implement SCIM service metadata commands for ServiceProviderConfig, ResourceTypes, and Schemas.
- [x] 5.4 Implement SCIM users list, get, create, replace, patch, and delete commands with SCIM pagination/filter support and JSON payload passthrough.
- [x] 5.5 Implement SCIM groups list, get, create, replace, patch, and delete commands with SCIM pagination/filter support and JSON payload passthrough.
- [x] 5.6 Preserve SCIM error bodies in JSON/error output and add tests for token handling, request paths, 204 delete success, and SCIM error responses.

## 6. Provider Passthrough Control

- [x] 6.1 Register the `providers` command family with `openai` and `ollama` subcommands and help text that explains requests route through Open WebUI.
- [x] 6.2 Implement provider config get/update and verify commands for OpenAI-compatible and Ollama-compatible routes where Open WebUI exposes them.
- [x] 6.3 Implement provider model discovery commands for OpenAI-compatible models and Ollama-compatible tags/version/process endpoints included in scope.
- [x] 6.4 Implement guarded raw request commands for OpenAI-compatible and Ollama-compatible passthrough paths with method, headers, JSON/file/stdin body, and streaming support.
- [x] 6.5 Reject arbitrary external provider URLs and unsafe Authorization header overrides unless an explicit safe override is added for a command.
- [x] 6.6 Add tests for passthrough path construction, raw response/status handling, streaming output, forbidden config updates, and credential redaction.

## 7. Documentation and Verification

- [x] 7.1 Update README or command documentation with examples for chats, analytics, automations, SCIM token usage, and provider passthrough commands.
- [x] 7.2 Run Go formatting on changed Go files.
- [x] 7.3 Run the full Go test suite.
- [x] 7.4 Run OpenSpec validation/status checks for `expand-admin-and-niche-surfaces`.
