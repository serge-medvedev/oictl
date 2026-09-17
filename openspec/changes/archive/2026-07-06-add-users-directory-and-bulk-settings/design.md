## Context

The CLI already routes `oictl users` commands through `internal/cli/surfaces.go` and supports current-user settings inspection/update plus a single-user admin UI settings patch. Open WebUI exposes user directory routes under `/api/v1/users/` and `/api/v1/users/search` with `query`, `order_by`, `direction`, and `page` query parameters. Bulk settings changes can be implemented client-side by expanding a target list and calling the existing admin UI settings patch endpoint once per user.

## Goals / Non-Goals

**Goals:**
- Add resource-first user directory commands that return server JSON unchanged and support existing global output/file behavior where practical.
- Add a bulk UI settings patch command with target validation, dry-run planning, existing sensitive UI key guardrails, and per-user results.
- Keep single-user settings commands stable and compatible with existing tests.

**Non-Goals:**
- Do not add server-side batch APIs or require Open WebUI changes.
- Do not introduce CSV parsing beyond a simple newline-delimited target file unless implementation discovers an existing project convention.
- Do not add declarative manifest support for users or settings in this change.

## Decisions

- Use `oictl users list` and `oictl users search` for directory access, mapping directly to `GET /api/v1/users/` and `GET /api/v1/users/search`. This follows the existing command taxonomy and preserves Open WebUI response shapes instead of defining an oictl-specific user record model.
- Add `oictl users ui-settings bulk-patch` rather than overloading `patch`. A distinct operation keeps one-user and many-user workflows clear, makes `--dry-run` semantics explicit, and avoids changing existing `patch <user-id>` behavior.
- Accept targets from repeated `--user-id`, positional user IDs, or `--users-file` with one user ID per non-empty line. This supports shell usage and auditable rollouts while keeping parsing minimal.
- Implement bulk execution sequentially. Sequential requests produce deterministic per-user reporting, avoid surprising rate spikes against Open WebUI, and are simpler to reason about for partial failures.
- Report bulk results as structured JSON by default, with one entry per target containing the user ID, action, status, and error detail when a request fails. This preserves automation friendliness and gives operators enough information to retry failures.

## Risks / Trade-offs

- Large target sets may be slow because requests run sequentially. Mitigation: keep the initial workflow deterministic and leave concurrency/rate-limit flags for a later change if needed.
- Client-side bulk operations can partially succeed. Mitigation: return per-user results and exit non-zero when any non-dry-run user update fails.
- User IDs from files may include accidental whitespace or duplicates. Mitigation: trim blank lines, reject an empty final target set, and de-duplicate targets while preserving first-seen order.
- Dry-run cannot prove remote authorization or per-user existence without sending mutation requests. Mitigation: document dry-run as local validation and target/payload planning only.

## Migration Plan

No migration is required. Existing `oictl users settings` and `oictl users ui-settings patch` commands remain unchanged, and the new commands are additive.

## Open Questions

- Should a later change add `--concurrency` or retry controls after real-world bulk rollout sizes are known?
