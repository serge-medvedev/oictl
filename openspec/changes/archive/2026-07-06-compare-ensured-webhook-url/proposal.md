## Why

Bootstrap and secret validation workflows need a way to confirm that an ensured channel incoming webhook resolves to a known URL without printing the secret-bearing webhook URL. Today `--verify-url` only verifies constructability, so callers cannot detect mismatched base URLs, webhook IDs, or tokens safely.

## What Changes

- Add expected URL comparison to `oictl webhooks channels ensure` via `--expected-url <url>`.
- Add `--expected-url-env <name>` so automation can read the expected URL from an environment variable without placing the secret value in command history.
- Make expected URL comparison verify the derived ensured webhook URL exactly while reporting only redacted or non-secret mismatch context.
- Treat expected URL comparison as an enhanced verification mode: it may be used with `--verify-url` for readability, but it cannot be combined with URL reveal or file output.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `webhooks-control`: Extend channel webhook ensure URL handling with secret-safe expected URL comparison modes.

## Impact

- Affects `oictl webhooks channels ensure` CLI flags, validation, and URL handling behavior.
- Affects tests for channel webhook ensure URL modes and secret redaction.
- No API endpoint or dependency changes are expected; comparison is client-side after the ensured webhook URL is derived.
