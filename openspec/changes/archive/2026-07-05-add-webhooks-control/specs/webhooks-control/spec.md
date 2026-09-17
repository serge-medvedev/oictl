## ADDED Requirements

### Requirement: CLI exposes webhook control commands
The CLI SHALL expose `oictl webhooks` as the command family for Open WebUI webhook management operations.

#### Scenario: Help lists webhook subdomains
- **WHEN** a user runs `oictl webhooks --help`
- **THEN** the help output lists channel incoming webhook and global event webhook subcommands that are implemented in this capability

### Requirement: Channel webhook commands manage incoming webhooks
The webhook command family SHALL support listing, creating, retrieving, updating, and deleting incoming webhooks for a selected Open WebUI channel.

#### Scenario: Channel manager lists webhooks
- **WHEN** a channel manager runs `oictl webhooks channels list <channel-id>`
- **THEN** the CLI requests the channel webhook list from Open WebUI and renders the returned webhook metadata without printing full webhook tokens or full webhook URLs in default output

#### Scenario: Channel manager creates a webhook
- **WHEN** a channel manager runs `oictl webhooks channels create <channel-id> --name ci-bot`
- **THEN** the CLI sends the channel webhook create request to Open WebUI and returns the created webhook metadata with secret-bearing values redacted in default output

#### Scenario: Channel manager updates a webhook
- **WHEN** a channel manager runs `oictl webhooks channels update <channel-id> <webhook-id> --name deploy-bot`
- **THEN** the CLI sends the channel webhook update request to Open WebUI and returns the updated webhook metadata

#### Scenario: Channel manager deletes a webhook with confirmation
- **WHEN** a channel manager runs `oictl webhooks channels delete <channel-id> <webhook-id> --yes`
- **THEN** the CLI sends the channel webhook delete request to Open WebUI and reports the server deletion result

#### Scenario: User omits delete confirmation
- **WHEN** a user runs `oictl webhooks channels delete <channel-id> <webhook-id>` in a non-interactive context without `--yes`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that confirmation is required

### Requirement: Channel webhook URL handling is explicit and safe
Channel webhook commands SHALL treat generated webhook tokens and constructed incoming webhook URLs as sensitive values.

#### Scenario: User requests a channel webhook URL
- **WHEN** a channel manager runs `oictl webhooks channels url <channel-id> <webhook-id> --show-url`
- **THEN** the CLI constructs the incoming webhook URL from the configured Open WebUI base URL, webhook ID, and server-returned token, then prints the full URL

#### Scenario: User writes a channel webhook URL to a file
- **WHEN** a channel manager runs `oictl webhooks channels url <channel-id> <webhook-id> --out webhook-url.txt`
- **THEN** the CLI writes the full incoming webhook URL to `webhook-url.txt` and does not print the full URL to stderr or default status output

#### Scenario: User does not explicitly reveal a URL
- **WHEN** a channel manager runs a channel webhook list, get, create, or update command without an explicit reveal or output-file option
- **THEN** the CLI redacts full webhook tokens and full incoming webhook URLs from default human-readable output

#### Scenario: Non-manager requests channel webhook metadata
- **WHEN** Open WebUI rejects a channel webhook request with 401 or 403
- **THEN** the CLI exits non-zero and prints the server authorization response without printing local credentials, webhook tokens, or full webhook URLs

### Requirement: Event webhook commands expose the event catalog
The webhook command family SHALL support inspecting the Open WebUI event catalog used by global event webhook filters.

#### Scenario: Admin lists event catalog
- **WHEN** an admin runs `oictl webhooks events catalog`
- **THEN** the CLI requests the Open WebUI event catalog and renders the returned event names and descriptions

#### Scenario: Non-admin lists event catalog
- **WHEN** Open WebUI rejects the event catalog request with 401 or 403
- **THEN** the CLI exits non-zero and surfaces the server authorization response

### Requirement: Event webhook commands manage global event webhooks
The webhook command family SHALL support listing, creating, retrieving, updating, enabling, disabling, and deleting global event webhooks through Open WebUI.

#### Scenario: Admin lists event webhooks
- **WHEN** an admin runs `oictl webhooks events list`
- **THEN** the CLI requests global event webhooks from Open WebUI and renders webhook names, enabled states, event filters, and targets while redacting destination URLs in default output

#### Scenario: Admin creates an event webhook
- **WHEN** an admin runs `oictl webhooks events create --file event-webhook.json`
- **THEN** the CLI sends the event webhook payload to Open WebUI and returns the created webhook metadata with the destination URL redacted in default output

#### Scenario: Admin updates event filters and targets
- **WHEN** an admin runs `oictl webhooks events update <webhook-id> --file event-webhook.json`
- **THEN** the CLI sends the replacement or partial update payload to Open WebUI and returns the updated webhook metadata

#### Scenario: Admin disables an event webhook
- **WHEN** an admin runs `oictl webhooks events disable <webhook-id>`
- **THEN** the CLI requests an update that sets the event webhook enabled state to false and reports the updated state

#### Scenario: Admin enables an event webhook
- **WHEN** an admin runs `oictl webhooks events enable <webhook-id>`
- **THEN** the CLI requests an update that sets the event webhook enabled state to true and reports the updated state

#### Scenario: Admin deletes an event webhook with confirmation
- **WHEN** an admin runs `oictl webhooks events delete <webhook-id> --yes`
- **THEN** the CLI sends the event webhook delete request to Open WebUI and reports the server deletion result

### Requirement: Event webhook URL handling is explicit and safe
Event webhook commands SHALL treat outbound destination webhook URLs as sensitive values.

#### Scenario: Admin requests event webhook JSON
- **WHEN** an admin runs `oictl webhooks events get <webhook-id> --output json`
- **THEN** the CLI writes the complete server-returned event webhook JSON, including URL fields if returned by Open WebUI

#### Scenario: Admin uses default event webhook output
- **WHEN** an admin runs an event webhook list, get, create, update, enable, or disable command without JSON output or an explicit URL reveal option
- **THEN** the CLI redacts full destination webhook URLs from default human-readable output

#### Scenario: Server rejects invalid event webhook payload
- **WHEN** Open WebUI rejects an event webhook create or update request because an event filter, target, or URL is invalid
- **THEN** the CLI exits non-zero and prints the server validation response without printing local credentials or unrelated webhook secret values

#### Scenario: Non-admin mutates event webhooks
- **WHEN** Open WebUI rejects an event webhook mutation with 401 or 403
- **THEN** the CLI exits non-zero and surfaces the server authorization response without attempting a client-side permission bypass
