# API behavior and compatibility

[Project overview and quick start](../README.md) · [Connection settings and profiles](configuration.md) · [Workflow examples](workflows.md) · [Manifest guide](manifests.md)

This page describes resource-specific behavior and deployment constraints. For exact flags, required fields, and input examples, use the action's offline `--help`. Connection precedence, timeout defaults, output protection, and common input rules are documented in [connection settings and profiles](configuration.md).

## Deployment requirements

`oictl` uses the APIs exposed by your Open WebUI instance. Endpoint availability, permissions, configured features, and request fields can vary by deployment. This guide does not establish a minimum Open WebUI release or guarantee compatibility with every version; check the APIs your deployment exposes before using a command.

Normal commands authenticate with an API key or JWT and remain subject to server authorization. SCIM uses a separate provisioning token. Provider commands route through the configured Open WebUI instance and reject external provider URLs or `Authorization` header overrides.

Cross-user UI settings patches are explicitly deployment-specific, as described [below](#user-settings). Do not assume that ordinary Open WebUI provides that extension.

## Model and function inventories

Model import/sync accepts `{"models":[...]}` or an array shorthand that the CLI wraps in that envelope. Omitted `params` defaults inside each model. Function sync similarly accepts `{"functions":[...]}` or an array.

Missing or wrongly shaped inventories are rejected. An intentionally empty model sync requires `--yes`; all function syncs require confirmation. These native inventory operations differ from [declarative manifest reconciliation](manifests.md).

## Export formats

### Chats

`chats export` exports only the authenticated user's chats and preserves the server's NDJSON format. `--all-users` explicitly selects the administrative database export instead, producing JSON for all users.

### Skills

`skills export` and `skills export <skill-id>` default to JSON **packages**, retaining the root `SKILL.md` and all supporting files on stdout or with `--out`.

Packages contain `id`, `name`, `description`, `meta`, `files`, and `is_active`. They do not contain ACL grants, owner, revision history, or `version_id`.

`skills get` returns root content and access/version metadata, not a package backup. `skills export <skill-id> --format manifest --out skill.md` writes only root Markdown and bounded frontmatter fields; it cannot back up supporting files or complex metadata/grants.

## Skill input and revisions

Skill create/update JSON requires `id` and `name`; `content` is optional.

- Creation accepts `files` or `data.files`, not both, as a package requiring UTF-8 `SKILL.md`. Without package/content input, it creates an empty root.
- Metadata-only updates preserve omitted description, metadata, active state, and files.
- Content-only updates replace the root while retaining supporting files.
- Non-null `files`, `data`, or `operations` updates are mutually exclusive. They require the current `expected_version_id` from `skills get`: absence returns 400, and a mismatch returns 409.
- Optional `commit_message` records a revision message.

Markdown input carries only supported frontmatter and the root body. Use JSON for packages, file operations, or complex metadata/grants.

## Task model configuration

Task model settings are managed through `oictl tasks config`. In the Open WebUI payload:

- `TASK_MODEL` maps to configuration key `task.model.default`.
- `TASK_MODEL_EXTERNAL` maps to `task.model.external`.

There is no separate `task-model` or `task-models` CLI resource. `oictl tasks skills attach <skill-id-or-name>...` attaches existing skills to the configured task model; `--external` targets `TASK_MODEL_EXTERNAL` rather than `TASK_MODEL`.

## Channels and webhooks

`channels list` returns the caller's complete accessible collection. It does not support page, query, or sort flags. Channel manifests use the administration inventory, subject to server authorization.

Channel webhook creation requires a name. Flag-based webhook updates preserve omitted current metadata; raw JSON updates remain deliberate replacement payloads.

Webhook table output redacts channel webhook tokens, constructed incoming webhook URLs, and event destination URLs by default. JSON preserves complete responses and may contain secrets. `webhooks channels url --show-url` explicitly prints a full incoming URL; `--out path` writes it without printing it to the terminal.

## User settings

Ordinary `users settings get/update` uses the original current-user API. `users settings update` shallow-patches `ui` keys:

- Omitted keys remain unchanged.
- A per-key `null` removes that key.
- A supplied nested object replaces that key's value rather than recursively merging it.
- `{}`, `{"ui":{}}`, and `{"ui":null}` do not clear stored UI preferences.

Server permissions may filter supplied settings. To remove `toolServers`, use `{"ui":{"toolServers":null}}` with `--allow-sensitive-ui-keys`; that acknowledgement does not bypass server permission checks.

Cross-user `users ui-settings patch` and `bulk-patch` require a deployment-specific `PATCH /api/v1/users/{id}/settings/ui` extension that the original Open WebUI API does not provide. Supply `--allow-ui-settings-extension` only for a deployment that implements it, including when previewing with `--dry-run`.

## Pagination and analytics

Chat list/search and automation list pagination uses `--page` with a server-defined page size. `--limit` and `--skip` are rejected on those actions. Chat search requires `--query TEXT` (alias `--q`). Automation **runs list** supports `--limit`/`--skip`.

Analytics filters are report-specific:

- Models, summary, and tokens accept dates and group.
- Users additionally accepts limit.
- Daily additionally accepts granularity.
- Model chats accepts dates, skip, limit, order, and direction.
- Model overview accepts days.
- Messages accepts dates, model, user, chat, skip, and limit.

Messages selector precedence is **chat > model > user**, not AND. The chat branch ignores dates/skip/limit; the user branch ignores dates; the model branch uses dates/skip/limit. No selector returns an empty list. Unsupported report options fail before HTTP.

## SCIM PATCH

SCIM user PATCH supports `add`/`replace` on:

- `active`
- `userName`
- `displayName`
- `emails[primary eq true].value`
- `name.formatted`
- `externalId`

Nonempty pathless objects keyed by supported paths are also accepted. Only `externalId` may be removed. Invalid operations, paths, or typed values return 400 protocol errors rather than being ignored; operations validate before persistence. The request uses a SCIM `Operations` envelope, not JSON Patch.

## File uploads

File uploads send a filename/media-derived multipart MIME type, with a binary fallback for unknown content. Open WebUI uses that type for media handling, such as configured audio transcription. `--metadata` is a separate JSON text field.

## Terminal server access grants

Terminal-server `access-grants get/diff/set --out path` writes the JSON result to the selected destination without also printing it on stdout. See the [terminal server manifest guide](manifests.md#terminal-server-connection) for declarative connection and grant management.
