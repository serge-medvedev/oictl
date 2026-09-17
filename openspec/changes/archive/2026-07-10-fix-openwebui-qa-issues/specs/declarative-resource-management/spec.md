## ADDED Requirements

### Requirement: Model manifest diff normalizes optional null metadata
The CLI SHALL treat absent and `null` values for optional model `spec.meta` fields as equivalent when comparing manifest-declared model state with current Open WebUI model state.

#### Scenario: Applied model manifest remains unchanged when server adds null profile image
- **WHEN** a `kind: Model` manifest omits `spec.meta.profile_image_url`, `oictl manifests apply` succeeds, and Open WebUI returns the model with `meta.profile_image_url: null`
- **THEN** a subsequent `oictl manifests diff --file model.json` reports the model as unchanged rather than planning an update for `meta`

#### Scenario: Explicit non-null metadata still diffs
- **WHEN** a `kind: Model` manifest declares a non-null optional value under `spec.meta` and the remote model has that field absent or `null`
- **THEN** `oictl manifests diff --file model.json` reports an update for `meta`
