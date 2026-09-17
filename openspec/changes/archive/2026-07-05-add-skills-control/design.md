## Context

`oictl` already exposes resource-first command families for Open WebUI workspace resources such as models, files, knowledge, chats, analytics, automations, SCIM, and provider passthrough. Open WebUI Skills are persisted workspace resources with native API endpoints under `/api/v1/skills`, access grants using resource type `skill`, active/inactive state, and import/export permissions.

The upstream Skills UI and docs describe Markdown-based skill content with optional YAML frontmatter for `name` and `description`, while the backend API stores Skills as JSON objects with `id`, `name`, `description`, `content`, `meta`, `is_active`, and `access_grants`. The CLI should use the API JSON shape as the source of truth and add Markdown manifest ergonomics only where it stays simple and reversible enough for command-line workflows.

## Goals / Non-Goals

**Goals:**
- Add a resource-first `oictl skills` command family that follows existing `oictl` routing, auth, output, and payload conventions.
- Cover list/search, get, create, update, access-update, toggle, delete, and export operations exposed by Open WebUI Skills APIs.
- Preserve Open WebUI server authorization, workspace feature, sharing, public-sharing, and import/export permission behavior.
- Support JSON payloads for full-fidelity create, update, access-update, and export workflows.
- Support a small Markdown Skill manifest workflow for create/update and per-skill export when it can be implemented without broad YAML support or a new dependency.

**Non-Goals:**
- Do not add declarative manifest diff/apply/sync for Skills in this change.
- Do not add model-to-skill binding management; that belongs with model configuration semantics.
- Do not reimplement Open WebUI permission checks or sharing policy enforcement locally.
- Do not define a new oictl-specific persisted Skill schema beyond the API JSON object and the optional Markdown import/export projection.

## Decisions

1. Add `skills` as the command family name.

Rationale: Skills is the upstream Open WebUI domain noun, and the existing taxonomy uses resource-first command families. Alternatives considered were `workspace skills` and `skill`; those would introduce hierarchy or singular naming inconsistent with current families.

2. Use native Skills API JSON as the canonical payload.

Rationale: JSON preserves `id`, `name`, `description`, `content`, `meta`, `is_active`, and `access_grants` without lossy conversion. It also matches existing `--data`, `--file`, and `--out` command behavior. The alternative was to introduce a new local manifest schema, but that would duplicate server models and create compatibility risk.

3. Make Markdown manifest support a bounded convenience layer.

Rationale: Upstream docs already support importing Markdown files with YAML frontmatter for human-authored Skills. The CLI can support this by converting a Markdown file into the native Skill JSON form for create/update and by projecting one fetched Skill to Markdown for export. To avoid a new dependency, the parser should support only a leading `---` frontmatter block with scalar `id`, `name`, `description`, optional `is_active`, and simple `tags`; the Markdown body becomes `content`. Full access-grant editing remains JSON via `create`, `update`, or `access-update` payloads.

Alternative considered: skip manifest support entirely and rely on JSON only. That would be smaller but would miss the primary human-authored Skill workflow documented upstream.

4. Keep deletion guarded with `--yes`.

Rationale: Existing destructive resource commands require explicit confirmation in non-interactive usage. Skills deletion is permanent and should follow the same safety convention. The alternative was direct API parity without local confirmation, but that would be inconsistent with `files`, `knowledge`, `chats`, `automations`, and SCIM deletion behavior.

5. Reuse existing request, output, and error handling.

Rationale: `executeStructured`, `requestBody`, `appendQuery`, `requireConfirmation`, and table rendering already provide consistent auth, output file handling, and server error surfacing. The Skills command should only add endpoint routing and small payload conversion where needed.

## Risks / Trade-offs

- Limited Markdown frontmatter parsing may not accept all YAML that the Open WebUI frontend accepts -> Document the supported subset in help/tests and tell users to use JSON for complex fields.
- Per-skill Markdown export requires fetching a single Skill because bulk `/api/v1/skills/export` returns a JSON array -> Keep bulk export JSON-only and limit Markdown export to `oictl skills export <skill-id> --format manifest`.
- Open WebUI Skills APIs may differ across server versions -> Surface server 404/401/403/400 responses directly and keep command logic thin.
- Access grants can be filtered by the server based on sharing policy -> Do not attempt to predict filtering locally; print the returned Skill so users can observe accepted grants.
- Search/list endpoints have overlapping semantics (`/` and `/list`) -> Use `skills list` for `/api/v1/skills/list` because it includes pagination/search/write access, and reserve a simple `--all-accessible` flag only if implementation finds `/api/v1/skills/` is necessary.

## Migration Plan

No persisted local migration is required. Implementation adds CLI commands and tests only. Rollback is removing the `skills` command routing and associated helper/tests before release.
