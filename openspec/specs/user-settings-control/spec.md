# user-settings-control Specification

## Purpose
TBD - created by archiving change add-user-settings-control. Update Purpose after archive.
## Requirements

### Requirement: Deployment extension capability gate
Cross-user UI settings PATCH is a deployment extension, absent from the original Open WebUI API. Both `users ui-settings patch` and `users ui-settings bulk-patch` SHALL require `--allow-ui-settings-extension` as an explicit operator assertion that the target deployment supplies this extension, including dry runs. The CLI SHALL never substitute the current-user settings endpoint.

#### Scenario: Extension support is not acknowledged
- **WHEN** either cross-user UI settings command is invoked without `--allow-ui-settings-extension`
- **THEN** the CLI exits non-zero before discovery or mutation and explains the extension requirement

Unless a scenario tests this gate itself, all cross-user UI settings scenarios below assume `--allow-ui-settings-extension` is supplied and the deployment implements the extension. This flag acknowledges deployment capability; it does not bypass server authentication or authorization.
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

### Requirement: Bulk admin user UI settings patching
The CLI SHALL provide an admin command to patch `settings.ui` for multiple target users using one UI settings payload.

#### Scenario: User UI settings are patched for multiple explicit targets
- **WHEN** the user runs `oictl users ui-settings bulk-patch user-1 user-2 --file ui-settings.json`
- **THEN** the CLI sends the JSON payload to `PATCH /api/v1/users/user-1/settings/ui`
- **THEN** the CLI sends the JSON payload to `PATCH /api/v1/users/user-2/settings/ui`
- **THEN** the CLI prints a result entry for each target user

#### Scenario: User UI settings are patched for targets from a file
- **WHEN** the user runs `oictl users ui-settings bulk-patch --users-file users.txt --file ui-settings.json`
- **THEN** the CLI reads target user IDs from non-empty lines in `users.txt`
- **THEN** the CLI sends one patch request per unique target user in first-seen order
- **THEN** the CLI prints a result entry for each target user

#### Scenario: Bulk UI settings patch requires at least one target
- **WHEN** the user runs `oictl users ui-settings bulk-patch --file ui-settings.json` without positional targets, `--user-id`, or `--users-file`
- **THEN** the CLI exits non-zero without sending a request
- **THEN** the CLI explains that at least one target user is required

#### Scenario: Bulk UI settings patch requires a payload
- **WHEN** the user runs `oictl users ui-settings bulk-patch user-1` without `--data`, `--file`, or `--file -`
- **THEN** the CLI exits non-zero without sending a request
- **THEN** the CLI explains that a UI settings payload is required

### Requirement: Bulk user UI settings dry-run
The CLI SHALL support a dry-run mode for bulk UI settings patching that validates local inputs and reports planned per-user actions without sending mutation requests.

#### Scenario: Bulk UI settings dry-run reports planned targets
- **WHEN** the user runs `oictl users ui-settings bulk-patch user-1 user-2 --file ui-settings.json --dry-run`
- **THEN** the CLI does not send any patch requests
- **THEN** the CLI prints a result entry for `user-1` and `user-2` marked as dry-run planned actions

#### Scenario: Bulk UI settings dry-run still validates sensitive keys
- **WHEN** the user runs `oictl users ui-settings bulk-patch user-1 --file ui-settings.json --dry-run` and the payload contains `toolServers`
- **THEN** the CLI exits non-zero without sending a request
- **THEN** the CLI names `toolServers` as a sensitive UI setting key and explains that `--allow-sensitive-ui-keys` is required

### Requirement: Bulk user UI settings per-user result reporting
The CLI SHALL report the outcome for every target user and return a non-zero exit code when any non-dry-run target fails.

#### Scenario: Bulk UI settings patch reports partial failure
- **WHEN** `oictl users ui-settings bulk-patch user-1 user-2 --file ui-settings.json` succeeds for `user-1` and Open WebUI returns 404 for `user-2`
- **THEN** the CLI prints a success result for `user-1`
- **THEN** the CLI prints a failed result for `user-2` with the response status and detail
- **THEN** the CLI exits non-zero

#### Scenario: Bulk UI settings patch redacts credentials in failures
- **WHEN** Open WebUI returns an error response for a bulk UI settings patch request
- **THEN** the CLI result output and diagnostics do not print configured credentials

### Requirement: Bulk UI settings directory target discovery
The CLI SHALL allow admin bulk UI settings patch targets to be discovered from the Open WebUI user directory using either all users or a server-supported query filter.

#### Scenario: Bulk UI settings dry-run targets all users
- **WHEN** the user runs `oictl users ui-settings bulk-patch --all --file ui-settings.json --dry-run`
- **THEN** the CLI calls `GET /api/v1/users/` to discover users
- **THEN** the CLI does not send any patch requests
- **THEN** the CLI prints one planned result entry per discovered user ID
- **THEN** each planned result identifies the target source as `all`

