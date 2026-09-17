## Context

Manifest access grants are represented with Open WebUI's API tuple: `principal_type`, `principal_id`, and `permission`. This is precise for API calls but inconvenient for declarative manifests because user and group IDs are often opaque, environment-specific, or hard to review.

Current manifest loading validates and normalizes access grants before contacting Open WebUI. Planning then compares desired grant tuples directly against remote grant tuples, and apply/sync sends the desired grant list to the resource-specific access update API.

Open WebUI exposes user and group inventory through existing APIs. Users can be queried through `/api/v1/users/`, and native groups can be listed through `/api/v1/groups/`. Grant update endpoints still require `principal_id`, so resolution belongs in `oictl` before planning and mutation.

## Goals / Non-Goals

**Goals:**
- Let manifests identify user principals by existing ID, email, name, or explicit ref.
- Let manifests identify group principals by existing ID, name, or explicit ref.
- Resolve all non-public grant principals to `principal_id` before diff, dry-run apply, apply, and sync compare or mutate resources.
- Preserve existing manifests that already use `principal_id`, including `principal_id: "*"` for public access.
- Fail before mutation when a referenced principal cannot be resolved uniquely.
- Avoid repeated user/group list requests within a single manifest workflow.

**Non-Goals:**
- Creating missing users or groups from access grant declarations.
- Changing Open WebUI access grant API payloads.
- Resolving `Function` or `Group` manifest `access_grants`, because those manifest kinds do not support grants.
- Supporting YAML manifests as part of this change.

## Decisions

1. Add manifest-only reference fields to access grants.

   Desired grants will continue to accept `principal_id`. They will also accept optional manifest-only reference fields such as `principal_ref`, `principal_email`, and `principal_name`. Validation will require each non-public grant to provide exactly one principal selector after trimming whitespace. `principal_email` is valid only with `principal_type: user`; `principal_name` is valid for users and groups; `principal_ref` is a convenience selector that is resolved within the declared `principal_type`.

   Alternative considered: overload `principal_id` to accept names and emails. That would be smaller but makes manifests ambiguous and makes it harder to tell whether an author meant an ID or a human-readable selector. Keeping `principal_id` as the raw API value preserves existing semantics.

2. Resolve into the existing `accessGrant` shape before planning.

   `loadManifestDocuments` will continue to parse and validate envelope structure locally. `runManifestWorkflow` will call a resolver after loading documents and before `buildManifestPlan`. The resolver will replace each reference-based desired grant with a normalized grant whose `PrincipalID` is the Open WebUI ID and whose manifest-only reference fields are cleared. From that point forward, `diffGrants`, `grantsEqual`, plan JSON, and resource handlers operate on the same ID-based grant tuples as today.

   Alternative considered: resolve lazily inside each resource handler when updating grants. That would miss `diff` and dry-run correctness and could make planning output disagree with actual apply behavior.

3. Resolve users and groups with per-workflow caches.

   A `principalResolver` will keep user and group candidate caches for the duration of one manifest workflow. User lookups can query `/api/v1/users/?query=<selector>` and match exact `id`, `email`, or `name` depending on the selector. Group lookups can use `/api/v1/groups/` and match exact `id` or `name`. The resolver should fetch only the principal type needed by manifests and reuse results across all documents.

   Alternative considered: fetch all users and all groups up front. That is simple but adds avoidable API calls to manifests that only use raw IDs or public grants.

4. Treat ambiguity and type mismatch as validation failures.

   A selector that matches no principal, matches multiple principals for the selected field, or uses a field incompatible with `principal_type` will return an error before `buildManifestPlan` performs remote resource lookup or any mutation occurs. Explicit `principal_ref` prefixes such as `user:` or `group:` must agree with `principal_type` when present.

   Alternative considered: choose the first match. That would make manifests non-deterministic and could grant access to the wrong principal.

5. Keep public grants out of resolver lookup.

   `principal_type: user` with `principal_id: "*"` remains the representation for public access. It must not be resolved through the user directory.

## Risks / Trade-offs

- User lookup API may require admin privileges -> surface Open WebUI's authorization error and fail before mutation; existing ID-based grants remain usable without directory lookup.
- User names may not be unique -> exact duplicate name matches fail as ambiguous; email or ID remains the recommended stable selector for users.
- Group names may not be globally unique in future Open WebUI versions -> exact duplicate group name matches fail as ambiguous; ID or explicit ref remains available.
- Extra API calls can slow large manifest diffs -> resolver caches users/groups and only resolves when a reference field is present.
- Plan output will show resolved IDs rather than original reference text -> this keeps plan JSON aligned with actual Open WebUI payloads, while docs will explain that reference fields are authoring-only.

## Migration Plan

No data migration is required. Existing manifests using `principal_id` continue to work. Users can incrementally replace raw principal IDs with `principal_email`, `principal_name`, or `principal_ref` where stable identifiers are preferable.
