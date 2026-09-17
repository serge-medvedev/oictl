## MODIFIED Requirements

### Requirement: Manifests model Open WebUI access grants
The CLI SHALL support an `access_grants` manifest field using Open WebUI grant entries with `principal_type`, `permission`, and either a resolved `principal_id` or one manifest-only principal selector for manifest kinds that support resource grant reconciliation.

#### Scenario: User grant is declared with principal id
- **WHEN** a manifest for a grant-capable kind includes an access grant with `principal_type: user`, a non-empty `principal_id`, and `permission: read`
- **THEN** the CLI accepts the grant as desired access state for that resource

#### Scenario: User grant is declared with email
- **WHEN** a manifest for a grant-capable kind includes an access grant with `principal_type: user`, `principal_email: user@example.com`, and `permission: read`
- **THEN** the CLI resolves the email to the matching Open WebUI user ID before comparing or applying grants

#### Scenario: User grant is declared with name
- **WHEN** a manifest for a grant-capable kind includes an access grant with `principal_type: user`, `principal_name: User One`, and `permission: read`
- **THEN** the CLI resolves the name to the matching Open WebUI user ID before comparing or applying grants

#### Scenario: Group grant is declared with principal id
- **WHEN** a manifest for a grant-capable kind includes an access grant with `principal_type: group`, a non-empty `principal_id`, and `permission: write`
- **THEN** the CLI accepts the grant as desired access state for that resource

#### Scenario: Group grant is declared with name
- **WHEN** a manifest for a grant-capable kind includes an access grant with `principal_type: group`, `principal_name: SRE`, and `permission: write`
- **THEN** the CLI resolves the name to the matching Open WebUI group ID before comparing or applying grants

#### Scenario: Explicit principal ref is declared
- **WHEN** a manifest for a grant-capable kind includes an access grant with `principal_type: group`, `principal_ref: group:SRE`, and `permission: read`
- **THEN** the CLI resolves the ref within the declared principal type before comparing or applying grants

#### Scenario: Unsupported grant kind is declared
- **WHEN** a `Function` or `Group` manifest includes `access_grants`
- **THEN** the CLI exits non-zero before mutation and reports that access grants are unsupported for that manifest kind

## ADDED Requirements

### Requirement: Grant principal references resolve before reconciliation
The CLI SHALL resolve manifest-only access grant principal references to Open WebUI principal IDs before comparing desired grants to current grants or sending grant update requests.

#### Scenario: Referenced user matches current grant by id
- **WHEN** a manifest declares a user grant by `principal_email` and the remote resource already has a grant for that user's Open WebUI ID and permission
- **THEN** `oictl manifests diff` reports the grant as unchanged

#### Scenario: Referenced group differs from current grant
- **WHEN** a manifest declares a group grant by `principal_name` and the remote resource has no grant for the resolved group ID and permission
- **THEN** `oictl manifests diff` reports a `replace-grants` action using the resolved group ID

#### Scenario: Public grant is not resolved through users
- **WHEN** a manifest declares `principal_type: user`, `principal_id: "*"`, and `permission: read`
- **THEN** the CLI preserves the public grant without querying the user directory for `*`

### Requirement: Grant principal references fail safely
The CLI SHALL reject unresolved, ambiguous, or type-incompatible principal references before mutation.

#### Scenario: Missing referenced user is rejected
- **WHEN** a manifest declares `principal_type: user` with a `principal_email` that does not match an Open WebUI user
- **THEN** the CLI exits non-zero before mutation and reports that the user principal could not be resolved

#### Scenario: Ambiguous referenced name is rejected
- **WHEN** a manifest declares a principal by `principal_name` and more than one principal of the declared type has that exact name
- **THEN** the CLI exits non-zero before mutation and reports that the principal reference is ambiguous

#### Scenario: Principal ref type mismatch is rejected
- **WHEN** a manifest declares `principal_type: user` with `principal_ref: group:SRE`
- **THEN** the CLI exits non-zero before mutation and reports that the principal ref type does not match the grant principal type

#### Scenario: Multiple principal selectors are rejected
- **WHEN** a manifest access grant includes both `principal_id` and `principal_email`
- **THEN** the CLI exits non-zero and reports that only one principal selector is allowed
