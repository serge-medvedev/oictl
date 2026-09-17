## 1. Core CLI and API Foundation

- [x] 1.1 Refactor `internal/cli.App.Run` into a domain/action dispatcher while preserving `help`, `--help`, `version`, and `--version` behavior.
- [x] 1.2 Add global flag parsing for `--profile`, `--base-url`, `--token`, `--output`, `--timeout`, and `--out` where applicable.
- [x] 1.3 Implement target resolution from flags, environment variables, and named profile configuration.
- [x] 1.4 Implement profile configuration storage under the user config directory with restrictive permissions for token-bearing files.
- [x] 1.5 Implement an internal Open WebUI API client for JSON requests, multipart uploads, raw downloads, bearer authentication, timeouts, and response error handling.
- [x] 1.6 Implement shared JSON input loading from `--data` and `--file` for complex request bodies.
- [x] 1.7 Implement shared output handling for JSON, table/list output, concise mutation status, and raw `--out` writes.
- [x] 1.8 Add unit tests for command dispatch, target resolution precedence, token redaction, JSON input loading, output formatting, and HTTP error handling.

## 2. Auth and Profile Commands

- [x] 2.1 Add `auth me`, `auth login`, and `auth logout` commands backed by `/api/v1/auths` session endpoints.
- [x] 2.2 Add optional token persistence for successful login according to the selected profile behavior.
- [x] 2.3 Add `auth profile update`, `auth timezone set`, and `auth password update` commands.
- [x] 2.4 Add `auth api-key get`, `auth api-key create`, and `auth api-key delete` commands.
- [x] 2.5 Add auth admin configuration commands for admin config, LDAP server config, LDAP config, and OAuth config get/set operations.
- [x] 2.6 Add HTTP-server tests for auth/profile command request paths, methods, payload handling, token persistence, and forbidden responses.

## 3. Model Commands

- [x] 3.1 Add model discovery commands for `models list`, `models base`, `models base-tags`, and `models tags`.
- [x] 3.2 Add `models get <model-id>` command for model inspection.
- [x] 3.3 Add model mutation commands for create, update, toggle, access update, delete, and delete-all.
- [x] 3.4 Add model import, export, and sync commands including `--out` support for export.
- [x] 3.5 Add tests covering model command endpoint paths, query parameters, JSON payload forwarding, export output, and authorization errors.

## 4. Config Commands

- [x] 4.1 Add config import, export, and namespace commands.
- [x] 4.2 Add connections config get/set commands.
- [x] 4.3 Add tool server and terminal server config commands, including verify, policy, lifecycle, and refresh operations.
- [x] 4.4 Add code execution config get/set commands.
- [x] 4.5 Add model UI config defaults/get/set commands.
- [x] 4.6 Add suggestions and banners config commands.
- [x] 4.7 Add OAuth client registration command under config control.
- [x] 4.8 Add tests covering config command request mapping, JSON passthrough payloads, raw export output, and admin authorization failures.

## 5. File Commands

- [x] 5.1 Add `files upload` with multipart upload, metadata, processing flags, and background processing flag support.
- [x] 5.2 Add file list, search, and count commands.
- [x] 5.3 Add file get and processing status commands.
- [x] 5.4 Add file content, HTML content, named content, and data-content retrieval commands with raw `--out` support.
- [x] 5.5 Add file data-content update command.
- [x] 5.6 Add file rename, delete, and delete-all commands.
- [x] 5.7 Add tests covering multipart upload, query parameters, raw download writes, JSON updates, deletion, and file authorization failures.

## 6. Knowledge Commands

- [x] 6.1 Add knowledge list, search, and search-files commands.
- [x] 6.2 Add knowledge create, get, update, access update, reset, delete, and export commands.
- [x] 6.3 Add knowledge file membership commands for list, pending, add, update, remove, batch add, and move.
- [x] 6.4 Add knowledge directory create, update, and delete commands.
- [x] 6.5 Add knowledge reindex, metadata reindex, sync diff, and sync cleanup commands.
- [x] 6.6 Add tests covering knowledge command endpoint paths, JSON payload forwarding, export output, and permission errors.

## 7. Documentation and Verification

- [x] 7.1 Update README usage examples for profiles, authentication, output formats, and representative Stage 1 commands.
- [x] 7.2 Ensure top-level and domain help output lists all Stage 1 command groups consistently.
- [x] 7.3 Run `gofmt` on changed Go files.
- [x] 7.4 Run `go test ./...` and fix any failures.
- [x] 7.5 Manually smoke-test `go run ./cmd/oictl --help`, `go run ./cmd/oictl version`, and representative command help for each Stage 1 domain.
