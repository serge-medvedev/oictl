## Context

Manifest loading currently parses JSON documents directly into `manifestDocument`, validates the envelope, and sends `spec` fields through the same planning pipeline used by diff, apply, and sync. Large resource bodies must be embedded as escaped JSON strings, and environment-specific values require separate manifest copies or external preprocessing. Several supported kinds already include `is_active` in their managed fields, but the docs do not explicitly say when that field is reconciled.

The change should improve authoring without changing the remote Open WebUI API payload contracts or the existing plan execution model.

## Goals / Non-Goals

**Goals:**
- Render environment placeholders in manifest string values before validation and planning.
- Support `spec.content_file` for resources that use `spec.content`, resolving local file contents into `spec.content` before diff, apply, and sync planning.
- Keep manifest plans based on the resolved desired state so table and JSON output remain accurate.
- Document active-state reconciliation for supported resources with `is_active` managed fields.

**Non-Goals:**
- Add YAML manifest support.
- Add a template language with conditionals, loops, filters, or external includes beyond `content_file`.
- Add new Open WebUI endpoints or change access grant semantics.
- Change destructive sync confirmation or pruning scopes.

## Decisions

1. Render parsed JSON string values instead of preprocessing raw files.

   Parsed rendering avoids corrupting JSON syntax and keeps substitution limited to manifest data values. The renderer will walk the manifest envelope, metadata, spec, and access grants, replacing `${VAR}` occurrences in strings with `os.Getenv("VAR")`. Missing variables fail validation before any remote request. Alternative considered: raw text substitution before JSON parsing. That would allow env values to affect JSON structure, but it is easier to produce invalid JSON and harder to reason about safely.

2. Use `${VAR}` as the only placeholder syntax.

   A single familiar form keeps docs and validation simple. Defaults and optional variables are intentionally out of scope for this slice; users can set environment defaults in their shell or invocation wrapper. Alternative considered: `${VAR:-default}` shell-style expansion. That adds parsing complexity and can hide configuration mistakes.

3. Resolve `content_file` during manifest normalization.

   `content_file` will be accepted only inside `spec`. The path is rendered for environment placeholders, then resolved relative to the manifest file directory unless absolute. The file bytes are read as UTF-8 text and assigned to `spec.content`; `content_file` is removed from the desired spec before planning so remote payloads and changed field names continue to use `content`. Alternative considered: keep `content_file` in the spec and teach each handler to ignore it. Central normalization is smaller and prevents accidental API payload leakage.

4. Reject ambiguous content declarations.

   A manifest that declares both `spec.content` and `spec.content_file` will fail before remote requests. This avoids unclear precedence and makes generated diffs predictable.

5. Treat active-state reconciliation as field-managed behavior.

   For kinds that manage `is_active`, a manifest reconciles active state only when `spec.is_active` is present in the desired spec. If omitted, diff and apply do not change active state for existing resources. This matches the current field-by-field diff model and should be documented rather than replaced by a separate active-state command path.

## Risks / Trade-offs

- Missing environment variables may break workflows that expected literal `${VAR}` text -> fail-fast behavior is safer for manifests; docs should call out escaping or avoiding the placeholder form when literals are needed.
- Relative `content_file` paths can be confusing when manifests are loaded from nested directories -> resolve relative to each manifest file and include the manifest source path in errors.
- Large content files increase memory use because manifests already operate on in-memory specs -> acceptable for CLI-sized resource bodies; no streaming is needed for current Open WebUI payloads.
- Removing `content_file` from resolved specs means exported plans do not show the original file reference -> this is intentional because the plan represents the API-visible desired state.

## Migration Plan

Existing manifests continue to work unchanged. Users can migrate embedded content by moving the text to a sibling file and replacing `spec.content` with `spec.content_file`. Rollback is to inline the content back into `spec.content` and avoid `${VAR}` placeholders.

## Open Questions

- None for the proposal. Implementation can choose the exact validation error wording as long as it identifies the source manifest and missing variable or unreadable content file.
