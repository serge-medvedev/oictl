## 1. Manifest Normalization

- [x] 1.1 Add a manifest normalization step after JSON parsing that walks manifest string values and renders `${VAR}` from the process environment.
- [x] 1.2 Return source-aware validation errors when an environment placeholder references an unset variable.
- [x] 1.3 Add `spec.content_file` resolution that renders the path, resolves relative paths from the manifest file directory, reads the file, assigns `spec.content`, and removes `content_file` before planning.
- [x] 1.4 Reject manifests that declare both `spec.content` and `spec.content_file` before any remote requests.

## 2. Tests

- [x] 2.1 Add manifest loading tests for successful environment rendering in spec fields and access grants.
- [x] 2.2 Add manifest loading tests for missing environment variables with source-aware errors.
- [x] 2.3 Add manifest tests proving relative and environment-rendered `content_file` paths populate `spec.content` and do not leave `content_file` in planned desired state.
- [x] 2.4 Add validation tests for ambiguous `content` plus `content_file` and unreadable content files.
- [x] 2.5 Add or update active-state reconciliation tests proving declared `is_active` diffs and omitted `is_active` is preserved.

## 3. Documentation

- [x] 3.1 Update `docs/manifests.md` with environment rendering syntax, missing-variable behavior, and examples.
- [x] 3.2 Update `docs/manifests.md` with `content_file` usage, relative path behavior, and mutual exclusion with `content`.
- [x] 3.3 Update `docs/manifests.md` to explicitly document `is_active` reconciliation semantics for manifest-managed resources.

## 4. Verification

- [x] 4.1 Run `go test ./...` and fix any failures.
- [x] 4.2 Run OpenSpec validation/status for `enhance-manifest-authoring` and confirm the change is apply-ready.
