## ADDED Requirements

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
