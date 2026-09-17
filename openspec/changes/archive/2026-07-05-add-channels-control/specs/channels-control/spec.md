## ADDED Requirements

### Requirement: CLI exposes channel control commands
The CLI SHALL expose `oictl channels` as the command family for typed Open WebUI channel, member, message, pin, and reaction operations.

#### Scenario: Help lists channel commands
- **WHEN** a user runs `oictl channels --help`
- **THEN** the help output lists channel list, get, create, update, delete, member, message, pin, and reaction operations implemented by this capability

#### Scenario: Unknown channel command fails predictably
- **WHEN** a user runs `oictl channels unknown`
- **THEN** the CLI exits non-zero and prints usage context for the channels command family

### Requirement: Channel commands manage channel records
The channel command family SHALL support listing, retrieving, creating, updating, and deleting Open WebUI channels while preserving the caller's effective server authorization scope.

#### Scenario: User lists accessible channels
- **WHEN** a verified user runs `oictl channels list --output json`
- **THEN** the CLI requests the Open WebUI channel list and returns only channels included by the server response

#### Scenario: User gets a channel
- **WHEN** a verified user runs `oictl channels get <channel-id> --output json`
- **THEN** the CLI requests `<channel-id>` from Open WebUI and prints the returned channel payload

#### Scenario: User creates a channel with payload passthrough
- **WHEN** a verified user runs `oictl channels create --file channel.json --output json`
- **THEN** the CLI sends the JSON payload to Open WebUI without applying an oictl-specific channel schema and prints the server response

#### Scenario: User updates a channel with payload passthrough
- **WHEN** a verified user runs `oictl channels update <channel-id> --data '{"name":"ops"}' --output json`
- **THEN** the CLI sends the JSON payload for `<channel-id>` to Open WebUI and prints the updated channel payload returned by the server

#### Scenario: Channel delete requires confirmation
- **WHEN** a user runs `oictl channels delete <channel-id>` without confirmation
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that destructive confirmation is required

#### Scenario: Confirmed channel delete is forwarded
- **WHEN** a user runs `oictl channels delete <channel-id> --yes`
- **THEN** the CLI sends the delete request to Open WebUI and prints the server deletion result

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

### Requirement: Channel message commands manage message content
The channel command family SHALL support listing, posting, retrieving, updating, threading, and deleting channel messages, and SHALL pass message mutation payloads through to Open WebUI.

#### Scenario: User lists channel messages
- **WHEN** a verified user runs `oictl channels messages list <channel-id> --skip 0 --limit 50 --output json`
- **THEN** the CLI requests channel messages from Open WebUI and prints the returned messages without redacting or reshaping message payload fields

#### Scenario: User posts a channel message
- **WHEN** a verified user runs `oictl channels messages post <channel-id> --file message.json --output json`
- **THEN** the CLI sends the JSON payload to Open WebUI and prints the created message returned by the server

#### Scenario: User gets one channel message
- **WHEN** a verified user runs `oictl channels messages get <channel-id> <message-id> --output json`
- **THEN** the CLI requests the message from Open WebUI and prints the server response

#### Scenario: User gets message data
- **WHEN** a verified user runs `oictl channels messages data <channel-id> <message-id> --output json`
- **THEN** the CLI requests the message data object from Open WebUI and prints the server response

#### Scenario: User lists thread replies
- **WHEN** a verified user runs `oictl channels messages thread <channel-id> <message-id> --skip 0 --limit 50 --output json`
- **THEN** the CLI requests thread replies for the message and prints the server response

#### Scenario: User updates a message with payload passthrough
- **WHEN** a verified user runs `oictl channels messages update <channel-id> <message-id> --file message.json --output json`
- **THEN** the CLI sends the JSON payload to Open WebUI and prints the updated message returned by the server

#### Scenario: Message delete requires confirmation
- **WHEN** a user runs `oictl channels messages delete <channel-id> <message-id>` without confirmation
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that destructive confirmation is required

#### Scenario: Confirmed message delete is forwarded
- **WHEN** a user runs `oictl channels messages delete <channel-id> <message-id> --yes`
- **THEN** the CLI sends the delete request to Open WebUI and prints the server deletion result

### Requirement: Channel pin commands manage pinned messages
The channel command family SHALL support listing pinned messages and pinning or unpinning channel messages using Open WebUI's server-backed pin state.

#### Scenario: User lists pinned messages
- **WHEN** a verified user runs `oictl channels pins list <channel-id> --page 1 --output json`
- **THEN** the CLI requests pinned messages from Open WebUI and prints the server response

#### Scenario: User pins a message
- **WHEN** a verified user runs `oictl channels pins set <channel-id> <message-id> --output json`
- **THEN** the CLI sends a pin request for the message to Open WebUI and prints the server response

#### Scenario: User unpins a message
- **WHEN** a verified user runs `oictl channels pins unset <channel-id> <message-id> --output json`
- **THEN** the CLI sends an unpin request for the message to Open WebUI and prints the server response

### Requirement: Channel reaction commands manage message reactions
The channel command family SHALL support adding and removing reactions on channel messages through Open WebUI.

#### Scenario: User adds a reaction
- **WHEN** a verified user runs `oictl channels reactions add <channel-id> <message-id> --name thumbs-up --output json`
- **THEN** the CLI sends the reaction add request to Open WebUI and prints the server response

#### Scenario: User removes a reaction
- **WHEN** a verified user runs `oictl channels reactions remove <channel-id> <message-id> --name thumbs-up --output json`
- **THEN** the CLI sends the reaction remove request to Open WebUI and prints the server response

#### Scenario: Reaction payload can be supplied as JSON
- **WHEN** a verified user runs `oictl channels reactions add <channel-id> <message-id> --data '{"name":"eyes"}' --output json`
- **THEN** the CLI sends the supplied JSON reaction payload to Open WebUI instead of inventing a local reaction schema

### Requirement: Channel commands preserve server authorization
The channel command family SHALL rely on Open WebUI for channel feature enablement, membership checks, access grants, author checks, admin checks, and resource existence decisions.

#### Scenario: Feature is disabled by server
- **WHEN** Open WebUI rejects a channel command because channels are disabled
- **THEN** the CLI exits non-zero and prints the server error context without printing credentials

#### Scenario: Server denies member-only access
- **WHEN** Open WebUI rejects a channel member, message, pin, or reaction operation with 401 or 403
- **THEN** the CLI exits non-zero and surfaces the server status without attempting a client-side bypass

#### Scenario: Message belongs to another channel
- **WHEN** Open WebUI rejects a message operation because the message does not belong to the requested channel
- **THEN** the CLI exits non-zero and surfaces the server status and response detail without rewriting the error as a local validation result

#### Scenario: Bearer token is used for channel requests
- **WHEN** a channel command runs with a resolved Open WebUI token
- **THEN** the CLI sends the request with bearer authentication using the shared target resolution behavior
