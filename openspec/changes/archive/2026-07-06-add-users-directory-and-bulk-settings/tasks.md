## 1. User Directory Commands

- [x] 1.1 Add `users list` routing, help text, query construction, and raw/structured output using `GET /api/v1/users/`.
- [x] 1.2 Add `users search` routing, help text, query construction, and raw/structured output using `GET /api/v1/users/search`.
- [x] 1.3 Add tests covering list/search endpoint paths, query parameters, help text, and authorization error propagation.

## 2. Bulk UI Settings Workflow

- [x] 2.1 Add `users ui-settings bulk-patch` routing and usage validation without changing existing `ui-settings patch <user-id>` behavior.
- [x] 2.2 Implement target collection from positional user IDs, repeated `--user-id`, and `--users-file`, including trimming, de-duplication, and empty-target errors.
- [x] 2.3 Reuse UI settings payload validation and sensitive key guardrails for bulk payloads, including `--allow-sensitive-ui-keys` behavior.
- [x] 2.4 Implement `--dry-run` planning output that validates local inputs and reports one planned result per target without sending patch requests.
- [x] 2.5 Implement sequential per-user patch execution with per-user result entries and non-zero exit when any target fails.

## 3. Verification

- [x] 3.1 Add tests for explicit-target bulk patch success and users-file bulk patch success.
- [x] 3.2 Add tests for dry-run no-request behavior, missing targets, missing payload, sensitive key rejection, duplicate target handling, and partial failure reporting.
- [x] 3.3 Run `go test ./...` and fix any regressions.
