## ADDED Requirements

### Requirement: CLI exposes nested valve commands for tools and functions
The CLI SHALL expose valve operations beneath the owning Open WebUI resource families using `oictl tools valves` and `oictl functions valves`.

#### Scenario: Tool valve help is available
- **WHEN** a user runs `oictl tools valves --help`
- **THEN** the help output lists global valve get, spec, and update operations and user valve get, spec, and update operations

#### Scenario: Function valve help is available
- **WHEN** a user runs `oictl functions valves --help`
- **THEN** the help output lists global valve get, spec, and update operations and user valve get, spec, and update operations

### Requirement: Tool global valves are inspectable and updatable
The CLI SHALL provide commands to get, inspect the spec for, and update global valves for a selected Open WebUI tool.

#### Scenario: Tool global valves are retrieved
- **WHEN** a user runs `oictl tools valves get <tool-id>`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/tools/id/<tool-id>/valves`
- **THEN** the CLI writes the returned valve object or null response to output

#### Scenario: Tool global valve spec is retrieved
- **WHEN** a user runs `oictl tools valves spec <tool-id>`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/tools/id/<tool-id>/valves/spec`
- **THEN** the CLI writes the server-generated valve schema response to output

#### Scenario: Tool global valves are updated
- **WHEN** a user runs `oictl tools valves update <tool-id> --file valves.json`
- **THEN** the CLI sends the JSON payload in an authenticated `POST` request to `/api/v1/tools/id/<tool-id>/valves/update`
- **THEN** the CLI writes the server-returned accepted valve values to output

### Requirement: Tool user valves are inspectable and updatable
The CLI SHALL provide commands to get, inspect the spec for, and update user-scoped valves for a selected Open WebUI tool.

#### Scenario: Tool user valves are retrieved
- **WHEN** a user runs `oictl tools valves user get <tool-id>`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/tools/id/<tool-id>/valves/user`
- **THEN** the CLI writes the returned user valve object or null response to output

#### Scenario: Tool user valve spec is retrieved
- **WHEN** a user runs `oictl tools valves user spec <tool-id>`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/tools/id/<tool-id>/valves/user/spec`
- **THEN** the CLI writes the server-generated user valve schema response to output

#### Scenario: Tool user valves are updated
- **WHEN** a user runs `oictl tools valves user update <tool-id> --file user-valves.json`
- **THEN** the CLI sends the JSON payload in an authenticated `POST` request to `/api/v1/tools/id/<tool-id>/valves/user/update`
- **THEN** the CLI writes the server-returned accepted user valve values to output

### Requirement: Function global valves are inspectable and updatable
The CLI SHALL provide commands to get, inspect the spec for, and update global valves for a selected Open WebUI function.

#### Scenario: Function global valves are retrieved
- **WHEN** a user runs `oictl functions valves get <function-id>`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/functions/id/<function-id>/valves`
- **THEN** the CLI writes the returned valve object or null response to output

#### Scenario: Function global valve spec is retrieved
- **WHEN** a user runs `oictl functions valves spec <function-id>`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/functions/id/<function-id>/valves/spec`
- **THEN** the CLI writes the server-generated valve schema response to output

#### Scenario: Function global valves are updated
- **WHEN** a user runs `oictl functions valves update <function-id> --file valves.json`
- **THEN** the CLI sends the JSON payload in an authenticated `POST` request to `/api/v1/functions/id/<function-id>/valves/update`
- **THEN** the CLI writes the server-returned accepted valve values to output

### Requirement: Function user valves are inspectable and updatable
The CLI SHALL provide commands to get, inspect the spec for, and update user-scoped valves for a selected Open WebUI function.

#### Scenario: Function user valves are retrieved
- **WHEN** a user runs `oictl functions valves user get <function-id>`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/functions/id/<function-id>/valves/user`
- **THEN** the CLI writes the returned user valve object or null response to output

#### Scenario: Function user valve spec is retrieved
- **WHEN** a user runs `oictl functions valves user spec <function-id>`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/functions/id/<function-id>/valves/user/spec`
- **THEN** the CLI writes the server-generated user valve schema response to output

#### Scenario: Function user valves are updated
- **WHEN** a user runs `oictl functions valves user update <function-id> --file user-valves.json`
- **THEN** the CLI sends the JSON payload in an authenticated `POST` request to `/api/v1/functions/id/<function-id>/valves/user/update`
- **THEN** the CLI writes the server-returned accepted user valve values to output

### Requirement: Valve updates require explicit JSON payload input
Valve update commands SHALL require a request body supplied through the standard payload flags and SHALL NOT synthesize default valve values locally.

#### Scenario: Global valve update omits payload
- **WHEN** a user runs `oictl tools valves update <tool-id>` without `--data`, `--file`, or `--file -`
- **THEN** the CLI exits non-zero before contacting Open WebUI
- **THEN** the CLI explains that a valve payload is required

#### Scenario: User valve update reads stdin
- **WHEN** a user runs `oictl functions valves user update <function-id> --file -`
- **THEN** the CLI reads the request body from stdin and sends it to Open WebUI unchanged

### Requirement: Valve commands preserve server authorization and validation behavior
Valve commands SHALL rely on Open WebUI for authorization, missing valve detection, schema generation, and payload validation, and SHALL surface non-2xx server responses consistently with other typed resource commands.

#### Scenario: Server rejects a valve update
- **WHEN** Open WebUI returns a validation error for a valve update command
- **THEN** the CLI exits non-zero and prints the response status and body to stderr

#### Scenario: Server denies valve access
- **WHEN** Open WebUI returns 401 or 403 for a valve command
- **THEN** the CLI exits non-zero and prints the response status and detail without exposing local credentials
