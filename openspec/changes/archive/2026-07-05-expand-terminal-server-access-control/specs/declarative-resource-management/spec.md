## ADDED Requirements

### Requirement: Terminal server connection manifests are supported as config-backed resources
The CLI SHALL accept `TerminalServerConnection` manifests that describe desired config-backed terminal server connection state and optional access grants for manifest diff and apply workflows.

#### Scenario: Terminal server connection manifest is diffed
- **WHEN** the user runs `oictl manifests diff --file terminal-server.json` and the manifest declares `kind: TerminalServerConnection`
- **THEN** the CLI resolves the connection from `TERMINAL_SERVER_CONNECTIONS` by `metadata.name`, compares declared `spec` fields and optional `access_grants`, and reports update or grant replacement actions without mutating Open WebUI

#### Scenario: Terminal server connection manifest is applied
- **WHEN** the user runs `oictl manifests apply --file terminal-server.json` and the manifest declares `kind: TerminalServerConnection`
- **THEN** the CLI creates or replaces only the declared connection inside `TERMINAL_SERVER_CONNECTIONS`, maps top-level manifest `access_grants` to the connection's nested `config.access_grants`, and preserves undeclared terminal server connections

#### Scenario: Terminal server connection manifest omits grants
- **WHEN** a `TerminalServerConnection` manifest omits `access_grants`
- **THEN** the CLI compares and applies declared connection spec fields without changing the connection's existing `config.access_grants`

### Requirement: Terminal server connection manifests are excluded from destructive sync pruning
The CLI SHALL NOT delete or detach terminal server connections merely because they are omitted from a manifest sync set in this change.

#### Scenario: Terminal server connection is omitted during sync
- **WHEN** the user runs `oictl manifests sync` with manifests that do not declare an existing terminal server connection
- **THEN** the CLI leaves the omitted terminal server connection unchanged
