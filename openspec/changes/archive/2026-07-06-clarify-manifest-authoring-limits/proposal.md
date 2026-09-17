## Why

Manifest authoring recently gained environment interpolation and content-file conveniences, but the intended limits are easy to misread as broader templating or YAML support. Clarifying the current contract now prevents users from depending on unsupported manifest formats or shell-style expansion behavior.

## What Changes

- Document that manifest documents are JSON-only in the current implementation, until the separate `add-yaml-manifest-input` change expands supported input formats.
- Clarify that environment interpolation intentionally supports only simple `${VAR}` placeholders.
- Clarify that shell-style default expressions such as `${VAR:-default}` are not supported and should fail rather than be interpreted as defaults.
- Add or update tests/docs to lock in the documented authoring limits.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `declarative-resource-management`: Clarify manifest format and environment interpolation requirements for declarative manifest authoring.

## Impact

- Affected docs: `docs/manifests.md`, potentially `README.md` manifest references.
- Affected tests: manifest loading/interpolation tests in `internal/cli/manifests_test.go`.
- Affected behavior: no new manifest language or format support; only clearer rejection/documentation of unsupported forms.
- Change ordering: implement this before `add-yaml-manifest-input`, or have the YAML change update the same docs from "JSON-only" to "JSON/YAML" while preserving the simple `${VAR}` interpolation contract.