#### Scenario: Bulk UI settings dry-run targets queried users
- **WHEN** the user runs `oictl users ui-settings bulk-patch --query alice --file ui-settings.json --dry-run`
- **THEN** the CLI calls `GET /api/v1/users/` with `query=alice` to discover users
- **THEN** the CLI does not send any patch requests
- **THEN** the CLI prints one planned result entry per discovered user ID
- **THEN** each planned result identifies the target source as `query` and includes the query text

#### Scenario: Directory target discovery follows pagination
- **WHEN** the user runs `oictl users ui-settings bulk-patch --all --file ui-settings.json --dry-run` and the users listing response spans multiple pages
- **THEN** the CLI fetches additional `GET /api/v1/users/` pages until it has collected the reported total or receives a page with no new users
- **THEN** the CLI de-duplicates discovered user IDs while preserving first-seen order
- **THEN** the CLI prints planned result entries for the discovered unique user IDs

#### Scenario: Directory target discovery requires extractable user IDs
- **WHEN** the user runs `oictl users ui-settings bulk-patch --query alice --file ui-settings.json --dry-run` and the users listing response does not contain any non-empty user `id` values
- **THEN** the CLI exits non-zero without sending any patch requests
- **THEN** the CLI explains that no target user IDs were discovered

### Requirement: Bulk UI settings directory target safety
The CLI SHALL prevent broad directory-derived bulk UI settings mutations unless the operator explicitly confirms the non-dry-run operation.

#### Scenario: All-user bulk UI settings patch requires confirmation
- **WHEN** the user runs `oictl users ui-settings bulk-patch --all --file ui-settings.json` without `--yes`, `--confirm`, or `--dry-run`
- **THEN** the CLI exits non-zero without sending any patch requests
- **THEN** the CLI explains that directory-derived bulk UI settings patches require `--yes` or `--dry-run`

#### Scenario: Queried bulk UI settings patch requires confirmation
- **WHEN** the user runs `oictl users ui-settings bulk-patch --query alice --file ui-settings.json` without `--yes`, `--confirm`, or `--dry-run`
- **THEN** the CLI exits non-zero without sending any patch requests
- **THEN** the CLI explains that directory-derived bulk UI settings patches require `--yes` or `--dry-run`

#### Scenario: Confirmed all-user bulk UI settings patch executes discovered targets
- **WHEN** the user runs `oictl users ui-settings bulk-patch --all --file ui-settings.json --yes`
- **THEN** the CLI discovers target users from `GET /api/v1/users/`
- **THEN** the CLI sends one `PATCH /api/v1/users/{user-id}/settings/ui` request per discovered unique user ID
- **THEN** the CLI prints a result entry for each discovered user

#### Scenario: Confirmed queried bulk UI settings patch executes discovered targets
- **WHEN** the user runs `oictl users ui-settings bulk-patch --query alice --file ui-settings.json --confirm`
- **THEN** the CLI discovers target users from `GET /api/v1/users/` with `query=alice`
- **THEN** the CLI sends one `PATCH /api/v1/users/{user-id}/settings/ui` request per discovered unique user ID
- **THEN** the CLI prints a result entry for each discovered user

#### Scenario: Directory discovery validates sensitive keys before remote requests
- **WHEN** the user runs `oictl users ui-settings bulk-patch --all --data '{"toolServers":{"x":"secret"}}' --dry-run` without `--allow-sensitive-ui-keys`
- **THEN** the CLI exits non-zero without sending directory discovery or patch requests
- **THEN** the CLI names `toolServers` as a sensitive UI setting key and explains that `--allow-sensitive-ui-keys` is required

### Requirement: Bulk UI settings target mode validation
The CLI SHALL reject ambiguous bulk UI settings target mode combinations before discovering remote users or sending mutation requests.

#### Scenario: All-user discovery cannot be combined with query discovery
- **WHEN** the user runs `oictl users ui-settings bulk-patch --all --query alice --file ui-settings.json --dry-run`
- **THEN** the CLI exits non-zero without sending any directory discovery or patch requests
- **THEN** the CLI explains that `--all` and `--query` cannot be combined

#### Scenario: Query discovery requires non-empty query text
- **WHEN** the user runs `oictl users ui-settings bulk-patch --query "" --file ui-settings.json --dry-run`
- **THEN** the CLI exits non-zero without sending any directory discovery or patch requests
- **THEN** the CLI explains that `--query` requires non-empty text

#### Scenario: Directory discovery cannot be combined with explicit targets
- **WHEN** the user runs `oictl users ui-settings bulk-patch user-1 --all --file ui-settings.json --dry-run`
- **THEN** the CLI exits non-zero without sending any directory discovery or patch requests
- **THEN** the CLI explains that directory discovery cannot be combined with positional targets, `--user-id`, or `--users-file`

#### Scenario: Bulk UI settings still requires a target mode
- **WHEN** the user runs `oictl users ui-settings bulk-patch --file ui-settings.json --dry-run` without positional targets, `--user-id`, `--users-file`, `--all`, or `--query`
- **THEN** the CLI exits non-zero without sending any request
- **THEN** the CLI explains that at least one target user or discovery mode is required

