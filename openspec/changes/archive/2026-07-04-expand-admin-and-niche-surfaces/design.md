## Context

`oictl` is planned as a thin Open WebUI control client. Earlier stages define shared target/profile/authentication behavior, direct resource commands, and declarative workspace automation. This change extends that foundation to advanced and niche API surfaces that are valuable for operators but carry higher data-sensitivity or integration risk.

The local Open WebUI source exposes first-class routers for chats, analytics, automations, SCIM, OpenAI-compatible provider traffic, and Ollama-compatible provider traffic. These surfaces are not uniform: chats mix user-owned and admin-enabled access, analytics is admin-only reporting, automations are feature-flagged scheduled execution, SCIM uses a dedicated bearer token and SCIM response shapes, and provider passthrough endpoints can stream or return raw upstream-provider payloads.

## Goals / Non-Goals

**Goals:**

- Add typed Stage 3+ command families for `chats`, `analytics`, `automations`, `scim`, and `providers`.
- Reuse existing target/profile/authentication, request execution, output formatting, body loading, and error handling foundations.
- Preserve canonical JSON output for automation while providing compact table output for list/report commands.
- Support raw request/response paths where passthrough semantics are the value, especially provider endpoints and SCIM payloads.
- Make destructive, sensitive, streaming, and admin-only operations explicit in command names, flags, help text, and tests.

**Non-Goals:**

- Do not add a new daemon, scheduler, analytics warehouse, or offline indexing layer.
- Do not reimplement Open WebUI permission checks, feature flags, automation limits, provider connection policies, or SCIM compliance logic in the CLI.
- Do not model every provider-specific request schema as CLI flags; complex payloads remain JSON file/stdin/body inputs.
- Do not make SCIM use normal user API tokens; it uses the SCIM bearer-token contract exposed by Open WebUI.
- Do not expand this change into all remaining Open WebUI routers such as audio, images, evaluations, calendar, notes, memories, and channels unless explicitly added by a later proposal.

## Decisions

1. Use resource-first command families.

   Commands will follow the established taxonomy: `oictl chats ...`, `oictl analytics ...`, `oictl automations ...`, `oictl scim ...`, and `oictl providers ...`. Deeper nouns are used for subresources, such as `oictl automations runs list`, `oictl analytics models overview`, `oictl scim users patch`, and `oictl providers openai request`. Alternative considered: a broad `admin` command tree, but that would hide user-owned operations such as personal chats and automations and conflict with the resource-first taxonomy.

2. Keep typed commands thin and endpoint-aligned.

   Each command maps closely to the corresponding Open WebUI route and accepts `--data`, `--file`, or stdin for complex payloads. The CLI should normalize command naming, target resolution, output selection, and error presentation, but not duplicate server validation. Alternative considered: fully modeled flags and structs for every payload, but these surfaces evolve quickly and include provider-specific and SCIM-specific schema details.

3. Separate normal API authentication from SCIM authentication.

   Most commands reuse the normal Open WebUI profile token/API key. `oictl scim` resolves a dedicated SCIM bearer token from an explicit flag, `OPEN_WEBUI_SCIM_TOKEN`, or SCIM-specific profile field if profile support already exists. It MUST NOT silently fall back to the normal Open WebUI user token. Alternative considered: reuse `--token` for SCIM, but that makes provisioning scripts more likely to send the wrong credential class.

4. Treat provider passthrough as guarded raw transport.

   Provider commands should support config/verify/model-discovery subcommands and a raw request mode for OpenAI-compatible and Ollama-compatible passthrough paths. Raw provider requests preserve status, headers where safe, streaming bodies, and provider payloads, while still redacting local credentials and applying Open WebUI authorization. Alternative considered: only expose provider configuration commands, but that leaves niche provider endpoints to ad hoc `api` calls and fails the requested passthrough scope.

5. Make destructive and sensitive operations opt-in.

   Commands that delete chats, automations, SCIM users/groups, or bulk-import/export sensitive chat records require explicit IDs or input files and confirmation flags when non-interactive safety would otherwise be ambiguous. Chat exports and analytics reports default to stdout JSON or explicit output files and never print credentials. Alternative considered: rely only on server permissions; server authorization is required but does not protect users from accidental local scripting mistakes.

6. Preserve JSON as the full output contract.

   All commands that return structured data support `--output json`. List/report commands may default to tables, raw provider calls default to raw body output unless `--output json` is explicitly meaningful, and streaming provider responses stream to stdout. Alternative considered: force table output for operator-facing reports, but that would make analytics and provisioning automation fragile.

## Risks / Trade-offs

- Sensitive data exposure through chats, analytics, SCIM records, and provider payloads -> Keep JSON explicit, redact credentials from CLI-generated output, require output-file flags where appropriate, and document admin-only assumptions in help.
- Upstream endpoint churn in advanced/niche routers -> Keep commands endpoint-aligned, use JSON passthrough for complex payloads, and prefer thin clients over deep local models.
- SCIM behavior is experimental upstream -> Scope CLI behavior to transport, metadata discovery, and server-returned SCIM response/error bodies rather than claiming full SCIM conformance.
- Raw provider passthrough can be mistaken for direct provider access -> Command help and errors must state that requests go through the configured Open WebUI instance and remain subject to server provider settings and access checks.
- Streaming responses complicate tests and output formatting -> Isolate streaming transport in the shared client and test it with Go HTTP test servers before wiring all commands.
- Broad scope creates implementation risk -> Implement command families in slices with shared fixtures, starting with read/list/config/verify commands before mutating or streaming commands.

## Migration Plan

No server data migration is required. Existing Stage 1/2 command behavior and profile configuration remain valid. Rollback is removing the new command families; remote mutations already performed through chats, automations, SCIM, or provider passthrough remain on the Open WebUI server and must be corrected there or through prior backups/manifests.

## Open Questions

- Should `providers` expose only `openai` and `ollama` initially, or reserve subcommands for additional provider routers in the first implementation slice?
- Should chat export commands default to stdout, require `--out`, or vary by size-sensitive operations?
- Which destructive commands require `--yes` unconditionally versus only when running without a TTY?
