## Purpose
Define the declarative manifest workflows used to preview, apply, and sync desired Open WebUI resource state.
## Requirements
### Requirement: Manifest documents define desired resource state
The CLI SHALL load declarative resource manifests from one or more files or directories and SHALL validate each document before contacting Open WebUI.

#### Scenario: Valid manifest is loaded
- **WHEN** the user runs `oictl manifests diff --file resource.json`
- **THEN** the CLI validates the manifest envelope and includes the declared resource in the plan

#### Scenario: Invalid manifest is rejected
- **WHEN** a manifest omits `apiVersion`, `kind`, `metadata.name`, or `spec`
- **THEN** the CLI exits non-zero and reports the missing manifest field without sending mutation requests

### Requirement: Manifest authoring uses JSON/YAML and simple environment placeholders
The CLI SHALL load declarative resource manifests as JSON or YAML documents from `.json`, `.yaml`, and `.yml` files and SHALL support environment interpolation only for simple `${VAR}` placeholders whose variable name is a standard environment identifier.

#### Scenario: JSON manifest is loaded
- **WHEN** the user runs `oictl manifests diff --file resource.json` with a valid JSON manifest
- **THEN** the CLI validates the JSON manifest envelope and includes the declared resource in the plan

#### Scenario: YAML manifest file can be selected
- **WHEN** the user runs `oictl manifests diff --file resource.yaml` and the file contains a valid manifest document
- **THEN** the CLI validates the manifest envelope and includes the declared resource in the plan

#### Scenario: YML manifest file can be selected
- **WHEN** the user runs `oictl manifests apply --dry-run --file resource.yml` and the file contains a valid manifest document
- **THEN** the CLI validates the manifest envelope and includes the declared resource in the dry-run plan without sending mutation requests

#### Scenario: Directory discovery includes YAML manifests
- **WHEN** the user runs `oictl manifests diff --directory manifests/` and the directory contains `.json`, `.yaml`, and `.yml` manifest files
- **THEN** the CLI loads all supported manifest files deterministically before validating duplicate identities and planning resources

#### Scenario: JSON manifest semantics are preserved
- **WHEN** the user runs `oictl manifests diff --file resource.json` and the file contains a valid JSON manifest document
- **THEN** the CLI processes the manifest with the same validation, interpolation, duplicate detection, content file handling, and planning semantics as before YAML support was added

#### Scenario: Simple environment placeholder is rendered
- **WHEN** a manifest string value contains `${TEAM_NAME}` and the environment contains `TEAM_NAME=sre`
- **THEN** the CLI renders the value as `sre` before validation, planning, or mutation requests

#### Scenario: YAML environment value is rendered before planning
- **WHEN** a YAML manifest string value contains `${OICTL_TEST_GROUP}` and the environment contains `OICTL_TEST_GROUP=sre`
- **THEN** `oictl manifests diff --file resource.yaml` plans the resource using the rendered value `sre`

#### Scenario: Missing YAML environment value fails before remote requests
- **WHEN** a YAML manifest string value contains `${OICTL_MISSING_VALUE}` and that environment variable is unset
- **THEN** the CLI exits non-zero, reports the missing environment variable and YAML manifest source, and sends no remote mutation requests

#### Scenario: Shell-style default placeholder is rejected
- **WHEN** a manifest string value contains `${TEAM_NAME:-sre}`
- **THEN** the CLI exits non-zero before contacting Open WebUI and does not render `sre` as a default value

#### Scenario: YAML shell-style default placeholder is rejected
- **WHEN** a YAML manifest string value contains `${TEAM_NAME:-sre}`
- **THEN** the CLI exits non-zero before contacting Open WebUI and does not render `sre` as a default value

### Requirement: Manifest command family provides diff apply and sync workflows
The CLI SHALL provide `oictl manifests diff`, `oictl manifests apply`, and `oictl manifests sync` commands for declarative resource workflows.

#### Scenario: Help lists manifest workflows
- **WHEN** the user runs `oictl manifests --help`
- **THEN** the help output lists `diff`, `apply`, and `sync` with short descriptions of preview, non-pruning apply, and reconciliation behavior

#### Scenario: Manifest file can be selected
- **WHEN** the user runs `oictl manifests apply --file resource.json`
- **THEN** the CLI reads desired state from `resource.json` and plans operations for its resources

### Requirement: Manifest workflows resolve access grant principals before planning
The CLI SHALL resolve reference-based access grant principals for all loaded manifests before producing a diff, dry-run apply, apply, or sync plan.

