## Context

Manifest workflows currently discover `.json` files from directories, reject `.yaml` and `.yml`, and parse every manifest through `encoding/json` before applying the existing normalization, environment rendering, validation, duplicate detection, content file expansion, planning, and execution logic. This change is limited to manifest input decoding and related user-facing text.

## Goals / Non-Goals

**Goals:**

- Accept `.yaml` and `.yml` manifests through existing `--file`, `--directory`, and `--dir` inputs.
- Keep JSON behavior unchanged, including error paths, validation timing, duplicate detection, and plan/apply/sync semantics.
- Apply existing simple `${VAR}` environment interpolation to YAML string values after decoding and before validation/planning.
- Keep directory loading deterministic across mixed JSON and YAML files.

**Non-Goals:**

- Introduce a new manifest schema version or change resource envelope fields.
- Add templating beyond current `${VAR}` string interpolation.
- Add shell-style defaults or optional variables; `${VAR:-default}` remains unsupported.
- Support multi-document YAML streams in a single file.
- Change manifest workflow flags or resource reconciliation behavior.

## Decisions

- Use extension-based parsing in `readManifestDocument`: `.json` continues to use `encoding/json`; `.yaml` and `.yml` use a YAML decoder targeting the existing `manifestDocument` structs and maps. Alternative considered: convert YAML to JSON first, but direct struct/map decoding is simpler and keeps source-aware error handling in one place.
- Include `.json`, `.yaml`, and `.yml` in `discoverManifestPaths`, then sort the complete path list once. Alternative considered: loading per-extension groups, but that could change mixed-directory ordering and duplicate reporting.
- Keep normalization and `renderManifestString` after decoding for both JSON and YAML. This preserves interpolation semantics and avoids a YAML-specific preprocessor.
- Preserve explicit `--file` JSON behavior for non-YAML extensions for compatibility: `.yaml` and `.yml` use the YAML decoder, while `.json` and other explicit file names continue through the JSON decoder. Directory discovery should load only `.json`, `.yaml`, and `.yml` files.
- Update help/docs to say JSON/YAML manifests rather than changing flags or examples wholesale.
- When combined with `clarify-manifest-authoring-limits`, update any JSON-only wording to JSON/YAML while retaining the simple placeholder syntax and unsupported-default guidance.

## Risks / Trade-offs

- YAML scalar typing can decode unquoted values into non-string types before validation. Mitigation: rely on the existing manifest struct/map validation and document that YAML authors should quote values that must remain strings.
- A new YAML dependency increases supply-chain surface. Mitigation: use the standard Go YAML package already common in Go CLIs, pin it in `go.mod`, and limit usage to manifest decoding.
- YAML supports anchors and aliases, which can make authoring less explicit. Mitigation: decoded output still passes through the same manifest validation and planning path; no new execution semantics are added.
- Multi-document YAML streams could create ambiguity for duplicate detection and source reporting. Mitigation: keep scope to one manifest document per file.
