# users-directory-control Specification

## Purpose
TBD - created by archiving change add-users-directory-and-bulk-settings. Update Purpose after archive.
## Requirements
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

### Requirement: Native users can be retrieved
The CLI SHALL retrieve one native Open WebUI user using normal Open WebUI authentication and an escaped user ID path segment.

#### Scenario: Administrator gets a user
- **WHEN** an administrator runs `oictl users get <user-id>` with exactly one ID
- **THEN** the CLI sends authenticated `GET /api/v1/users/{escaped-user-id}`
- **THEN** the CLI writes the complete successful server JSON through the existing users output pipeline

#### Scenario: Get requires exactly one ID
- **WHEN** an administrator runs `oictl users get` with zero or multiple IDs
- **THEN** the CLI exits non-zero before contacting Open WebUI and prints the command usage

### Requirement: Native users can be created
The CLI SHALL create a native Open WebUI user from pass-through JSON supplied by `--data`, `--file path`, or `--file -`, using normal Open WebUI authentication.

#### Scenario: Administrator creates a user
- **WHEN** an administrator runs `oictl users create --file user.json`
- **THEN** the CLI sends authenticated `POST /api/v1/auths/add` with the supplied payload bytes
- **THEN** the CLI writes the complete successful server JSON through the existing users output pipeline

#### Scenario: Create payload is required
- **WHEN** an administrator runs `oictl users create` without `--data`, `--file path`, or `--file -`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that a user payload is required

### Requirement: Native users can be updated
The CLI SHALL update one native Open WebUI user from pass-through JSON using normal Open WebUI authentication and an escaped user ID path segment.

#### Scenario: Administrator updates a user
- **WHEN** an administrator runs `oictl users update <user-id> --data '{"name":"Updated"}'`
- **THEN** the CLI sends authenticated `POST /api/v1/users/{escaped-user-id}/update` with the supplied payload bytes
- **THEN** the CLI writes the complete successful server JSON through the existing users output pipeline

#### Scenario: Update requires exactly one ID
- **WHEN** an administrator runs `oictl users update` with zero or multiple IDs
- **THEN** the CLI exits non-zero before contacting Open WebUI and prints the command usage

#### Scenario: Update payload is required
- **WHEN** an administrator runs `oictl users update <user-id>` without `--data`, `--file path`, or `--file -`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that a user payload is required

### Requirement: Native users can be deleted safely
The CLI SHALL delete one native Open WebUI user using normal Open WebUI authentication and an escaped user ID path segment, and MUST require explicit confirmation before contacting Open WebUI.

#### Scenario: Delete without confirmation is rejected
- **WHEN** an administrator runs `oictl users delete <user-id>` without `--yes` or `--confirm`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that destructive operations require confirmation

#### Scenario: Confirmed delete is sent
- **WHEN** an administrator runs `oictl users delete <user-id> --yes` or `--confirm`
- **THEN** the CLI sends authenticated `DELETE /api/v1/users/{escaped-user-id}`
- **THEN** the CLI writes the complete successful server JSON through the existing users output pipeline

#### Scenario: Delete requires exactly one ID
- **WHEN** an administrator runs `oictl users delete` with zero or multiple IDs
- **THEN** the CLI exits non-zero before contacting Open WebUI and prints the command usage

### Requirement: Native user CRUD delegates policy and preserves failures
The CLI SHALL delegate admin authorization, user-field validation, duplicate-email handling, password and role rules, target existence, and protected deletion rules to Open WebUI. It SHALL surface server 400, 401, 403, and not-found status/details while redacting configured API or JWT credentials from CLI-generated diagnostics.

#### Scenario: Open WebUI rejects a CRUD request
- **WHEN** Open WebUI returns a validation, authentication, authorization, or not-found response for a native user CRUD command
- **THEN** the CLI exits non-zero and prints the response status and safe detail
- **THEN** any configured credential present in the diagnostic is replaced with `<redacted>`

#### Scenario: Unknown users command is local
- **WHEN** a user runs an unknown `oictl users` subcommand
- **THEN** the CLI exits non-zero before contacting Open WebUI

### Requirement: Users help discovers native CRUD and existing commands
The CLI SHALL document native user CRUD alongside the existing directory and nested settings commands without changing those existing behaviors.

#### Scenario: Users help lists all command groups
- **WHEN** a user runs `oictl users --help`
- **THEN** help lists `get`, `create`, `update`, and `delete` with JSON input and deletion confirmation syntax
- **THEN** help continues to list `list`, `search`, `settings get`, `settings update`, `ui-settings patch`, and `ui-settings bulk-patch`

#### Scenario: Native and SCIM users remain distinct
- **WHEN** a user compares native users help and `oictl scim users`
- **THEN** native CRUD uses normal Open WebUI authentication while SCIM behavior and dedicated SCIM authentication remain unchanged
