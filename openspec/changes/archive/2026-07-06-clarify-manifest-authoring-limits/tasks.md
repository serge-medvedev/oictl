## 1. Placeholder Validation

- [x] 1.1 Constrain manifest environment placeholder names to simple environment identifiers in `renderManifestString`.
- [x] 1.2 Return a clear manifest-loading error for unsupported placeholder expressions such as `${VAR:-default}` before any remote request.
- [x] 1.3 Preserve existing behavior for valid `${VAR}` interpolation and missing simple variables.

## 2. Documentation

- [x] 2.1 Update `docs/manifests.md` to state that manifests are JSON-only currently.
- [x] 2.2 Update `docs/manifests.md` to state that interpolation supports simple `${VAR}` only and not shell-style defaults.
- [x] 2.3 Update any concise manifest references in `README.md` if they imply broader format or interpolation support.
- [x] 2.4 If `add-yaml-manifest-input` has already landed, keep the interpolation-limit documentation but do not reintroduce JSON-only wording.

## 3. Tests

- [x] 3.1 Add a manifest-loading test showing YAML or another non-JSON manifest body is rejected before remote requests.
- [x] 3.2 Add a manifest interpolation test showing `${VAR:-default}` fails and is not rendered as a default value.
- [x] 3.3 Run the manifest-related Go tests and fix any regressions.
