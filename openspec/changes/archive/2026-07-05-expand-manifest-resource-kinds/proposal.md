## Why

Declarative manifests currently cover only `Knowledge`, `Prompt`, and `Tool`, leaving common Open WebUI workspace configuration such as models outside the diff/apply/sync workflow. Expanding manifest kinds makes repeatable workspace automation useful for model records first, then for other resource surfaces where Open WebUI exposes compatible create, update, delete, and grant APIs.

## What Changes

- Add manifest support for `Model` with `diff`, `apply`, and `sync` coverage, including access grant reconciliation through Open WebUI's model access API.
- Add manifest support for `Skill`, `Function`, `Group`, and `Channel` where the upstream API supports the required lifecycle operations.
- Preserve existing manifest envelope rules: `apiVersion`, `kind`, `metadata.name`, `spec`, optional `access_grants`, JSON input only, deterministic directory loading, and duplicate `kind/name` rejection.
- Extend `sync --scope` to accept the new resource scopes and include them in `all`.
- Treat access grants as supported only for manifest kinds backed by Open WebUI access-grant APIs or mutable `access_grants` fields; unsupported grant declarations fail before mutation.
- Update documentation and tests to cover new manifest examples, scope validation, planning, apply, sync, and unsupported grant behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `declarative-resource-management`: Expand supported manifest kinds, sync scopes, and workflow expectations for `Model`, `Skill`, `Function`, `Group`, and `Channel`.
- `access-grant-modeling`: Define which new manifest kinds reconcile grants and how unsupported grant declarations are rejected.
- `models-control`: Require model manifest support to use the existing model lifecycle and access endpoints consistently with imperative model commands.

## Impact

- Affected code: manifest handler registry and resource-specific handlers in `internal/cli/manifests.go`, manifest tests, CLI help text, README, and `docs/manifests.md`.
- Affected APIs: Open WebUI model, skill, function, group, and channel endpoints for list/get/create/update/delete and access update where available.
- No new runtime dependencies are expected.
