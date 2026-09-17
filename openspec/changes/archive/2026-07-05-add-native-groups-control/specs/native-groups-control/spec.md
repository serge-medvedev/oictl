## ADDED Requirements

### Requirement: CLI exposes native groups commands
The CLI SHALL expose `oictl groups` as the command family for native Open WebUI group operations, distinct from SCIM group provisioning commands.

#### Scenario: Help lists native group commands
- **WHEN** a user runs `oictl groups --help`
- **THEN** the help output lists list, create, get, info, export, update, delete, preview, and users membership operations

#### Scenario: Native groups use normal API authentication
- **WHEN** a user runs `oictl groups list` with `OPEN_WEBUI_API_KEY` configured and no `OPEN_WEBUI_SCIM_TOKEN`
- **THEN** the outgoing request uses the normal Open WebUI bearer token and does not require a SCIM token

#### Scenario: SCIM group commands remain separate
- **WHEN** a user runs `oictl scim groups list`
- **THEN** the command continues to use the SCIM command family and SCIM authentication rather than the native groups endpoint

### Requirement: Native groups can be listed and inspected
The native groups command family SHALL support listing groups and retrieving native group detail, info, export, and access preview responses from Open WebUI.

#### Scenario: User lists native groups
- **WHEN** a user runs `oictl groups list --share true --output json`
- **THEN** the CLI sends `GET /api/v1/groups/` with the `share=true` query parameter and writes the native groups response as JSON

#### Scenario: User gets a native group
- **WHEN** a user runs `oictl groups get group-a --output json`
- **THEN** the CLI sends `GET /api/v1/groups/id/group-a` and writes the native group response

#### Scenario: User gets native group info
- **WHEN** a user runs `oictl groups info group-a --output json`
- **THEN** the CLI sends `GET /api/v1/groups/id/group-a/info` and writes the native group info response

#### Scenario: User exports a native group
- **WHEN** a user runs `oictl groups export group-a --out group-a.json`
- **THEN** the CLI sends `GET /api/v1/groups/id/group-a/export` and writes the raw export response to `group-a.json`

#### Scenario: User previews native group access
- **WHEN** a user runs `oictl groups preview group-a --output json`
- **THEN** the CLI sends `GET /api/v1/groups/id/group-a/preview` and writes the native preview response without mutating remote state

### Requirement: Native groups can be created and updated
The native groups command family SHALL support creating and updating native Open WebUI groups using JSON payloads supplied inline, from a file, or from standard input.

#### Scenario: User creates a native group
- **WHEN** a user runs `oictl groups create --file group.json`
- **THEN** the CLI sends `POST /api/v1/groups/create` with the supplied JSON payload and writes the created native group response

#### Scenario: User updates a native group
- **WHEN** a user runs `oictl groups update group-a --file group.json`
- **THEN** the CLI sends `POST /api/v1/groups/id/group-a/update` with the supplied JSON payload and writes the updated native group response

#### Scenario: Mutation payload is missing
- **WHEN** a user runs `oictl groups create` without `--data`, `--file`, or `--file -`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that a group payload is required

### Requirement: Native groups can be deleted safely
The native groups command family SHALL support deleting native Open WebUI groups and SHALL require explicit confirmation before contacting Open WebUI for deletion.

#### Scenario: Delete without confirmation is rejected
- **WHEN** a user runs `oictl groups delete group-a` without `--yes` or `--confirm`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that destructive operations require confirmation

#### Scenario: Delete with confirmation is sent
- **WHEN** a user runs `oictl groups delete group-a --yes`
- **THEN** the CLI sends `DELETE /api/v1/groups/id/group-a/delete` and treats a successful Open WebUI response as success

### Requirement: Native group memberships can be managed
The native groups command family SHALL support listing native group members and adding or removing users from native groups.

#### Scenario: User lists native group members
- **WHEN** a user runs `oictl groups users list group-a --output json`
- **THEN** the CLI sends `POST /api/v1/groups/id/group-a/users` and writes the native user info list response as JSON

#### Scenario: User adds one native group member
- **WHEN** a user runs `oictl groups users add group-a user-1`
- **THEN** the CLI sends `POST /api/v1/groups/id/group-a/users/add` with JSON body `{"user_ids":["user-1"]}`

#### Scenario: User adds multiple native group members
- **WHEN** a user runs `oictl groups users add group-a user-1 user-2`
- **THEN** the CLI sends `POST /api/v1/groups/id/group-a/users/add` with both user ids in the `user_ids` list

#### Scenario: User removes native group members
- **WHEN** a user runs `oictl groups users remove group-a user-1 user-2`
- **THEN** the CLI sends `POST /api/v1/groups/id/group-a/users/remove` with both user ids in the `user_ids` list

#### Scenario: Membership mutation requires users
- **WHEN** a user runs `oictl groups users add group-a` without any user ids or payload
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that at least one user id is required

### Requirement: Native group errors preserve server details
Native group commands SHALL surface Open WebUI HTTP status and response details without leaking configured bearer tokens.

#### Scenario: Native group is not found
- **WHEN** Open WebUI returns an error for `oictl groups get missing --output json`
- **THEN** the CLI exits non-zero and writes the HTTP status plus server response detail to stderr

#### Scenario: Token is redacted from native group errors
- **WHEN** Open WebUI returns an error body containing the configured Open WebUI API token during a native group command
- **THEN** the CLI writes the error with the token replaced by `<redacted>`
