## ADDED Requirements

### Requirement: Channel webhook commands ensure named incoming webhooks
The webhook command family SHALL support ensuring an incoming webhook for a selected Open WebUI channel by webhook name.

#### Scenario: Existing named webhook is reused
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot`
- **THEN** the CLI lists the channel's incoming webhooks, finds the single webhook whose `name` equals `ci-bot`, skips creation, and returns that webhook metadata with secret-bearing values redacted in default output

#### Scenario: Missing named webhook is created
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --profile-image-url https://images.example/bot.png` and no existing channel webhook named `ci-bot` is found
- **THEN** the CLI creates a channel incoming webhook with the supplied fields and returns the created webhook metadata with secret-bearing values redacted in default output

#### Scenario: Duplicate named webhooks fail safely
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot` and more than one existing channel webhook has the name `ci-bot`
- **THEN** the CLI exits non-zero without creating or updating a webhook and reports that the webhook name is ambiguous without printing full webhook tokens or full webhook URLs

#### Scenario: Webhook name is required
- **WHEN** a user runs `oictl webhooks channels ensure <channel-id>` without a non-empty `--name`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that `--name` is required

### Requirement: Channel webhook ensure handles URLs explicitly and safely
The channel webhook ensure workflow SHALL treat generated webhook tokens and constructed incoming webhook URLs as sensitive values.

#### Scenario: User requests ensured webhook URL on stdout
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --show-url`
- **THEN** the CLI ensures the named webhook, constructs the incoming webhook URL from the configured Open WebUI base URL, ensured webhook ID, and server-returned token, then prints only the full URL

#### Scenario: User writes ensured webhook URL to a file
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --out webhook-url.txt`
- **THEN** the CLI ensures the named webhook, writes the full incoming webhook URL to `webhook-url.txt`, and does not print the full URL to stderr or default status output

#### Scenario: User verifies ensured webhook URL without revealing it
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --verify-url`
- **THEN** the CLI ensures the named webhook, verifies that the ensured webhook response includes enough information to construct an incoming webhook URL, and reports success without printing the full token or full URL

#### Scenario: URL handling modes are exclusive
- **WHEN** a user runs `oictl webhooks channels ensure <channel-id> --name ci-bot` with more than one of `--show-url`, `--out`, or `--verify-url`
- **THEN** the CLI exits non-zero before revealing or writing a webhook URL and explains that only one URL handling mode may be used

#### Scenario: User does not request URL handling
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot` without `--show-url`, `--out`, or `--verify-url`
- **THEN** the CLI returns ensured webhook metadata using default redaction and does not print the full webhook token or full incoming webhook URL
