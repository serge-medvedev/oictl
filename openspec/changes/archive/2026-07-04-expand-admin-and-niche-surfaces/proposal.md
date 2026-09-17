## Why

Stage 3+ needs typed `oictl` coverage for advanced Open WebUI surfaces that operators use during administration, migration, reporting, and integration work. Chats, analytics, automations, SCIM, and provider passthroughs are currently reachable only through low-level API calls, which makes scripts brittle and hides safety expectations around admin-only data, scheduled execution, identity provisioning, and provider proxy traffic.

## What Changes

- Add Stage 3+ chat administration commands for listing, searching, inspecting, exporting/importing, sharing, tagging, archiving, deleting, and compacting chats where Open WebUI exposes these operations.
- Add analytics commands for admin reports over models, users, messages, summaries, daily activity, token usage, and model-specific chat/overview data.
- Add automation commands for scheduled prompt workflows, including list, create, inspect, update, toggle, run-now, delete, and run-history operations.
- Add SCIM commands for provisioning diagnostics and SCIM 2.0 user/group operations using the dedicated SCIM bearer token and SCIM response/error shapes.
- Add provider passthrough commands for guarded OpenAI-compatible and Ollama-compatible calls, including configuration inspection, model discovery, connection verification, and raw passthrough requests for niche provider endpoints.
- Preserve existing Stage 1/2 command contracts and keep Open WebUI server authorization, feature flags, quotas, and provider access checks authoritative.
- No breaking changes.

## Capabilities

### New Capabilities
- `chats-control`: Typed control of Open WebUI chat records, chat metadata/workflows, imports/exports, sharing, tags, archival state, deletion, compaction, and admin chat access behavior.
- `analytics-control`: Admin reporting commands for Open WebUI analytics endpoints, including model, user, message, summary, daily, token, and model-detail reports.
- `automations-control`: Typed control of scheduled automation resources, including lifecycle, activation state, manual execution, schedule validation, and run history.
- `scim-control`: SCIM 2.0 provisioning commands for service metadata, users, groups, patch/update/delete operations, and SCIM-specific authentication/error handling.
- `provider-passthrough-control`: Guarded provider passthrough commands for OpenAI-compatible and Ollama-compatible endpoints where raw provider semantics are needed.

### Modified Capabilities

None.

## Impact

- Affected code: `cmd/oictl`, `internal/cli`, shared API transport, output formatting, request body loading, streaming/raw response handling, command tests, and likely new internal packages for chat, analytics, automation, SCIM, and provider passthrough clients.
- Affected commands: new Stage 3+ resource families under `chats`, `analytics`, `automations`, `scim`, and `providers`.
- Affected APIs: Open WebUI REST endpoints under `/api/v1/chats`, `/api/v1/analytics`, `/api/v1/automations`, `/api/v1/scim/v2`, `/openai`, and `/ollama` as exposed by the target instance.
- Affected security posture: commands may expose sensitive chat content, analytics, identity provisioning data, schedules, and provider responses; implementation must preserve token redaction, explicit destructive-operation confirmation, and server-side authorization boundaries.
- Dependencies: Prefer existing shared client and standard-library behavior; add dependencies only if needed for robust table/report rendering or SCIM/provider payload handling.
