## Why

Open WebUI now exposes Skills as first-class workspace resources for reusable model instructions, sharing, activation, and export. `oictl` currently has controls for adjacent workspace resources such as models, files, knowledge, and automations, but no dedicated way to manage Skills without falling back to raw API calls.

## What Changes

- Add a `skills` command family for listing/searching, inspecting, creating, updating, toggling, deleting, exporting, and access-grant updates for Open WebUI Skills.
- Map commands to Open WebUI's native Skills API endpoints and preserve server-side permission checks for workspace, import/export, sharing, and public-sharing policies.
- Support JSON payloads as the canonical input and output format for Skills operations.
- Add optional Markdown Skill manifest support for create/update/export if implementation confirms it can be done with a small, dependency-light parser/serializer. The manifest format should be limited to documented frontmatter fields plus Markdown content, not a new persisted resource model.
- Require explicit confirmation for destructive skill deletion in non-interactive usage.

## Capabilities

### New Capabilities
- `skills-control`: Defines commands for managing Open WebUI Skills, including list, get, create, update, toggle, delete, export, and access-update operations with optional Markdown Skill manifest support.

### Modified Capabilities
- None.

## Impact

- Affected CLI command routing and help output: `internal/cli/app.go` and `internal/cli/surfaces.go`.
- Affected tests: CLI routing, command-to-endpoint mapping, payload handling, deletion confirmation, export output, and optional manifest conversion tests.
- Open WebUI APIs used: `/api/v1/skills/`, `/api/v1/skills/list`, `/api/v1/skills/export`, `/api/v1/skills/create`, `/api/v1/skills/id/{id}`, `/api/v1/skills/id/{id}/update`, `/api/v1/skills/id/{id}/access/update`, `/api/v1/skills/id/{id}/toggle`, and `/api/v1/skills/id/{id}/delete`.
- No new external service dependencies are required.
