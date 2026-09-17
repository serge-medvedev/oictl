## Why

Terminal server connections are configured through the Open WebUI config API, but their access grants are not yet manageable as a first-class `oictl` workflow. Operators need to inspect, replace, and preview terminal server sharing changes without editing the entire terminal server config blob by hand.

## What Changes

- Add first-class grant inspection for a named terminal server connection from the config-backed terminal server connection set.
- Add grant replacement for a named terminal server connection using the existing `access_grants` tuple model and validation rules.
- Add a terminal server grant diff workflow that compares desired grants with current config state without mutating Open WebUI.
- Add optional manifest support for terminal server connections so declarative workflows can manage terminal server connection spec fields and, when declared, access grants.
- Keep terminal server creation/update backed by `/api/v1/configs/terminal_servers`; do not introduce a separate terminal server resource API.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `config-control`: Add terminal server connection access grant get, set, and diff workflows over config-backed terminal server connections.
- `declarative-resource-management`: Allow terminal server connection manifests to participate in diff/apply workflows when manifest support is enabled for this resource kind.
- `access-grant-modeling`: Clarify that grant reconciliation applies to config-backed terminal server connections as well as resource-specific access endpoints.

## Impact

- Affected code: `internal/cli` config command handling, manifest resource handlers/planning, shared access grant validation/diff helpers, documentation, and tests.
- Affected commands: `oictl config terminal-servers` subcommands for access grant get/set/diff, plus optional `oictl manifests diff/apply` support for a terminal server connection kind.
- Affected APIs: Open WebUI `/api/v1/configs/terminal_servers` get/set endpoint; existing verification/policy/lifecycle/refresh endpoints remain unchanged.
- Dependencies: No new third-party dependency is expected.
