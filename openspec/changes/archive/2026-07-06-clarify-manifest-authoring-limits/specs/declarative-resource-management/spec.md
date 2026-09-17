## ADDED Requirements

### Requirement: Manifest authoring uses JSON and simple environment placeholders
The CLI SHALL load declarative resource manifests as JSON documents and SHALL support environment interpolation only for simple `${VAR}` placeholders whose variable name is a standard environment identifier.

#### Scenario: JSON manifest is loaded
- **WHEN** the user runs `oictl manifests diff --file resource.json` with a valid JSON manifest
- **THEN** the CLI validates the JSON manifest envelope and includes the declared resource in the plan

#### Scenario: Non-JSON manifest is rejected
- **WHEN** the user runs `oictl manifests diff --file resource.yaml` with YAML or another non-JSON manifest body
- **THEN** the CLI exits non-zero during manifest loading and sends no remote mutation requests

#### Scenario: Simple environment placeholder is rendered
- **WHEN** a manifest string value contains `${TEAM_NAME}` and the environment contains `TEAM_NAME=sre`
- **THEN** the CLI renders the value as `sre` before validation, planning, or mutation requests

#### Scenario: Shell-style default placeholder is rejected
- **WHEN** a manifest string value contains `${TEAM_NAME:-sre}`
- **THEN** the CLI exits non-zero before contacting Open WebUI and does not render `sre` as a default value
