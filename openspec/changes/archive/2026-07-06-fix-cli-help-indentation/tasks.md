## 1. Help Text Updates

- [x] 1.1 Inspect current help output for `tasks`, `tools`, `users`, `webhooks`, and `manifests --help` to identify tab-indented or misaligned rows.
- [x] 1.2 Normalize the affected static help strings in `internal/cli/app.go`, `internal/cli/surfaces.go`, and `internal/cli/manifests.go` using the existing two-space section indentation convention.
- [x] 1.3 Confirm no command names, flags, arguments, parsing paths, exit codes, HTTP requests, or non-help output behavior were changed.

## 2. Tests

- [x] 2.1 Add or update focused tests for the affected help output that assert representative aligned rows, absence of tab-indented help rows, and continued presence of documented commands.
- [x] 2.2 Run the CLI test suite with `go test ./...`.
- [x] 2.3 Run OpenSpec validation for `fix-cli-help-indentation`.
