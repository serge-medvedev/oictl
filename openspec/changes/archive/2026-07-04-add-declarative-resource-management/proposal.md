## Why

Stage 2 needs repeatable workspace automation beyond one-off CRUD commands. Operators need a declarative way to describe Open WebUI resources, preview drift, apply intended state, and safely reconcile access grants across environments.

## What Changes

- Add a declarative manifest format for Stage 2 Open WebUI resources with resource identity, desired spec fields, metadata, and access grant declarations.
- Add `apply`, `diff`, and `sync` workflows that read manifest files or directories, compare desired state with remote state, and execute controlled mutations.
- Add dry-run and machine-readable plan output so CI jobs and operators can inspect changes before mutation.
- Add access grant modeling for users, groups, and public/private visibility where supported by Open WebUI resources.
- Keep Open WebUI as the source of authorization truth; manifest grants describe desired server state but do not bypass server permissions.
- Scope this change to Stage 2 Workspace Automation and build on Stage 1 target/profile/API foundations.

## Capabilities

### New Capabilities
- `declarative-resource-management`: Manifest loading, validation, diffing, applying, and syncing desired Open WebUI resource state.
- `access-grant-modeling`: Declarative representation and reconciliation of Open WebUI resource access grants.

### Modified Capabilities

None.

## Impact

- Affected code: `cmd/oictl`, `internal/cli`, and new internal packages for manifest parsing, resource planning, diff rendering, apply execution, and tests.
- Affected commands: new Stage 2 declarative workflows under the `manifests` resource family, such as `oictl manifests apply`, `oictl manifests diff`, and `oictl manifests sync`.
- Affected APIs: Open WebUI REST endpoints for Stage 2 workspace resources, initially including resources with stable create/update/delete and access grant support from models, knowledge, prompts, tools, functions, pipelines, chats, channels, and automations as implemented.
- Dependencies: May add YAML parsing support if manifests support YAML in addition to JSON; any dependency must be justified during implementation.
