## ADDED Requirements

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
