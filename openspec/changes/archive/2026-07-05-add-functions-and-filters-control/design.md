## Context

`oictl` already has a resource-first command taxonomy and shared expectations for Open WebUI API access, JSON file/stdin payloads, output files, and server-backed error handling. Open WebUI exposes function administration through `/api/v1/functions`; function records include an `id`, `name`, server-derived `type`, Python `content`, metadata, active/global flags, timestamps, and optional valves on export or sync payloads.

Open WebUI filters are not a separate persisted resource. They are function records whose loaded plugin type is `filter`, and toggleable-filter behavior is represented in returned metadata when the server detects `self.toggle = True` during plugin loading. The CLI must therefore avoid executing plugin code locally or inventing a separate filter data model.

## Goals / Non-Goals

**Goals:**

- Add `oictl functions` commands for listing, retrieving, creating, updating, deleting, exporting, loading source from a URL, syncing, activating/deactivating, and toggling global scope.
- Represent filters as function records with `type: filter`, including list filtering and output fields that make active/global/toggleable state visible.
- Reuse existing API client, body loading, output formatting, output-file writing, and server error handling patterns.
- Keep function payloads close to Open WebUI request/response JSON so plugin metadata and future function types are not lost.
- Make destructive sync and delete behavior explicit in help, flags, and tests.

**Non-Goals:**

- Do not add a top-level `filters` command family in this change.
- Do not execute, parse, sandbox, lint, or classify Python plugin code locally; Open WebUI remains responsible for loading and validating functions.
- Do not add function valves or user-valves management commands unless a later proposal scopes them explicitly.
- Do not extend declarative manifests for functions in this change.

## Decisions

1. Use one `functions` command family for functions and filters.

   Open WebUI stores filters in the same `function` table and exposes them through the same `/api/v1/functions` routes. The CLI will use `oictl functions list --type filter` for filter-focused discovery and normal `functions` verbs for individual filter records. Alternative considered: add `oictl filters`, but that would duplicate command semantics, imply a separate upstream API, and make mixed export/sync payloads harder to explain.

2. Preserve server-derived function type and metadata.

   Create, update, and sync commands submit JSON payloads containing source and metadata to Open WebUI; the server loads the plugin and returns the resulting `type`, manifest metadata, active/global state, and toggle metadata. The CLI does not accept a `--type` override for create/update because that could diverge from the server's plugin loader. Alternative considered: require users to pass `--type filter` when creating filters, but upstream ignores local intent and determines the type from the loaded module.

3. Keep payload interfaces JSON-first.

   Commands that mutate full function records use `--file` or stdin JSON compatible with Open WebUI forms and export/sync models. Export supports output-file writing and an option to request server-included valves when available. Alternative considered: expose source, name, description, active, and global as many flags, but plugin metadata and future record shape changes make JSON payloads safer and more scriptable.

4. Treat `sync` as destructive reconciliation.

   Open WebUI's sync route updates or inserts supplied functions and removes functions omitted from the submitted list. The CLI will require an explicit file/stdin input and a confirmation flag for non-interactive sync execution. Alternative considered: rename it to `import`, but the upstream route performs pruning semantics that match the taxonomy's `sync` meaning rather than additive import.

5. Delegate plugin-fetch and plugin-load security to Open WebUI while making trust boundaries clear.

   `load-url`, create, update, and sync can involve arbitrary Python code that executes on the Open WebUI server when loaded. The CLI forwards requests only to the configured Open WebUI instance, surfaces server errors, and documents that admins must trust plugin sources. Alternative considered: pre-fetch or inspect plugin code in the CLI, but that moves security-sensitive behavior into a local client and can disagree with server behavior.

## Risks / Trade-offs

- Function source can execute arbitrary Python on the server -> Limit commands to authenticated Open WebUI API calls, keep admin-only server errors visible, and include trust warnings in help/docs for create, update, load-url, and sync.
- Sync can delete omitted remote functions -> Require explicit input and `--yes` for non-interactive sync, document pruning semantics, and test confirmation failures before any HTTP request.
- Upstream function record shapes may evolve -> Preserve JSON payloads and full JSON output rather than over-modeling every field as typed flags.
- Filters may look like a missing command family to users -> Provide `--type filter` examples and help text stating filters are Open WebUI functions of type `filter`.
- Server list endpoints expose different details for verified users and admins -> Keep command behavior endpoint-aligned and surface authorization failures instead of silently downgrading responses.

## Migration Plan

No local or server data migration is required. Existing profiles and API behavior remain unchanged. Rollback is removing the `functions` command family; any remote function records created, updated, synced, toggled, globalized, or deleted through the CLI remain on the Open WebUI server and must be corrected through Open WebUI or backups.

## Open Questions

- Should the first implementation expose both the verified-user list endpoint and the admin list-with-users endpoint, or make `functions list` admin-oriented and rely on server authorization for non-admin failures?
