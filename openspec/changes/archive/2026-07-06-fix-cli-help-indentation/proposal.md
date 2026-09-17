## Why

Some CLI help output for recently added command families has odd indentation, which makes usage examples and command lists harder to scan. Normalizing this presentation keeps help text consistent and professional without changing command parsing or API behavior.

## What Changes

- Normalize help indentation for `oictl tasks --help`, `oictl tools --help`, `oictl users --help`, `oictl webhooks --help`, and `oictl manifests sync --help` or the sync help section under manifests.
- Preserve all command names, flags, arguments, exit codes, request behavior, and response formatting outside of help text.
- Add or update tests that assert the affected help text has consistent indentation and still includes the documented commands.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `cli-core-api`: Require help output for affected command families to use consistent indentation while preserving existing help availability and command behavior.

## Impact

- Affected code: `internal/cli/app.go`, `internal/cli/surfaces.go`, and `internal/cli/manifests.go` help text literals or helpers.
- Affected tests: CLI help tests under `internal/cli/*_test.go`.
- APIs/dependencies: none.
