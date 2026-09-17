## ADDED Requirements

### Requirement: Manifest documents define desired resource state
The CLI SHALL load declarative resource manifests from one or more files or directories and SHALL validate each document before contacting Open WebUI.

#### Scenario: Valid manifest is loaded
- **WHEN** the user runs `oictl manifests diff --file resource.json`
- **THEN** the CLI validates the manifest envelope and includes the declared resource in the plan

#### Scenario: Invalid manifest is rejected
- **WHEN** a manifest omits `apiVersion`, `kind`, `metadata.name`, or `spec`
- **THEN** the CLI exits non-zero and reports the missing manifest field without sending mutation requests

### Requirement: Manifest command family provides diff apply and sync workflows
The CLI SHALL provide `oictl manifests diff`, `oictl manifests apply`, and `oictl manifests sync` commands for declarative resource workflows.

#### Scenario: Help lists manifest workflows
- **WHEN** the user runs `oictl manifests --help`
- **THEN** the help output lists `diff`, `apply`, and `sync` with short descriptions of preview, non-pruning apply, and reconciliation behavior

#### Scenario: Manifest file can be selected
- **WHEN** the user runs `oictl manifests apply --file resource.json`
- **THEN** the CLI reads desired state from `resource.json` and plans operations for its resources

### Requirement: Diff previews remote drift without mutation
The CLI SHALL compare desired manifest state with current Open WebUI state and SHALL report create, update, delete, access-grant, unchanged, and unsupported actions without mutating remote state.

#### Scenario: Missing remote resource is planned for creation
- **WHEN** a manifest declares a supported resource that does not exist remotely
- **THEN** `oictl manifests diff` reports a create action for that resource and exits successfully

#### Scenario: Existing remote resource differs
- **WHEN** a manifest declares a supported resource whose remote state has different managed fields
- **THEN** `oictl manifests diff` reports an update action with the changed fields

### Requirement: Apply reconciles declared resources without pruning omitted resources
The CLI SHALL create or update resources declared in manifests and SHALL NOT delete supported remote resources merely because they are omitted from the manifest set.

#### Scenario: Declared resource is applied
- **WHEN** the user runs `oictl manifests apply --file resource.json`
- **THEN** the CLI creates or updates the declared resource using the resolved Open WebUI target

#### Scenario: Omitted resource is preserved during apply
- **WHEN** a remote resource exists in the apply scope but no manifest declares it
- **THEN** `oictl manifests apply` leaves that remote resource unchanged

### Requirement: Sync reconciles a scoped remote set to manifests
The CLI SHALL reconcile the selected remote scope to the manifest set and SHALL make destructive delete or detach actions explicit in the plan before execution.

#### Scenario: Omitted resource is planned for deletion during sync
- **WHEN** the user runs `oictl manifests sync --scope prompts --file manifests/` and a prompt in that scope is absent from the manifests
- **THEN** the CLI includes a delete action for that prompt in the sync plan

#### Scenario: Destructive sync requires explicit approval
- **WHEN** a sync plan contains delete or detach actions and the user has not supplied the required confirmation flag
- **THEN** the CLI exits non-zero and prints the destructive actions that require approval

### Requirement: Dry run and JSON plan output are supported
The CLI SHALL support dry-run execution and machine-readable JSON plan output for manifest workflows.

#### Scenario: Apply dry run does not mutate resources
- **WHEN** the user runs `oictl manifests apply --dry-run --file resource.json`
- **THEN** the CLI prints the planned actions and does not send create, update, delete, or grant update requests

#### Scenario: JSON plan is requested
- **WHEN** the user runs `oictl manifests diff --output json --file manifests/`
- **THEN** the CLI prints valid JSON containing the planned action list, resource identifiers, and changed field summaries

### Requirement: Unsupported manifest kinds fail clearly
The CLI SHALL reject manifest kinds that do not have a registered resource handler.

#### Scenario: Unsupported kind is declared
- **WHEN** a manifest declares `kind: UnknownResource`
- **THEN** the CLI exits non-zero and reports that the manifest kind is unsupported
