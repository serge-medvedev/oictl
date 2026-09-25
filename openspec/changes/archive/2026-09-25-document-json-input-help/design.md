## Context

See `proposal.md` for motivation and `specs/cli-core-api/spec.md` for behavior. This is cross-cutting help work in the existing hand-written Go CLI, not a new command system.

At oictl revision `32c0469ba54f7d7c79a7b96cc282a15f69bb12c6`, `internal/cli/command_flags.go` assigns `bodyFlags` to 107 command keys. That is a discovery seed, not proof that every allowed flag is consumed. Actual input handling lives in `app.go`, `surfaces.go`, `terminal_servers.go`, and `manifests.go`. `files upload --metadata` and manifest documents demonstrate why scanning `bodyFlags` alone is insufficient. Existing valve `spec` commands provide resource-specific schema inspection.

The current users dispatcher serves family help, then passes action arguments to the ordinary flag parser; `users create --help` fails with `--help requires a value`. Nested commands have different action depths and selector counts. Help routing must recognize actual grammar rather than blindly joining the first words or scanning every argument for a help-looking string.

The user-create example was checked against `backend/open_webui/models/auths.py` (`SignupForm`, `AddUserForm`) and `routers/auths.py` (`add_user`) in the available Open WebUI source revision `0aa65cd1c9d42d6458976598b0551f3e37968ace`. This identifies the inspected source, not a universal version guarantee or a claim about a deployed server. Existing CLI normalization and local document parsing remain authoritative for what the caller supplies.

## Goals / Non-Goals

**Goals:**
- A user can ask any JSON-consuming action for help and construct its input from that reference.
- All current input surfaces receive explicit coverage, including optional bodies, nested actions, aliases, and local/ancillary JSON.
- Help stays static, offline, concise, and independent of the action's execution prerequisites.

**Non-Goals:**
- Changing execution contracts, repairing unrelated API mismatches, adding endpoints, or implementing new input formats.
- Machine-readable schema export, runtime schema discovery, automatic validation/coercion/defaulting, or support-version negotiation.
- Refactoring the CLI into a framework, generating an SDK/help site, adding services/dependencies, or making live infrastructure a help-test prerequisite.

## Decisions

### 1. Extend the existing help path with minimal private routing

Retain current dispatch, parser, flag allowlist, loaders, and family help. Add only the help-specific routing needed to identify an existing action and print its reference before input/target/execution checks. A small private help lookup and shared formatting/text helpers are acceptable; a general command registry owning dispatch, validation, routes, and schemas is not.

Support `oictl <action-path> --help` and `-h` without required selectors, and the corresponding actual invocation grammar with selectors. Preserve existing supported help spellings. Do not add a new `schema` command, `--help-schema`, a `help <path>` grammar, or positional `help` aliases as part of this change. Parse enough option boundaries to distinguish help flags from values; do not treat string content within JSON as a flag. For optional-body actions, absence of a body does not remove the input reference.

Alternative rejected: rewriting dispatch around a third-party framework would expand the change and risk unrelated behavior. Fetching OpenAPI at help time would require a server and credentials and still miss local envelopes.

### 2. Use one authoritative help entry per action, sharing text only where contracts match

Use ordinary Go text/constants plus small helpers. Render sections for Usage, Input, and Examples, with source-option descriptions where relevant. Document root type, known fields, nested objects/array elements, required/optional/nullable values, default owner, conditional fields, omission semantics, and actual accepted alternatives. Do not infer enums from common values or call an unconstrained string an enum. Defaults describe existing behavior; documentation does not insert them.

Include compact valid examples for distinct input forms. Required nested fields must be visible; a skeletal `meta: object` is insufficient when that object has a known contract. Free-form extension maps can be identified as such instead of enumerating every arbitrary key. Shared shapes such as access grants can reuse one text fragment, but each action's rendered help must be useful without opening source code or searching another document.

