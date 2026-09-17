## ADDED Requirements

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
