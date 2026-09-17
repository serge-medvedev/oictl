## Why

Recent Open WebUI QA found three correctness gaps that make otherwise successful workflows confusing or non-idempotent: applied model manifests can immediately diff due to server-added nullable metadata fields, channel webhook ensure can produce empty stdout when structured output is requested with `--out`, and missing model deletes can surface Open WebUI's misleading 401 response instead of a clear not-found result.

These are focused polish fixes for existing model manifest, webhook ensure, and model delete behavior before broader usage relies on them in CI.

## What Changes

- Normalize absent and `null` optional model `meta` fields during manifest comparison so Open WebUI responses such as `meta.profile_image_url: null` do not cause post-apply drift when the manifest omits that optional field.
- Preserve webhook URL secrecy for `oictl webhooks channels ensure --out`, while returning non-secret status and metadata on stdout when `--output json` or table output is requested.
- Surface missing model deletion as clear not-found text/status when Open WebUI responds with its misleading `401` response whose detail is Open WebUI's not-found message.
- Add focused tests for idempotent model manifest diff, webhook ensure structured/table output with `--out`, and model delete not-found handling where feasible.
- Ensure any touched manifest docs or examples continue to use `apiVersion: oictl.openwebui/v1`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `declarative-resource-management`: Model manifest diff SHALL treat absent and null optional `meta` fields as equivalent for idempotent apply/diff behavior.
- `webhooks-control`: Channel webhook ensure with `--out` SHALL still emit non-secret structured/table metadata when requested.
- `models-control`: Model delete SHALL report missing models as not found instead of generic authorization failure when Open WebUI returns the known misleading not-found response.

## Impact

- Affected code likely includes model manifest normalization/diff planning, channel webhook ensure output rendering, model delete error handling, and related CLI tests.
- No API contract or command syntax changes are expected.
- Secret-bearing webhook URLs and tokens remain excluded from stdout except for explicit URL reveal modes.
- Documentation impact is limited to correcting any touched manifest examples that do not use `oictl.openwebui/v1`.

## Grill-Me Conclusions

- No blocking clarification is needed before implementation.
- Model metadata normalization should start with the observed optional nullable field `meta.profile_image_url`; broader null removal is out of scope unless another optional nullable field is identified during implementation.
- `webhooks channels ensure --out` must keep default URL secrecy. Explicit JSON/table output should contain sanitized metadata/status only, not a redacted or full URL field.
- The model delete special case is scoped to the Open WebUI source-confirmed missing-model shape: `/api/v1/models/model/delete` returns `401` with not-found detail when the model does not exist. Authorization details that are not the not-found detail remain authorization failures.
