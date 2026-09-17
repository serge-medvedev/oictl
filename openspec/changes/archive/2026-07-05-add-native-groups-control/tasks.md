## 1. Command Surface

- [x] 1.1 Add `groups` to top-level command dispatch and help text as a native Open WebUI resource family.
- [x] 1.2 Add `oictl groups --help` covering list, create, get, info, export, update, delete, preview, and users membership commands.
- [x] 1.3 Ensure native `groups` commands use normal Open WebUI target resolution and API token authentication, not SCIM token resolution.

## 2. Native Group Operations

- [x] 2.1 Implement `groups list` using `GET /api/v1/groups/` with supported query flags such as `--share`.
- [x] 2.2 Implement `groups get`, `groups info`, `groups export`, and `groups preview` against the native `/api/v1/groups/id/{id}` endpoints.
- [x] 2.3 Implement `groups create` and `groups update` with required JSON payload handling from `--data`, `--file`, or `--file -`.
- [x] 2.4 Implement `groups delete` with `--yes` or `--confirm` required before sending `DELETE /api/v1/groups/id/{id}/delete`.

## 3. Membership Operations

- [x] 3.1 Implement `groups users list <group-id>` using `POST /api/v1/groups/id/{id}/users`.
- [x] 3.2 Implement `groups users add <group-id> <user-id...>` with a `{"user_ids":[...]}` JSON body.
- [x] 3.3 Implement `groups users remove <group-id> <user-id...>` with a `{"user_ids":[...]}` JSON body.
- [x] 3.4 Reject membership add/remove commands before contacting Open WebUI when no user ids or payload are supplied.

## 4. Output And Errors

- [x] 4.1 Reuse structured output handling for table and JSON output on native group responses.
- [x] 4.2 Support `--out` for export and other raw response writing where existing command helpers support it.
- [x] 4.3 Surface native group HTTP errors with status and server response details while redacting configured API tokens.

## 5. Tests And Documentation

- [x] 5.1 Add command help tests for top-level `groups` and `oictl groups --help`.
- [x] 5.2 Add request-construction tests for list, create, get, info, export, update, delete, preview, and users list/add/remove.
- [x] 5.3 Add tests proving native group commands do not require SCIM tokens and `oictl scim groups` remains separate.
- [x] 5.4 Add validation tests for required payloads, required membership user ids, delete confirmation, `--out`, and token redaction.
- [x] 5.5 Update user-facing command documentation if the repository has a generated or maintained CLI reference for typed command families.
