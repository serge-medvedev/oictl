## ADDED Requirements

### Requirement: Current-user settings inspection
The CLI SHALL provide a command to fetch the authenticated user's Open WebUI settings object.

#### Scenario: Current user settings are displayed
- **WHEN** the user runs `oictl users settings get`
- **THEN** the CLI calls `GET /api/v1/users/user/settings`
- **THEN** the CLI prints the returned settings response

### Requirement: Current-user settings update
The CLI SHALL provide a command to replace or update the authenticated user's Open WebUI settings with a JSON payload accepted by the server.

#### Scenario: Current user settings are updated from a file
- **WHEN** the user runs `oictl users settings update --file settings.json`
- **THEN** the CLI sends the JSON payload to `POST /api/v1/users/user/settings/update`
- **THEN** the CLI prints the returned settings response

#### Scenario: Current user settings update requires a payload
- **WHEN** the user runs `oictl users settings update` without `--data`, `--file`, or `--file -`
- **THEN** the CLI exits non-zero without sending a request
- **THEN** the CLI explains that a settings payload is required

### Requirement: Admin user UI settings patching
The CLI SHALL provide an admin command to patch another user's `settings.ui` object without replacing the rest of that user's settings object.

#### Scenario: User UI settings are patched by admin
- **WHEN** the user runs `oictl users ui-settings patch user-123 --file ui-settings.json`
- **THEN** the CLI sends the JSON payload to `PATCH /api/v1/users/user-123/settings/ui`
- **THEN** the CLI prints the returned settings response

#### Scenario: User UI settings patch requires a target user
- **WHEN** the user runs `oictl users ui-settings patch --file ui-settings.json`
- **THEN** the CLI exits non-zero without sending a request
- **THEN** the CLI prints usage for `oictl users ui-settings patch <user-id>`

#### Scenario: User UI settings patch requires a payload
- **WHEN** the user runs `oictl users ui-settings patch user-123` without `--data`, `--file`, or `--file -`
- **THEN** the CLI exits non-zero without sending a request
- **THEN** the CLI explains that a UI settings payload is required

### Requirement: Sensitive UI setting key guardrails
The CLI SHALL reject user settings mutations that include known sensitive UI setting keys unless the user explicitly opts in with `--allow-sensitive-ui-keys`.

#### Scenario: Sensitive key in current-user settings update is rejected by default
- **WHEN** the user runs `oictl users settings update --file settings.json` and the payload contains `ui.toolServers`
- **THEN** the CLI exits non-zero without sending a request
- **THEN** the CLI names `toolServers` as a sensitive UI setting key and explains that `--allow-sensitive-ui-keys` is required

#### Scenario: Sensitive key in admin UI settings patch is rejected by default
- **WHEN** the user runs `oictl users ui-settings patch user-123 --file ui-settings.json` and the payload contains `toolServers`
- **THEN** the CLI exits non-zero without sending a request
- **THEN** the CLI names `toolServers` as a sensitive UI setting key and explains that `--allow-sensitive-ui-keys` is required

#### Scenario: Sensitive key mutation is sent when explicitly allowed
- **WHEN** the user runs `oictl users ui-settings patch user-123 --allow-sensitive-ui-keys --file ui-settings.json` and the payload contains `toolServers`
- **THEN** the CLI sends the payload to `PATCH /api/v1/users/user-123/settings/ui`
- **THEN** the CLI does not print submitted sensitive values in CLI-generated diagnostics

### Requirement: User settings authorization and errors
The CLI SHALL preserve Open WebUI authentication, authorization, validation, and not-found failures for user settings commands.

#### Scenario: User settings request is unauthorized
- **WHEN** Open WebUI returns 401 or 403 for a user settings command
- **THEN** the CLI exits non-zero and prints the response status and detail without printing configured credentials

#### Scenario: Admin UI patch target is missing
- **WHEN** Open WebUI returns 404 for `oictl users ui-settings patch missing-user --file ui-settings.json`
- **THEN** the CLI exits non-zero and prints the response status and detail
