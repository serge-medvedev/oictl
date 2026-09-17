## Context

`oictl` already has shared request execution, JSON payload loading, profile/token resolution, and broad resource-oriented command families. Current-user profile updates are available under `auth`, but Open WebUI user settings are a separate users API surface under `/api/v1/users`.

Open WebUI exposes three relevant endpoints: `GET /api/v1/users/user/settings`, `POST /api/v1/users/user/settings/update`, and admin-only `PATCH /api/v1/users/{user_id}/settings/ui`. User settings are schemaless beyond the top-level `ui` object, and UI settings can include operationally sensitive keys such as `toolServers` that may contain external server definitions or credentials.

## Goals / Non-Goals

**Goals:**

- Add a `users` command family for typed user settings operations that follows the existing resource-first taxonomy.
- Support current-user settings get/update and admin user UI settings patching with existing JSON payload conventions.
- Keep the CLI a thin transport layer over Open WebUI while adding client-side guardrails for sensitive UI setting keys.
- Avoid leaking submitted sensitive settings through CLI-generated validation or error messages.

**Non-Goals:**

- Do not add full user administration, user listing, group management, role changes, deletion, status updates, or permission editing.
- Do not invent a typed schema for Open WebUI settings or transform settings keys beyond explicit safety validation.
- Do not override Open WebUI authorization, permission filtering, event publishing, or server-side stripping of unauthorized settings.
- Do not persist fetched or submitted user settings in local profiles or other client state.

## Decisions

1. Introduce `users` as the command family.

   Commands will use `oictl users settings get`, `oictl users settings update`, and `oictl users ui-settings patch <user-id>`. This follows the existing resource-first taxonomy and leaves room for future user administration without placing settings operations under `auth`. Alternative considered: add `auth settings`, but admin patching another user's UI settings is a user administration action rather than authentication/profile state.

2. Reuse raw JSON passthrough for payloads.

   Settings are intentionally flexible in Open WebUI, so update and patch commands will accept `--data`, `--file`, and `--file -` payloads and send the JSON object to the server. The CLI will validate only enough JSON structure to detect sensitive UI keys before mutation. Alternative considered: typed flags for common UI settings, but that would lag upstream settings and risk lossy writes.

3. Require explicit opt-in for sensitive UI setting key writes.

   Payloads that include known sensitive UI setting keys require `--allow-sensitive-ui-keys`; the initial sensitive key set includes `toolServers`. This applies to both current-user settings updates and admin UI patches. The CLI will fail before sending the request when the flag is absent. Alternative considered: rely only on Open WebUI server filtering, but that still allows accidental admin writes and produces behavior that is hard to distinguish from server-side permission stripping.

4. Preserve server responses as the output contract.

   Successful commands will print the server response using existing output handling. The CLI will not redact normal stdout data returned by the requested settings endpoint because the operator explicitly asked for that data, but CLI-generated validation errors and transport diagnostics must not echo payload contents. Alternative considered: redact sensitive keys in all output, but that would make settings backup, review, and admin patch verification incomplete.

5. Keep implementation in the current CLI structure.

   The existing `App.Run`, `parseCommandFlags`, `requestBody`, `execute`, and `executeStructured` helpers are sufficient. The change can add a small `runUsers` handler and a JSON inspection helper without introducing new packages or dependencies. Alternative considered: refactor command routing first, but this change is narrow and does not justify a larger architecture change.

## Risks / Trade-offs

- Sensitive-key list may become stale as Open WebUI adds new UI settings -> Mitigation: keep the helper centralized and cover it with tests so new keys can be added safely.
- Requiring `--allow-sensitive-ui-keys` may surprise admins patching `toolServers` intentionally -> Mitigation: produce a clear validation error naming the key and required flag without printing values.
- JSON passthrough can send malformed or semantically invalid payloads -> Mitigation: validate object shape only where needed and let Open WebUI return authoritative schema or permission errors.
- Server responses may contain sensitive settings on stdout -> Mitigation: treat stdout as the explicit command result, keep files written with restrictive permissions through existing output helpers, and avoid duplicating values in stderr diagnostics.

## Migration Plan

No migration is required. The change only adds commands and does not alter existing command behavior or local profile data. Rollback is removing the new `users` command handler and tests.

## Open Questions

- Should future work add a `users get/list` administration surface under the same command family, or keep this change's `users` family settings-only until a broader users proposal is accepted?