For fixed-shape inputs, trace the selected request route to its form/model and relevant transformation. For local inputs, trace the parser and normalization. Retain a compact source locator with the help/test entry, including an identified upstream revision when used; do not add a separate provenance service, pinned source dependency, or copied upstream repository. Correct any discovered documentation assumptions; report genuine runtime contract defects separately rather than fixing them under help scope.

Alternative rejected: a formal JSON Schema catalog plus renderer/validator would create a second contract system. Static reference text is sufficient for this task.

### 3. Treat dynamic shapes as documented cases, not exclusions

- Tool/function valves: explain that the body is the selected resource's valve-values object; show the existing matching `valves spec` or `valves user spec` command. Label examples illustrative.
- `api` and provider `request`: describe endpoint-owned JSON, input sources, and a method/path-specific illustration. Do not invent a universal body or enumerate all possible remote endpoints.
- Open settings, provider options, and extension maps: document known enclosing fields and any locally enforced constraints; identify which nested keys depend on configuration/resource/provider. Do not use "dynamic" to omit an otherwise fixed form.
- Manifest inputs: describe the local envelope and every currently supported kind's `spec` fields, identity/defaulting, optional access grants, and create-versus-update conditions. Reuse existing kind contracts; do not add a manifest schema command.

Runtime help performs no schema requests. If an existing schema-inspection command is shown, it is a reference the operator may separately execute, not a hidden preflight.

### 4. Establish a closed coverage inventory from consuming paths

Use one table-driven test inventory of actual JSON-consuming actions, with invocation templates, input sources, shape classification, example(s), and source locators. Reconcile it against both the `bodyFlags` allowlist and actual loader/parser call sites. Explicitly account for aliases; do not count raw command keys as a verified command total. An allowed-but-unused body flag is not permission to start consuming input or to document a fictional contract.

The baseline discovery families are:

- `api`.
- `auth`: `login`, `profile update`, `password update`, `admin-config set`, `ldap-server-config set`, `ldap-config set`, `oauth-config set`.
- `tasks config set`.
- `models`: `create`, `import`, `sync`, `update`, `toggle`, `access-update`, `delete`; ID-based actions inject the positional ID into an optional object and reject conflicting body IDs.
- `config`: `import`, `connections set`, `tool-servers set/verify`, `terminal-servers set/verify/policy/lifecycle/refresh`, `terminal-servers access-grants diff/set`, `code-execution set`, `models set`, `suggestions`, `banners set`, `oauth-client register`.
- `files`: `update-content`, plus `upload --metadata`.
- `knowledge`: `create`, `reindex`, `metadata-reindex`, `get`, `export`, `delete`, `reset`, `update`, `access-update`, `files update/remove/move/batch-add`, `dirs create/update/delete`, `sync diff/cleanup`. The get/export/delete/reset and directory-delete paths forward optional bodies; explain whether the identified upstream route defines a body rather than inventing fields.
- `manifests`: `diff`, `apply`, `sync`, covering JSON documents from each supported file/directory source and every supported kind: Channel, Function, FunctionValve, Group, Knowledge, Model, Prompt, Skill, TerminalServerConnection, Tool, ToolValve. Each file contains one envelope; files/directories may be combined, and comma-separated paths are supported. Unlike request-body files, manifest `--file -` is a literal path, not stdin.
- `channels`: `create/update`, `members add/remove/active`, `messages post/update`, `reactions add/remove`.
- `webhooks`: `channels create/ensure/update`, `events create/update`, including explicit-JSON precedence over convenience fields. `channels ensure` consumes the JSON only when creating a missing webhook; `--name` still selects the existing-name lookup.
- `groups`: `create/update`, `users add/remove`.
- `users`: `create/update`, `settings update`, `ui-settings patch/bulk-patch`; the bulk `--users-file` is newline-delimited IDs, not JSON; current-user settings use a nested `ui` field, whereas UI patches take the direct UI object.
- `functions`: `create/update/sync`, `valves update`, `valves user update`.
- `skills`: `create/update/access-update`; distinguish JSON bodies from the separate Markdown `--manifest` format.
- `tools`: `create/update/access-update/load-url`, `valves update`, `valves user update`.
- `chats`: `import/compact`, `tags set/delete`.
- `automations`: `create/update`.
- `scim`: `users create/replace/patch`, `groups create/replace/patch`.
- `providers`: `openai` and `ollama`, each with `config update/set`, `verify`, and `request`.

