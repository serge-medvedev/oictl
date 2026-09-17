## Why

`oictl` does not currently expose Open WebUI function administration, so operators must use the UI or raw API calls to install, audit, export, synchronize, enable, or globalize functions and filters. Adding this control surface closes a Stage 3 extensibility gap while keeping filters aligned with Open WebUI's existing function model.

## What Changes

- Add a `functions` command family for Open WebUI function records, including list, get, create, update, delete, export, load-url, sync, toggle, and toggle-global operations.
- Treat filters as function records whose server-derived `type` is `filter`; do not add a separate `filters` command family unless implementation discovers an upstream API distinction that cannot be represented cleanly under `functions`.
- Preserve Open WebUI's plugin loading and validation behavior by submitting function source and metadata to the server rather than locally executing or reclassifying function code.
- Include filter-specific behavior in command output where returned by the server, including active/global state and toggleable-filter metadata.
- Surface Open WebUI authorization, validation, import, and plugin-load errors consistently for function and filter operations.

## Capabilities

### New Capabilities
- `functions-control`: CLI control of Open WebUI functions, including CRUD, export, load-from-URL, sync, activation, global assignment, and filter handling as function records of type `filter`.

### Modified Capabilities

None.

## Impact

- Affected code: `cmd/oictl`, `internal/cli`, Open WebUI API client code, serializers, output formatting, and tests for command routing and HTTP behavior.
- Affected APIs: Open WebUI `/api/v1/functions` endpoints for function listing, admin listing, export, load-url, sync, create, get, update, delete, toggle, and toggle-global.
- Affected user interface: CLI help output, flags for JSON file input and output files, and JSON/table rendering for function records.
- Dependencies: No new dependency is expected; use existing CLI/API/file-output foundations where possible.
