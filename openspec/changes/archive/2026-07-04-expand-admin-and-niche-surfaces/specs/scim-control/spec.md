## ADDED Requirements

### Requirement: CLI exposes SCIM commands
The CLI SHALL expose `oictl scim` as the command family for Open WebUI SCIM 2.0 provisioning operations.

#### Scenario: Help lists SCIM commands
- **WHEN** a user runs `oictl scim --help`
- **THEN** the help output lists service-provider-config, resource-types, schemas, users, and groups operations that are implemented in this capability

### Requirement: SCIM commands use dedicated SCIM authentication
SCIM commands SHALL authenticate with the Open WebUI SCIM bearer token and SHALL NOT silently fall back to the normal Open WebUI user API token.

#### Scenario: SCIM token environment variable is supplied
- **WHEN** `OPEN_WEBUI_SCIM_TOKEN` is set and a user runs `oictl scim users list`
- **THEN** the outgoing SCIM request includes `Authorization: Bearer <scim-token>`

#### Scenario: SCIM token is missing
- **WHEN** no SCIM token source is available and a user runs `oictl scim users list`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that a SCIM token is required

#### Scenario: Normal API token exists but SCIM token is missing
- **WHEN** a normal Open WebUI API token is configured and no SCIM token is configured
- **THEN** `oictl scim users list` exits non-zero instead of using the normal API token

### Requirement: SCIM commands expose service metadata
SCIM commands SHALL support retrieving SCIM service provider configuration, resource types, and schemas from Open WebUI.

#### Scenario: User gets service provider config
- **WHEN** a user runs `oictl scim service-provider-config --output json`
- **THEN** the CLI requests `/api/v1/scim/v2/ServiceProviderConfig` and writes the SCIM JSON response

#### Scenario: User gets resource types
- **WHEN** a user runs `oictl scim resource-types --output json`
- **THEN** the CLI requests `/api/v1/scim/v2/ResourceTypes` and writes the SCIM JSON response

#### Scenario: User gets schemas
- **WHEN** a user runs `oictl scim schemas --output json`
- **THEN** the CLI requests `/api/v1/scim/v2/Schemas` and writes the SCIM JSON response

### Requirement: SCIM commands manage users
The SCIM command family SHALL support listing, retrieving, creating, replacing, patching, and deleting SCIM users using SCIM request and response payloads.

#### Scenario: User lists SCIM users
- **WHEN** a user runs `oictl scim users list --start-index 1 --count 50`
- **THEN** the CLI sends SCIM pagination parameters and returns the SCIM ListResponse body

#### Scenario: User creates a SCIM user
- **WHEN** a user runs `oictl scim users create --file user.json`
- **THEN** the CLI sends the SCIM user creation payload and returns the created SCIM User body

#### Scenario: User patches a SCIM user
- **WHEN** a user runs `oictl scim users patch <user-id> --file patch.json`
- **THEN** the CLI sends the SCIM PatchOp payload and returns the updated SCIM User body

#### Scenario: User deletes a SCIM user with confirmation
- **WHEN** a user runs `oictl scim users delete <user-id> --yes`
- **THEN** the CLI sends the SCIM delete request and treats HTTP 204 as success

### Requirement: SCIM commands manage groups
The SCIM command family SHALL support listing, retrieving, creating, replacing, patching, and deleting SCIM groups using SCIM request and response payloads.

#### Scenario: User lists SCIM groups
- **WHEN** a user runs `oictl scim groups list --start-index 1 --count 50`
- **THEN** the CLI sends SCIM pagination parameters and returns the SCIM ListResponse body

#### Scenario: User creates a SCIM group
- **WHEN** a user runs `oictl scim groups create --file group.json`
- **THEN** the CLI sends the SCIM group creation payload and returns the created SCIM Group body

#### Scenario: User patches SCIM group membership
- **WHEN** a user runs `oictl scim groups patch <group-id> --file patch.json`
- **THEN** the CLI sends the SCIM PatchOp payload and returns the updated SCIM Group body

#### Scenario: User deletes a SCIM group with confirmation
- **WHEN** a user runs `oictl scim groups delete <group-id> --yes`
- **THEN** the CLI sends the SCIM delete request and treats HTTP 204 as success

### Requirement: SCIM commands preserve SCIM error bodies
SCIM commands SHALL surface server-returned SCIM error bodies without converting them into generic CLI errors.

#### Scenario: SCIM object is not found
- **WHEN** Open WebUI returns a SCIM error for a missing user or group
- **THEN** the CLI exits non-zero and writes the SCIM error JSON body to stderr for JSON output mode
