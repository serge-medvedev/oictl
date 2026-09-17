## Context

`oictl` is currently a small Go CLI with only help and version handling. Stage 1 needs to turn it into a practical Open WebUI control client while keeping the implementation simple enough to extend across later domains.

Open WebUI already exposes the needed REST APIs under `/api/v1`. The CLI should be a thin, predictable control layer over those APIs rather than a second business-logic implementation. The implementation must support both interactive operator use and automation in scripts.

## Goals / Non-Goals

**Goals:**

- Add a reusable API client, profile/config resolution, authentication token handling, input loading, output formatting, and consistent errors.
- Implement Stage 1 commands for models, config, files, knowledge, and auth/profile basics.
- Preserve machine-readable JSON output for all commands and useful table output for list-style commands.
- Keep command behavior close to Open WebUI API semantics so new endpoints can be added with little redesign.

**Non-Goals:**

- Do not implement chats, users/groups administration, prompts, tools/functions, channels, analytics, evaluations, audio/images, or Open Terminal file-system operations.
- Do not add a long-lived daemon, plugin system, or generated client in Stage 1.
- Do not persist passwords. Sign-in may persist only the returned bearer token when explicitly requested or when setting a profile token.
- Do not mask or reinterpret Open WebUI authorization rules; the server remains authoritative.

## Decisions

1. Use a domain/action command tree.

   Commands will follow `oictl <domain> <action>` with deeper nouns only when needed, such as `oictl auth api-key get` or `oictl knowledge dirs create`. This keeps command names discoverable and maps cleanly to Open WebUI router groupings. Alternative considered: one command per endpoint, but that would produce an unwieldy flat namespace.

2. Build a small internal HTTP client instead of generating one.

   The API surface needed for Stage 1 is broad but uses conventional JSON, multipart upload, and streaming/download responses. A hand-written client wrapper can centralize base URL joining, bearer tokens, request timeouts, JSON encoding/decoding, multipart uploads, and HTTP error handling without introducing OpenAPI generation complexity. Alternative considered: generate from OpenAPI, but the local source is the primary contract and not all operational endpoints are equally suited to generated types.

3. Resolve target settings through explicit flags, environment, then named profiles.

   Global flags such as `--base-url`, `--token`, `--profile`, `--output`, and `--timeout` take precedence over environment variables, which take precedence over the selected profile. Profiles will live under the user's config directory, store named base URLs and optional tokens, and use restrictive permissions when token material is written. Alternative considered: environment-only configuration, but it is inconvenient for operators managing multiple Open WebUI instances.

4. Prefer JSON passthrough forms for complex Open WebUI payloads.

   Commands that create or update complex resources will accept `--data` JSON or `--file` JSON input and send the payload to the corresponding endpoint. Simple high-frequency values can have first-class flags where they are unambiguous. Alternative considered: model every request as CLI flags, but that would quickly lag Open WebUI's evolving schema and produce brittle command parsing.

5. Make JSON the complete output contract and table output a convenience layer.

   Every command that returns structured data must support `--output json`. List commands may default to a compact table, while detail and mutation commands may default to JSON or concise status text where appropriate. Downloads and file-content commands must support writing raw bytes to `--out` without table formatting. Alternative considered: table-only human output, but that would make automation unreliable.

6. Keep dependencies minimal at first.

   The initial implementation should use the Go standard library unless a dependency materially reduces risk or complexity. A CLI framework or table renderer can be introduced during implementation only if the command tree becomes error-prone without it. Alternative considered: immediately adopting a CLI framework, but the current codebase is small and Stage 1 can validate structure before committing to a dependency.

## Risks / Trade-offs

- Broad command scope → Mitigation: implement shared request/input/output primitives first, then add thin endpoint commands in small tested slices.
- Open WebUI payload schemas may evolve → Mitigation: support JSON passthrough for complex forms and avoid overfitting request structs unless the CLI adds value.
- Token persistence can leak credentials → Mitigation: use restrictive file permissions, redact tokens from errors, and avoid storing passwords.
- Table output can become inconsistent → Mitigation: make JSON canonical and test table headers only for list commands.
- Multipart and download behavior differs from JSON endpoints → Mitigation: isolate upload/download helpers and cover them with HTTP test servers.
- Some commands require admin permissions → Mitigation: document and surface server 401/403 responses directly with clear command context.

## Migration Plan

No data migration is required. Existing `help` and `version` behavior should continue to work. Rollback is reverting the CLI changes; any profile config files created by users are local client state and do not affect Open WebUI server data.

## Open Questions

- Should `auth login` save the token by default when a named profile is selected, or require an explicit `--save` flag?
- Should destructive commands require `--yes` in Stage 1, or rely on non-interactive command intent?
