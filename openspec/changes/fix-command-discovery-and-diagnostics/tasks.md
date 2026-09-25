## 1. Lock the bounded regression inventory

- [x] 1.1 Reconcile `audit.md` against the current command dispatch, aliases, intermediate groups, global options and command allowlist. Retain one finite command/help test inventory, with affected rows and already-correct controls distinguished; account for both production missing-value emitters.
- [x] 1.2 Add focused failing tests for every inventoried help/version failure class and both help spellings, with omitted and representative supplied selectors. Include local-profile operations, nested groups and existing JSON-help controls.
- [x] 1.3 Add failing diagnostic tests for global/local genuine missing values, unknown options with/without a value, known-but-unsupported options, unsupported explicit-false safety flags, and literal help/version-like option values. Retain existing valid boolean and dash-leading value behavior. Cover the overloaded manifest boundary explicitly: tool-export boolean followed by either help flag is help; skill-input `-h` remains a path value and terminal missing paths remain errors. Preserve version output with appended help and existing dynamic passthrough/sync boundaries.

## 2. Correct informational routing and option diagnostics

- [x] 2.1 Handle existing top-level version spellings before action flag validation; verify identical stdout and zero exit without resolving target/profile or touching input/output paths.
- [x] 2.2 Extend the existing help-only routing/static content to every supported command and group in the audit. Provide relevant local usage, required arguments, options and child actions; preserve aliases, existing positional help forms, selectors and JSON-input references without inventing new public syntax.
- [x] 2.3 Correct option classification so genuinely absent required values, unknown options and options unsupported by the selected action each produce their own nonzero diagnostic with appropriate usage/help discovery. Preserve value boundaries and strict unsupported boolean checks, and do not echo secret values.
- [x] 2.4 Verify informational handling precedes operation admission and required-input/confirmation checks. Use existing test seams and disposable paths to prove no profile/config resolution, input-file/directory/stdin reads, output writes, local mutations or HTTP requests for recognized help/version.

## 3. Keep rendered help end-user-focused

- [x] 3.1 Remove source footers and unused runtime-only provenance data; retain necessary accuracy evidence in maintainer comments/tests. Remove irrelevant upstream links, revision hashes, module/class/function locators and backend-route explanations from all rendered help fragments.
- [x] 3.2 Preserve meaningful field names, input constraints/defaults/omission behavior, examples, CLI schema-inspection recipes and operator-supplied API method/path syntax. Keep input-source wording neutral and free of unsolicited operational/security guidance.
- [x] 3.3 Update existing footer-dependent tests and minimal README discovery/provenance claims. Verify rendered help content rather than substituting a blanket keyword/URL blacklist for relevance review.

## 4. Verify and review the completed scope

- [x] 4.1 Run the finite command/help inventory for `--help` and `-h`, all existing top-level version spellings and bounded diagnostic/value-boundary cases. Account programmatically for every audited row and inspect representative leaf/group output for useful context, not just zero exit.
- [x] 4.2 Run focused RED/GREEN regressions, the existing full Go suite, `go vet ./...`, a task-local build and offline real-binary help/version/error smoke checks. Retain existing execution/request/confirmation and JSON-help regression coverage; no live server or credentials are required.
- [x] 4.3 Validate the named OpenSpec change and all project specifications strictly, check diff hygiene, and independently review the exact implementation candidate for missing outcomes, misleading help, side effects, parser/value-boundary regressions and contradictory contracts.
- [x] 4.4 Enforce the stop rule: every production edit must support an audited informational/diagnostic correction, preserved behavior or removal of irrelevant rendered references. Reuse existing parser, allowlist, static help, serializers, admission and execution ownership; add no framework rewrite, generalized command/schema registry, generator, runtime OpenAPI fetching, dependency/service/configuration, public source/schema/completion surface, auto-correction/retry, compatibility layer or unrelated API repair. Stop when the complete bounded acceptance set passes; extra sophistication requires an explicit request.
