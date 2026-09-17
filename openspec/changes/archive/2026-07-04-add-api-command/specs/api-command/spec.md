## ADDED Requirements

### Requirement: CLI exposes a low-level API command
The CLI SHALL expose `oictl api` as a low-level command for making HTTP requests to an Open WebUI API endpoint.

#### Scenario: Help lists the API command
- **WHEN** a user runs `oictl --help`
- **THEN** the help output includes the `api` command with a short description

#### Scenario: API command help is available
- **WHEN** a user runs `oictl api --help`
- **THEN** the help output describes the required endpoint argument and supported request flags

### Requirement: API command resolves the target Open WebUI URL
The API command SHALL resolve the Open WebUI base URL from an explicit `--url` flag or the `OPEN_WEBUI_URL` environment variable, with the flag taking precedence.

#### Scenario: URL flag is supplied
- **WHEN** a user runs `oictl api --url http://localhost:3000 /api/models`
- **THEN** the request target is `http://localhost:3000/api/models`

#### Scenario: URL environment variable is supplied
- **WHEN** `OPEN_WEBUI_URL` is `http://localhost:3000` and the user runs `oictl api /api/models`
- **THEN** the request target is `http://localhost:3000/api/models`

#### Scenario: URL is missing
- **WHEN** no `--url` flag is supplied and `OPEN_WEBUI_URL` is unset
- **THEN** the command exits non-zero and explains that an Open WebUI URL is required

### Requirement: API command authenticates requests by default with bearer credentials
The API command SHALL read the Open WebUI API credential from `OPEN_WEBUI_API_KEY` by default and send it as `Authorization: Bearer <credential>` unless a custom API key header is selected.

#### Scenario: API key environment variable is supplied
- **WHEN** `OPEN_WEBUI_API_KEY` is `sk-test` and the user runs `oictl api /api/models`
- **THEN** the outgoing request includes `Authorization: Bearer sk-test`

#### Scenario: API key is missing
- **WHEN** no API credential is available and the user runs `oictl api /api/models`
- **THEN** the command exits non-zero and explains that an Open WebUI API credential is required

### Requirement: API command supports custom API key header authentication
The API command SHALL support an explicit custom API key header option for Open WebUI deployments that cannot use the `Authorization` header.

#### Scenario: Custom API key header is supplied
- **WHEN** `OPEN_WEBUI_API_KEY` is `sk-test` and the user runs `oictl api --api-key-header x-api-key /api/models`
- **THEN** the outgoing request includes `x-api-key: sk-test`
- **THEN** the outgoing request does not include an automatically generated `Authorization` header

#### Scenario: Custom API key header is empty
- **WHEN** a user runs `oictl api --api-key-header "" /api/models`
- **THEN** the command exits non-zero and explains that the custom API key header name cannot be empty

### Requirement: API command supports scriptable request construction
The API command SHALL support method selection, repeated request headers, and request body input suitable for shell scripts.

#### Scenario: Method flag is supplied
- **WHEN** a user runs `oictl api --method POST /api/v1/models/import`
- **THEN** the outgoing request uses the `POST` method

#### Scenario: Method flag is omitted
- **WHEN** a user runs `oictl api /api/models`
- **THEN** the outgoing request uses the `GET` method

#### Scenario: Header flag is repeated
- **WHEN** a user runs `oictl api --header Accept:application/json --header Content-Type:application/json /api/models`
- **THEN** both headers are included in the outgoing request

#### Scenario: Inline body is supplied
- **WHEN** a user runs `oictl api --method POST --data '{"name":"example"}' /api/example`
- **THEN** the outgoing request body is the supplied JSON string

#### Scenario: Body file is supplied
- **WHEN** a user runs `oictl api --method POST --data-file payload.json /api/example`
- **THEN** the outgoing request body is read from `payload.json`

### Requirement: API command returns response content and status-oriented exits
The API command SHALL stream successful response bodies to stdout, report failed HTTP responses on stderr, and return non-zero for transport errors and non-2xx HTTP responses.

#### Scenario: API returns success with a body
- **WHEN** Open WebUI responds with HTTP 200 and a JSON body
- **THEN** the command writes the response body to stdout
- **THEN** the command exits with status 0

#### Scenario: API returns an error with a body
- **WHEN** Open WebUI responds with HTTP 401 and a response body
- **THEN** the command writes the HTTP status and response body to stderr
- **THEN** the command exits non-zero

#### Scenario: Transport fails
- **WHEN** the request cannot connect to the target Open WebUI URL
- **THEN** the command writes the transport error to stderr
- **THEN** the command exits non-zero

### Requirement: API command protects credentials in normal output
The API command SHALL NOT print API credentials in help, validation errors, transport errors, or HTTP status diagnostics generated by the CLI.

#### Scenario: Credential is invalid
- **WHEN** `OPEN_WEBUI_API_KEY` contains an invalid credential and Open WebUI returns HTTP 401
- **THEN** the CLI-generated stderr output does not include the credential value

#### Scenario: User requests help
- **WHEN** a user runs `oictl api --help`
- **THEN** the help output names supported credential sources without printing any credential value
