## MODIFIED Requirements

### Requirement: Channel member commands manage memberships
The channel command family SHALL support server-backed member listing, member add, member remove, and caller active-state update operations where the target Open WebUI instance exposes them.

#### Scenario: User lists channel members
- **WHEN** a verified user runs `oictl channels members list <channel-id> --page 1 --query alice --output json`
- **THEN** the CLI requests the channel members from Open WebUI with the supplied filters and prints the server response

#### Scenario: Owner or admin adds channel members
- **WHEN** an authorized user runs `oictl channels members add <channel-id> --file members.json --output json`
- **THEN** the CLI sends the member payload to Open WebUI without locally deciding whether the caller may add members

#### Scenario: Owner or admin removes channel members
- **WHEN** an authorized user runs `oictl channels members remove <channel-id> --data '{"user_ids":["user-1"]}' --output json`
- **THEN** the CLI sends the removal payload to Open WebUI and prints the server response

#### Scenario: User updates own active member state
- **WHEN** a verified user runs `oictl channels members active <channel-id> --data '{"is_active":false}' --output json`
- **THEN** the CLI forwards the active-state payload to Open WebUI and prints the server response

#### Scenario: User updates own active member state with active alias
- **WHEN** a verified user runs `oictl channels members active <channel-id> --data '{"active":false}' --output json`
- **THEN** the CLI sends `is_active: false` to Open WebUI instead of `active`
- **AND** the CLI prints the server response

#### Scenario: Explicit is_active takes precedence over active alias
- **WHEN** a verified user runs `oictl channels members active <channel-id> --data '{"active":true,"is_active":false}' --output json`
- **THEN** the CLI sends `is_active: false` to Open WebUI
- **AND** the CLI does not let the `active` alias override the explicit `is_active` value