These are action labels, not literal command recipes: implementation must use each dispatcher's actual placement of IDs/subactions. The inventory is complete only after every actual consuming path is classified and rendered help is tested. No family is deferred merely because it is less common.

### 5. Bound tests to discovery, content, and execution preservation

Use existing Go test facilities. Every inventory entry gets a help success/content check for both help flags, with and without selectors where applicable. Check actual usage grammar, nonempty input reference, field/shape-specific content, and parseable example JSON. For dynamic cases, assert the variable-contract explanation and matching existing inspection recipe rather than fixed invented fields.

Cover representative routing edges: deep nesting, multiple positional selectors, aliases, optional bodies, conflicting/missing execution prerequisites with help, unreadable/missing input paths, stdin not consumed, `--out` not written, and help-like values. Prove help does not invoke the HTTP client or target/configuration resolution. Preserve existing tests for unsupported flags, confirmation, normalization, pass-through bodies, and output errors. Do not require exhaustive combinations of all options or exact snapshots of every whitespace character.

Use source-backed assertions for meaningful required fields, defaults and nested forms; parseable JSON alone is not schema evidence. Execute representative documentation examples through the real CLI against existing local HTTP fixtures for fixed, nested, envelope, metadata and manifest cases. Keep these fixtures local and bounded; no running Open WebUI installation, live mutations, or new schema-validation framework is needed.

Run focused help tests, `go test ./... -count=1`, `go vet ./...`, build a task-local binary, and smoke its top-level, family, and representative nested help without server configuration. Broad hardware/deployment/benchmark or race/coverage campaigns are not acceptance requirements for this help-only increment.

## Anti-overengineering Guardrails and Stop Rule

Produce the smallest complete solution that satisfies the accepted task. Additional sophistication belongs in this implementation only when the user explicitly requested it as part of this task. Lead with the observable help behavior.

Expected production additions are help text and the minimum private routing/formatting support. Reuse existing serializers, body loading, parsers, admission checks, command dispatch, and request execution. Do not add dependencies, services, configuration knobs, public schema APIs/flags, runtime fetch/cache/version machinery, code generators, generalized documentation tooling, or parser/framework migrations. The test inventory is for coverage, not a new runtime resource model.

Stop when every actual JSON-input path has accurate offline action help, the bounded checks pass, and review finds no missing accepted outcome or behavior regression. Review blockers must identify an uncovered input, inaccurate contract, unreachable/mutating help, contradiction, or introduced regression. Additional architecture, generalized hardening, unrelated API repairs, and live acceptance are outside this change.

## Risks / Trade-offs

- **Static references can drift from upstream** → Keep concise source locators, review form changes with the related help entry, and retain source-backed regression expectations. Do not promise every server version or add runtime negotiation.
- **A broad allowlist can hide unusual inputs or unused flags** → Reconcile actual consumers, auxiliary options and local documents with the test inventory; report unrelated defects separately.
- **Naive help detection can intercept values or resolve the wrong nested action** → Exercise actual parser boundaries and positional grammar through the CLI.
- **Some schemas are genuinely runtime-defined** → State known structure and variable ownership and reuse existing inspection commands without blocking offline help.
- **Large inputs can produce long help** → Share authored fragments, use clear nested sections and compact examples; do not sacrifice completeness or build a browsing subsystem.

## Migration Plan

No data, configuration, or deployment migration is required. Implement within the existing CLI, verify the help and preserved execution paths, then deliver through the normal build process. This planning change leaves runtime code and canonical specs unchanged; implementation and later lifecycle closure require separate authorization.
