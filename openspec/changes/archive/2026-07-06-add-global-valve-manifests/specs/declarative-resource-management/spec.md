## ADDED Requirements

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
