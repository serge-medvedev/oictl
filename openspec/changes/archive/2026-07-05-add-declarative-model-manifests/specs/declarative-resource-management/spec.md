## ADDED Requirements

### Requirement: Model manifests define desired model state
The CLI SHALL accept `kind: Model` manifest documents in `oictl.openwebui/v1` manifests and SHALL treat `metadata.name` as the Open WebUI model identifier.

#### Scenario: Valid model manifest is loaded
- **WHEN** a manifest declares `kind: Model`, `metadata.name: llama-ops`, and a `spec` containing model fields
- **THEN** the CLI validates the document and includes `Model/llama-ops` in the manifest plan

#### Scenario: Model identity mismatch is rejected
- **WHEN** a model manifest declares `metadata.name: llama-ops` and `spec.id: other-model`
- **THEN** the CLI exits non-zero and reports that the model manifest identity is inconsistent without sending mutation requests

### Requirement: Manifest workflows reconcile model resources
The CLI SHALL include `Model` resources in `oictl manifests diff`, `oictl manifests apply`, and `oictl manifests sync` workflows using the same plan and execution semantics as other supported manifest resources.

#### Scenario: Missing model is planned for creation
- **WHEN** a `kind: Model` manifest declares a model that does not exist remotely
- **THEN** `oictl manifests diff` reports a create action for that model and exits successfully

#### Scenario: Declared model is applied
- **WHEN** the user runs `oictl manifests apply --file model.json` for a valid `kind: Model` manifest
- **THEN** the CLI creates or updates the Open WebUI model using the resolved Open WebUI target

#### Scenario: Model sync scope prunes omitted models
- **WHEN** the user runs `oictl manifests sync --scope models --directory manifests/ --yes` and a remote model in scope is absent from the manifest set
- **THEN** the CLI deletes that omitted model through the model delete endpoint

### Requirement: Model diff reports managed field drift
The CLI SHALL compare manifest-declared model fields against current Open WebUI model state and SHALL report changed managed fields, including `id`, `base_model_id`, `name`, `meta`, `params`, and `is_active`, while ignoring server-generated fields.

#### Scenario: Active state differs
- **WHEN** a model manifest declares `spec.is_active: false` and the remote model has `is_active: true`
- **THEN** `oictl manifests diff` reports an update action with `is_active` in the changed fields

#### Scenario: Metadata and params differ
- **WHEN** a model manifest changes nested values under `spec.meta` or `spec.params`
- **THEN** `oictl manifests diff` reports an update action with the changed top-level managed field names

#### Scenario: Server fields are ignored
- **WHEN** the remote model only differs by `user_id`, `created_at`, `updated_at`, or `write_access`
- **THEN** `oictl manifests diff` reports the model as unchanged
