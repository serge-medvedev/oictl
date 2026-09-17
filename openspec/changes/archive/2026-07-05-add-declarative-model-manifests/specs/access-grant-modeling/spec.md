## ADDED Requirements

### Requirement: Model manifests reconcile access grants
The CLI SHALL support `access_grants` on `kind: Model` manifests using the same grant representation, validation, normalization, comparison, and replacement semantics as other manifest resources.

#### Scenario: Model public read grant is declared
- **WHEN** a model manifest includes `access_grants` with `principal_type: user`, `principal_id: "*"`, and `permission: read`
- **THEN** the CLI treats the model as publicly readable where Open WebUI allows public model sharing

#### Scenario: Model grants differ
- **WHEN** a model manifest declares grants that differ from the remote model grants
- **THEN** `oictl manifests diff` reports a `replace-grants` action with grants added and removed for that model

#### Scenario: Model grants are applied
- **WHEN** `oictl manifests apply --file model.json` applies a model manifest with changed `access_grants`
- **THEN** the CLI replaces the model grants through Open WebUI's model access update API

#### Scenario: Model grant filtering is reported
- **WHEN** Open WebUI accepts a model access update but returns fewer grants than requested because of sharing policy
- **THEN** the CLI exits non-zero or reports the server-side difference in the command output without leaking credentials
