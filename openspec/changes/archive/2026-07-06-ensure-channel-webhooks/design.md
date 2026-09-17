## Context

`oictl webhooks channels` already supports list, get, create, update, delete, and URL reveal operations for Open WebUI channel incoming webhooks. Those commands work by channel ID and webhook ID, redact webhook tokens and incoming URLs in default human-readable output, and only reveal a full incoming URL through explicit `channels url` options.

Automation that needs a webhook for a named integration currently has to list webhooks, match by name externally, decide whether to create, and then separately reveal or store the URL. The ensure workflow should make that sequence first-class without weakening the existing secret redaction behavior.

## Goals / Non-Goals

**Goals:**

- Add `oictl webhooks channels ensure <channel-id> --name <name>` as an idempotent, name-based workflow.
- Reuse a single existing webhook with the requested name.
- Create a webhook when no matching name exists.
- Refuse to proceed when duplicate matching names make the result ambiguous.
- Support explicit URL reveal, file output, and non-revealing URL verification for the ensured webhook.
- Preserve default redaction for tokens and full webhook URLs.

**Non-Goals:**

- Enforce global uniqueness of channel webhook names on the server.
- Update or reconcile an existing webhook's mutable fields during ensure.
- Add declarative manifest support for channel webhooks.
- Change event webhook behavior.

## Decisions

- Implement ensure as a new `channels ensure` action rather than overloading `create`.
  Alternative considered: add `--if-absent` to `channels create`. A dedicated action is clearer for scripts because it communicates lookup-before-create behavior and can have ensure-specific duplicate and URL handling rules.

- Match existing webhooks by exact trimmed `name` equality.
  Alternative considered: case-insensitive or partial matching. Exact matching avoids surprising collisions and mirrors the name value sent to Open WebUI.

- Treat more than one matching webhook as a hard error.
  Alternative considered: reuse the first returned match. Failing is safer because Open WebUI does not make webhook names unique and server ordering may not be stable.

- Do not update an existing webhook during ensure.
  Alternative considered: update fields like `profile_image_url` when a match exists. Not updating keeps ensure idempotent and avoids mutating an existing integration unexpectedly; users can run `channels update` when they intend reconciliation.

- Share the existing URL construction and redaction logic where practical.
  Alternative considered: return any `url` field supplied by Open WebUI. Constructing from base URL, webhook ID, and token keeps behavior consistent with `channels url` and avoids trusting an arbitrary returned URL shape.

- Add `--verify-url` as a non-revealing URL handling mode.
  Alternative considered: always verify URL construction silently. Making verification explicit keeps default ensure output focused on metadata and gives automation a clear mode that confirms token availability without exposing the resulting URL.

## Risks / Trade-offs

- Duplicate names can already exist on the server, causing ensure to fail for some users. Mitigation: print a clear ambiguous-name error that identifies the channel/name context without secret-bearing values, and leave cleanup to explicit list/delete/update commands.
- Existing webhooks may not include tokens in list responses on all Open WebUI versions. Mitigation: URL reveal and verification should report that ID/token data is missing without printing partial secrets; metadata-only ensure can still succeed when a single match is found.
- `--out` overlaps with the global output-file flag. Mitigation: preserve existing command behavior by resolving command `--out` and global `--out` consistently, and treat it as the explicit URL file target for ensure when used with URL handling.
- Ensure performs list-before-create and is not atomic. Mitigation: document duplicate detection behavior and avoid client-side claims of server-enforced uniqueness.
