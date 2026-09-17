## 1. Model Manifest Idempotency

- [x] 1.1 Add model manifest comparison normalization for allowlisted optional `meta` fields where absent and `null` are equivalent, starting with `profile_image_url`.
- [x] 1.2 Add a regression test showing `oictl manifests apply` followed by `oictl manifests diff` is unchanged when Open WebUI returns `meta.profile_image_url: null` and the manifest omits it.
- [x] 1.3 Add or preserve test coverage showing non-null desired optional `meta` values still produce a `meta` diff when remote state is absent or `null`.

## 2. Webhook Ensure Output

- [x] 2.1 Update `webhooks channels ensure --out` so requested `--output json` renders non-secret ensured webhook metadata/status to stdout after the URL file write succeeds.
- [x] 2.2 Update `webhooks channels ensure --out` so requested `--output table` renders non-secret ensured webhook metadata/status to stdout after the URL file write succeeds.
- [x] 2.3 Add regression tests for JSON and table output with `--out`, asserting the URL file contains the full URL and stdout does not contain the full token or full URL.

## 3. Model Delete Not Found Handling

- [x] 3.1 Detect HTTP `401` with not-found detail from `/api/v1/models/model/delete` for `oictl models delete <model-id>` and report a clear not-found error for the requested model ID.
- [x] 3.2 Preserve generic authorization handling for real 401/403 model delete responses that do not match the missing-model response.
- [x] 3.3 Add regression tests for HTTP `401` not-found model delete handling and real HTTP `401` authorization failure handling.

## 4. Documentation And Verification

- [x] 4.1 Check touched docs/examples for manifest `apiVersion` values and keep them on `oictl.openwebui/v1`.
- [x] 4.2 Run targeted Go tests covering manifest, webhook, and model command behavior.
- [x] 4.3 Run the broader relevant test suite if targeted tests pass.
