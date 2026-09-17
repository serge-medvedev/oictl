## 1. Command Surface

- [x] 1.1 Add the `manifests` command family to top-level help and command dispatch.
- [x] 1.2 Add `manifests diff`, `manifests apply`, and `manifests sync` help output with file, directory, output, dry-run, scope, and confirmation flags.
- [x] 1.3 Add command tests covering help output, unknown manifest subcommands, and missing required manifest inputs.

## 2. Manifest Loading and Validation

- [x] 2.1 Implement manifest file and directory discovery with deterministic ordering.
- [x] 2.2 Implement manifest document parsing for JSON and decide whether YAML support is included in this slice.
- [x] 2.3 Validate `apiVersion`, `kind`, `metadata.name`, `spec`, and duplicate desired identities before contacting Open WebUI.
- [x] 2.4 Add a supported-kind registry that rejects unsupported manifest kinds with clear diagnostics.
- [x] 2.5 Add unit tests for valid manifests, invalid envelopes, duplicate identities, and unsupported kinds.

## 3. Planning and Diff Output

- [x] 3.1 Define the resource handler interface for identity lookup, current-state fetch, create, update, delete, and access-grant update operations.
- [x] 3.2 Implement plan action types for create, update, delete, replace-grants, unchanged, and unsupported.
- [x] 3.3 Implement desired-versus-current comparison for managed resource spec fields and access grants.
- [x] 3.4 Implement human-readable diff output and `--output json` plan output.
- [x] 3.5 Add planner tests for create, update, grant replacement, unchanged resources, unsupported resources, and JSON output.

## 4. Access Grant Modeling

- [x] 4.1 Implement access grant validation for `principal_type`, `principal_id`, and `permission` using Open WebUI `access_grants` semantics.
- [x] 4.2 Normalize duplicate grants and compare grants without considering server-generated IDs or timestamps.
- [x] 4.3 Represent public read/write grants as `principal_type: user` with `principal_id: "*"` and private access as an empty grant list.
- [x] 4.4 Surface server authorization failures and server-filtered grant differences after grant updates.
- [x] 4.5 Add tests for user grants, group grants, public grants, private grants, invalid grants, deduplication, and server-filtered responses.

## 5. Apply and Sync Execution

- [x] 5.1 Implement `manifests diff` as a non-mutating planner command.
- [x] 5.2 Implement `manifests apply` so it creates or updates declared resources and does not prune omitted remote resources.
- [x] 5.3 Implement `manifests apply --dry-run` and `manifests sync --dry-run` so they produce plans without mutation requests.
- [x] 5.4 Implement `manifests sync --scope` so it reconciles a bounded remote set and plans delete actions for omitted resources.
- [x] 5.5 Require explicit confirmation for sync plans with destructive delete or detach actions.
- [x] 5.6 Add integration-style HTTP tests for diff, apply, dry-run, sync pruning, confirmation failures, and non-2xx Open WebUI responses.

## 6. Initial Resource Handlers

- [x] 6.1 Implement the first supported manifest handlers for knowledge, prompts, and tools using their Open WebUI REST endpoints.
- [x] 6.2 Wire each handler to resource-specific create, update, delete, lookup, and access-grant update APIs.
- [x] 6.3 Ensure unsupported Stage 2 kinds such as channels or automations fail clearly until handlers are implemented.
- [x] 6.4 Add handler tests using HTTP test servers for resource lookup, create, update, delete, grant update, and post-mutation re-fetch behavior.

## 7. Documentation and Verification

- [x] 7.1 Document manifest examples for knowledge, prompts, tools, public grants, group grants, private access, apply, diff, and sync.
- [x] 7.2 Update README usage examples for the `manifests` command family.
- [x] 7.3 Run `go test ./...` and fix regressions.
- [x] 7.4 Run OpenSpec validation/status commands and confirm the change is ready to archive after implementation.
