## ADDED Requirements

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
