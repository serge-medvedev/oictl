## Purpose
Define how declarative manifests represent, compare, validate, and reconcile Open WebUI access grants.
## Requirements
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

### Requirement: Grant values are validated before mutation
The CLI SHALL reject access grants whose principal type or permission is not supported by Open WebUI's access grant model.

#### Scenario: Unsupported principal type is rejected
- **WHEN** a manifest includes an access grant with `principal_type: role`
- **THEN** the CLI exits non-zero and reports that only `user` and `group` grant principal types are supported

#### Scenario: Unsupported permission is rejected
- **WHEN** a manifest includes an access grant with `permission: admin`
- **THEN** the CLI exits non-zero and reports that only `read` and `write` permissions are supported

### Requirement: Public and private access are represented consistently
The CLI SHALL represent public access as a `user` grant with `principal_id: "*"` and SHALL represent private owner-only access as an empty access grant list.

#### Scenario: Public read grant is declared
- **WHEN** a manifest includes `principal_type: user`, `principal_id: "*"`, and `permission: read`
- **THEN** the CLI treats the resource as publicly readable where Open WebUI allows public sharing for that resource

#### Scenario: Private access is declared
- **WHEN** a manifest includes `access_grants: []`
- **THEN** the CLI plans to remove non-owner grants for that resource where the resource supports grant replacement

### Requirement: Grant comparison ignores server-generated grant metadata
The CLI SHALL compare desired and current access grants by principal type, principal id, and permission, and SHALL ignore server-generated grant identifiers and timestamps.

#### Scenario: Same grant has different server id
- **WHEN** the remote grant has the same `principal_type`, `principal_id`, and `permission` as the manifest but a different `id`
- **THEN** the CLI reports the grant as unchanged

#### Scenario: Duplicate desired grants are declared
- **WHEN** a manifest repeats the same `principal_type`, `principal_id`, and `permission` grant tuple
- **THEN** the CLI normalizes the duplicate entries into one desired grant for planning

### Requirement: Grant reconciliation uses resource-specific Open WebUI APIs
The CLI SHALL apply access grant changes through the supported resource's Open WebUI API or, for config-backed terminal server connections, through the terminal server configuration API, and SHALL surface server authorization or sharing-policy filtering in the command result.

#### Scenario: Grant update is authorized
- **WHEN** the user applies a manifest with changed grants for a supported resource and Open WebUI accepts the update
- **THEN** the CLI reports the grant replacement as applied and includes the resource in the success output

#### Scenario: Config-backed terminal server grant update is authorized
- **WHEN** the user replaces grants for a terminal server connection and Open WebUI accepts the updated terminal server configuration
- **THEN** the CLI reports the grant replacement as applied for the selected terminal server connection

#### Scenario: Grant update is forbidden
- **WHEN** Open WebUI returns 401 or 403 for a manifest grant update or terminal server config grant update
- **THEN** the CLI exits non-zero and prints the response status and detail without leaking credentials

#### Scenario: Server filters requested grants
- **WHEN** Open WebUI accepts a grant update but returns fewer grants than requested because of sharing policy
- **THEN** the CLI reports the server-side difference in the command output

### Requirement: Grant-capable manifest kinds are explicit
The CLI SHALL reconcile `access_grants` for `Knowledge`, `Prompt`, `Tool`, `Model`, `Skill`, and `Channel` manifests and SHALL reject `access_grants` for manifest kinds without grant support.

#### Scenario: Model grants are reconciled
- **WHEN** a `Model` manifest includes `access_grants` and the remote model has different grants
- **THEN** the CLI plans and applies a replace-grants action through the Open WebUI model access update endpoint

#### Scenario: Skill grants are reconciled
- **WHEN** a `Skill` manifest includes `access_grants` and the remote skill has different grants
- **THEN** the CLI plans and applies a replace-grants action through the Open WebUI skill access update endpoint

#### Scenario: Function grants are rejected
- **WHEN** a `Function` manifest includes `access_grants`
- **THEN** the CLI rejects the manifest before sending create, update, delete, or grant update requests

#### Scenario: Group grants are rejected
- **WHEN** a `Group` manifest includes `access_grants`
- **THEN** the CLI rejects the manifest before sending create, update, delete, or grant update requests

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
