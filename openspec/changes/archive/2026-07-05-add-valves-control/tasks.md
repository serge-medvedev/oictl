## 1. Command Wiring

- [x] 1.1 Add `tools` and `functions` top-level dispatch cases and help entries limited to valve operations.
- [x] 1.2 Add help text for `oictl tools valves` and `oictl functions valves` showing global and user-scoped operations.
- [x] 1.3 Ensure unknown or incomplete valve commands return clear usage errors without contacting Open WebUI.

## 2. Valve Request Handling

- [x] 2.1 Implement shared valve command dispatch for `get`, `spec`, and `update` operations under `tools valves` and `functions valves`.
- [x] 2.2 Map global tool valve commands to `/api/v1/tools/id/<tool-id>/valves`, `/api/v1/tools/id/<tool-id>/valves/spec`, and `/api/v1/tools/id/<tool-id>/valves/update`.
- [x] 2.3 Map user tool valve commands to `/api/v1/tools/id/<tool-id>/valves/user`, `/api/v1/tools/id/<tool-id>/valves/user/spec`, and `/api/v1/tools/id/<tool-id>/valves/user/update`.
- [x] 2.4 Map global function valve commands to `/api/v1/functions/id/<function-id>/valves`, `/api/v1/functions/id/<function-id>/valves/spec`, and `/api/v1/functions/id/<function-id>/valves/update`.
- [x] 2.5 Map user function valve commands to `/api/v1/functions/id/<function-id>/valves/user`, `/api/v1/functions/id/<function-id>/valves/user/spec`, and `/api/v1/functions/id/<function-id>/valves/user/update`.
- [x] 2.6 Require `--data`, `--file`, or `--file -` for valve update commands and forward the payload unchanged.

## 3. Output And Error Behavior

- [x] 3.1 Reuse structured authenticated request execution so successful valve responses support existing `--output` and `--out` behavior.
- [x] 3.2 Preserve server status and response body for validation, missing valve, missing resource, and authorization failures.
- [x] 3.3 Verify CLI-generated diagnostics do not print local API credentials.

## 4. Tests And Verification

- [x] 4.1 Add tests for help output and dispatch for `tools valves` and `functions valves` commands.
- [x] 4.2 Add request-mapping tests for all global and user-scoped tool valve endpoints.
- [x] 4.3 Add request-mapping tests for all global and user-scoped function valve endpoints.
- [x] 4.4 Add tests that valve update commands require a payload and support stdin payloads.
- [x] 4.5 Run the Go test suite and OpenSpec validation for `add-valves-control`.
