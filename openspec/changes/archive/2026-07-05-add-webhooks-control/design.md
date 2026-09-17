## Context

`oictl` is a thin Open WebUI control client with resource-oriented command families and server-authoritative authorization. Open WebUI now has two webhook control planes that are useful to operators: channel incoming webhooks for posting messages into channels, and global event webhooks for sending selected Open WebUI events to external systems.

Channel webhook management is scoped to a channel and is available to channel managers and admins. Event webhook management is admin-only and includes event filters, user/group target filters, enabled state, and outbound destination URLs. Both surfaces involve sensitive data: channel webhook tokens are embedded in public posting URLs, and event webhook URLs often contain external service secrets.

## Goals / Non-Goals

**Goals:**

- Add a discoverable `oictl webhooks` command family covering channel incoming webhooks and global event webhooks.
- Keep commands endpoint-aligned with Open WebUI and reuse existing profile, authentication, request, JSON input, output formatting, confirmation, and error handling foundations.
- Preserve complete JSON output for automation while redacting secret-bearing webhook tokens and URLs from default human-readable output.
- Require explicit user intent before printing or writing full channel webhook URLs, channel webhook tokens, or event destination URLs.
- Surface server 401, 403, 404, and validation responses without replacing Open WebUI authorization or webhook validation locally.

**Non-Goals:**

- Do not implement a webhook receiver, delivery retry system, or outbound event dispatcher in `oictl`.
- Do not add personal user notification webhook settings unless a later proposal scopes user account notification settings.
- Do not post messages through incoming channel webhook URLs as a first-class feature beyond optionally exposing the URL needed by external services.
- Do not reimplement Open WebUI event matching, target matching, URL safety validation, or channel manager rules in the CLI.
- Do not store generated webhook URLs or tokens in local profiles.

## Decisions

1. Use one webhook command family with two explicit subdomains.

   Commands will use `oictl webhooks channels ...` for incoming channel webhooks and `oictl webhooks events ...` for global event webhooks. This keeps webhook discovery in one place while making the different authorization and data models clear. Alternative considered: putting channel webhooks under `oictl channels webhooks`, but a dedicated webhook family better matches the requested scope and can explain shared secret-handling behavior.

2. Keep mutation payloads mostly endpoint-aligned.

   Channel webhook create/update commands can expose simple flags such as `--name` and `--profile-image-url`, while event webhook create/update commands should accept JSON file/stdin payloads and may also expose obvious scalar flags such as `--name`, `--url`, `--enabled`, `--event`, `--target-user`, and `--target-group` if the existing command style supports repeated flags. Alternative considered: fully typed local event webhook schemas only, but event catalog and target formats are server-owned and may evolve.

3. Redact secret-bearing values by default.

   Table and default human output will not print full channel webhook tokens, constructed channel webhook URLs, or event webhook URLs. Commands that need to expose a usable secret-bearing value will require explicit flags such as `--show-url`, `--show-token`, or `--out`, and help text will label those flags as sensitive. JSON output remains the complete server response for automation, but command documentation must warn that JSON can include secret values. Alternative considered: redact JSON too, but that would break backup, auditing, and automation use cases where the server response is the canonical contract.

4. Construct channel webhook URLs client-side only when explicitly requested.

   Open WebUI returns the channel webhook identifier and token, while the UI constructs `{base}/api/v1/channels/webhooks/{webhook_id}/{token}`. The CLI should construct that URL from the configured base URL only for explicit reveal/copy commands and otherwise render non-secret metadata. Alternative considered: require users to manually combine ID and token, but that is error-prone and encourages copying tokens from general output.

5. Preserve server authority for permissions and validation.

   The CLI will send authenticated requests to the Open WebUI instance and surface server responses for non-manager channel access, non-admin event webhook access, invalid event filters, invalid targets, invalid URLs, and missing resources. Local validation should be limited to required CLI arguments, mutually exclusive flags, JSON parsing, and confirmation behavior. Alternative considered: mirroring permission and event catalog checks locally, but that risks drift and false negatives against different Open WebUI versions.

## Risks / Trade-offs

- Secret leakage through default output -> Redact tokens and URLs in table/default output, require explicit reveal flags, and add tests that failures and logs do not include sensitive values.
- JSON output can still contain sensitive values -> Keep JSON canonical for automation, document the risk in help, and avoid making JSON the default for secret-bearing list views.
- Open WebUI endpoint paths or payloads may change -> Keep commands thin and endpoint-aligned, rely on JSON payloads for complex event webhook forms, and test path construction with local HTTP fixtures.
- Channel and event webhook permissions differ -> Use explicit `channels` and `events` subcommands, and surface server authorization errors directly.
- Constructed channel webhook URLs can be wrong behind proxies or custom base URLs -> Build from the resolved CLI base URL and provide `--out` so operators can verify and distribute the value intentionally.

## Migration Plan

No server data migration is required. Existing Open WebUI webhooks remain unchanged until users invoke the new commands. Rollback is removing the new CLI commands; any webhook resources created, updated, or deleted before rollback remain server-side and must be corrected through Open WebUI or API calls.

## Open Questions

- Should event webhook create/update expose first-class repeated filter flags in the first implementation, or start with JSON file/stdin payloads plus minimal scalar flags?
- Should full webhook URL reveal be limited to a dedicated `url` subcommand, or allowed on create/get/list through `--show-url`?
