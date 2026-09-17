## ADDED Requirements

### Requirement: CLI exposes function commands
The CLI SHALL expose `oictl functions` as the command family for Open WebUI function records, including filter records represented by server-derived function type `filter`.

#### Scenario: Help lists function commands
- **WHEN** a user runs `oictl functions --help`
- **THEN** the help output lists list, get, create, update, delete, export, load-url, sync, toggle, and toggle-global operations

#### Scenario: Help explains filter handling
- **WHEN** a user reads `oictl functions` help output
- **THEN** the help output states that Open WebUI filters are managed as functions whose returned `type` is `filter`

### Requirement: Function commands list and inspect records
The CLI SHALL provide commands to list function records and retrieve a single function record by identifier while preserving the server-returned record fields.

#### Scenario: User lists functions
- **WHEN** a user runs `oictl functions list`
- **THEN** the CLI requests the Open WebUI function list endpoint and renders returned function identifiers, names, types, active state, global state, and timestamps

#### Scenario: User lists filters
- **WHEN** a user runs `oictl functions list --type filter`
- **THEN** the CLI renders only returned function records whose server-returned `type` is `filter`

#### Scenario: User retrieves a function
- **WHEN** a user runs `oictl functions get <function-id>`
- **THEN** the CLI returns the matching Open WebUI function record without dropping source content or metadata in JSON output

### Requirement: Function commands manage lifecycle records
The CLI SHALL provide create, update, and delete commands for Open WebUI function records using JSON payloads compatible with the server function form.

#### Scenario: Function is created
- **WHEN** an admin runs `oictl functions create --file function.json`
- **THEN** the CLI submits the JSON payload to Open WebUI and prints the created function response with the server-derived `type`

#### Scenario: Filter is created as a function
- **WHEN** an admin creates a function whose source is loaded by Open WebUI as a filter
- **THEN** the CLI prints the created function response with `type` equal to `filter`

#### Scenario: Function is updated
- **WHEN** an admin runs `oictl functions update <function-id> --file function.json`
- **THEN** the CLI submits the JSON payload to the Open WebUI update endpoint and prints the updated function response

#### Scenario: Function is deleted with confirmation
- **WHEN** an admin runs `oictl functions delete <function-id> --yes`
- **THEN** the CLI requests deletion of that function and reports the server result

#### Scenario: Function delete lacks confirmation
- **WHEN** an admin runs `oictl functions delete <function-id>` in a non-interactive context without confirmation
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that confirmation is required

### Requirement: Function commands support export load-url and sync workflows
The CLI SHALL provide Open WebUI-backed export, load-url, and sync commands for function records.

#### Scenario: Functions are exported
- **WHEN** an admin runs `oictl functions export --out functions.json`
- **THEN** the CLI requests the Open WebUI function export endpoint and writes the returned JSON to the requested output path

#### Scenario: Function export includes valves
- **WHEN** an admin runs `oictl functions export --include-valves --out functions.json`
- **THEN** the CLI requests exported functions with valves included where Open WebUI supports them

#### Scenario: Function source is loaded from a URL
- **WHEN** an admin runs `oictl functions load-url https://example.invalid/function.py`
- **THEN** the CLI asks Open WebUI to fetch the URL and renders the returned function name and source content

#### Scenario: Functions are synchronized
- **WHEN** an admin runs `oictl functions sync --file functions.json --yes`
- **THEN** the CLI submits the function list to Open WebUI sync and prints the reconciled function records returned by the server

#### Scenario: Function sync lacks confirmation
- **WHEN** an admin runs `oictl functions sync --file functions.json` in a non-interactive context without confirmation
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that sync can remove remote functions omitted from the input

### Requirement: Function commands toggle active and global state
The CLI SHALL provide commands to toggle a function's active state and global state using Open WebUI server operations.

#### Scenario: Function active state is toggled
- **WHEN** an admin runs `oictl functions toggle <function-id>`
- **THEN** the CLI requests the Open WebUI function toggle operation and prints the resulting function state

#### Scenario: Function global state is toggled
- **WHEN** an admin runs `oictl functions toggle-global <function-id>`
- **THEN** the CLI requests the Open WebUI global-toggle operation and prints the resulting function state

### Requirement: Function commands preserve server validation and authorization
The CLI SHALL rely on Open WebUI authorization, plugin loading, and validation for function operations and SHALL surface server failures consistently.

#### Scenario: Admin-only function action is forbidden
- **WHEN** Open WebUI returns 401 or 403 for a function command
- **THEN** the CLI exits non-zero and includes the response status and detail in stderr

#### Scenario: Plugin loading fails
- **WHEN** Open WebUI rejects a create, update, load-url, or sync request because function source cannot be fetched, loaded, or validated
- **THEN** the CLI exits non-zero and prints the server validation detail without executing the plugin source locally
