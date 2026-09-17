## 1. CLI Surface

- [x] 1.1 Update webhooks help and ensure usage text to show `--expected-url` and `--expected-url-env` as optional inputs to non-revealing URL verification.
- [x] 1.2 Confirm command flag parsing accepts both expected URL flags as value flags without changing unrelated flag behavior.

## 2. Ensure URL Comparison

- [x] 2.1 Extend `ensureChannelWebhook` URL mode validation so `--show-url`, URL `--out`, and verification/comparison are exclusive, while `--verify-url` may be combined with one expected URL source.
- [x] 2.2 Resolve `--expected-url-env` before contacting Open WebUI and fail when the env var name or value is empty.
- [x] 2.3 Compare the constructed ensured webhook URL exactly against the direct or environment-sourced expected URL.
- [x] 2.4 Print a non-secret success message on match and a non-secret error message on mismatch.
- [x] 2.5 Reject `--expected-url` and `--expected-url-env` together before contacting Open WebUI.

## 3. Tests

- [x] 3.1 Add successful `--expected-url` and `--expected-url-env` tests that assert no full token, derived URL, or expected URL leaks to stdout or stderr.
- [x] 3.2 Add mismatch and missing environment variable tests that assert non-zero exit and secret-safe diagnostics.
- [x] 3.3 Add coverage that `--verify-url --expected-url` and `--verify-url --expected-url-env` are valid, and that expected URL flags without `--verify-url` imply verification.
- [x] 3.4 Add exclusivity coverage for reveal/write versus verify/compare modes and for conflicting expected URL sources, ensuring invalid combinations fail before contacting Open WebUI.
- [x] 3.5 Run the Go test suite covering CLI webhook behavior.
