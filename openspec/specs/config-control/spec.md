## Purpose
Define commands for inspecting and updating Open WebUI configuration surfaces.
## Requirements
### Requirement: Configuration import and export
The CLI SHALL provide commands to export all configuration, import configuration JSON, and fetch configuration by namespace.

#### Scenario: Configuration is exported
- **WHEN** the user runs `oictl config export --out config.json`
- **THEN** the CLI writes the full exported configuration response to `config.json`

#### Scenario: Namespace is fetched
- **WHEN** the user runs `oictl config namespace <namespace>`
- **THEN** the CLI prints the configuration values returned for that namespace

### Requirement: Connection configuration
The CLI SHALL provide get and set commands for Open WebUI connection configuration.

#### Scenario: Connections config is updated
- **WHEN** the user runs `oictl config connections set --file connections.json`
- **THEN** the CLI submits the JSON payload to the connections configuration endpoint

### Requirement: Tool and terminal server configuration
The CLI SHALL provide commands for tool server and terminal server configuration, verification, policy, lifecycle, and refresh endpoints included in `/api/v1/configs`.

#### Scenario: Tool servers are listed
- **WHEN** the user runs `oictl config tool-servers get`
- **THEN** the CLI displays the configured tool server connections

#### Scenario: Terminal server verification is requested
- **WHEN** the user runs `oictl config terminal-servers verify --file request.json`
- **THEN** the CLI submits the verification request and prints the verification response

### Requirement: Code execution configuration
The CLI SHALL provide get and set commands for code execution and code interpreter configuration.

#### Scenario: Code execution config is read
- **WHEN** the user runs `oictl config code-execution get`
- **THEN** the CLI prints the returned code execution configuration

### Requirement: Model UI configuration
The CLI SHALL provide commands for model defaults and model UI configuration exposed under `/api/v1/configs/models`.

#### Scenario: Model config is updated
- **WHEN** the user runs `oictl config models set --file models-config.json`
- **THEN** the CLI submits the payload to the model configuration endpoint and prints the updated configuration

### Requirement: Suggestions and banners configuration
The CLI SHALL provide commands to update prompt suggestions and get or update banners.

#### Scenario: Banners are updated
- **WHEN** the user runs `oictl config banners set --file banners.json`
- **THEN** the CLI submits the banner payload and prints the updated banner list

### Requirement: OAuth client registration
The CLI SHALL provide a command for registering OAuth clients through the configuration API.

#### Scenario: OAuth client is registered
- **WHEN** the user runs `oictl config oauth-client register --file oauth-client.json`
- **THEN** the CLI submits the registration request and prints the registration response

### Requirement: Configuration authorization
The CLI SHALL surface Open WebUI authorization failures for admin-only configuration commands.

#### Scenario: Non-admin config update is rejected
- **WHEN** Open WebUI returns 401 or 403 for a configuration command
- **THEN** the CLI exits non-zero and prints the response status and detail

### Requirement: Terminal server connection access grants are first-class config workflows
The CLI SHALL provide commands under `oictl config terminal-servers access-grants` to inspect, replace, and diff access grants for a config-backed terminal server connection.

#### Scenario: Terminal server connection grants are inspected
- **WHEN** the user runs `oictl config terminal-servers access-grants get <connection-id-or-name>`
- **THEN** the CLI fetches `/api/v1/configs/terminal_servers`, locates the terminal server connection by exact id or unique name, and prints the grants from `config.access_grants`

#### Scenario: Terminal server connection grants are replaced
- **WHEN** the user runs `oictl config terminal-servers access-grants set <connection-id-or-name> --file grants.json`
- **THEN** the CLI validates the desired grants, replaces only `config.access_grants` on the selected connection, preserves unrelated connection fields and other terminal server connections, and submits the updated `TERMINAL_SERVER_CONNECTIONS` payload to `/api/v1/configs/terminal_servers`

#### Scenario: Terminal server connection grant diff is previewed
- **WHEN** the user runs `oictl config terminal-servers access-grants diff <connection-id-or-name> --file grants.json`
- **THEN** the CLI compares normalized desired grants with the current `config.access_grants` and prints added and removed grants without submitting a mutation request

### Requirement: Terminal server grant commands validate identity and desired grants before mutation
The CLI SHALL reject ambiguous connection identities and invalid grant files before posting terminal server configuration changes.

#### Scenario: Terminal server connection is not found
- **WHEN** the user runs a terminal server grant command for a connection id or name that is absent from `TERMINAL_SERVER_CONNECTIONS`
- **THEN** the CLI exits non-zero and reports that the terminal server connection was not found without submitting a mutation request

#### Scenario: Terminal server connection name is ambiguous
- **WHEN** more than one terminal server connection has the requested name and no exact id match exists
- **THEN** the CLI exits non-zero and reports that the terminal server connection name is ambiguous without submitting a mutation request

#### Scenario: Invalid terminal server grant file is rejected
- **WHEN** the user runs `oictl config terminal-servers access-grants set <connection-id-or-name> --file grants.json` and the file contains an unsupported principal type or permission
- **THEN** the CLI exits non-zero and reports the grant validation error without submitting a mutation request

