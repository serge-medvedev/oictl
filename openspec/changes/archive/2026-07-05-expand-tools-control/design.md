## Context

`oictl` already has a reusable Open WebUI API client, JSON input loading, output handling, and declarative manifest support. The manifest registry includes `Tool` resources mapped to `/api/v1/tools/list`, `/api/v1/tools/id/{id}`, `/api/v1/tools/create`, `/api/v1/tools/id/{id}/update`, `/api/v1/tools/id/{id}/delete`, and `/api/v1/tools/id/{id}/access/update`, but there is no imperative `oictl tools` command family.

Open WebUI exposes the needed tool routes under `/api/v1/tools`, including `GET /list`, `GET /id/{id}`, `POST /create`, `POST /id/{id}/update`, `DELETE /id/{id}/delete`, `POST /id/{id}/access/update`, `GET /export`, and `POST /load/url`. The CLI should remain a thin operator-focused layer over those APIs.

## Goals / Non-Goals

**Goals:**

- Add `oictl tools` as a resource-first command family for the requested imperative operations.
- Reuse existing global target/auth flags, JSON passthrough input, output handling, HTTP error reporting, and test patterns.
- Keep create and update payloads compatible with Open WebUI `ToolForm` fields: `id`, `name`, `content`, `meta`, and optional `access_grants`.
- Provide a manifest export mode that produces current `Tool` manifest documents accepted by `oictl manifests apply` and `sync --scope tools`.

**Non-Goals:**

- Do not add tool execution, runtime validation, valves management, function management, prompt management, or tool server configuration commands.
- Do not invent a typed local schema that diverges from Open WebUI's tool request and response models.
- Do not change the existing manifest envelope, manifest diff/apply/sync behavior, or access grant model.

## Decisions

1. Implement `tools` as a thin endpoint-backed command family.

   Commands will map directly to Open WebUI routes: `list` to `GET /api/v1/tools/list`, `get` to `GET /api/v1/tools/id/{id}`, `create` to `POST /api/v1/tools/create`, `update` to `POST /api/v1/tools/id/{id}/update`, `delete` to `DELETE /api/v1/tools/id/{id}/delete`, `access-update` to `POST /api/v1/tools/id/{id}/access/update`, `export` to `GET /api/v1/tools/export`, and `load-url` to `POST /api/v1/tools/load/url`. Alternative considered: route through manifest handlers for all operations, but direct commands should surface raw imperative API behavior and avoid manifest planning side effects.

2. Use JSON passthrough for complex payloads.

   `create`, `update`, `access-update`, and `load-url` will accept `--data` or `--file` input and submit the body unchanged except for existing JSON loading and request encoding. This matches existing command conventions and avoids hard-coding evolving tool metadata fields. Alternative considered: first-class flags for every `ToolForm` field, but Python source content and nested metadata are better represented as JSON.

3. Keep raw export as the default and add explicit manifest export.

   `tools export` will call the Open WebUI export endpoint and print or write the server response. `tools export --manifest` will transform exported tool models into `oictl.openwebui/v1` `Tool` manifest documents. For a single tool, the CLI can print one manifest document; for multiple tools, it should write deterministic per-tool JSON files to a requested directory rather than emitting a bundle the current manifest loader cannot consume. Alternative considered: changing manifest loading to accept arrays, but that would broaden this change beyond tools control.

4. Use stable manifest identity and managed fields.

   Manifest export will use the tool `id` as `metadata.name` to avoid duplicate display-name collisions. The manifest `spec` should include only fields accepted by `ToolForm` (`id`, `name`, `content`, `meta`) and place access grants in top-level `access_grants` when present. Generated fields such as `user_id`, `specs`, `created_at`, `updated_at`, and `write_access` should not be exported into `spec` because the manifest planner treats desired `spec` fields as managed drift targets. Alternative considered: exporting the full tool model, but that would create noisy or invalid updates.

5. Let Open WebUI remain authoritative for authorization, import safety, and sharing policy.

   The CLI should validate only command shape and JSON readability before sending requests. Server responses for forbidden operations, invalid tool IDs, invalid Python content, and filtered access grants should be surfaced with existing HTTP error handling. Alternative considered: duplicating server validation client-side, but that would drift from Open WebUI behavior.

## Risks / Trade-offs

- Tool payload schemas may evolve → Mitigation: use JSON passthrough and avoid local request structs unless needed for manifest transformation.
- Manifest export may omit fields a future Open WebUI `ToolForm` adds → Mitigation: keep the transformation focused on current accepted managed fields and update it with upstream model changes.
- `load-url` fetches remote Python source through Open WebUI and may require admin permissions → Mitigation: document that the command returns the server result and surfaces 401/403/4xx/5xx responses directly.
- Multiple tools may share the same display name → Mitigation: use tool IDs for manifest file names and `metadata.name`.

## Migration Plan

No data migration is required. Existing commands and manifest workflows remain unchanged. Rollback is removing the new `tools` command family and tests; server-side tool records are only changed when users invoke the new mutation commands.

## Open Questions

- Should `tools export --manifest` support printing multiple newline-delimited JSON manifest documents, or only require `--directory` for multi-tool manifest export to stay strictly compatible with current manifest loading?
