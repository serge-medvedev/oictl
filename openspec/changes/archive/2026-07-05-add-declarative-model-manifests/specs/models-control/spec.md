## ADDED Requirements

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
