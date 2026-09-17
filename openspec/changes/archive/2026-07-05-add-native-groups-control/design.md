## Context

`oictl` already exposes SCIM groups under `oictl scim groups`, authenticated with a dedicated SCIM token and backed by `/api/v1/scim/v2/Groups`. Open WebUI also exposes native groups under `/api/v1/groups`, which are used by Open WebUI access grants, share permissions, and membership checks. The native API has a separate data shape, uses normal Open WebUI bearer authentication, and includes native-only operations such as `info`, `export`, `preview`, and direct user membership changes.

This change adds a top-level `groups` resource family aligned with the existing command taxonomy. It does not alter SCIM provisioning semantics.

## Goals / Non-Goals

**Goals:**

- Add first-class CLI coverage for native Open WebUI groups.
- Keep native groups clearly distinct from SCIM groups in command names, authentication, endpoints, and payload expectations.
- Cover native group list, create, get, info, export, update, delete, preview, and user membership add/remove/list operations.
- Reuse existing target resolution, token handling, structured output, `--file`/`--data` request bodies, `--out`, and destructive confirmation behavior.

**Non-Goals:**

- Do not change `oictl scim groups` commands or SCIM authentication.
- Do not add declarative manifest support for groups in this change.
- Do not implement client-side role or permission policy; Open WebUI remains authoritative for authorization.
- Do not invent a local group schema beyond passing the native Open WebUI request and response JSON.

## Decisions

### Native command family

Use `oictl groups` for native Open WebUI groups and keep SCIM operations under `oictl scim groups`.

Rationale: the command taxonomy reserves domain nouns for Open WebUI resources and already lists `groups` as a future domain noun. Keeping SCIM nested under `scim` prevents accidental use of SCIM credentials or SCIM payloads against native group endpoints.

Alternative considered: `oictl native-groups`. This is more explicit but less aligned with the resource-first taxonomy and would make the normal Open WebUI resource family name less discoverable.

### Endpoint mapping

Map commands directly to Open WebUI native group endpoints:

- `groups list` -> `GET /api/v1/groups/`
- `groups create` -> `POST /api/v1/groups/create`
- `groups get <id>` -> `GET /api/v1/groups/id/{id}`
- `groups info <id>` -> `GET /api/v1/groups/id/{id}/info`
- `groups export <id>` -> `GET /api/v1/groups/id/{id}/export`
- `groups update <id>` -> `POST /api/v1/groups/id/{id}/update`
- `groups delete <id>` -> `DELETE /api/v1/groups/id/{id}/delete`
- `groups preview <id>` -> `GET /api/v1/groups/id/{id}/preview`
- `groups users list <id>` -> `POST /api/v1/groups/id/{id}/users`
- `groups users add <id> <user-id...>` -> `POST /api/v1/groups/id/{id}/users/add`
- `groups users remove <id> <user-id...>` -> `POST /api/v1/groups/id/{id}/users/remove`

Rationale: the native router uses action-style endpoints rather than REST-only collection URLs. Matching upstream avoids translation ambiguity and keeps request construction testable.

Alternative considered: hide upstream action names behind a more REST-like internal client. That adds indirection without improving the CLI contract for this small command surface.

### Payload handling

Use existing structured input helpers for create and update payloads, requiring `--file`, `--data`, or stdin through `--file -`. Membership add/remove accepts one or more positional user ids and sends `{"user_ids":[...]}`. If a payload is supplied for membership commands, implementation may also support it, but positional ids are the primary documented path.

Rationale: create and update payloads mirror native `GroupForm` and `GroupUpdateForm`; membership commands are common enough to deserve a shell-friendly path that does not force users to write JSON for a single id.

Alternative considered: require JSON payloads for membership changes. That is simpler internally but makes routine operations unnecessarily awkward.

### Output and safety

Use structured output defaults consistent with existing administrative resource commands: table output for list-style responses where rendering succeeds, JSON when requested with `--output json`, and raw body writing via `--out`. Require `--yes` or `--confirm` before native group deletion contacts the server.

Rationale: users can inspect groups interactively while automation can request JSON or write exports to a file. Delete safety matches other destructive command families.

Alternative considered: always output JSON. That is automation-friendly but inconsistent with newer typed command surfaces that default list responses to tables.

## Risks / Trade-offs

- Native endpoint names may differ across Open WebUI versions -> Keep mappings centralized in the command implementation and add request-construction tests for every command.
- Membership list uses a POST endpoint even though it is read-like -> Preserve upstream behavior and document/test the method explicitly.
- Native and SCIM groups can represent different concepts with overlapping names -> Keep command families and authentication flows separate, and include help text that names native Open WebUI groups.
- Preview output can grow with accessible resources -> Do not transform the server response beyond existing structured output handling; allow `--out` and `--output json` for automation.
- Server authorization varies by user role -> Surface Open WebUI status and response details without client-side policy guesses.

## Migration Plan

No data migration is required. Existing commands continue to work. Rollback removes the new `groups` command family without affecting SCIM provisioning commands or stored profiles.

## Open Questions

- None for the proposal. Endpoint behavior is based on the local Open WebUI native groups router.
