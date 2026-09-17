## ADDED Requirements

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