#### Scenario: Diff resolves grant principal references
- **WHEN** the user runs `oictl manifests diff --file resource.json` and the manifest declares an access grant by user email or group name
- **THEN** the CLI resolves the grant principal to an Open WebUI principal ID before comparing desired state with current resource state

#### Scenario: Apply dry run resolves without mutation
- **WHEN** the user runs `oictl manifests apply --dry-run --file resource.json` and the manifest declares an access grant by principal reference
- **THEN** the CLI resolves the principal and prints the planned action without sending create, update, delete, or grant update requests

#### Scenario: Apply sends resolved grant IDs
- **WHEN** the user runs `oictl manifests apply --file resource.json` and the manifest declares an access grant by principal reference
- **THEN** the CLI sends the grant update request with the resolved `principal_id` value

#### Scenario: Sync resolves before destructive confirmation
- **WHEN** the user runs `oictl manifests sync --scope prompts --file manifests/` and the manifests declare access grants by principal reference
- **THEN** the CLI resolves the grant principals before printing the sync plan and before requiring destructive action confirmation

#### Scenario: Resolution failure prevents mutation
- **WHEN** the user runs `oictl manifests apply --file resource.json` and an access grant principal reference cannot be resolved uniquely
- **THEN** the CLI exits non-zero before sending create, update, delete, or grant update requests

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

### Requirement: Manifests support additional workspace resource kinds
The CLI SHALL support `Model`, `Skill`, `Function`, `Group`, and `Channel` manifests in addition to `Knowledge`, `Prompt`, and `Tool`.

#### Scenario: Model manifest is planned
- **WHEN** the user runs `oictl manifests diff --file model.json` and the manifest declares `kind: Model`
- **THEN** the CLI validates the manifest and plans create, update, replace-grants, unchanged, or delete actions using the Open WebUI model resource state

#### Scenario: Skill manifest is planned
- **WHEN** the user runs `oictl manifests diff --file skill.json` and the manifest declares `kind: Skill`
- **THEN** the CLI validates the manifest and plans create, update, replace-grants, unchanged, or delete actions using the Open WebUI skill resource state

#### Scenario: Function manifest is planned
- **WHEN** the user runs `oictl manifests diff --file function.json` and the manifest declares `kind: Function`
- **THEN** the CLI validates the manifest and plans create, update, unchanged, or delete actions using the Open WebUI function resource state

#### Scenario: Group manifest is planned
- **WHEN** the user runs `oictl manifests diff --file group.json` and the manifest declares `kind: Group`
- **THEN** the CLI validates the manifest and plans create, update, unchanged, or delete actions using the Open WebUI group resource state

#### Scenario: Channel manifest is planned
- **WHEN** the user runs `oictl manifests diff --file channel.json` and the manifest declares `kind: Channel`
- **THEN** the CLI validates the manifest and plans create, update, replace-grants, unchanged, or delete actions using the Open WebUI channel resource state

### Requirement: Sync scopes include all supported manifest kinds
The CLI SHALL accept sync scopes for every supported manifest kind and SHALL include all supported kinds when `--scope all` is selected.

#### Scenario: Model sync scope is selected
- **WHEN** the user runs `oictl manifests sync --scope models --file manifests/ --dry-run`
- **THEN** the CLI reconciles remote `Model` resources against the manifest set without pruning other supported kinds

#### Scenario: All sync scope includes new kinds
- **WHEN** the user runs `oictl manifests sync --scope all --file manifests/ --dry-run`
- **THEN** the CLI includes `Knowledge`, `Prompt`, `Tool`, `Model`, `Skill`, `Function`, `Group`, and `Channel` resources in the sync plan

#### Scenario: Singular sync scope is accepted
- **WHEN** the user runs `oictl manifests sync --scope channel --file manifests/ --dry-run`
- **THEN** the CLI treats the scope as the `Channel` resource kind

### Requirement: Resource-specific manifest handlers preserve existing workflow semantics
The CLI SHALL apply resource-specific Open WebUI endpoint shapes while preserving existing manifest diff, apply, sync, dry-run, JSON output, and duplicate identity behavior.

#### Scenario: Apply does not prune omitted new-kind resources
- **WHEN** the user runs `oictl manifests apply --file model.json` and another remote model is omitted from the manifest set
- **THEN** the CLI leaves the omitted remote model unchanged

#### Scenario: Sync plans omitted new-kind resources for deletion
- **WHEN** the user runs `oictl manifests sync --scope skills --file manifests/ --dry-run` and a remote skill in scope is absent from the manifests
- **THEN** the CLI includes a delete action for that skill in the sync plan

