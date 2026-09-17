## ADDED Requirements

### Requirement: File upload
The CLI SHALL provide a command to upload a local file to Open WebUI with optional metadata and processing flags.

#### Scenario: File is uploaded
- **WHEN** the user runs `oictl files upload ./document.pdf`
- **THEN** the CLI uploads the file using multipart form data and prints the created file response

#### Scenario: File is uploaded without processing
- **WHEN** the user runs `oictl files upload ./document.pdf --process=false`
- **THEN** the CLI sends the upload request with processing disabled

### Requirement: File listing and search
The CLI SHALL provide commands to list files, search files, and count files accessible to the authenticated user.

#### Scenario: Files are listed
- **WHEN** the user runs `oictl files list`
- **THEN** the CLI displays the returned file list response

#### Scenario: Files are searched
- **WHEN** the user runs `oictl files search --query report`
- **THEN** the CLI submits the query to the file search endpoint and prints matching files

### Requirement: File inspection and processing status
The CLI SHALL provide commands to inspect a file record and fetch processing status by file identifier.

#### Scenario: File is inspected
- **WHEN** the user runs `oictl files get <file-id>`
- **THEN** the CLI prints the file record returned by Open WebUI

#### Scenario: Processing status is inspected
- **WHEN** the user runs `oictl files status <file-id>`
- **THEN** the CLI prints the file processing status response

### Requirement: File content retrieval
The CLI SHALL provide commands to retrieve file content, HTML content, named content, and extracted data content.

#### Scenario: File content is downloaded
- **WHEN** the user runs `oictl files content <file-id> --out downloaded.bin`
- **THEN** the CLI writes the raw content response to `downloaded.bin`

#### Scenario: Extracted text is printed
- **WHEN** the user runs `oictl files data-content <file-id>`
- **THEN** the CLI prints the extracted data content response

### Requirement: File content update
The CLI SHALL provide a command to update extracted file data content.

#### Scenario: Extracted content is updated
- **WHEN** the user runs `oictl files update-content <file-id> --file content.json`
- **THEN** the CLI submits the JSON payload to the file data content update endpoint

### Requirement: File rename and deletion
The CLI SHALL provide commands to rename a file, delete one file, and delete all files when authorized.

#### Scenario: File is renamed
- **WHEN** the user runs `oictl files rename <file-id> --name new-name.pdf`
- **THEN** the CLI submits the rename request and prints the updated file response

#### Scenario: File is deleted
- **WHEN** the user runs `oictl files delete <file-id>`
- **THEN** the CLI requests deletion and reports the server result
