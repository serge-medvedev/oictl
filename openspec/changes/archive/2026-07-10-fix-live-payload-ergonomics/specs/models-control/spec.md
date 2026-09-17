## MODIFIED Requirements

### Requirement: Model mutation
The CLI SHALL provide commands to create, update, toggle, update access grants for, delete, and delete all model records using Open WebUI model endpoints.

#### Scenario: Model is created
- **WHEN** the user runs `oictl models create --file model.json` and `model.json` omits `params`
- **THEN** the CLI submits the JSON payload to the model creation endpoint with `params` defaulted to an empty object
- **AND** the CLI prints the created model response

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

#### Scenario: Models are imported with omitted params
- **WHEN** the user runs `oictl models import --file models.json` and an imported model omits `params`
- **THEN** the CLI sends that model import payload with `params` defaulted to an empty object

#### Scenario: Models are synchronized
- **WHEN** the user runs `oictl models sync` and a synchronized model payload omits `params`
- **THEN** the CLI calls the model sync endpoint with `params` defaulted to an empty object for that model payload
- **AND** the CLI prints the synchronized models response
