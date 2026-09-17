## 1. Manifest Decoding

- [x] 1.1 Add a YAML decoder dependency and wire `.yaml`/`.yml` parsing into `readManifestDocument` while keeping `.json` on `encoding/json`.
- [x] 1.2 Return clear parse errors that include the manifest source path for both JSON and YAML decode failures.
- [x] 1.3 Keep `normalizeManifestDocument`, environment interpolation, validation, duplicate detection, and content file expansion shared after decoding.
- [x] 1.4 Preserve simple placeholder validation for YAML, including rejection of shell-style defaults such as `${VAR:-default}`.

## 2. Manifest Discovery And Help

- [x] 2.1 Update directory discovery to include `.json`, `.yaml`, and `.yml` files in one sorted deterministic path list.
- [x] 2.2 Preserve current handling of unsupported file extensions: directories only discover supported manifest extensions, while explicit non-YAML file paths continue through JSON decoding for compatibility.
- [x] 2.3 Update manifest help/docs wording to describe JSON/YAML manifest inputs.
- [x] 2.4 If `clarify-manifest-authoring-limits` has landed, replace its JSON-only wording with JSON/YAML wording while keeping the simple interpolation-limit language.

## 3. Tests

- [x] 3.1 Add loader tests for explicit `.yaml` and `.yml` manifest files.
- [x] 3.2 Add directory discovery coverage for mixed `.json`, `.yaml`, and `.yml` manifests with deterministic loading and duplicate detection.
- [x] 3.3 Add YAML environment interpolation tests for rendered values and missing variables failing before remote requests.
- [x] 3.4 Add a YAML interpolation test proving `${VAR:-default}` fails and is not rendered as a default value.
- [x] 3.5 Add or update regression tests proving JSON manifest loading semantics remain unchanged.
- [x] 3.6 Run `go test ./...`.
