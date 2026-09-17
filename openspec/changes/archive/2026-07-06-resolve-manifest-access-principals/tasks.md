## 1. Manifest Model And Validation

- [x] 1.1 Extend the manifest access grant struct with authoring-only principal selector fields such as `principal_ref`, `principal_email`, and `principal_name`.
- [x] 1.2 Update access grant validation to require exactly one principal selector for non-public grants while preserving `principal_type: user` with `principal_id: "*"` for public access.
- [x] 1.3 Reject selector/type mismatches, including user-only email selectors on group grants and explicit refs whose type prefix conflicts with `principal_type`.
- [x] 1.4 Normalize duplicate desired grants after selector validation without treating authoring-only fields as part of the final grant identity.

## 2. Principal Resolution

- [x] 2.1 Add a manifest principal resolver that runs after manifest loading and before plan construction.
- [x] 2.2 Resolve user selectors to Open WebUI user IDs using existing user listing/query APIs and exact matching on ID, email, or name.
- [x] 2.3 Resolve group selectors to Open WebUI group IDs using existing group listing APIs and exact matching on ID or name.
- [x] 2.4 Cache user and group lookup data for the duration of one manifest workflow and skip resolver API calls when all grants already use raw IDs or public access.
- [x] 2.5 Return clear pre-mutation errors for missing, ambiguous, or type-incompatible principal references.

## 3. Workflow Integration

- [x] 3.1 Invoke principal resolution from `diff`, dry-run `apply`, `apply`, and `sync` before `buildManifestPlan`.
- [x] 3.2 Ensure plan comparison and JSON/table output use resolved `principal_id` values for reference-based grants.
- [x] 3.3 Ensure create and replace-grants execution sends only Open WebUI grant fields with resolved principal IDs to grant update endpoints.
- [x] 3.4 Preserve unsupported-kind validation so `Function` and `Group` manifest `access_grants` fail before any resolver lookup or mutation.

## 4. Tests And Documentation

- [x] 4.1 Add unit tests for valid selector forms, public grants, multiple selector rejection, user email/name resolution, group name/ref resolution, and ambiguity failures.
- [x] 4.2 Add workflow tests proving `diff`, dry-run `apply`, `apply`, and `sync` resolve grants before planning or mutation and send resolved IDs to Open WebUI.
- [x] 4.3 Add regression tests proving existing `principal_id` manifests do not require user/group resolver API calls.
- [x] 4.4 Update `docs/manifests.md` with the new authoring fields, resolution behavior, and examples for user email and group name grants.
- [x] 4.5 Run the relevant Go test suite for manifest workflows.
