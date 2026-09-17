## 1. CLI Validation And Help

- [x] 1.1 Update users help text to document `bulk-patch --all`, `bulk-patch --query text`, and `--yes`/`--confirm` for non-dry-run directory discovery.
- [x] 1.2 Add `--all` to CLI boolean flag parsing if it is not already recognized.
- [x] 1.3 Validate bulk target modes so `--all` and `--query` are mutually exclusive.
- [x] 1.4 Validate bulk target modes so directory discovery cannot be combined with positional targets, `--user-id`, or `--users-file`.
- [x] 1.5 Require `--yes` or `--confirm` for non-dry-run `--all` and `--query` bulk patches before discovery or mutation requests are sent.
- [x] 1.6 Reject empty or whitespace-only `--query` values before directory discovery.

## 2. Directory Discovery Implementation

- [x] 2.1 Refactor bulk target collection to represent explicit and directory-derived target sources.
- [x] 2.2 Implement user directory discovery through `GET /api/v1/users/`, passing `query` for query mode.
- [x] 2.3 Page through directory responses until the reported `total` is collected or no new users are returned.
- [x] 2.4 Extract non-empty user `id` values from the `users` array and fail clearly when no target IDs are discovered.
- [x] 2.5 Preserve first-seen ordering and de-duplicate discovered user IDs before dry-run or patch execution.

## 3. Bulk Patch Results

- [x] 3.1 Extend bulk UI settings result entries with optional target-source metadata for directory-derived runs.
- [x] 3.2 Include `target_source=all` for `--all` dry-run and mutation result entries.
- [x] 3.3 Include `target_source=query` and the query text for `--query` dry-run and mutation result entries.
- [x] 3.4 Preserve existing explicit-target result JSON shape apart from compatible omitted optional fields.

## 4. Tests And Verification

- [x] 4.1 Add tests for `--all --dry-run` discovery, output metadata, no patch requests, and pagination/de-duplication.
- [x] 4.2 Add tests for `--query --dry-run` passing the query parameter and including query metadata.
- [x] 4.3 Add tests that non-dry-run `--all` and `--query` require `--yes` or `--confirm` and send no remote requests when missing confirmation.
- [x] 4.4 Add tests that confirmed `--all` and `--query` patch each discovered unique user and report per-user results.
- [x] 4.5 Add tests for invalid target-mode combinations, empty query values, and no-discovered-target errors.
- [x] 4.6 Run the relevant Go test package and fix regressions.
