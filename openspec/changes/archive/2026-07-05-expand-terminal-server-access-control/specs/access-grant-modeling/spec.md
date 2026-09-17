## MODIFIED Requirements

### Requirement: Grant reconciliation uses resource-specific Open WebUI APIs
The CLI SHALL apply access grant changes through the supported resource's Open WebUI API or, for config-backed terminal server connections, through the terminal server configuration API, and SHALL surface server authorization or sharing-policy filtering in the command result.

#### Scenario: Grant update is authorized
- **WHEN** the user applies a manifest with changed grants for a supported resource and Open WebUI accepts the update
- **THEN** the CLI reports the grant replacement as applied and includes the resource in the success output

#### Scenario: Config-backed terminal server grant update is authorized
- **WHEN** the user replaces grants for a terminal server connection and Open WebUI accepts the updated terminal server configuration
- **THEN** the CLI reports the grant replacement as applied for the selected terminal server connection

#### Scenario: Grant update is forbidden
- **WHEN** Open WebUI returns 401 or 403 for a manifest grant update or terminal server config grant update
- **THEN** the CLI exits non-zero and prints the response status and detail without leaking credentials

#### Scenario: Server filters requested grants
- **WHEN** Open WebUI accepts a grant update but returns fewer grants than requested because of sharing policy
- **THEN** the CLI reports the server-side difference in the command output
