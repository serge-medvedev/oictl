## MODIFIED Requirements

### Requirement: Chat commands manage chat metadata and workflow state
The chat command family SHALL support server-backed chat metadata and workflow operations, including sharing, tags, archiving, deletion, and compaction where the target Open WebUI instance exposes them.

#### Scenario: User shares a chat
- **WHEN** a user runs `oictl chats share <chat-id>`
- **THEN** the CLI sends the share request to Open WebUI and prints the returned shared-chat metadata

#### Scenario: User updates a tag
- **WHEN** a user runs `oictl chats tags set <chat-id> --tag ops`
- **THEN** the CLI sends the tag update request as a single `TagForm` object containing `name`
- **AND** the CLI returns the updated chat metadata from Open WebUI

#### Scenario: User supplies multiple tag convenience flags
- **WHEN** a user runs `oictl chats tags set <chat-id> --tag ops --tag review`
- **THEN** the CLI rejects the command with a clear error before contacting Open WebUI

#### Scenario: User compacts a chat
- **WHEN** a user runs `oictl chats compact <chat-id> --model <model-id>`
- **THEN** the CLI requests server-side compaction for the chat and model and reports the server result
