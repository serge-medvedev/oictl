# tools-control Specification

## Purpose
TBD - created by archiving change expand-tools-control. Update Purpose after archive.
## Requirements
### Requirement: Tools command family is exposed
The CLI SHALL provide an `oictl tools` resource family for imperative Open WebUI tool operations and SHALL list the supported subcommands in help output.

#### Scenario: Tools help lists supported operations
- **WHEN** the user runs `oictl tools --help`
- **THEN** the help output lists `list`, `get`, `create`, `update`, `delete`, `export`, `load-url`, and `access-update`

#### Scenario: Unknown tools command is rejected
- **WHEN** the user runs `oictl tools unknown`
- **THEN** the CLI exits non-zero and reports that the tools command is unknown

### Requirement: Tools can be listed and inspected
The CLI SHALL provide commands to list accessible tools and inspect one tool by identifier.

#### Scenario: Tools are listed
- **WHEN** the user runs `oictl tools list`
- **THEN** the CLI sends `GET /api/v1/tools/list` to the resolved Open WebUI target and displays the returned tools

#### Scenario: Tool is inspected
- **WHEN** the user runs `oictl tools get weather_tool`
- **THEN** the CLI sends `GET /api/v1/tools/id/weather_tool` and displays the returned tool record

#### Scenario: Tool identifier is required for inspection
- **WHEN** the user runs `oictl tools get`
- **THEN** the CLI exits non-zero before sending a request and reports that a tool id is required

### Requirement: Tools can be created and updated from JSON payloads
The CLI SHALL provide create and update commands that submit Open WebUI tool form JSON loaded from `--data` or `--file`.

#### Scenario: Tool is created from inline JSON
- **WHEN** the user runs `oictl tools create --data '{"id":"weather_tool","name":"Weather","content":"...","meta":{"description":"Weather lookup"}}'`
- **THEN** the CLI sends the JSON payload to `POST /api/v1/tools/create` and displays the created tool response

#### Scenario: Tool is updated from a file
- **WHEN** the user runs `oictl tools update weather_tool --file tool.json`
- **THEN** the CLI sends the loaded JSON payload to `POST /api/v1/tools/id/weather_tool/update` and displays the updated tool response

#### Scenario: Mutation payload is required
- **WHEN** the user runs `oictl tools create` without `--data` or `--file`
- **THEN** the CLI exits non-zero before sending a request and reports that a JSON payload is required

### Requirement: Tools can be deleted
The CLI SHALL provide a delete command that removes one Open WebUI tool by identifier.

#### Scenario: Tool is deleted
- **WHEN** the user runs `oictl tools delete weather_tool`
- **THEN** the CLI sends `DELETE /api/v1/tools/id/weather_tool/delete` and reports the server result

#### Scenario: Tool identifier is required for deletion
- **WHEN** the user runs `oictl tools delete`
- **THEN** the CLI exits non-zero before sending a request and reports that a tool id is required

### Requirement: Tool access grants can be updated
The CLI SHALL provide an access update command that replaces a tool's Open WebUI access grants using the server access update endpoint.

#### Scenario: Tool access grants are updated
- **WHEN** the user runs `oictl tools access-update weather_tool --data '{"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}'`
- **THEN** the CLI sends the JSON payload to `POST /api/v1/tools/id/weather_tool/access/update` and displays the returned tool response

#### Scenario: Access update payload is required
- **WHEN** the user runs `oictl tools access-update weather_tool` without `--data` or `--file`
- **THEN** the CLI exits non-zero before sending a request and reports that a JSON payload is required

#### Scenario: Server filters requested grants
- **WHEN** Open WebUI accepts an access update but returns a tool with fewer grants than requested because of sharing policy
- **THEN** the CLI displays the server response without claiming the filtered grants were applied

### Requirement: Tool source can be loaded from a URL
The CLI SHALL provide a load-url command that asks Open WebUI to fetch tool source from a URL and returns the server-generated tool form data.

#### Scenario: Tool source is loaded from URL
- **WHEN** the user runs `oictl tools load-url --data '{"url":"https://github.com/example/tools/blob/main/weather.py"}'`
- **THEN** the CLI sends the JSON payload to `POST /api/v1/tools/load/url` and displays the returned `name` and `content` fields

#### Scenario: Load-url payload is required
- **WHEN** the user runs `oictl tools load-url` without `--data` or `--file`
- **THEN** the CLI exits non-zero before sending a request and reports that a JSON payload is required

### Requirement: Tools can be exported
The CLI SHALL provide a tools export command that retrieves tool records from Open WebUI and supports raw server output and manifest-compatible output.

#### Scenario: Tools are exported as raw server records
- **WHEN** the user runs `oictl tools export --out tools.json`
- **THEN** the CLI sends `GET /api/v1/tools/export` and writes the returned response to `tools.json`

#### Scenario: One tool is exported as a manifest
- **WHEN** the user runs `oictl tools export weather_tool --manifest --out weather-tool.json`
- **THEN** the CLI writes one JSON manifest with `apiVersion: oictl.openwebui/v1`, `kind: Tool`, `metadata.name: weather_tool`, a `spec` containing ToolForm-managed fields, and top-level `access_grants` when grants are present

#### Scenario: Multiple tools are exported as manifest files
- **WHEN** the user runs `oictl tools export --manifest --directory manifests/tools`
- **THEN** the CLI writes deterministic per-tool JSON manifest files that can be read by `oictl manifests apply --directory manifests/tools`

#### Scenario: Manifest export excludes generated fields
- **WHEN** a remote tool contains `user_id`, `specs`, `created_at`, `updated_at`, or `write_access` fields
- **THEN** `oictl tools export --manifest` excludes those fields from the manifest `spec`

### Requirement: Tools commands preserve common CLI behavior
The CLI SHALL apply existing global target resolution, authentication, output formatting, raw output writing, and HTTP error handling conventions to all tools commands.

#### Scenario: JSON output is requested
- **WHEN** the user runs `oictl tools list --output json`
- **THEN** the CLI prints valid JSON for the returned tool list

#### Scenario: Authorization failure is surfaced
- **WHEN** Open WebUI returns 401 or 403 for a tools command
- **THEN** the CLI exits non-zero and prints the response status and detail without leaking credentials

