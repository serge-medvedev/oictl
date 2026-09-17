## MODIFIED Requirements

### Requirement: Channel webhook ensure handles URLs explicitly and safely
The channel webhook ensure workflow SHALL treat generated webhook tokens and constructed incoming webhook URLs as sensitive values and SHALL support secret-safe validation of the derived URL against an expected value.

#### Scenario: User requests ensured webhook URL on stdout
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --show-url`
- **THEN** the CLI ensures the named webhook, constructs the incoming webhook URL from the configured Open WebUI base URL, ensured webhook ID, and server-returned token, then prints only the full URL

#### Scenario: User writes ensured webhook URL to a file
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --out webhook-url.txt`
- **THEN** the CLI ensures the named webhook, writes the full incoming webhook URL to `webhook-url.txt`, and does not print the full URL to stderr or default status output

#### Scenario: User verifies ensured webhook URL without revealing it
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --verify-url`
- **THEN** the CLI ensures the named webhook, verifies that the ensured webhook response includes enough information to construct an incoming webhook URL, and reports success without printing the full token or full URL

#### Scenario: User verifies ensured webhook URL against an expected value
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --verify-url --expected-url <expected-url>`
- **THEN** the CLI ensures the named webhook, constructs the incoming webhook URL, compares it exactly to `<expected-url>`, and reports success without printing the full token, full derived URL, or full expected URL

#### Scenario: User verifies ensured webhook URL against an environment value
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --verify-url --expected-url-env WEBHOOK_URL` and `WEBHOOK_URL` contains the expected incoming webhook URL
- **THEN** the CLI reads the expected URL from `WEBHOOK_URL`, ensures the named webhook, compares the constructed incoming webhook URL exactly to the environment value, and reports success without printing the full token, full derived URL, or full expected URL

#### Scenario: Expected URL implies verification
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --expected-url <expected-url>`
- **THEN** the CLI treats the command as non-revealing URL verification with exact expected URL comparison

#### Scenario: Expected environment value is missing
- **WHEN** a user runs `oictl webhooks channels ensure <channel-id> --name ci-bot --expected-url-env WEBHOOK_URL` and `WEBHOOK_URL` is unset or empty
- **THEN** the CLI exits non-zero before contacting Open WebUI and reports that the expected URL environment variable is required without printing webhook URL secrets

#### Scenario: Expected ensured webhook URL does not match
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --expected-url <expected-url>` and the constructed incoming webhook URL differs from `<expected-url>`
- **THEN** the CLI exits non-zero and reports the mismatch without printing the full token, full derived URL, or full expected URL

#### Scenario: Expected URL sources are exclusive
- **WHEN** a user runs `oictl webhooks channels ensure <channel-id> --name ci-bot --verify-url --expected-url <expected-url> --expected-url-env WEBHOOK_URL`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that only one expected URL source may be used

#### Scenario: URL handling modes are exclusive
- **WHEN** a user runs `oictl webhooks channels ensure <channel-id> --name ci-bot --show-url --verify-url --expected-url <expected-url>`
- **THEN** the CLI exits non-zero before revealing, writing, verifying, or comparing a webhook URL and explains that URL reveal, URL file output, and URL verification/comparison modes are exclusive

#### Scenario: User does not request URL handling
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot` without `--show-url`, `--out`, `--verify-url`, `--expected-url`, or `--expected-url-env`
- **THEN** the CLI returns ensured webhook metadata using default redaction and does not print the full webhook token or full incoming webhook URL
