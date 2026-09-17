## 1. Manifest Handler Implementation

- [x] 1.1 Add specialized `ToolValve` and `FunctionValve` resource handlers to the manifest handler registry.
- [x] 1.2 Implement global valve lookup by authenticated `GET /api/v1/{tools|functions}/id/{id}/valves`, using `metadata.name` as `{id}` and treating JSON `null` as no current resource.
- [x] 1.3 Implement create and update plan execution by authenticated `POST /api/v1/{tools|functions}/id/{id}/valves/update` with the manifest `spec` object as the request body.
- [x] 1.4 Ensure valve handlers reject access grants, do not delete resources, and do not participate in sync pruning scopes.

## 2. Validation And Planning Tests

- [x] 2.1 Add manifest load validation tests for valid `ToolValve` and `FunctionValve` documents.
- [x] 2.2 Add validation tests that `ToolValve` and `FunctionValve` reject `access_grants` and user-scoped valve manifest kinds remain unsupported.
- [x] 2.3 Add planning tests for `create` on remote `null`, `update` on changed global valves, and `unchanged` on matching global valves.
- [x] 2.4 Add sync scope tests proving `--scope all` excludes valve kinds and valve-specific scopes are rejected.

## 3. Apply Workflow Tests

- [x] 3.1 Add dry-run workflow tests proving valve update requests are not sent.
- [x] 3.2 Add apply workflow tests proving `ToolValve` and `FunctionValve` send POST requests to the global valve update endpoints with the manifest `spec` payload.
- [x] 3.3 Add error handling tests proving Open WebUI validation or authorization failures surface consistently.
- [x] 3.4 Add tests proving declarative valve workflows do not call user-scoped valve endpoints.

## 4. Documentation And Verification

- [x] 4.1 Update manifest documentation and help text as needed to list `ToolValve` and `FunctionValve`, describe examples, and note that per-user valves are out of scope.
- [x] 4.2 Document that `ToolValve.metadata.name` and `FunctionValve.metadata.name` are owner IDs, not display names.
- [x] 4.3 Run Go tests for the CLI package.
- [x] 4.4 Run OpenSpec validation/status checks for `add-global-valve-manifests`.
