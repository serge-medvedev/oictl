## 1. Terminal Server Grant Command Foundation

- [x] 1.1 Add helpers to fetch and decode `/api/v1/configs/terminal_servers` as a `TERMINAL_SERVER_CONNECTIONS` collection while preserving unknown fields.
- [x] 1.2 Implement terminal server connection lookup by exact `id` first and unique `name` second, including not-found and ambiguous-name errors.
- [x] 1.3 Add helpers to read desired grants from either a JSON array or an object containing `access_grants`.
- [x] 1.4 Reuse existing access grant validation, normalization, comparison, and rendering behavior for terminal server grant workflows.

## 2. Config Command Workflows

- [x] 2.1 Add `oictl config terminal-servers access-grants get <connection-id-or-name>` to print the selected connection's `config.access_grants`.
- [x] 2.2 Add `oictl config terminal-servers access-grants diff <connection-id-or-name> --file grants.json` to report added and removed grants without mutation.
- [x] 2.3 Add `oictl config terminal-servers access-grants set <connection-id-or-name> --file grants.json` to replace only the selected connection's `config.access_grants` and post the preserved collection.
- [x] 2.4 Compare returned terminal server grants with desired grants after set and surface server-side filtering or normalization differences.
- [x] 2.5 Update config command help output to document the terminal server access grant workflows.

## 3. Optional Manifest Support

- [x] 3.1 Add a config-backed manifest handler for `TerminalServerConnection` that lists, looks up, creates, updates, and fetches entries from `TERMINAL_SERVER_CONNECTIONS` without a per-resource endpoint.
- [x] 3.2 Map manifest `metadata.name` to terminal server connection identity and ensure identity fields are present when creating a connection.
- [x] 3.3 Map top-level manifest `access_grants` to nested `config.access_grants` while preserving existing grants when the manifest omits `access_grants`.
- [x] 3.4 Include `TerminalServerConnection` in manifest diff and apply workflows while excluding it from destructive sync pruning.
- [x] 3.5 Update manifest help, scope text, and documentation examples for terminal server connection manifests.

## 4. Tests and Verification

- [x] 4.1 Add HTTP-server tests for terminal server grant get, set, and diff request methods, paths, payload preservation, and no-mutation diff behavior.
- [x] 4.2 Add tests for connection lookup by id, lookup by unique name, not-found errors, ambiguous-name errors, and empty or missing `config.access_grants`.
- [x] 4.3 Add tests for grant file parsing, invalid grants, duplicate normalization, public grants, private empty grants, and server-filtered responses.
- [x] 4.4 Add manifest planner/apply tests for `TerminalServerConnection` create, update, grant replacement, unchanged, omitted-grants preservation, and sync non-pruning behavior.
- [x] 4.5 Run `gofmt` on changed Go files and `go test ./...`.
