## 1. Close the input coverage inventory

- [x] 1.1 Reconcile the design's discovery inventory with every actual JSON consumer in `app.go`, `surfaces.go`, `terminal_servers.go`, and `manifests.go`, and the `commandFlagAllowlist`; record actual invocation templates, aliases, sources, fixed/dynamic/local shape classification, and allowed-but-unused flags in one bounded test inventory.
- [x] 1.2 Trace each fixed form to an identified upstream router/model and each local form to its parser/normalizer; capture concise source locators and required/optional/nullable fields, nested structure, defaults, alternatives, and omission semantics. Include metadata and every supported manifest kind; distinguish non-JSON files and resource-defined fields.

## 2. Make action-level help reachable

- [x] 2.1 Add failing help tests for `users create --help`, nested actions with multiple selectors, both help flags, absent selectors, and selectors supplied through the existing grammar.
- [x] 2.2 Implement the smallest private help routing needed before execution prerequisites, reusing existing dispatch/parser conventions. Preserve family/top-level help and recognize option values correctly; add no new public command/flag or general command registry.
- [x] 2.3 Verify help does not consume stdin, open JSON/input directories, resolve profiles/targets, contact HTTP, or write `--out`; cover help-like values and representative confirmation/input prerequisites without building an option-combination matrix.

## 3. Document every consuming action

- [x] 3.1 Add source-grounded input references and compact valid examples for `auth`, `tasks`, `models`, `config`, `files`, and `knowledge`, including optional bodies, ancillary metadata, nested structures, and existing shorthand/envelopes.
- [x] 3.2 Add equivalent references for `channels`, `webhooks`, `groups`, `users`, `chats`, `automations`, and `scim`, including convenience-field interactions, replacement/omission behavior, and deployment-extension distinctions.
- [x] 3.3 Add references for `functions`, `skills`, and `tools`; explain resource-defined valves with their existing schema-inspection recipes and distinguish Markdown manifests from JSON bodies.
- [x] 3.4 Document the local JSON manifest envelope and every supported kind's input fields/conditions in each manifest-consuming action's help, reusing shared authored fragments without adding a schema browsing subsystem.
- [x] 3.5 Document `api` and provider `request` inputs as endpoint-owned contracts with specific illustrative examples; document fixed provider configuration/verification envelopes and clearly identify variable nested options.
- [x] 3.6 Make family help point to action help, add minimal README discovery examples, and review all new copy for neutral input-source descriptions and absence of unsolicited operational/security guidance.

## 4. Verify completeness and preserved behavior

- [x] 4.1 Exercise every reconciled inventory entry through `--help` and `-h`, checking action-specific usage/input content and valid example JSON; verify dynamic-input explanations and existing schema recipes. Account for every actual consumer and alias, not only a sampled family or raw allowlist count.
- [x] 4.2 Check required fields/defaults/nested shapes against their recorded source contracts; exercise representative published fixed, nested, envelope, metadata, and manifest examples through the actual CLI using bounded local fixtures. Reuse existing non-help request/normalization/confirmation/output regression tests.
- [x] 4.3 Run focused help tests, `go test ./... -count=1`, `go vet ./...`, build a task-local binary, and smoke top-level/family/nested help without server configuration. Run strict named/all OpenSpec validation and `git diff --check`.
- [x] 4.4 Review the complete implementation against the finite inventory and guardrails: every production edit must support accurate help, its routing, or preserved behavior. Require no missing JSON surface, inaccurate contract, unreachable/mutating help, contradiction, or execution regression; stop when these outcomes and the bounded checks pass.

## 5. Enforce the implementation scope ceiling

- [x] 5.1 Confirm the implementation is the smallest complete solution: static help plus minimal private helpers, existing loaders/serializers/admission/request paths retained, and no new dependency, framework migration, general registry/schema engine, code generation, runtime fetch/cache/version mechanism, schema export surface, new validation/coercion/defaulting, or unrelated API repair. Additional sophistication requires an explicit new request.
