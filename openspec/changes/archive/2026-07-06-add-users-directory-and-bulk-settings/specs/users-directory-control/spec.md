## ADDED Requirements

### Requirement: Admin user directory listing
The CLI SHALL provide a command to enumerate Open WebUI users with server-supported pagination and ordering filters.

#### Scenario: Users are listed
- **WHEN** the user runs `oictl users list --page 2 --order-by name --direction asc`
- **THEN** the CLI calls `GET /api/v1/users/` with `page=2`, `order_by=name`, and `direction=asc`
- **THEN** the CLI prints the returned users response

#### Scenario: Users list accepts a query filter
- **WHEN** the user runs `oictl users list --query alice`
- **THEN** the CLI calls `GET /api/v1/users/` with `query=alice`
- **THEN** the CLI prints the returned users response

### Requirement: User directory search
The CLI SHALL provide a command to search Open WebUI users with server-supported query, pagination, and ordering filters.

#### Scenario: Users are searched
- **WHEN** the user runs `oictl users search --query alice --page 1`
- **THEN** the CLI calls `GET /api/v1/users/search` with `query=alice` and `page=1`
- **THEN** the CLI prints the returned users response

#### Scenario: User search preserves authorization failures
- **WHEN** Open WebUI returns 401 or 403 for `oictl users search --query alice`
- **THEN** the CLI exits non-zero and prints the response status and detail without printing configured credentials

### Requirement: User directory command help
The CLI SHALL document user directory commands under the `users` command family.

#### Scenario: Users help includes directory commands
- **WHEN** the user runs `oictl users help`
- **THEN** the help output includes `users list` and `users search`
- **THEN** the help output describes supported query, pagination, and ordering flags