#### Scenario: Duplicate identity is rejected across new kinds
- **WHEN** two input files declare the same `kind` and `metadata.name` for `Model`, `Skill`, `Function`, `Group`, or `Channel`
- **THEN** the CLI exits non-zero before contacting Open WebUI and reports the duplicate manifest identity

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

### Requirement: Model manifest diff normalizes optional null metadata
The CLI SHALL treat absent and `null` values for optional model `spec.meta` fields as equivalent when comparing manifest-declared model state with current Open WebUI model state.

#### Scenario: Applied model manifest remains unchanged when server adds null profile image
- **WHEN** a `kind: Model` manifest omits `spec.meta.profile_image_url`, `oictl manifests apply` succeeds, and Open WebUI returns the model with `meta.profile_image_url: null`
- **THEN** a subsequent `oictl manifests diff --file model.json` reports the model as unchanged rather than planning an update for `meta`

#### Scenario: Explicit non-null metadata still diffs
- **WHEN** a `kind: Model` manifest declares a non-null optional value under `spec.meta` and the remote model has that field absent or `null`
- **THEN** `oictl manifests diff --file model.json` reports an update for `meta`

### Requirement: Terminal server connection manifests are supported as config-backed resources
The CLI SHALL accept `TerminalServerConnection` manifests that describe desired config-backed terminal server connection state and optional access grants for manifest diff and apply workflows.

#### Scenario: Terminal server connection manifest is diffed
- **WHEN** the user runs `oictl manifests diff --file terminal-server.json` and the manifest declares `kind: TerminalServerConnection`
- **THEN** the CLI resolves the connection from `TERMINAL_SERVER_CONNECTIONS` by `metadata.name`, compares declared `spec` fields and optional `access_grants`, and reports update or grant replacement actions without mutating Open WebUI

#### Scenario: Terminal server connection manifest is applied
- **WHEN** the user runs `oictl manifests apply --file terminal-server.json` and the manifest declares `kind: TerminalServerConnection`
- **THEN** the CLI creates or replaces only the declared connection inside `TERMINAL_SERVER_CONNECTIONS`, maps top-level manifest `access_grants` to the connection's nested `config.access_grants`, and preserves undeclared terminal server connections

#### Scenario: Terminal server connection manifest omits grants
- **WHEN** a `TerminalServerConnection` manifest omits `access_grants`
- **THEN** the CLI compares and applies declared connection spec fields without changing the connection's existing `config.access_grants`

### Requirement: Terminal server connection manifests are excluded from destructive sync pruning
The CLI SHALL NOT delete or detach terminal server connections merely because they are omitted from a manifest sync set in this change.

#### Scenario: Terminal server connection is omitted during sync
- **WHEN** the user runs `oictl manifests sync` with manifests that do not declare an existing terminal server connection
- **THEN** the CLI leaves the omitted terminal server connection unchanged

### Requirement: Manifest string values render environment variables
The CLI SHALL render `${VAR}` placeholders in manifest string values from the process environment before manifest validation, planning, or mutation requests.

#### Scenario: Environment value is rendered before planning
- **WHEN** a manifest string value contains `${OICTL_TEST_GROUP}` and the environment contains `OICTL_TEST_GROUP=sre`
- **THEN** `oictl manifests diff` plans the resource using the rendered value `sre`

#### Scenario: Missing environment value fails before remote requests
- **WHEN** a manifest string value contains `${OICTL_MISSING_VALUE}` and that environment variable is unset
- **THEN** the CLI exits non-zero, reports the missing environment variable and manifest source, and sends no remote mutation requests

### Requirement: Manifest content files populate content fields
The CLI SHALL support `spec.content_file` for manifest resources that use `spec.content`, reading the referenced file and using its contents as the desired `spec.content` value before diff, apply, or sync planning.

#### Scenario: Relative content file is loaded
- **WHEN** a `Skill`, `Function`, `Prompt`, or `Tool` manifest declares `spec.content_file` with a relative path
- **THEN** the CLI resolves the path relative to that manifest file, reads the file, removes `content_file` from the desired spec, and plans against `spec.content`

#### Scenario: Content file path can be environment-rendered
- **WHEN** a manifest declares `spec.content_file: "${OICTL_CONTENT_DIR}/body.py"` and `OICTL_CONTENT_DIR` is set
- **THEN** the CLI renders the path before reading the content file

#### Scenario: Content and content file are mutually exclusive
- **WHEN** a manifest declares both `spec.content` and `spec.content_file`
- **THEN** the CLI exits non-zero and reports the ambiguous content declaration before contacting Open WebUI

