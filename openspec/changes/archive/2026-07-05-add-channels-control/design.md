## Context

oictl already has shared command parsing, target resolution, authenticated HTTP execution, structured JSON input, table/JSON output, and confirmation helpers for destructive operations. Open WebUI exposes channel collaboration APIs under `/api/v1/channels`, including channel records, members, messages, pinned messages, threads, and message reactions. Those APIs already enforce global channel enablement, feature permissions, channel membership, access grants, author checks, and admin-only behavior.

## Goals / Non-Goals

**Goals:**
- Add a typed `oictl channels` command family for Open WebUI channel collaboration resources.
- Preserve Open WebUI as the source of authorization decisions by forwarding requests with the resolved bearer token and surfacing server denials.
- Reuse existing oictl input and output behavior for JSON payloads, files, stdin, `--output`, and `--out`.
- Keep mutation payloads pass-through so channel and message schemas track the target Open WebUI instance.
- Require explicit confirmation before destructive channel or message deletion requests are sent.

**Non-Goals:**
- Do not implement a local channel data model, cache, or offline state reconciliation.
- Do not add declarative manifest support for channels in this change.
- Do not expose channel webhooks, channel file attachment management, or socket/event streaming in this change.
- Do not bypass, duplicate, or weaken Open WebUI authorization checks in the CLI.

## Decisions

1. Use a resource-first command family rooted at `oictl channels`.

   Commands will use nested subresources for channel-owned collections, such as `oictl channels members list <channel-id>`, `oictl channels messages post <channel-id>`, `oictl channels pins list <channel-id>`, and `oictl channels reactions add <channel-id> <message-id>`. This follows the existing taxonomy while avoiding a new top-level `messages` family that could be confused with chat messages or analytics reports.

   Alternative considered: add top-level `oictl channel-messages`. This was rejected because the messages are scoped by channel and the existing taxonomy prefers Open WebUI domain nouns grouped by resource family.

2. Map typed commands directly to Open WebUI channel endpoints.

   The implementation should build request paths under `/api/v1/channels` and let the server return the authoritative payload. oictl may provide ergonomic flags for common query parameters like `--page`, `--skip`, `--limit`, `--query`, `--order-by`, and `--direction`, but it should not reshape channel, member, message, pin, or reaction response bodies beyond existing output rendering.

   Alternative considered: use the generic `api` command only. This was rejected because the change's purpose is a discoverable, stable typed CLI surface for channel operations.

3. Treat create and update bodies as pass-through JSON.

   Commands that create or update channels and messages, update member active state, pin/unpin messages, or add/remove reactions should accept existing structured input sources (`--data`, `--file`, and stdin where supported by shared helpers). Convenience flags may synthesize the same server payload only for small scalar operations, such as `--name` for a reaction or `--active true` for the caller's member activity state, but explicit JSON input remains authoritative.

   Alternative considered: define complete CLI-native structs for channel and message payloads. This was rejected because Open WebUI channel payloads include evolving `data`, `meta`, `access_grants`, threading, and file metadata fields that should pass through unchanged.

4. Preserve server authorization and error semantics.

   Every channel command requiring Open WebUI access should use the normal authenticated target resolution path and set authentication required. The CLI should not infer membership, grant access, author ownership, or admin capability locally. Server 401, 403, and 404 responses should cause non-zero exits with redacted error output.

   Alternative considered: pre-check access locally to print friendlier errors. This was rejected because it risks diverging from the server and leaking information about resources the caller cannot access.

5. Use explicit confirmation for destructive operations.

   `channels delete` and `channels messages delete` should require `--yes` before contacting Open WebUI. Non-destructive updates, including pin/unpin and reaction add/remove, do not require confirmation because they are reversible and consistent with existing mutation command behavior.

   Alternative considered: require confirmation for all mutations. This was rejected as inconsistent with existing typed commands and too noisy for routine collaboration workflows.

## Risks / Trade-offs

- Endpoint shape differs across Open WebUI versions -> Keep command request construction thin, preserve raw server errors, and document that unsupported endpoints fail with the server status.
- Pass-through payloads are less guided than fully typed flags -> Provide clear help text and support JSON output so users can inspect server payloads, while avoiding stale local schemas.
- Nested subcommands increase parser complexity -> Reuse existing manual command parsing patterns and add focused tests for every route.
- Channel operations can expose sensitive collaboration content -> Rely on server authorization and existing secret redaction; avoid logging tokens or request bodies in error paths.

## Migration Plan

No data migration is required. The change adds CLI commands that call existing Open WebUI APIs. Rollback is removing the new command dispatch and tests; no remote state is transformed by installing the CLI.

## Open Questions

- None for proposal scope. Webhooks, channel file attachment management, declarative manifests, and socket streaming remain intentionally out of scope for future changes.
