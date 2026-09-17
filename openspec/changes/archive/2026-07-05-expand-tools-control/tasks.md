## 1. Command Surface

- [x] 1.1 Add `tools` to the top-level command dispatcher and help output.
- [x] 1.2 Add `oictl tools --help` output documenting `list`, `get`, `create`, `update`, `delete`, `export`, `load-url`, and `access-update`.
- [x] 1.3 Implement tools subcommand argument validation for required tool ids and required JSON payloads.

## 2. Endpoint Commands

- [x] 2.1 Implement `tools list` using `GET /api/v1/tools/list` with existing output conventions.
- [x] 2.2 Implement `tools get <tool-id>` using `GET /api/v1/tools/id/{id}`.
- [x] 2.3 Implement `tools create` using `POST /api/v1/tools/create` with JSON loaded from `--data` or `--file`.
- [x] 2.4 Implement `tools update <tool-id>` using `POST /api/v1/tools/id/{id}/update` with JSON loaded from `--data` or `--file`.
- [x] 2.5 Implement `tools delete <tool-id>` using `DELETE /api/v1/tools/id/{id}/delete`.
- [x] 2.6 Implement `tools access-update <tool-id>` using `POST /api/v1/tools/id/{id}/access/update` with JSON loaded from `--data` or `--file`.
- [x] 2.7 Implement `tools load-url` using `POST /api/v1/tools/load/url` with JSON loaded from `--data` or `--file`.

## 3. Export and Manifest Alignment

- [x] 3.1 Implement default `tools export` using `GET /api/v1/tools/export` with `--out` support for raw server output.
- [x] 3.2 Implement `tools export <tool-id> --manifest` to write one `Tool` manifest JSON document for the selected tool.
- [x] 3.3 Implement `tools export --manifest --directory <path>` to write deterministic per-tool `Tool` manifest JSON files.
- [x] 3.4 Ensure manifest export uses `apiVersion: oictl.openwebui/v1`, `kind: Tool`, tool id as `metadata.name`, ToolForm-managed fields in `spec`, and top-level `access_grants` when present.
- [x] 3.5 Ensure manifest export excludes generated fields such as `user_id`, `specs`, `created_at`, `updated_at`, and `write_access` from `spec`.

## 4. Tests and Verification

- [x] 4.1 Add HTTP-server tests covering tools endpoint paths, methods, payload forwarding, raw export output, and authorization error handling.
- [x] 4.2 Add tests for manifest export shape, deterministic file names, generated-field exclusion, and compatibility with existing manifest loading.
- [x] 4.3 Add tests for tools help output and command validation errors.
- [x] 4.4 Run `gofmt` on changed Go files.
- [x] 4.5 Run `go test ./...` and fix any failures.
- [x] 4.6 Smoke-test `go run ./cmd/oictl --help` and `go run ./cmd/oictl tools --help`.
