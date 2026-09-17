## ADDED Requirements

### Requirement: Model listing and discovery
The CLI SHALL provide model discovery commands for listing accessible models, base models, base model tags, and model tags.

#### Scenario: Accessible models are listed
- **WHEN** the user runs `oictl models list`
- **THEN** the CLI calls the Open WebUI models list endpoint and displays accessible models with pagination-aware response data

#### Scenario: Base models are listed
- **WHEN** the user runs `oictl models base`
- **THEN** the CLI calls the base models endpoint and displays the returned base models

### Requirement: Model inspection
The CLI SHALL allow a user to inspect a single model record by model identifier.

#### Scenario: Model is found
- **WHEN** the user runs `oictl models get <model-id>`
- **THEN** the CLI returns the matching model response from Open WebUI

### Requirement: Model mutation
The CLI SHALL provide commands to create, update, toggle, update access grants for, delete, and delete all model records using Open WebUI model endpoints.

#### Scenario: Model is created
- **WHEN** the user runs `oictl models create --file model.json`
- **THEN** the CLI submits the JSON payload to the model creation endpoint and prints the created model response

#### Scenario: Model is updated
- **WHEN** the user runs `oictl models update <model-id> --file model.json`
- **THEN** the CLI submits the JSON payload with the target model identifier to the model update endpoint

#### Scenario: Model is deleted
- **WHEN** the user runs `oictl models delete <model-id>`
- **THEN** the CLI requests deletion of that model and reports the boolean server result

### Requirement: Model import export and sync
The CLI SHALL provide commands to export models, import models, and synchronize model records from Open WebUI.

#### Scenario: Models are exported
- **WHEN** the user runs `oictl models export --out models.json`
- **THEN** the CLI writes the exported model JSON to the requested output path

#### Scenario: Models are synchronized
- **WHEN** the user runs `oictl models sync`
- **THEN** the CLI calls the model sync endpoint and prints the synchronized models response

### Requirement: Model command authorization
The CLI SHALL rely on Open WebUI authorization for model commands and SHALL surface authorization failures consistently.

#### Scenario: Admin-only model action is forbidden
- **WHEN** Open WebUI returns 401 or 403 for a model command
- **THEN** the CLI exits non-zero and includes the response status and detail in stderr
