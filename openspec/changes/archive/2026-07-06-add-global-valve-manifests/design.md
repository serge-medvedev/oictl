## Context

`oictl manifests` uses a registry of `resourceHandler` implementations to validate manifest kinds, load current state, build create/update/unchanged/delete plans, and execute plans. Tool and function global valve commands already exist as imperative operations that call `/api/v1/{tools|functions}/id/{id}/valves` and `/api/v1/{tools|functions}/id/{id}/valves/update`.

Global valve values are configuration attached to an existing tool or function, not independently owned resources. User-scoped valves have different ownership and targeting semantics, so this change intentionally limits declarative support to global valve values.

## Goals / Non-Goals

**Goals:**
- Add `ToolValve` and `FunctionValve` manifest kinds whose `spec` is the desired global valve object for the owning tool or function ID in `metadata.name`.
- Reuse existing manifest diff, apply, dry-run, duplicate identity, environment rendering, and output behavior.
- Reuse Open WebUI global valve endpoints for current-state lookup and updates.
- Avoid pruning, deletion, or reset semantics for omitted valve manifests.

**Non-Goals:**
- No `ToolUserValve`, `FunctionUserValve`, or other user-scoped desired-state manifests.
- No local valve schema interpretation, default synthesis, or client-side coercion beyond manifest envelope validation.
- No creation or deletion of owning Tool or Function records through valve manifests.

## Decisions

1. Add specialized valve manifest handlers instead of extending `endpointHandler`.

`ToolValve` and `FunctionValve` need endpoint-backed lookup/update behavior, but their lifecycle differs from normal resources: they cannot create or delete an owner, and their remote state is fetched from a nested valve endpoint. A small dedicated handler can implement `resourceHandler` while keeping existing generic planning and execution code intact.

Alternative considered: add flags or hooks to `endpointHandler`. That would make the generic handler understand nested, non-owning resources and deletion exceptions, increasing complexity for a two-kind special case.

2. Represent absent or null remote valve values as a missing manifest resource for planning, but apply via the global valve update endpoint.

When `/valves` returns `null`, the handler should return `nil` from `Lookup` so `diff` reports `create`. The handler's `Create` implementation should still POST the manifest `spec` to `/valves/update`, because Open WebUI does not expose a separate valve creation operation. Existing non-null objects compare as normal `update` or `unchanged` actions.

Alternative considered: always return an empty current state and report `update`. Reporting `create` better communicates that no current global valve values exist while still preserving the actual server operation.

3. Keep valve manifests out of sync pruning scopes.

`sync --scope all` should continue to prune only independently enumerable resource kinds. Since omitted valve manifests do not mean desired deletion or reset, `ToolValve` and `FunctionValve` should not be returned by `syncScopeKinds`, and no valve-specific sync scope should be added.

Alternative considered: add `tool-valves` and `function-valves` scopes that reset omitted valves. This is ambiguous and potentially destructive, especially without a server-level delete/reset contract.

4. Reject top-level `access_grants` through existing capability checks.

Valve handlers should return `SupportsAccessGrants() == false`. Global valves inherit the authorization of their owning resource and existing Open WebUI endpoints; valve manifests manage only the valve object.

Alternative considered: allow grants to pass through to the owning Tool or Function. That would conflate separate resources and create surprising side effects from a valve-only manifest.

5. Treat `metadata.name` as the endpoint identifier, not a display-name lookup.

Existing imperative valve commands accept `<tool-id>` and `<function-id>` and call ID-based endpoints directly. The manifest handler should do the same, sending `metadata.name` unchanged in `/api/v1/{tools|functions}/id/{id}/valves` paths. Alternative considered: resolve display names by listing owner resources first. That adds ambiguity, duplicate-name behavior, and extra requests; it can be added later if operators need friendlier authoring.

## Risks / Trade-offs

- Remote `null` semantics may differ across Open WebUI versions -> Treat only JSON `null` as missing and let non-object responses fail clearly during decode.
- Operators may expect `metadata.name` to accept display names like some resource manifests do -> Document that valve manifests use owner IDs in this slice.
- Valve schemas are server-defined and may contain defaults not present in manifests -> Do not synthesize defaults locally; compare and apply concrete returned values only.
- Applying a `create` plan through an update endpoint may be surprising in logs -> Document that `ToolValve` and `FunctionValve` are non-owning configuration resources and use global valve update endpoints for both create and update plan actions.
- Sync users may expect `--scope all` to include every supported manifest kind -> Document that valve manifests are apply/diff resources and are intentionally excluded from destructive pruning.

## Migration Plan

No data migration is required. Add handlers, validation tests, workflow tests, and documentation. Existing imperative valve commands and existing manifests keep their behavior.

Rollback is removal of the two manifest handlers and related documentation/tests; no persisted local state is introduced.

## Open Questions

- None for this slice; per-user valve desired-state ownership and selection semantics remain explicitly deferred.
