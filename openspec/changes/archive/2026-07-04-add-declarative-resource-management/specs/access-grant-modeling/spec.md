## ADDED Requirements

### Requirement: Manifests model Open WebUI access grants
The CLI SHALL support an `access_grants` manifest field using Open WebUI grant entries with `principal_type`, `principal_id`, and `permission` fields.

#### Scenario: User grant is declared
- **WHEN** a manifest includes an access grant with `principal_type: user`, a non-empty `principal_id`, and `permission: read`
- **THEN** the CLI accepts the grant as desired access state for that resource

#### Scenario: Group grant is declared
- **WHEN** a manifest includes an access grant with `principal_type: group`, a non-empty `principal_id`, and `permission: write`
- **THEN** the CLI accepts the grant as desired access state for that resource

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
The CLI SHALL apply access grant changes through the supported resource's Open WebUI API and SHALL surface server authorization or sharing-policy filtering in the command result.

#### Scenario: Grant update is authorized
- **WHEN** the user applies a manifest with changed grants for a supported resource and Open WebUI accepts the update
- **THEN** the CLI reports the grant replacement as applied and includes the resource in the success output

#### Scenario: Grant update is forbidden
- **WHEN** Open WebUI returns 401 or 403 for a manifest grant update
- **THEN** the CLI exits non-zero and prints the response status and detail without leaking credentials

#### Scenario: Server filters requested grants
- **WHEN** Open WebUI accepts a grant update but returns fewer grants than requested because of sharing policy
- **THEN** the CLI reports the server-side difference in the command output
