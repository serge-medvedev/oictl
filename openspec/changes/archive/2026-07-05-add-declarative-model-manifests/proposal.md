## Why

`oictl manifests` can declaratively manage prompts, knowledge, and tools, but models still require imperative `oictl models` commands for create, update, sync, access grants, and active-state changes. Adding `kind: Model` manifests lets operators preview and reconcile model configuration consistently with the existing declarative resource workflow while preserving the current imperative model command family.

## What Changes

- Add support for `kind: Model` documents in `oictl.openwebui/v1` manifests.
- Allow model manifests to participate in `oictl manifests diff`, `apply`, and `sync` plans, including create, update, unchanged, delete during scoped sync, and access-grant replacement actions.
- Define model field comparison for managed model fields, including active state, metadata, params, base model identity, display/name fields, and other supported Open WebUI model payload fields.
- Preserve existing imperative `oictl models` commands and their endpoint behavior.
- Extend manifest documentation and examples with model-specific manifest shape and sync scope usage.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `declarative-resource-management`: Add `Model` as a supported manifest kind and sync scope for declarative diff/apply/sync workflows.
- `models-control`: Require declarative model manifests to reconcile model records without changing existing imperative `oictl models` commands.
- `access-grant-modeling`: Require model manifest access grants to use the existing grant representation, comparison, validation, and reconciliation behavior.

## Impact

- Affected CLI surface: `oictl manifests diff`, `oictl manifests apply`, `oictl manifests sync`, `oictl manifests --help`, and `oictl models` preservation tests.
- Affected implementation areas: manifest handler registration, model endpoint mapping, model identity lookup, managed field comparison, access grant update handling, plan output, docs, and tests.
- No new external dependencies are expected.
