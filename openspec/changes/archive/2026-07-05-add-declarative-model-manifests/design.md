## Context

`oictl manifests` currently registers resource handlers for `Knowledge`, `Prompt`, and `Tool` and drives all declarative workflows through the shared manifest loader, planner, executor, plan output, and sync scope logic. `oictl models` already exposes Open WebUI model endpoints imperatively, including list, get, create, update, toggle, access-update, delete, import, export, and sync.

Open WebUI model records use `id` as the stable API identifier and include `base_model_id`, `name`, `meta`, `params`, `access_grants`, `is_active`, owner and timestamp fields. Model create/update/access/delete endpoints are structurally different from the generic manifest handler paths because model get uses a query parameter and update/delete/access operations post an `id` in the JSON body.

## Goals / Non-Goals

**Goals:**
- Add `kind: Model` support to existing JSON manifests and `diff`, `apply`, and `sync` workflows.
- Compare and reconcile managed model fields: `id`, `base_model_id`, `name`, `meta`, `params`, `is_active`, and manifest-declared `access_grants`.
- Support model sync scopes without changing non-model manifest behavior.
- Preserve all existing imperative `oictl models` commands and request routing.

**Non-Goals:**
- Add YAML manifest support.
- Replace `oictl models import`, `export`, or `sync` with manifest commands.
- Add new Open WebUI API endpoints or dependencies.
- Manage server-generated fields such as `user_id`, `created_at`, `updated_at`, or `write_access` from manifests.

## Decisions

1. Implement `Model` as another manifest resource handler, not as a separate command family.
   - Rationale: Keeps diff/apply/sync, dry-run, JSON output, duplicate identity checks, and destructive sync confirmation consistent with existing resources.
   - Alternative considered: Add `oictl models apply`. Rejected because it would duplicate manifest planning semantics and fragment declarative workflows.

2. Use a model-specific handler instead of forcing models into the generic endpoint handler.
   - Rationale: Model endpoints use query parameters and JSON body IDs rather than resource IDs embedded in URL paths, and model list responses are paginated under `items`.
   - Alternative considered: Generalize `endpointHandler` with more knobs. Rejected for this change because it would complicate existing prompt, knowledge, and tool behavior without a current need.

3. Treat `metadata.name` as the manifest identity and require it to match `spec.id` when `spec.id` is provided.
   - Rationale: Manifest identity is already `kind/name`; Open WebUI models are keyed by `id`; requiring consistency prevents accidental creation or update of the wrong model.
   - Alternative considered: Allow `metadata.name` to be an alias for `spec.name`. Rejected because model display names are mutable and not guaranteed unique.

4. Compare only managed model fields from `spec` and ignore server-generated fields.
   - Rationale: Owners, timestamps, and read/write decoration are server-owned and would create constant drift if included in desired state.
   - Alternative considered: Compare the full model response. Rejected because Open WebUI responses include volatile or authorization-derived fields.

5. Reconcile active state through the model create/update payload instead of invoking the imperative toggle command.
   - Rationale: Declarative apply must set desired state idempotently. Toggle is state-dependent and not safe for reconciliation.
   - Alternative considered: Use `/api/v1/models/model/toggle` when `is_active` differs. Rejected because retries or stale reads could invert the desired state.

## Risks / Trade-offs

- Model list pagination could hide existing remote models if only one page is fetched -> The model handler must account for the API's paginated `items` response when listing for lookup and sync.
- Open WebUI may filter requested public or group grants by sharing policy -> Apply must surface a server-side grant mismatch the same way existing manifest grant reconciliation does.
- Some model params or metadata may be read-restricted for read-only users -> Authorization failures or stripped fields should be surfaced as Open WebUI responses; manifest reconciliation requires sufficient write/admin access.
- Model `meta.profile_image_url` may be normalized by Open WebUI validation -> Diff tests should account for server-normalized metadata and report real remote drift.

## Migration Plan

No data migration is required. Existing `oictl models` commands continue to work. Users can opt in by adding `kind: Model` JSON manifests and running existing manifest workflows.

Rollback is limited to removing the model manifest handler and docs; existing Open WebUI model records are normal model records created through existing APIs.

## Open Questions

- None.
