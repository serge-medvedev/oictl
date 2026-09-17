## MODIFIED Requirements

### Requirement: Manifests model Open WebUI access grants
The CLI SHALL support an `access_grants` manifest field using Open WebUI grant entries with `principal_type`, `principal_id`, and `permission` fields for manifest kinds that support resource grant reconciliation.

#### Scenario: User grant is declared
- **WHEN** a manifest for a grant-capable kind includes an access grant with `principal_type: user`, a non-empty `principal_id`, and `permission: read`
- **THEN** the CLI accepts the grant as desired access state for that resource

#### Scenario: Group grant is declared
- **WHEN** a manifest for a grant-capable kind includes an access grant with `principal_type: group`, a non-empty `principal_id`, and `permission: write`
- **THEN** the CLI accepts the grant as desired access state for that resource

#### Scenario: Unsupported grant kind is declared
- **WHEN** a `Function` or `Group` manifest includes `access_grants`
- **THEN** the CLI exits non-zero before mutation and reports that access grants are unsupported for that manifest kind

### Requirement: Grant reconciliation uses resource-specific Open WebUI APIs
The CLI SHALL apply access grant changes through the supported resource's Open WebUI API or mutable resource payload and SHALL surface server authorization or sharing-policy filtering in the command result.

#### Scenario: Grant update is authorized
- **WHEN** the user applies a manifest with changed grants for a supported resource and Open WebUI accepts the update
- **THEN** the CLI reports the grant replacement as applied and includes the resource in the success output

#### Scenario: Grant update is forbidden
- **WHEN** Open WebUI returns 401 or 403 for a manifest grant update
- **THEN** the CLI exits non-zero and prints the response status and detail without leaking credentials

#### Scenario: Server filters requested grants
- **WHEN** Open WebUI accepts a grant update but returns fewer grants than requested because of sharing policy
- **THEN** the CLI reports the server-side difference in the command output

#### Scenario: Channel grants use channel mutation payloads
- **WHEN** a `Channel` manifest changes only `access_grants`
- **THEN** the CLI reconciles the channel grants through the Open WebUI channel mutation surface and verifies the resulting remote grants

## ADDED Requirements

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
