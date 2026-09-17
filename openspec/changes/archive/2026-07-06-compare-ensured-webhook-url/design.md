## Context

`oictl webhooks channels ensure` already finds or creates a named channel incoming webhook and can derive the secret-bearing incoming webhook URL for `--show-url`, `--out`, or `--verify-url`. `--verify-url` currently proves only that the response contains enough data to construct a URL; it does not compare the derived URL with an expected bootstrap secret.

The CLI parser stores non-boolean flags in `commandFlags.values`, so `--expected-url` and `--expected-url-env` can be added without parser structural changes. The sensitive value must not be printed in default output, errors, or mismatch diagnostics.

## Goals / Non-Goals

**Goals:**

- Add exact derived URL comparison for ensured channel webhooks.
- Support direct expected URL input and environment-sourced expected URL input.
- Preserve existing URL redaction guarantees and avoid printing the expected or derived URL during comparison.
- Validate incompatible URL modes before contacting Open WebUI.

**Non-Goals:**

- Add expected URL comparison to `webhooks channels url` or event webhook commands.
- Normalize, canonicalize, or partially compare URLs.
- Change Open WebUI API requests or webhook URL derivation rules.

## Decisions

- Treat `--expected-url` and `--expected-url-env` as comparison inputs for the existing verification mode. They may be supplied with `--verify-url` for readability, and they also imply verification when `--verify-url` is omitted. The effective URL handling modes remain: reveal to stdout, write to file, or verify/compare without revealing. Alternative considered: make expected URL flags mutually exclusive with `--verify-url`; rejected because `--verify-url --expected-url-env WEBHOOK_URL` is the natural reading of the client feedback and is safe.
- Resolve `--expected-url-env` before contacting Open WebUI and require a non-empty environment variable value. This fails fast for misconfigured automation and avoids unnecessary remote mutations. Alternative considered: allow empty values and let comparison fail; rejected because webhook URLs are never expected to be empty and the resulting mismatch would obscure the configuration error.
- Compare the derived URL and expected URL as exact strings. This avoids changing semantics around base URL paths, trailing slashes, URL encoding, or token casing. Alternative considered: parse and canonicalize URLs; rejected because callers need to validate the exact secret value they plan to store or consume.
- On success, print a short confirmation with the webhook ID only. On mismatch, exit non-zero with a redacted message that does not include either URL. Alternative considered: print redacted URL prefixes; rejected because even partial URL material can expose deployment details and is unnecessary for secret validation.
- Reject `--expected-url` with `--expected-url-env` before contacting Open WebUI. They are two sources for the same expected secret and allowing both would create unclear precedence.

## Risks / Trade-offs

- Exact comparison can fail when semantically equivalent URLs differ by formatting -> Document through tests and messages that comparison is exact.
- Direct `--expected-url` can place a secret URL in shell history -> Provide `--expected-url-env` for automation and describe it in help.
- Env lookup failures could otherwise occur after ensure creation -> Validate env mode before ensure work so missing secrets do not cause remote changes.
