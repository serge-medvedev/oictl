## Context

The QA issues are isolated to existing CLI behavior:

- Model manifest comparison currently uses `modelManagedSpec` and treats `meta` as a managed field. Open WebUI may return optional keys such as `profile_image_url` with a JSON `null` value even when a manifest omits the key, causing a false `meta` diff after apply.
- `ensureChannelWebhook` writes the generated secret URL to `--out` and returns before writing normal command output. That preserves secrecy, but leaves stdout empty even when `--output json` or `--output table` was requested.
- Imperative `models delete` goes through the generic model mutation path and generic HTTP status surfacing. Open WebUI can return a 401-shaped response for a missing model delete, which reads as an authorization failure to users.

## Goals / Non-Goals

**Goals:**

- Make model manifest apply/diff idempotent when Open WebUI only adds nullable optional metadata fields.
- Keep channel webhook URL secrecy intact while making `ensure --out --output json|table` useful for automation.
- Reclassify only the known misleading model-delete not-found response into clear not-found text/status.
- Add regression tests at the CLI behavior level.
- Keep touched manifest docs/examples on `apiVersion: oictl.openwebui/v1`.

**Non-Goals:**

- Do not add new webhook URL reveal modes or print secret URLs in JSON/table output.
- Do not broadly ignore arbitrary `meta` differences for models.
- Do not change model create/update/get authorization handling.
- Do not introduce local state, caching, or compatibility aliases for old manifest API versions.

## Decisions

### Normalize Only Optional Null Model Metadata During Comparison

Implement normalization at the model manifest comparison boundary, likely near `modelManagedSpec` or the model-specific diff preparation path, so both desired and remote model specs are compared with equivalent optional absent/null metadata fields collapsed.

The initial normalization target is `meta.profile_image_url`, because this is the observed Open WebUI response field. The implementation should be easy to extend with additional optional nullable `meta` keys if future QA identifies them.

Alternative considered: remove all null-valued `meta` keys before comparison. This is broader and could hide meaningful user-managed nulls, so the focused allowlist is safer.

### Continue Separating Secret URL Output From Command Metadata

For `webhooks channels ensure --out`, write the full incoming webhook URL only to the requested file. After the write succeeds, render non-secret ensured webhook metadata/status to stdout when `--output json` or `--output table` is requested, using the existing redaction/structured output path where possible.

Default behavior may remain quiet if that is the current contract, but explicit structured/table output must not be empty.

Alternative considered: include a redacted URL placeholder in JSON output. That is unnecessary for the QA issue and risks confusing automation; metadata/status without secret URL is sufficient.

### Special-Case Missing Model Delete Without Weakening Auth Errors

Handle `models delete` separately enough to detect the Open WebUI source-confirmed missing-model response from `/api/v1/models/model/delete`: HTTP `401` with not-found detail. Print a clear not-found message for the requested model ID in that case. Other 401/403 responses, including Open WebUI's unauthorized detail, must still be reported as authorization failures with the server detail.

Alternative considered: globally reclassify model 401 responses containing not-found text. That would be too broad and could mislabel real auth failures on get/update/toggle/access-update.

## Risks / Trade-offs

- Over-normalizing model metadata could hide real drift. Mitigation: use a small optional-null allowlist and test that non-null desired values still diff.
- Webhook ensure output could accidentally include token-bearing fields returned by Open WebUI. Mitigation: route through existing webhook redaction/sanitization and assert stdout does not contain the token or full URL.
- The Open WebUI missing-delete response shape may vary by version. Mitigation: detect the source-confirmed `401` plus not-found detail shape and keep fallback generic status handling for unknown responses.
- Tests may need fake server responses that match current command wiring rather than real Open WebUI behavior. Mitigation: cover user-visible stdout/stderr/status and request paths at the CLI test layer.

## Migration Plan

No migration is required. Existing commands and manifest schemas remain unchanged.

Rollback is limited to reverting the implementation and tests for this change.

## Open Questions

None. The grill-me pass resolved the main ambiguity: model metadata normalization starts with the observed `profile_image_url` optional-null field rather than broad null stripping.

## Grill-Me Conclusions

- No blocking user decision is needed before implementation.
- Normalize only allowlisted optional-null model metadata keys, beginning with `profile_image_url`, so meaningful null/non-null metadata differences are not hidden.
- For webhook ensure with `--out`, return sanitized metadata/status only when structured/table output is explicitly requested; do not include full or redacted URL fields in stdout.
- For missing model delete, special-case only the `/api/v1/models/model/delete` response with HTTP `401` and not-found detail. Do not reclassify unrelated model command authorization failures.
