## 1. Payload Normalization

- [x] 1.0 Keep implementation scoped to the three outbound payload normalization fixes covered by this proposal: chat tag `TagForm`, omitted model `params`, and channel member `active` alias normalization.
- [x] 1.1 Update chat tag `--tag` payload construction in `internal/cli/surfaces.go` so one convenience tag sends a single object with `name` instead of a wrapped array.
- [x] 1.2 Add model payload normalization for `models create`, `models import`, and `models sync` in `internal/cli/app.go` so omitted `params` is sent as `{}`.
- [x] 1.3 Update channel member active payload handling in `internal/cli/surfaces.go` so `active` is normalized to `is_active` and explicit `is_active` takes precedence.

## 2. Tests

- [x] 2.1 Update the chat workflow test in `internal/cli/app_test.go` to expect a single `TagForm` object containing `name` for `chats tags set --tag` and reject multiple convenience tags.
- [x] 2.2 Add or update model command tests in `internal/cli/app_test.go` to verify `create`, `import`, and `sync` send `params: {}` when omitted.
- [x] 2.3 Add or update channel member tests in `internal/cli/app_test.go` to verify `active` alias normalization and explicit `is_active` precedence.

## 3. Verification

- [x] 3.1 Run `go test ./...`.
- [x] 3.2 Run `PATH="$HOME/.local/bin:$PATH" openspec status --change fix-live-payload-ergonomics` and confirm the change is apply-ready.
