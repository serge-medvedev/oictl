## ADDED Requirements

### Requirement: CLI exposes provider passthrough commands
The CLI SHALL expose `oictl providers` as the command family for guarded Open WebUI provider passthrough operations.

#### Scenario: Help lists provider families
- **WHEN** a user runs `oictl providers --help`
- **THEN** the help output lists OpenAI-compatible and Ollama-compatible provider subcommands that are implemented in this capability

### Requirement: Provider commands route through Open WebUI
Provider passthrough commands SHALL send requests to the configured Open WebUI instance and SHALL NOT contact arbitrary upstream provider URLs directly.

#### Scenario: User requests OpenAI-compatible models
- **WHEN** a user runs `oictl providers openai models`
- **THEN** the CLI requests the Open WebUI OpenAI-compatible models endpoint and returns the server response

#### Scenario: User requests Ollama tags
- **WHEN** a user runs `oictl providers ollama tags`
- **THEN** the CLI requests the Open WebUI Ollama-compatible tags endpoint and returns the server response

#### Scenario: User supplies an external provider URL
- **WHEN** a user attempts to pass a raw external provider URL to a provider passthrough command
- **THEN** the CLI rejects the command before sending the request and explains that passthrough requests are routed through Open WebUI

### Requirement: Provider commands expose configuration and verification operations
Provider commands SHALL support inspecting and updating provider configuration where Open WebUI exposes those operations and SHALL support server-backed connection verification.

#### Scenario: Admin gets OpenAI provider config
- **WHEN** an admin runs `oictl providers openai config get --output json`
- **THEN** the CLI requests the Open WebUI OpenAI provider configuration endpoint and writes the JSON response

#### Scenario: Admin verifies an Ollama connection
- **WHEN** an admin runs `oictl providers ollama verify --file verify.json`
- **THEN** the CLI sends the verification payload to Open WebUI and returns the server verification response

#### Scenario: Non-admin updates provider config
- **WHEN** a non-admin runs a provider config update command
- **THEN** the CLI sends the authenticated request and surfaces the server 401 or 403 response without masking it as a local validation error

### Requirement: Provider raw requests preserve passthrough semantics
Provider raw request commands SHALL preserve request body, method, streaming behavior, response body, and status-oriented exit behavior for Open WebUI provider passthrough paths.

#### Scenario: User sends OpenAI-compatible chat completion
- **WHEN** a user runs `oictl providers openai request --method POST /chat/completions --file request.json`
- **THEN** the CLI sends the JSON payload through Open WebUI and writes the provider response body to stdout

#### Scenario: User sends Ollama-compatible request
- **WHEN** a user runs `oictl providers ollama request --method POST /api/pull --file request.json`
- **THEN** the CLI sends the request through Open WebUI's Ollama passthrough path and writes the response body to stdout

#### Scenario: Provider response streams
- **WHEN** Open WebUI returns a streaming provider response
- **THEN** the CLI streams response chunks to stdout without waiting for the entire response body to be buffered

#### Scenario: Provider returns an error
- **WHEN** Open WebUI or the upstream provider returns a non-2xx response
- **THEN** the CLI exits non-zero and writes the status and response body to stderr

### Requirement: Provider commands protect credentials and unsafe headers
Provider commands SHALL use the configured Open WebUI credential for the request to Open WebUI and SHALL avoid exposing local credentials in help, diagnostics, or raw header output.

#### Scenario: User requests help
- **WHEN** a user runs `oictl providers openai request --help`
- **THEN** the help output documents credential sources without printing credential values

#### Scenario: User supplies Authorization header override
- **WHEN** a user attempts to pass an `Authorization` header override to a provider passthrough command
- **THEN** the CLI rejects the unsafe header unless the implementation explicitly supports a safe override flag for that command

#### Scenario: Provider request fails authentication
- **WHEN** Open WebUI rejects a provider passthrough request because the local credential is invalid
- **THEN** the CLI-generated stderr output does not include the credential value