#### Scenario: Unreadable content file fails before remote requests
- **WHEN** a manifest declares `spec.content_file` that cannot be read
- **THEN** the CLI exits non-zero, reports the manifest source and content file path, and sends no remote mutation requests

### Requirement: Active state reconciliation is explicit
The CLI SHALL document and preserve field-managed active-state reconciliation for manifest kinds that expose `spec.is_active` as a managed field.

#### Scenario: Declared active state is reconciled
- **WHEN** a manifest for a kind that supports `spec.is_active` declares `spec.is_active: false` and the remote resource is active
- **THEN** `oictl manifests diff` reports an update for `is_active`, and `oictl manifests apply` reconciles the remote active state through the resource update workflow

#### Scenario: Omitted active state is preserved
- **WHEN** a manifest for a kind that supports `spec.is_active` omits `spec.is_active`
- **THEN** `oictl manifests diff` and `oictl manifests apply` do not plan or send an active-state change solely because the remote resource has a different active state

### Requirement: Global valve manifests define desired valve state
The CLI SHALL accept `ToolValve` and `FunctionValve` manifest documents in `oictl.openwebui/v1` manifests and SHALL treat `metadata.name` as the owning Open WebUI tool or function ID whose global valve values are managed by `spec`.

#### Scenario: Valid tool valve manifest is loaded
- **WHEN** a manifest declares `kind: ToolValve`, `metadata.name: weather_tool`, and `spec` containing valve fields
- **THEN** the CLI validates the document and includes `ToolValve/weather_tool` in the manifest plan

#### Scenario: Valid function valve manifest is loaded
- **WHEN** a manifest declares `kind: FunctionValve`, `metadata.name: fn_a`, and `spec` containing valve fields
- **THEN** the CLI validates the document and includes `FunctionValve/fn_a` in the manifest plan

#### Scenario: Valve manifest rejects access grants
- **WHEN** a `ToolValve` or `FunctionValve` manifest declares top-level `access_grants`
- **THEN** the CLI exits non-zero before contacting Open WebUI and reports that access grants are unsupported for valve manifests

#### Scenario: Valve manifest identity is used as the owner ID
- **WHEN** a manifest declares `kind: ToolValve` and `metadata.name: weather_tool`
- **THEN** the CLI uses `weather_tool` as the owner ID for global tool valve endpoint requests without resolving a separate display name

### Requirement: Manifest workflows reconcile global valve resources
The CLI SHALL include `ToolValve` and `FunctionValve` resources in `oictl manifests diff` and `oictl manifests apply` workflows using existing manifest plan and dry-run semantics.

#### Scenario: Missing global valves are planned for creation
- **WHEN** a `ToolValve` manifest declares global valves for an existing tool whose global valve endpoint returns no values
- **THEN** `oictl manifests diff` reports a create action for that `ToolValve` without mutating Open WebUI

#### Scenario: Existing global valves differ
- **WHEN** a `FunctionValve` manifest declares valve values that differ from the remote global function valve values
- **THEN** `oictl manifests diff` reports an update action with the changed valve field names

#### Scenario: Declared global valves are applied
- **WHEN** the user runs `oictl manifests apply --file tool-valves.json` for a valid `ToolValve` manifest
- **THEN** the CLI updates the owning tool's global valves using the resolved Open WebUI target

#### Scenario: Dry run does not update global valves
- **WHEN** the user runs `oictl manifests apply --dry-run --file function-valves.json` for a valid `FunctionValve` manifest
- **THEN** the CLI reports the planned action and does not send a valve update request

### Requirement: Global valve manifests are excluded from pruning
The CLI SHALL NOT delete, reset, or detach global valve state merely because a `ToolValve` or `FunctionValve` manifest is omitted from a manifest sync set.

#### Scenario: Tool valve is omitted during sync all
- **WHEN** the user runs `oictl manifests sync --scope all --file manifests/ --dry-run` and a remote tool has global valve values but no manifest declares `ToolValve` for that tool
- **THEN** the CLI leaves the omitted global tool valve values unchanged

#### Scenario: Valve-specific sync scope is rejected
- **WHEN** the user runs `oictl manifests sync --scope tool-valves --file manifests/`
- **THEN** the CLI exits non-zero and reports that the sync scope is unsupported

### Requirement: Per-user valve desired state is unsupported
The CLI SHALL NOT accept declarative manifest kinds for user-scoped tool or function valve values in this change.

#### Scenario: User valve manifest kind is rejected
- **WHEN** a manifest declares `kind: ToolUserValve` or `kind: FunctionUserValve`
- **THEN** the CLI exits non-zero before contacting Open WebUI and reports that the manifest kind is unsupported
