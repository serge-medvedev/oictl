## Context

See `proposal.md` for the requested outcome. Baseline: `cad8b3e91162e2343e5d794a09d058b17c64387f`. This follows the JSON-input help change but extends discovery to the whole supported command tree and removes its rendered source provenance.

There are two production emit sites for the literal diagnostic in `internal/cli/app.go`: the global-option setter in `parseGlobalFlags`, and ordinary non-boolean parsing in `parseCommandFlags`. The latter runs through `validateCommandFlags` before top-level version dispatch. The help-only map in `json_help.go` covers JSON consumers, not every leaf or intermediate group. Consequently changing only the version branch or adding one help string cannot close the reported class of failure.

The audit companion `audit.md` records the finite command inventory, actual baseline classifications and legitimate missing-value cases. It is implementation scope evidence, not a new runtime command model. Arbitrary endpoint paths, user identifiers and unknown option spellings are not enumerable commands; they are represented by their owning command and parser equivalence classes.

## Goals / Non-Goals

**Goals:**
- Match informational intent with the correct local result: version for version, relevant usage for help, and a precise nonzero diagnostic for malformed options.
- Cover all supported leaves, intermediate groups and aliases, including commands with no JSON input and local profile operations.
- Keep help useful to end users while preserving existing input detail and request behavior.

**Non-Goals:**
- Replacing the CLI grammar/framework, adding new command families/help aliases/completion, changing APIs or credential policy, repairing unrelated runtime quirks, or installing/publishing an executable during this planning phase.
- Silencing every occurrence of `requires a value`: a genuinely missing value remains an error with actionable context.

## Decisions

### 1. Classify failures before changing behavior

Use the audit inventory to assign one disposition per path/class:
- Recognized help flag: relevant leaf/group help, stdout, zero, no side effects.
- Top-level version spelling: existing version output, stdout, zero, no side effects.
- Known required value missing: nonzero diagnostic naming that option plus local usage or a usable help command.
- Unknown option or known option unsupported here: distinguish that defect before looking for a value; nonzero with relevant help.
- Already-correct informational behavior: preserve and cover as regression controls.

Do not treat `-h` as safe merely because it avoids the exact long-option error: it can enter positional parsing or execution. Do not turn malformed ordinary invocations into successful help. Keep malformed boolean diagnostics and explicit false unsupported safety options enforced. Value-bearing options still consume literal values, including dash-leading strings, according to supported grammar; new heuristic value guessing is outside scope.

### 2. Extend existing informational handling, not operation dispatch

Reuse the existing early help scanner and static JSON references. Add the smallest private help routing/content for missing leaves and groups, and a narrow top-level version branch before action flag validation. Reuse existing family summaries and private fragments when they really describe the selected path. A group summary must identify children; a leaf must identify its action, required arguments and relevant options rather than simply dumping the whole family.

Keep help-only metadata separate from execution authority. If recognizing supported option names/arity requires sharing a small existing flag definition or lookup, share that boundary; do not introduce a generalized registry owning commands, schemas, HTTP routes, dependencies, or lifecycle. For ordinary execution, determine command scope using existing grammar before classifying unsupported/unknown flags. Preserve global option placement and known input-value consumption. Give malformed global options top-level context and malformed action options the nearest unambiguous action/group context.

Precedence is syntax-aware: standalone help can bypass operation prerequisites; a token already consumed as an option value is not standalone help. Top-level version is a command spelling, not a magic token anywhere in argv. Preserve existing supported positional help forms without inventing `help <arbitrary-path>` or trailing positional aliases. Existing bare group behavior is unchanged unless the inventory demonstrates it is part of the corrected informational path; this change does not require a new implicit-help policy for incomplete ordinary operations.

The audit closes over 328 finite paths and 72 ordinary option names, with 5,734 isolated probes. It identifies 177 long-help failures and confirms all 563 supported local valued-option omissions remain genuine errors. These are baseline observations, not new runtime count constraints.

For overloaded `--manifest`, determine help-token boundaries from the selected action: tools export owns a boolean, so both following help flags request help; skills create/update owns a path, so a following `-h` is still the path value. Terminal missing skill paths stay nonzero errors with context. This narrow arity correction does not authorize broader manifest parsing changes. Preserve the existing version result for top-level version spellings followed by help. Do not introduce nested version support, a new `--` delimiter contract, or reinterpret arbitrary dash-leading positional IDs as unknown short options. Arbitrary knowledge-sync actions and passthrough routes remain existing dynamic boundaries, not new finite commands to constrain.

