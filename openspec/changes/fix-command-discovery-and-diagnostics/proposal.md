## Why

`oictl --version` is intercepted by ordinary option parsing and fails with `--version requires a value`. The previous JSON-input help improvement covered only JSON-consuming actions, leaving other help paths and parser diagnostics inconsistent; rendered source-reference footers also expose implementation research instead of useful operator information.

## What Changes

- Audit every finite implemented/documented command path, intermediate command group, existing alias, and production `requires a value` emitter. Retain a complete affected-path inventory and distinguish legitimate missing option values from misclassified help, version, and unknown options.
- Make `--help` and `-h` work for every supported command/group, with useful local usage, arguments/options, and child actions where applicable. Preserve existing JSON-input references and examples.
- Route top-level `version`, `--version`, and `-v` to version output before command-specific validation. Informational requests remain offline and inert.
- Give genuinely missing values and unknown/unsupported options their respective nonzero diagnostics plus relevant usage/help discovery. Do not replace input errors with success or execute incomplete operations.
- Remove source-provenance footers and irrelevant links/references to Open WebUI source/API internals from rendered help. Retain useful input behavior and operator-facing CLI recipes; keep accuracy evidence in maintainer comments/tests.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cli-core-api`: Complete informational dispatch, contextual option diagnostics, and end-user-focused help across the command tree.

## Impact

Expected implementation changes are confined to existing parser/dispatch help seams, static help content, focused tests, and minimal README discovery guidance. Existing commands, value-bearing options, JSON contracts, request execution, authorization, confirmation, and output behavior remain intact except for the explicitly corrected informational/diagnostic paths.

## Anti-overengineering Guardrails

Produce the smallest complete solution that satisfies the accepted task. Additional sophistication requires an explicit request. Reuse the current parser, allowlist, static help, and test facilities; every production edit must fix an inventoried informational/diagnostic defect or remove irrelevant rendered references.

No parser framework migration, generalized command/schema registry, generator, runtime OpenAPI fetching, dependency, new public discovery commands/flags, completion system, automatic correction/retry, or unrelated API repair. No source URLs, upstream commit IDs, Python paths/classes, provenance footers, or irrelevant backend-route explanations in end-user help. API method/path examples remain only where the operator actually supplies them, such as passthrough commands. Do not remove meaningful field names, input constraints, or existing resource-schema inspection recipes in the name of cleanup.

## Delivery Boundary

This request delivers a written, validated, independently reviewed, locally committed OpenSpec change with its audit inventory. Runtime implementation, canonical synchronization, archive, installation, and push require later authorization. Implementation tasks remain unchecked.
