## ADDED Requirements

### Requirement: CLI exposes analytics commands
The CLI SHALL expose `oictl analytics` as the command family for Open WebUI administrative analytics reports.

#### Scenario: Help lists analytics reports
- **WHEN** an admin runs `oictl analytics --help`
- **THEN** the help output lists model, user, message, summary, daily, token, and model-detail analytics reports that are implemented in this capability

### Requirement: Analytics commands require server-side admin authorization
Analytics commands SHALL use normal Open WebUI API authentication and SHALL rely on Open WebUI to authorize admin-only analytics access.

#### Scenario: Non-admin requests analytics
- **WHEN** a non-admin user runs `oictl analytics summary`
- **THEN** the CLI sends the authenticated request and surfaces the server 401 or 403 response without fabricating analytics data

#### Scenario: Credential is invalid
- **WHEN** the configured Open WebUI credential is invalid and the user runs an analytics command
- **THEN** the CLI exits non-zero and its generated diagnostics do not include the credential value

### Requirement: Analytics commands support common report filters
Analytics commands SHALL support server-backed time range and group filters where the corresponding Open WebUI analytics endpoint accepts them.

#### Scenario: Admin filters by time range
- **WHEN** an admin runs `oictl analytics models --start-date 1700000000 --end-date 1700600000`
- **THEN** the CLI sends the start and end timestamps as query parameters and renders the returned model analytics

#### Scenario: Admin filters by group
- **WHEN** an admin runs `oictl analytics users --group-id <group-id>`
- **THEN** the CLI sends the group filter to Open WebUI and renders the returned user analytics

### Requirement: Analytics commands produce machine-readable reports
Analytics commands SHALL support complete JSON output for every report and MAY provide table output for list-style summaries.

#### Scenario: Admin requests JSON output
- **WHEN** an admin runs `oictl analytics tokens --output json`
- **THEN** the CLI writes the full Open WebUI token analytics response as JSON

#### Scenario: Admin requests default table output for a list report
- **WHEN** an admin runs `oictl analytics users`
- **THEN** the CLI renders a compact table containing stable columns from the returned user analytics entries

### Requirement: Analytics commands expose message and model detail reports
Analytics commands SHALL expose message query and model-detail reports with the filters supported by Open WebUI.

#### Scenario: Admin queries messages by model
- **WHEN** an admin runs `oictl analytics messages --model-id <model-id> --limit 25`
- **THEN** the CLI requests analytics messages for that model and limit and returns the server response

#### Scenario: Admin gets model overview
- **WHEN** an admin runs `oictl analytics models overview <model-id>`
- **THEN** the CLI requests the model overview analytics endpoint and returns the server response

#### Scenario: Message query has no filter
- **WHEN** an admin runs `oictl analytics messages` without a model, user, or chat filter
- **THEN** the CLI either returns the server's empty result or validation error exactly according to the target Open WebUI behavior
