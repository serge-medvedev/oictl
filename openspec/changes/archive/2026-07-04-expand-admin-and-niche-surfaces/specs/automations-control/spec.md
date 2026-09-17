## ADDED Requirements

### Requirement: CLI exposes automation commands
The CLI SHALL expose `oictl automations` as the command family for Open WebUI scheduled automation resources.

#### Scenario: Help lists automation commands
- **WHEN** a user runs `oictl automations --help`
- **THEN** the help output lists automation list, create, get, update, toggle, run, delete, and runs operations that are implemented in this capability

### Requirement: Automation commands honor feature flags and permissions
Automation commands SHALL use normal Open WebUI API authentication and SHALL surface server responses for disabled automation features, exceeded limits, and missing permissions.

#### Scenario: Automations are disabled
- **WHEN** a user runs `oictl automations list` against an Open WebUI instance with automations disabled
- **THEN** the CLI exits non-zero and prints the server authorization or feature-disabled error

#### Scenario: User exceeds automation limit
- **WHEN** Open WebUI rejects an automation creation because the user has reached an automation limit
- **THEN** the CLI exits non-zero and prints the server error without retrying or altering the schedule locally

### Requirement: Automation commands manage automation lifecycle
The automation command family SHALL support listing, creating, retrieving, updating, toggling, manually running, and deleting automations through Open WebUI.

#### Scenario: User lists automations
- **WHEN** a user runs `oictl automations list --page 1 --status active`
- **THEN** the CLI sends the page and status filters to Open WebUI and renders the returned automation list

#### Scenario: User creates an automation
- **WHEN** a user runs `oictl automations create --file automation.json`
- **THEN** the CLI reads the JSON payload, sends it to Open WebUI, and returns the created automation response

#### Scenario: User updates an automation
- **WHEN** a user runs `oictl automations update <automation-id> --file automation.json`
- **THEN** the CLI sends the replacement automation payload to Open WebUI and returns the updated automation response

#### Scenario: User toggles an automation
- **WHEN** a user runs `oictl automations toggle <automation-id>`
- **THEN** the CLI requests the server toggle operation and prints the resulting automation state

#### Scenario: User runs an automation immediately
- **WHEN** a user runs `oictl automations run <automation-id>`
- **THEN** the CLI requests immediate server execution and reports the server result

### Requirement: Automation commands preserve server schedule validation
Automation commands SHALL send recurrence rule and timezone-sensitive schedule data to Open WebUI and SHALL NOT replace server-side schedule validation with local-only validation.

#### Scenario: Server rejects an invalid recurrence rule
- **WHEN** a user creates an automation with an invalid RRULE payload
- **THEN** the CLI exits non-zero and prints the server validation message

#### Scenario: Server returns next runs
- **WHEN** a user runs `oictl automations get <automation-id> --output json`
- **THEN** the CLI includes the server-returned schedule, last-run, and next-run data in JSON output

### Requirement: Automation run history is inspectable
The automation command family SHALL expose automation run history for a selected automation.

#### Scenario: User lists automation runs
- **WHEN** a user runs `oictl automations runs list <automation-id>`
- **THEN** the CLI requests the automation run history from Open WebUI and renders the returned run entries

### Requirement: Automation destructive operations require explicit intent
The automation command family SHALL require an explicit automation identifier and confirmation for deletion in non-interactive contexts.

#### Scenario: User deletes an automation with confirmation
- **WHEN** a user runs `oictl automations delete <automation-id> --yes`
- **THEN** the CLI sends the delete request to Open WebUI and prints the server result

#### Scenario: User omits delete confirmation
- **WHEN** a user runs `oictl automations delete <automation-id>` in a non-interactive context without `--yes`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that confirmation is required
