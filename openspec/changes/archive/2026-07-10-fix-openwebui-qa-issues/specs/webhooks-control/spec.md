## ADDED Requirements

### Requirement: Channel webhook ensure returns non-secret output when writing URLs
The channel webhook ensure workflow SHALL keep generated webhook URLs and tokens out of stdout when `--out` is used, while still returning non-secret command results when the user requests structured or table output.

#### Scenario: JSON output is returned while URL is written to a file
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --out webhook-url.txt --output json`
- **THEN** the CLI writes the full incoming webhook URL only to `webhook-url.txt`
- **AND** stdout contains valid JSON describing non-secret ensured webhook metadata and status
- **AND** stdout does not contain the full webhook token or full incoming webhook URL

#### Scenario: Table output is returned while URL is written to a file
- **WHEN** a channel manager runs `oictl webhooks channels ensure <channel-id> --name ci-bot --out webhook-url.txt --output table`
- **THEN** the CLI writes the full incoming webhook URL only to `webhook-url.txt`
- **AND** stdout contains non-secret ensured webhook metadata and status in table form
- **AND** stdout does not contain the full webhook token or full incoming webhook URL
