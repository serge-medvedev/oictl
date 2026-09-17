## 1. Command Routing And Help

- [x] 1.1 Add `users` to top-level command routing and help output.
- [x] 1.2 Add `runUsers` help for `settings` and `ui-settings` subcommands.
- [x] 1.3 Add usage validation for `oictl users settings <get|update>` and `oictl users ui-settings patch <user-id>`.

## 2. User Settings Operations

- [x] 2.1 Implement `oictl users settings get` using `GET /api/v1/users/user/settings`.
- [x] 2.2 Implement `oictl users settings update` using `POST /api/v1/users/user/settings/update` with required JSON payload input.
- [x] 2.3 Implement `oictl users ui-settings patch <user-id>` using `PATCH /api/v1/users/{user_id}/settings/ui` with required JSON payload input and URL-escaped user IDs.
- [x] 2.4 Reuse existing output handling so successful commands print the server response and support existing global output/file flags where applicable.

## 3. Sensitive UI Setting Safeguards

- [x] 3.1 Add `--allow-sensitive-ui-keys` as a parsed boolean command flag.
- [x] 3.2 Add a centralized helper that detects known sensitive UI keys in mutation payloads, initially including `toolServers`.
- [x] 3.3 Reject current-user settings updates containing `ui.toolServers` unless `--allow-sensitive-ui-keys` is set.
- [x] 3.4 Reject admin UI settings patch payloads containing `toolServers` unless `--allow-sensitive-ui-keys` is set.
- [x] 3.5 Ensure CLI-generated validation errors name sensitive keys but do not print submitted sensitive values.

## 4. Tests And Verification

- [x] 4.1 Add command tests for current-user settings get and update request method/path/body behavior.
- [x] 4.2 Add command tests for admin UI settings patch request method/path/body behavior and missing user-id or payload validation.
- [x] 4.3 Add tests for sensitive-key rejection and explicit `--allow-sensitive-ui-keys` opt-in.
- [x] 4.4 Add tests confirming Open WebUI 401, 403, and 404 responses are surfaced without credential leakage.
- [x] 4.5 Run the repository Go test suite.
