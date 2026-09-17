## Context

`oictl manifests` currently uses a small handler registry in `internal/cli/manifests.go` for `Knowledge`, `Prompt`, and `Tool`. The generic endpoint handler assumes REST shapes that are compatible with those resources, but Open WebUI uses different lifecycle shapes for models, skills, functions, groups, and channels. Existing specs already define declarative workflows, access grant modeling, and imperative model commands; this change extends those contracts without replacing the manifest envelope or command family.

Open WebUI exposes the strongest fit for `Model`: list/get/create/update/delete plus `access/update`. `Skill` is similar and has direct access update. `Channel` has list/get/create/update/delete and mutable `access_grants` on create/update, but no dedicated access-update route. `Function` and `Group` expose lifecycle APIs but do not expose Open WebUI access-grant reconciliation for the resource itself.

## Goals / Non-Goals

**Goals:**

- Register manifest handlers for `Model`, `Skill`, `Function`, `Group`, and `Channel`, prioritizing complete `Model` behavior.
- Preserve `diff`, non-pruning `apply`, pruning `sync`, dry-run, and JSON plan semantics for all newly supported kinds.
- Reconcile grants for `Model`, `Skill`, and `Channel`, and reject `access_grants` for `Function` and `Group` before sending mutation requests.
- Extend sync scope parsing and help/docs to include singular/plural scopes for the new kinds and include them in `all`.
- Cover resource-specific endpoint quirks with tests rather than broad compatibility shims.

**Non-Goals:**

- YAML manifest support.
- New Open WebUI API endpoints or server-side schema changes.
- Declarative reconciliation of channel messages, channel webhooks, function valves, group access previews, model base-model discovery, or global delete-all operations.
- Backward compatibility for misspelled or previously unsupported manifest kinds.

## Decisions

1. Add resource-specific manifest handlers where endpoint shapes differ from the existing generic handler.

   Rationale: `Model` uses query/body `id` payloads, `Channel` uses path IDs without `/id/`, and `Function` has code-loading side effects that should be handled deliberately. Extending the generic handler with many optional knobs would obscure these differences.

   Alternative considered: add more fields to `endpointHandler` for query IDs, body IDs, and access modes. This was rejected because it would make the common path harder to reason about and still not cover every resource cleanly.

2. Use `metadata.name` as the stable manifest identity and map it to the natural Open WebUI identifier for each kind.

   Rationale: existing manifests identify resources as `kind/name`. For model and function records, `metadata.name` maps to the Open WebUI `id`; for skills, groups, and channels it is matched against `id` or human-readable name fields where available.

   Alternative considered: require `metadata.id` or resource-specific identity fields. This was rejected to preserve the existing manifest envelope and duplicate detection model.

3. Treat grant reconciliation as kind-specific.

   Rationale: `Model` and `Skill` expose dedicated access update endpoints, while `Channel` stores `access_grants` through create/update payloads. `Function` and `Group` do not have compatible resource grant APIs. Failing unsupported grant declarations during validation avoids partial mutation and false confidence.

   Alternative considered: silently ignore `access_grants` for unsupported kinds. This was rejected because manifests are desired state and ignored grants would create unsafe drift.

4. Keep `sync` destructive behavior uniform across all supported kinds.

   Rationale: users already understand that `sync` prunes omitted resources after explicit confirmation. New scopes should produce delete actions consistently and require `--yes` for destructive execution.

   Alternative considered: ship new kinds as `diff`/`apply` only. This was rejected because the requested change explicitly includes `sync` and the upstream APIs expose deletes for these resource types.

## Risks / Trade-offs

- Upstream endpoint schemas differ by Open WebUI version -> Keep handlers narrow, validate via tests against expected paths/payloads, and surface server errors without hiding response details.
- Channel manifests can affect live collaboration spaces -> Require the existing destructive sync confirmation and avoid message, webhook, and membership reconciliation in this change.
- Function create/update executes Open WebUI function loading and validation -> Do not manage valves or global toggles through manifests, and rely on Open WebUI admin authorization and validation responses.
- Generic spec diffing may compare server-generated fields returned by list/get -> Normalize or ignore known read-only fields in handlers so unchanged resources do not churn plans.
- Public sharing policies can filter requested grants -> Preserve existing filtered-grants detection and report server-side differences after mutation.

## Migration Plan

No data migration is required. Existing manifests continue to work unchanged. Rollback is limited to removing the new handler registrations, scope values, tests, and docs if endpoint compatibility proves insufficient before release.
