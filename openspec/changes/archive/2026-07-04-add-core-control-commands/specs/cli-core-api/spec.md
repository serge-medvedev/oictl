## ADDED Requirements

### Requirement: Domain command dispatch
The CLI SHALL dispatch commands using a domain/action command tree and SHALL keep existing `help` and `version` behavior available.

#### Scenario: Existing help remains available
- **WHEN** the user runs `oictl help` or `oictl --help`
- **THEN** the CLI exits with status 0 and prints top-level usage including Stage 1 domains

#### Scenario: Unknown command fails predictably
- **WHEN** the user runs an unknown command
- **THEN** the CLI exits non-zero and prints the unknown command plus relevant usage to stderr

### Requirement: Target resolution
The CLI SHALL resolve the Open WebUI target from explicit global flags first, environment variables second, and the selected named profile third.

#### Scenario: Explicit target overrides profile
- **WHEN** a command is run with `--base-url` and `--token` while a profile is selected
- **THEN** the request uses the explicit base URL and token values

#### Scenario: Missing target is rejected
- **WHEN** a command that requires Open WebUI access has no resolvable base URL
- **THEN** the CLI exits non-zero and explains how to provide a base URL

### Requirement: Profile storage
The CLI SHALL support named profiles containing at least a base URL and optional bearer token, stored under the user's config directory with restrictive permissions when secrets are present.

#### Scenario: Profile is used for API command
- **WHEN** the user runs an API command with `--profile prod`
- **THEN** the CLI loads the `prod` profile and uses its target settings for the request

#### Scenario: Token is not exposed
- **WHEN** the CLI prints profile information or an error
- **THEN** stored bearer token values are not printed in full

### Requirement: Authenticated HTTP requests
The CLI SHALL send HTTP requests to the resolved Open WebUI base URL, include bearer authentication when a token is available, enforce a timeout, and support JSON, multipart, and raw response bodies.

#### Scenario: Bearer token is sent
- **WHEN** a token is resolved for an API command
- **THEN** the CLI sends an `Authorization: Bearer <token>` header

#### Scenario: Server error is surfaced
- **WHEN** Open WebUI returns a non-2xx response
- **THEN** the CLI exits non-zero and prints the HTTP status plus any response detail without leaking tokens

### Requirement: Structured input
The CLI SHALL allow complex request bodies to be supplied as inline JSON or from a JSON file for create, update, import, and configuration commands.

#### Scenario: JSON file input is submitted
- **WHEN** the user runs a mutation command with `--file payload.json`
- **THEN** the CLI reads the file as JSON and submits it as the request body

### Requirement: Output formats
The CLI SHALL support `--output json` for all structured responses and MAY provide compact table output for list-style responses.

#### Scenario: JSON output is requested
- **WHEN** a command returns structured data and the user passes `--output json`
- **THEN** the CLI prints valid JSON representing the response

#### Scenario: Raw output is written
- **WHEN** a content or export command is run with `--out path`
- **THEN** the CLI writes the raw response body to the specified path and does not table-format it
