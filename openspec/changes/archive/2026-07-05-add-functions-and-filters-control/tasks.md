## 1. Command Surface

- [x] 1.1 Review existing command registration, API request helpers, JSON body loading, output-file writing, confirmation handling, and table/JSON output patterns.
- [x] 1.2 Register the `functions` command family with help text for list, get, create, update, delete, export, load-url, sync, toggle, and toggle-global.
- [x] 1.3 Add help text and examples explaining that Open WebUI filters are managed as function records with server-returned `type: filter`.

## 2. Read and Output Commands

- [x] 2.1 Implement `functions list` against the Open WebUI function list endpoint with JSON output and compact table output including id, name, type, active, global, and timestamps.
- [x] 2.2 Implement `functions list --type filter` by filtering returned records by server-returned type without adding a separate `filters` command family.
- [x] 2.3 Implement `functions get <function-id>` with full JSON output that preserves source content and metadata.
- [x] 2.4 Add tests for list, filter-list, get, output formatting, and not-found or forbidden responses.

## 3. Lifecycle Commands

- [x] 3.1 Implement `functions create --file <path>` and stdin JSON payload support using Open WebUI's function create route.
- [x] 3.2 Implement `functions update <function-id> --file <path>` and stdin JSON payload support using Open WebUI's function update route.
- [x] 3.3 Implement `functions delete <function-id> --yes` with confirmation failure before any HTTP request in non-interactive contexts.
- [x] 3.4 Add tests for create/update payload forwarding, server-derived `type` preservation for filters, delete confirmation, delete success, and server validation errors.

## 4. Export Load-URL and Sync

- [x] 4.1 Implement `functions export --out <path>` with output-file writing and JSON stdout support.
- [x] 4.2 Implement `functions export --include-valves --out <path>` by passing the Open WebUI export option for valves.
- [x] 4.3 Implement `functions load-url <url>` by forwarding the URL to Open WebUI and rendering the returned name/content payload.
- [x] 4.4 Implement `functions sync --file <path> --yes` and stdin JSON payload support, documenting that sync can remove omitted remote functions.
- [x] 4.5 Add tests for export output files, include-valves query parameters, load-url payloads and errors, sync payload forwarding, and sync confirmation failures before any HTTP request.

## 5. Toggle Commands

- [x] 5.1 Implement `functions toggle <function-id>` using the Open WebUI function active-state toggle route.
- [x] 5.2 Implement `functions toggle-global <function-id>` using the Open WebUI global-state toggle route.
- [x] 5.3 Add tests for toggle request paths, returned active/global state rendering, and forbidden/not-found responses.

## 6. Documentation and Verification

- [x] 6.1 Update command documentation or README examples for function CRUD, filter listing, export, load-url, sync, toggle, and toggle-global.
- [x] 6.2 Ensure help/docs warn that function source is arbitrary Python loaded by Open WebUI and should only come from trusted sources.
- [x] 6.3 Run Go formatting on changed Go files.
- [x] 6.4 Run the full Go test suite.
- [x] 6.5 Run OpenSpec validation/status checks for `add-functions-and-filters-control`.
