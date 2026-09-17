## ADDED Requirements

### Requirement: CLI exposes chat control commands
The CLI SHALL expose `oictl chats` as the command family for typed Open WebUI chat record and chat workflow operations.

#### Scenario: Help lists chat commands
- **WHEN** a user runs `oictl chats --help`
- **THEN** the help output lists chat list, search, get, export, import, share, tags, archive, delete, and compact operations that are implemented in this capability

### Requirement: Chat commands list and search chat records
The chat command family SHALL support listing and searching chat records with server-supported pagination and filters while preserving the user's effective server authorization scope.

#### Scenario: User lists own chats
- **WHEN** a verified user runs `oictl chats list --page 1`
- **THEN** the CLI requests the Open WebUI chat list for that user and renders the returned chat identifiers and titles

#### Scenario: User searches chats
- **WHEN** a verified user runs `oictl chats search --query "tag:ops incident" --page 1`
- **THEN** the CLI sends the query and pagination parameters to Open WebUI and returns only server-authorized matches

#### Scenario: Admin lists another user's chats where enabled
- **WHEN** an admin runs a chat list command with an explicit user filter supported by the target Open WebUI instance
- **THEN** the CLI sends the admin-scoped request and surfaces any server 403 response without attempting a client-side bypass

### Requirement: Chat commands inspect and export chat content
The chat command family SHALL support retrieving and exporting chat content in machine-readable form without redacting or reshaping server chat payloads by default.

#### Scenario: User gets a chat
- **WHEN** a user runs `oictl chats get <chat-id> --output json`
- **THEN** the CLI returns the Open WebUI chat payload for `<chat-id>` as JSON

#### Scenario: User exports chats
- **WHEN** a user runs `oictl chats export --out chats.json`
- **THEN** the CLI writes the server export payload to `chats.json` and exits successfully only after the file is written

#### Scenario: Export is forbidden
- **WHEN** Open WebUI rejects a chat export request with 403
- **THEN** the CLI exits non-zero and prints the server error context without printing credentials

### Requirement: Chat commands import chat content
The chat command family SHALL support importing chat payloads from a file or stdin and SHALL send the payload to Open WebUI without inventing local chat schema transformations.

#### Scenario: User imports from a file
- **WHEN** a user runs `oictl chats import --file chats.json`
- **THEN** the CLI reads `chats.json`, sends the payload to Open WebUI, and prints the server import result

#### Scenario: Import input is missing
- **WHEN** a user runs `oictl chats import` without a file or stdin payload
- **THEN** the CLI exits non-zero and explains that an import payload is required

### Requirement: Chat commands manage chat metadata and workflow state
The chat command family SHALL support server-backed chat metadata and workflow operations, including sharing, tags, archiving, deletion, and compaction where the target Open WebUI instance exposes them.

#### Scenario: User shares a chat
- **WHEN** a user runs `oictl chats share <chat-id>`
- **THEN** the CLI sends the share request to Open WebUI and prints the returned shared-chat metadata

#### Scenario: User updates tags
- **WHEN** a user runs `oictl chats tags set <chat-id> --tag ops --tag review`
- **THEN** the CLI sends the tag update request and returns the updated chat metadata from Open WebUI

#### Scenario: User compacts a chat
- **WHEN** a user runs `oictl chats compact <chat-id> --model <model-id>`
- **THEN** the CLI requests server-side compaction for the chat and model and reports the server result

### Requirement: Chat destructive operations require explicit intent
The chat command family SHALL require explicit chat identifiers and destructive confirmation for operations that delete or irreversibly alter chat records.

#### Scenario: User deletes a chat with confirmation
- **WHEN** a user runs `oictl chats delete <chat-id> --yes`
- **THEN** the CLI sends the delete request and returns the server deletion result

#### Scenario: User omits delete confirmation
- **WHEN** a user runs `oictl chats delete <chat-id>` in a non-interactive context without `--yes`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that confirmation is required
