## ADDED Requirements

### Requirement: Manifest inputs accept JSON and YAML files
The CLI SHALL accept manifest documents from `.json`, `.yaml`, and `.yml` files through the existing manifest workflow inputs, decoding all supported formats into the same manifest document model before validation and planning.

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

### Requirement: YAML manifest strings render environment variables
The CLI SHALL render `${VAR}` placeholders in YAML manifest string values from the process environment using the same interpolation semantics as JSON manifests before manifest validation, planning, or mutation requests.

#### Scenario: YAML environment value is rendered before planning
- **WHEN** a YAML manifest string value contains `${OICTL_TEST_GROUP}` and the environment contains `OICTL_TEST_GROUP=sre`
- **THEN** `oictl manifests diff --file resource.yaml` plans the resource using the rendered value `sre`

#### Scenario: Missing YAML environment value fails before remote requests
- **WHEN** a YAML manifest string value contains `${OICTL_MISSING_VALUE}` and that environment variable is unset
- **THEN** the CLI exits non-zero, reports the missing environment variable and YAML manifest source, and sends no remote mutation requests

#### Scenario: YAML shell-style default placeholder is rejected
- **WHEN** a YAML manifest string value contains `${TEAM_NAME:-sre}`
- **THEN** the CLI exits non-zero before contacting Open WebUI and does not render `sre` as a default value
