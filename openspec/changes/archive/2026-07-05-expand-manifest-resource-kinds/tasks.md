## 1. Handler Foundation

- [x] 1.1 Add manifest handler metadata for grant support and unsupported grant validation before any remote mutation.
- [x] 1.2 Register `Model`, `Skill`, `Function`, `Group`, and `Channel` manifest kinds with resource-specific handlers where endpoint shapes differ.
- [x] 1.3 Normalize remote state per kind so server-generated fields do not produce spurious manifest diffs.

## 2. Model Manifest Support

- [x] 2.1 Implement `Model` list, lookup, fetch, create, update, delete, and access-grant update behavior using Open WebUI model endpoints.
- [x] 2.2 Ensure `metadata.name` and `spec.id` identity handling produces stable model create/update/delete payloads.
- [x] 2.3 Add model manifest tests for diff, apply, sync delete planning, grant replacement, and filtered grant reporting.

## 3. Additional Resource Kinds

- [x] 3.1 Implement `Skill` manifest lifecycle and dedicated access-grant update behavior.
- [x] 3.2 Implement `Function` manifest lifecycle and reject `access_grants` for function manifests.
- [x] 3.3 Implement `Group` manifest lifecycle and reject `access_grants` for group manifests.
- [x] 3.4 Implement `Channel` manifest lifecycle and access-grant reconciliation through channel create/update payloads.
- [x] 3.5 Add tests for each new kind covering valid planning, apply paths, unsupported grants where applicable, and resource-specific endpoint shapes.

## 4. Sync Scopes and UX

- [x] 4.1 Extend sync scope parsing to accept singular and plural scopes for all supported kinds and include new kinds in `all`.
- [x] 4.2 Update manifests help text, README examples, and `docs/manifests.md` with the new kinds, scopes, and grant support limits.
- [x] 4.3 Add tests for scope validation, `--scope all`, destructive sync confirmation, and duplicate identity rejection across new kinds.

## 5. Verification

- [x] 5.1 Run manifest-focused Go tests and fix any regressions.
- [x] 5.2 Run the repository's standard test suite or document any unavailable verification.
- [x] 5.3 Confirm OpenSpec status reports the change as ready for apply.
