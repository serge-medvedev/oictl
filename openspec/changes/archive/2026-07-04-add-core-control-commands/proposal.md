## Why

`oictl` currently only exposes help and version output, so operators cannot use it to inspect or control an Open WebUI instance. Stage 1 establishes the first practical control surface for the highest-frequency administration and workspace domains: models, config, files, knowledge, and authentication/profile basics.

## What Changes

- Add reusable CLI foundations for Open WebUI API access, including profile-based target configuration, token handling, request execution, output formatting, and consistent errors.
- Add model control commands for listing, inspecting, creating, updating, toggling, deleting, importing, exporting, and syncing model records.
- Add configuration control commands for exporting/importing config and managing Stage 1 config groups such as connections, model defaults, code execution, banners, tool servers, terminal servers, and OAuth client registration.
- Add file control commands for upload, list/search, inspect, content retrieval, rename, delete, processing status, and content update.
- Add knowledge control commands for listing/searching knowledge bases, creating/updating/deleting knowledge bases, managing file membership, access grants, directories, export, reset, sync, and reindex operations.
- Add authentication/profile basics for sign-in, sign-out, current session inspection, API key management, profile updates, timezone updates, password updates, and admin auth configuration inspection/update where supported.
- Keep Stage 1 scoped to direct Open WebUI REST control commands; do not add chats, users/groups administration, tools/functions/prompts, channels, evaluations, analytics, audio/images, or terminal file-system operations in this change.

## Capabilities

### New Capabilities
- `cli-core-api`: Shared command behavior for target profiles, authentication, HTTP requests, output formats, errors, and help conventions.
- `models-control`: CLI control of Open WebUI model records and model configuration endpoints.
- `config-control`: CLI control of Open WebUI instance configuration endpoints included in Stage 1.
- `files-control`: CLI control of Open WebUI file records, uploads, content, processing, and deletion.
- `knowledge-control`: CLI control of Open WebUI knowledge bases, files, directories, access grants, sync, export, reset, and reindex operations.
- `auth-profile-control`: CLI control of sign-in/sign-out, current session/profile basics, API keys, password/timezone updates, and auth configuration basics.

### Modified Capabilities

None.

## Impact

- Affected code: `cmd/oictl`, `internal/cli`, and new internal packages for configuration, API transport, command parsing, serialization, and tests.
- Affected APIs: Open WebUI REST endpoints under `/api/v1/models`, `/api/v1/configs`, `/api/v1/files`, `/api/v1/knowledge`, and `/api/v1/auths`.
- Affected user interface: CLI command tree, help output, flags, exit codes, and JSON/table output.
- Dependencies: May add small Go standard-library-based helpers first; any third-party CLI, config, or table-rendering dependency must be justified during implementation.
