## Purpose
Define commands for discovering, inspecting, mutating, importing, exporting, and syncing Open WebUI model records.
## Requirements
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

### Requirement: Model command authorization
The CLI SHALL rely on Open WebUI authorization for model commands and SHALL surface authorization failures consistently.

#### Scenario: Admin-only model action is forbidden
- **WHEN** Open WebUI returns 401 or 403 for a model command
- **THEN** the CLI exits non-zero and includes the response status and detail in stderr

### Requirement: Model delete reports missing models clearly
The CLI SHALL translate Open WebUI's known misleading missing-model delete response, HTTP `401` with not-found detail from `/api/v1/models/model/delete`, into a clear not-found result for `oictl models delete` without masking ordinary authorization failures.

#### Scenario: Missing model delete reports not found
- **WHEN** the user runs `oictl models delete <model-id>` and Open WebUI returns HTTP `401` with not-found detail from the model delete endpoint
- **THEN** the CLI exits non-zero and reports that the model was not found, including the target model identifier where appropriate

#### Scenario: Real authorization failure remains authorization failure
- **WHEN** the user runs `oictl models delete <model-id>` and Open WebUI returns an authorization failure that does not match the known missing-model response
- **THEN** the CLI exits non-zero and surfaces the authorization status and detail without reclassifying it as not found

### Requirement: Model records are manageable by declarative manifests
The CLI SHALL allow Open WebUI model records to be managed through `oictl manifests diff`, `oictl manifests apply`, and `oictl manifests sync` using `kind: Model` manifests.

#### Scenario: Model manifest creates a model record
- **WHEN** the user runs `oictl manifests apply --file model.json` and the `Model` manifest does not match an existing remote model
- **THEN** the CLI sends the manifest `spec` to the Open WebUI model creation endpoint with the model identifier resolved from `metadata.name` or `spec.id`

#### Scenario: Model manifest updates a model record
- **WHEN** the user runs `oictl manifests apply --file model.json` and the remote model has different managed fields
- **THEN** the CLI sends the updated model payload to the Open WebUI model update endpoint and reports the changed fields

#### Scenario: Model manifest deletes an omitted model during sync
- **WHEN** the user runs `oictl manifests sync --scope models --file manifests/ --yes` and a remote model in scope is omitted from the manifest set
- **THEN** the CLI deletes that model through the Open WebUI model delete endpoint

#### Scenario: Model manifest updates access grants
- **WHEN** a `Model` manifest includes `access_grants` that differ from the remote model grants
- **THEN** the CLI applies those grants through the Open WebUI model access update endpoint and reports filtered grants if the server does not retain the requested grant set

### Requirement: Declarative model manifests preserve imperative model commands
The CLI SHALL add declarative model reconciliation without changing existing imperative `oictl models` commands or their direct Open WebUI endpoint behavior.

#### Scenario: Imperative model create still uses model create endpoint
- **WHEN** the user runs `oictl models create --file model.json`
- **THEN** the CLI submits the JSON payload to the model creation endpoint and prints the created model response

#### Scenario: Imperative model update remains available
- **WHEN** the user runs `oictl models update <model-id> --file model.json`
- **THEN** the CLI submits the JSON payload with the target model identifier to the model update endpoint

### Requirement: Model manifests use Open WebUI model APIs
The CLI SHALL reconcile `kind: Model` manifests through Open WebUI model list, get, create, update, access-update, and delete APIs without introducing separate model storage or local state.

#### Scenario: Model manifest creates model payload
- **WHEN** `oictl manifests apply --file model.json` creates a model from a `kind: Model` manifest
- **THEN** the CLI sends a create request containing the desired `id`, `base_model_id`, `name`, `meta`, `params`, and `is_active` fields

#### Scenario: Model manifest updates active state idempotently
- **WHEN** `oictl manifests apply --file model.json` updates an existing model whose `is_active` value differs from the manifest
- **THEN** the CLI sends an update payload with the desired `is_active` value rather than toggling the model state

#### Scenario: Model manifest delete uses model delete endpoint
- **WHEN** model manifest sync plans a delete action for an omitted model
- **THEN** the CLI deletes that model through the existing Open WebUI model delete endpoint

### Requirement: Model manifests validate required payload fields
The CLI SHALL validate model manifest payloads before mutation so that required Open WebUI model fields are present and consistent.

#### Scenario: Model id defaults from metadata name
- **WHEN** a model manifest omits `spec.id` and declares `metadata.name: llama-ops`
- **THEN** the CLI uses `llama-ops` as the model `id` in create and update payloads

#### Scenario: Model name is required
- **WHEN** a model manifest omits `spec.name`
- **THEN** the CLI exits non-zero and reports that `spec.name` is required for `kind: Model`

#### Scenario: Model meta and params default to objects
- **WHEN** a model manifest omits `spec.meta` or `spec.params`
- **THEN** the CLI plans and applies the model with empty object values for the omitted fields
