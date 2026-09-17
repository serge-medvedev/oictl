## Purpose
Define commands for listing, searching, managing, exporting, indexing, and syncing Open WebUI knowledge resources.

## Requirements

### Requirement: Knowledge listing and search
The CLI SHALL provide commands to list knowledge bases, search knowledge bases, and search knowledge files.

#### Scenario: Knowledge bases are listed
- **WHEN** the user runs `oictl knowledge list`
- **THEN** the CLI displays the knowledge bases returned for the authenticated user

#### Scenario: Knowledge files are searched
- **WHEN** the user runs `oictl knowledge search-files --query handbook`
- **THEN** the CLI prints matching knowledge files returned by Open WebUI

### Requirement: Knowledge base lifecycle
The CLI SHALL provide commands to create, inspect, update, update access grants for, reset, delete, and export local knowledge bases.

#### Scenario: Knowledge base is created
- **WHEN** the user runs `oictl knowledge create --file knowledge.json`
- **THEN** the CLI submits the payload to the knowledge creation endpoint and prints the created knowledge base

#### Scenario: Knowledge base is exported
- **WHEN** the user runs `oictl knowledge export <knowledge-id> --out knowledge.zip`
- **THEN** the CLI writes the exported response to the requested file path

### Requirement: Knowledge file membership
The CLI SHALL provide commands to list, add, update, remove, batch add, and inspect pending files for a knowledge base.

#### Scenario: File is added to knowledge base
- **WHEN** the user runs `oictl knowledge files add <knowledge-id> <file-id>`
- **THEN** the CLI requests that Open WebUI attach the file to the knowledge base

#### Scenario: Knowledge files are listed
- **WHEN** the user runs `oictl knowledge files list <knowledge-id>`
- **THEN** the CLI displays files attached to the knowledge base

### Requirement: Knowledge directories
The CLI SHALL provide commands to create, update, delete, and move files among knowledge directories.

#### Scenario: Directory is created
- **WHEN** the user runs `oictl knowledge dirs create <knowledge-id> --file dir.json`
- **THEN** the CLI submits the directory creation payload and prints the created directory

#### Scenario: File is moved
- **WHEN** the user runs `oictl knowledge files move <knowledge-id> --file move.json`
- **THEN** the CLI submits the move request and prints the server result

### Requirement: Knowledge indexing and sync
The CLI SHALL provide commands for knowledge reindexing, metadata reindexing, reset, sync diff, and sync cleanup operations.

#### Scenario: Knowledge base is reindexed
- **WHEN** the user runs `oictl knowledge reindex --file request.json`
- **THEN** the CLI submits the reindex request and prints the boolean server result

#### Scenario: Sync diff is requested
- **WHEN** the user runs `oictl knowledge sync diff <knowledge-id> --file request.json`
- **THEN** the CLI submits the sync diff request and prints the diff response

### Requirement: Knowledge authorization
The CLI SHALL rely on Open WebUI permissions for knowledge commands and SHALL surface authorization failures consistently.

#### Scenario: Knowledge write is forbidden
- **WHEN** Open WebUI returns 401 or 403 for a knowledge mutation command
- **THEN** the CLI exits non-zero and prints the response status and detail
