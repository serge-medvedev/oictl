## Why

Global tool and function valve values can be changed imperatively today, but they cannot be represented as declarative desired state alongside other Open WebUI resources. Adding manifest support lets operators review, apply, and version global valve configuration while avoiding unresolved semantics for user-scoped valve state.

## What Changes

- Add `ToolValve` manifests for declaring desired global valve values for an existing Open WebUI tool identified by tool ID.
- Add `FunctionValve` manifests for declaring desired global valve values for an existing Open WebUI function identified by function ID.
- Include global valve manifests in manifest load, validate, diff, apply, and dry-run workflows.
- Treat valve manifests as non-owning configuration resources that update only global valve values and do not create, delete, or prune owning tools/functions.
- Defer per-user tool/function valve desired state; user-scoped valves remain available only through existing imperative valve commands.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities
- `declarative-resource-management`: Add manifest kinds for desired global tool and function valve state.
- `valves-control`: Define how declarative global valve reconciliation uses the existing global valve get/spec/update behavior while excluding user-scoped valves.

## Impact

- Manifest registry, validation, diff planning, apply execution, and sync scope handling for the new `ToolValve` and `FunctionValve` kinds.
- Existing tool/function global valve client operations reused by declarative handlers.
- Tests for manifest validation, diff/apply planning, dry-run behavior, and exclusion of per-user valve manifests.
