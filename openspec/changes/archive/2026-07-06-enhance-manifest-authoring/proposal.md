## Why

Large manifest-managed resources such as tools, skills, functions, and prompts are awkward to author because code or prompt bodies must be embedded directly in JSON strings. Manifest users also need a documented way to render environment-specific values and a clearer contract for how declared active-state fields are reconciled.

## What Changes

- Add manifest environment rendering so users can substitute values from the execution environment while keeping manifests reusable across Open WebUI instances.
- Add `content_file` support for manifest kinds with large `content` fields, resolving file content into `spec.content` before validation, diffing, and apply.
- Document explicit active-state reconciliation semantics for manifest-managed resources that expose `is_active`.
- Preserve existing JSON manifest workflows, planning output, sync behavior, and access grant behavior.

## Capabilities

### New Capabilities

### Modified Capabilities
- `declarative-resource-management`: Manifest loading, validation, diff, apply, and sync behavior changes to support environment rendering, `content_file`, and documented active-state reconciliation.

## Impact

- Affects manifest loading and normalization in `internal/cli/manifests.go`.
- Affects manifest tests in `internal/cli/manifests_test.go` and resource-specific manifest coverage where content fields are exercised.
- Affects user documentation in `docs/manifests.md`.
- No API dependency changes are expected; rendered manifests continue to use existing Open WebUI endpoints and payload shapes.
