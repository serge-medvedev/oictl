## Context

`oictl users` already provides authenticated directory listing/search and nested settings operations. Other native command families implement CRUD as thin adapters over shared flag parsing, body loading, path escaping, confirmation, structured output, and HTTP error redaction. This change follows that established pattern and is verified only against `httptest`; no live user mutation is needed.

## Goals / Non-Goals

**Goals:**

- Add the four native user CRUD commands with exact endpoint, authentication, payload, confirmation, output, and failure behavior.
- Preserve all existing `users` commands and use focused request-contract tests.
- Keep implementation and documentation small enough to audit as one immutable candidate.

**Non-Goals:**

- User manifests, bulk CRUD, SCIM behavior changes, client-side user-field schemas, interactive password input, compatibility layers, or changes to settings/UI-settings behavior.
- Transforming successful server JSON, including token-bearing create responses.
- Exercising create, update, or delete against a live Open WebUI instance.

## Decisions

### Reuse the existing users dispatch and execution pipeline

Add four cases to `runUsers`, retaining JSON as that command family's default format and forwarding `--out` through `executeStructured`. A new command type or transport helper would duplicate existing behavior without adding a reusable abstraction.

### Keep request validation structural

Use `requireArgs`, `requiredBody`, `url.PathEscape`, and `requireConfirmation`. Open WebUI remains responsible for field schemas, roles, password policy, duplicate email, authorization, target existence, and deletion protections.

### Treat server responses as pass-through data

Return complete successful JSON consistently with current structured commands. Keep shared status/detail reporting and configured-credential redaction for failures rather than introducing user-specific output or error handling.

### Verify vertical slices with isolated HTTP contracts

Implement help, get, create, update, delete, validation/failures, and documentation in sequence. Each command slice starts with a focused failing test, then the smallest implementation that makes it pass; the final gate runs focused tests, the full suite, a build, built help inspection, OpenSpec validation, and diff checks.

### Bounded grill conclusions

The approved plan supplied the answers to 25 essential design questions: command family (`users`); operations (get/create/update/delete); get method/path; create method/path; update method/path; delete method/path; normal API authentication; server-side admin authorization; exact one-ID arity; ID path escaping; pass-through JSON inputs; required create payload; required update payload; no client field schema; complete success JSON; JSON-compatible default output; existing global/local output handling; `--out` handling; explicit delete confirmation; no request before confirmation; shared status/detail errors; configured API/JWT credential redaction; preservation of list/search/settings/UI-settings; SCIM separation; and `httptest`-only acceptance with no live mutations. No answer requires scope beyond the proposal; richer user schemas, prompting, bulk operations, manifests, and compatibility behavior remain optional follow-ups.

## Risks / Trade-offs

- Create responses may contain a newly issued token -> Preserve the established complete JSON contract and document sensitive payload handling rather than adding a one-off transformation.
- Upstream user forms may evolve -> Pass payloads through and delegate schema validation to Open WebUI.
- IDs containing path separators could alter routing -> Escape every user ID before path construction and assert `EscapedPath` in tests.
- Destructive requests could be accidental -> Validate arity and confirmation before constructing or sending the request.
- Existing nested users behavior could regress -> Retain its dispatch cases and assert old and new commands in help plus run the full suite.
