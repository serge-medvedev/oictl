## ADDED Requirements

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
