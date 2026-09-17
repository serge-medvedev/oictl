## ADDED Requirements

### Requirement: Model delete reports missing models clearly
The CLI SHALL translate Open WebUI's known misleading missing-model delete response, HTTP `401` with not-found detail from `/api/v1/models/model/delete`, into a clear not-found result for `oictl models delete` without masking ordinary authorization failures.

#### Scenario: Missing model delete reports not found
- **WHEN** the user runs `oictl models delete <model-id> --yes` and Open WebUI returns HTTP `401` with not-found detail from the model delete endpoint
- **THEN** the CLI exits non-zero and reports that the model was not found, including the target model identifier where appropriate

#### Scenario: Real authorization failure remains authorization failure
- **WHEN** the user runs `oictl models delete <model-id> --yes` and Open WebUI returns an authorization failure that does not match the known missing-model response
- **THEN** the CLI exits non-zero and surfaces the authorization status and detail without reclassifying it as not found
