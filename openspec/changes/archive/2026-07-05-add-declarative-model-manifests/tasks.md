## 1. Manifest Model Handler

- [x] 1.1 Add a `Model` manifest resource handler that lists, looks up, fetches, creates, updates, deletes, and replaces grants using existing Open WebUI model APIs.
- [x] 1.2 Handle model list responses with `items` pagination data when resolving current state for lookup and sync planning.
- [x] 1.3 Register `Model` in manifest handlers without changing `Knowledge`, `Prompt`, or `Tool` behavior.

## 2. Model Manifest Validation and State Mapping

- [x] 2.1 Validate `kind: Model` identity so `metadata.name` is used as `spec.id` when omitted and mismatched `metadata.name`/`spec.id` values are rejected.
- [x] 2.2 Validate required model manifest fields, including `spec.name`, before sending mutation requests.
- [x] 2.3 Default omitted `spec.meta` and `spec.params` to empty objects in create and update payloads.
- [x] 2.4 Map remote model state to managed manifest fields while excluding `user_id`, `created_at`, `updated_at`, and `write_access` from diff comparison.

## 3. Diff Apply and Sync Behavior

- [x] 3.1 Ensure model diffs report managed field drift for `id`, `base_model_id`, `name`, `meta`, `params`, and `is_active`.
- [x] 3.2 Apply model create and update actions idempotently, including active state through update payloads rather than toggle operations.
- [x] 3.3 Add `model` and `models` sync scope support and include models in `all` scope pruning.
- [x] 3.4 Delete omitted models during confirmed model sync through the existing model delete endpoint.

## 4. Access Grants

- [x] 4.1 Support model manifest `access_grants` through existing grant validation, normalization, comparison, and plan output.
- [x] 4.2 Replace model grants through the Open WebUI model access update API and report server-side grant filtering.

## 5. Documentation and Tests

- [x] 5.1 Update manifest help and `docs/manifests.md` with `kind: Model` examples and model sync scope usage.
- [x] 5.2 Add manifest tests for valid model load, identity mismatch rejection, required field validation, field diffing, create/update/delete planning, active state, and ignored server fields.
- [x] 5.3 Add model grant tests for public grants, replace-grants planning, grant application, and server-filtered grants.
- [x] 5.4 Add regression tests proving existing imperative `oictl models` commands still route to the same endpoints.
- [x] 5.5 Run the repository test suite and any OpenSpec validation commands required by the project.
