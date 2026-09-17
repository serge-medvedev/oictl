## Context

Manifest loading currently decodes files with Go's JSON decoder into the manifest envelope, then walks parsed string values to render environment placeholders. The manifest docs mention JSON examples and `${VAR}` interpolation, while the prior authoring design explicitly kept YAML and shell-style defaults out of scope. This change turns those limits into user-facing, test-backed contract details.

This change intentionally captures the current pre-YAML contract. If `add-yaml-manifest-input` is implemented later, it should supersede only the JSON-only format text while keeping the simple placeholder syntax and unsupported-default behavior defined here.

## Goals / Non-Goals

**Goals:**
- State clearly that manifest files are JSON documents in the current implementation.
- State clearly that environment interpolation accepts only simple `${VAR}` placeholders with normal environment variable names.
- Reject or document shell-style expressions such as `${VAR:-default}` as unsupported instead of treating them as default expansion.
- Keep existing valid JSON manifests and simple `${VAR}` interpolation working unchanged.

**Non-Goals:**
- Add YAML, JSON5, HCL, or another manifest format.
- Add a template language, expression evaluation, optional variables, or default values.
- Change `content_file`, access grant resolution, sync pruning, or resource-specific API payload behavior.

## Decisions

1. Keep JSON parsing as the manifest format boundary.

   Documentation should describe manifests as JSON-only and examples should continue using `.json`. No new parser should be introduced. Alternative considered: adding YAML while clarifying interpolation. That expands the supported surface and conflicts with the goal of documenting the current contract.

2. Validate placeholder names as simple environment variable identifiers.

   Interpolation should continue replacing `${VAR}` after JSON parsing, but the placeholder name should be constrained to a simple variable identifier such as `VAR` or `TEAM_NAME`. Expressions containing `:`, `-`, whitespace, or other shell syntax should fail before remote requests. Alternative considered: continue looking up the entire text between braces as an environment variable name. That technically fails for typical `${VAR:-default}` input, but it produces misleading errors and leaves unsupported syntax ambiguous.

3. Cover unsupported shell defaults with tests and docs.

   A regression test should demonstrate that `${VAR:-default}` is not expanded to `default` and fails before any Open WebUI request. Docs should tell users to set defaults in their shell or wrapper before invoking `oictl`. Alternative considered: docs-only clarification. Tests are small and protect the intentionally narrow interpolation contract.

## Risks / Trade-offs

- Users who expected YAML support may still try it -> docs will explicitly say manifests are JSON-only today.
- Users may want inline defaults for reusable manifests -> recommend setting environment defaults outside `oictl`; this keeps manifest evaluation predictable.
- Tightening placeholder validation could reject exotic process environment names that previously could be referenced -> acceptable because the documented supported syntax is simple `${VAR}` only.

## Migration Plan

Existing JSON manifests that use simple `${VAR}` placeholders continue to work. Manifests using shell-style defaults should move defaults to the invoking shell, CI environment, or wrapper script before running `oictl`. Rollback is to remove the stricter placeholder-name validation while keeping the documentation clarification.

## Open Questions

- None.

## Dependency Notes

- Prefer implementing this before `add-yaml-manifest-input` so the current contract is made explicit before YAML expands the accepted formats.
- If YAML support is implemented first, implementation of this change should skip reintroducing JSON-only user-facing text and should keep only the placeholder validation/docs/tests.
