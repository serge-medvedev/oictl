## Context

`oictl users ui-settings bulk-patch` already applies one UI settings payload to an explicit target set collected from positional user IDs, repeated `--user-id`, or `--users-file`. Open WebUI's admin user listing endpoint, `GET /api/v1/users/`, returns paginated responses containing `users` records with stable `id` fields and a `total` count. The existing `oictl users list` command exposes the same endpoint with query, page, ordering, and direction parameters.

## Goals / Non-Goals

**Goals:**

- Let operators build bulk UI settings target sets directly from the user directory with `--all` or `--query`.
- Keep broad, directory-derived mutations non-interactive but protected by an explicit `--yes` or `--confirm` acknowledgement unless `--dry-run` is used.
- Preserve deterministic sequential patch execution, de-duplication, sensitive-key validation, credential redaction, and per-user result reporting.
- Make dry-run output useful for audits by including discovered target IDs and source metadata.

**Non-Goals:**

- Do not add server-side batch mutation APIs or require Open WebUI changes.
- Do not add a general user query language beyond the server-supported `query` text filter.
- Do not add concurrency, retries, or rate-limit controls in this change.
- Do not change `oictl users list` or `oictl users search` response passthrough behavior.

## Decisions

- Use `--all` and `--query text` on `bulk-patch` rather than a separate discovery subcommand. This keeps discovery attached to the mutation workflow and mirrors existing target flags.
- Treat explicit target sources and directory discovery as mutually exclusive modes. Mixing positional or file targets with `--all` or `--query` would make the effective target set less obvious, so the CLI should fail before reading remote users or sending patches.
- Treat `--all` and `--query` as mutually exclusive. `--all` means no query filter; `--query` means the non-empty, trimmed server query filter selects the target set. Empty `--query` values should fail rather than silently becoming all-user discovery.
- Require `--yes` or `--confirm` for non-dry-run directory-derived target sets. This follows the repository's established confirmation convention for broad or destructive actions while remaining non-interactive and automation-friendly.
- Discover targets from `GET /api/v1/users/`, passing `query` only for `--query`. Page through results by incrementing `page` until the number of collected users reaches the response `total` or a page returns no new users. De-duplicate by `id` while preserving first-seen order.
- Keep result output as the existing JSON array of per-user result entries. Add optional metadata fields for directory-derived runs, such as `target_source` and `query`, so existing explicit-target consumers are not forced into a new envelope shape.

## Risks / Trade-offs

- Large tenants may require many listing requests before mutation begins. Mitigation: use the endpoint's `total` and stop as soon as all reported users have been collected; leave page-size changes or concurrency for a later change if needed.
- Directory contents can change between discovery and patch execution. Mitigation: patch only the concrete IDs discovered for that invocation and show the planned IDs in dry-run output.
- A query that matches more users than intended can still be risky. Mitigation: require `--yes` for non-dry-run directory-derived patches and recommend `--dry-run` before `--yes` in help and tests.
- Response-shape drift could break target extraction. Mitigation: validate that the listing response contains a `users` array with non-empty string `id` fields and fail before mutation if targets cannot be extracted.
- Empty or whitespace query values can look like a scoped rollout while behaving like a broad rollout. Mitigation: reject empty `--query` before discovery.

## Migration Plan

No migration is required. Existing explicit bulk-patch invocations, output fields, and validation behavior remain valid. New confirmation requirements apply only when targets are discovered via `--all` or `--query` and `--dry-run` is not present.

## Open Questions

- Should a later change add filters beyond `query`, such as role or group membership, after upstream support and operator needs are clearer?
