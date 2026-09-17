## Why

Manifest authors currently have to write declarative resources as JSON, even though YAML is more readable for multi-line content and nested configuration. Adding YAML input support improves authoring ergonomics while preserving the existing JSON contract and environment interpolation behavior.

## What Changes

- Accept `.yaml` and `.yml` files anywhere manifest JSON files are accepted.
- Include YAML files during deterministic directory manifest discovery.
- Decode YAML manifests into the same manifest document model used by JSON.
- Preserve existing JSON semantics, validation, duplicate detection, content file handling, plan generation, and mutation behavior.
- Preserve simple `${VAR}` environment interpolation after manifest decoding and before validation/planning.
- Preserve the clarified interpolation limit from `clarify-manifest-authoring-limits`: shell-style defaults such as `${VAR:-default}` remain unsupported for both JSON and YAML.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `declarative-resource-management`: Manifest loading accepts YAML files in addition to JSON while retaining existing manifest workflow semantics.

## Impact

- Affected code: manifest discovery and parsing in `internal/cli/manifests.go`, manifest tests in `internal/cli/manifests_test.go`, and manifest help/docs that describe supported input formats.
- Dependencies: likely adds or uses a YAML decoder dependency for Go.
- APIs: CLI flags remain unchanged; accepted file extensions expand from `.json` to `.json`, `.yaml`, and `.yml`.
- Related change: this supersedes the JSON-only documentation from `clarify-manifest-authoring-limits` but depends on the same simple placeholder contract.