### 3. Remove rendered provenance while retaining useful contracts

Stop rendering the `source` footer and move any necessary evidence to ordinary comments or test expectations. Do not retain a runtime provenance field solely for a footer that no longer exists, or add an opt-in `--show-source` replacement. Update tests that currently require the footer and README prose claiming sources are identified in help.

Review the rendered input fragments as well as the footer: remove irrelevant backend route explanations and class/module/function locators. Replace implementation-oriented terms with their user-facing meaning without deleting actual fields, type/default/omission behavior, extension availability, or valid examples. References to tool/function valve inspection remain CLI recipes. Method/path syntax remains for `api` and provider `request`, where the operator supplies it. A blanket ban on the word "API" or every URL would erase valid input examples; relevance to the operator's invocation, not a lexical blacklist, is the rule.

No unsolicited password storage, shell-history, workflow, or security advice. Keep error context limited to the option/path and usage; do not echo raw values containing credentials.

### 4. Keep acceptance finite and proportional

Carry the audited command list into one table-driven regression inventory using existing Go tests, not a new audit service or generator. Reconcile finite dispatch branches, aliases, group prefixes, existing help text, and current JSON-help inventory; don't replace the actual command inventory with only the flag allowlist. Test `--help` and `-h` for every supported path, with omitted and representative supplied selectors. Preserve successful JSON help and its examples.

Use one focused case per parser failure class plus relevant variations: all existing version spellings; global versus local missing values; unknown versus unsupported options with/without values; strict booleans; dash-leading/help-like values; local profiles and remote destructive help with no side effects. Prove no config/file/stdin/output/HTTP effects through existing injected clients and disposable paths. Do not run exploratory help against configured production targets: a broken `-h` can execute an operation.

Read actual rendered output to confirm relevant action/group context and no source/API provenance. Assert useful CLI schema recipes and passthrough path examples remain. Test source comments internally rather than requiring provenance in stdout. Run focused RED/GREEN regressions, existing full Go suite, vet, one task-local build and offline real-binary help/version/error smoke. No live Open WebUI instance, new dependency, network acquisition, benchmark/race campaign, or exhaustive arbitrary option-combination matrix is required.

## Anti-overengineering Guardrails and Stop Rule

Produce the smallest complete solution that satisfies the accepted task. Additional sophistication belongs only when explicitly requested. Every production edit must map to an inventoried discovery/diagnostic defect, necessary preservation of argument boundaries, or removal of irrelevant rendered references.

Reuse authored help, existing parser/allowlist, serializers, request paths, and tests. No CLI framework rewrite, general command/schema registry, code generation, runtime introspection/OpenAPI retrieval, new dependencies/configuration/services, public source/schema flags, completion system, retries, typo auto-correction, compatibility layer, or unrelated API changes. Keep maintainer source evidence out of end-user output.

Stop once every affected inventory row has its appropriate result, regression controls retain their contracts, help is inert and useful, provenance noise is absent, and bounded checks pass. Review may block on missing inventory coverage, inaccurate help, conflicting behavior, option-value misclassification, side effects, or introduced regression—not requests for more architecture or unrelated hardening.

## Risks / Trade-offs

- **String matching misses real command semantics** → Classify supported paths and token positions from dispatch; preserve value boundaries and test both help spellings.
- **Misleading success hides input errors** → Only recognized informational requests exit zero; real missing/unknown/unsupported input stays nonzero with context.
- **Content cleanup removes useful details** → Retain field/type/omission semantics, existing inspection recipes and operator-supplied API paths while removing research provenance.
- **Static help can drift** → Reuse the audited test inventory and ordinary maintainer comments; avoid a second schema/dispatch system.

## Migration Plan

No stored-data or configuration migration. Apply focused code/help/test changes after separate implementation authorization. Existing users keep their command names and inputs; previously broken informational forms become reliable and malformed option diagnostics become actionable. Planning leaves runtime files, canonical specs, archives, installed binaries and remote branches unchanged.
