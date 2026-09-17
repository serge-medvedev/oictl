## ADDED Requirements

### Requirement: Task command attaches skills to the configured task model
The CLI SHALL provide `oictl tasks skills attach [--external] <skill-id-or-name>...` to attach existing skills to the configured Open WebUI task model.

#### Scenario: Skills are attached to the default task model
- **WHEN** the user runs `oictl tasks skills attach review-helper ops-helper`
- **THEN** the CLI fetches task config, resolves `TASK_MODEL` to a model, resolves each skill argument by exact ID or unique exact name, and posts an updated model payload whose `meta.skillIds` includes the resolved skill IDs

#### Scenario: Skills are attached to the external task model
- **WHEN** the user runs `oictl tasks skills attach --external review-helper`
- **THEN** the CLI resolves `TASK_MODEL_EXTERNAL` from task config and attaches the resolved skill ID to that model's `meta.skillIds`

### Requirement: Skill attachment is idempotent
The CLI SHALL preserve existing model metadata and SHALL NOT duplicate skill IDs already present in `meta.skillIds`.

#### Scenario: Skill is already attached
- **WHEN** the resolved task model already has `meta.skillIds` containing the requested skill ID
- **THEN** the CLI leaves that ID present once and does not add a duplicate entry

#### Scenario: New skills are appended
- **WHEN** the resolved task model has existing `meta.skillIds` and the user attaches additional skills
- **THEN** the CLI preserves existing skill ID order and appends newly resolved IDs in argument order

### Requirement: Task skill attachment validates resources before mutation
The CLI SHALL reject missing task model config, unresolved task models, unknown skill references, ambiguous skill names, and malformed existing `meta.skillIds` before submitting a model update.

#### Scenario: Task model config is missing
- **WHEN** the selected task config field is empty
- **THEN** the CLI exits non-zero with a clear error and does not submit a model update

#### Scenario: Skill name is ambiguous
- **WHEN** a skill argument matches more than one skill by exact name and does not match a skill ID
- **THEN** the CLI exits non-zero with an ambiguity error and does not submit a model update

#### Scenario: Existing skill IDs are malformed
- **WHEN** the resolved model has `meta.skillIds` that is not an array of strings
- **THEN** the CLI exits non-zero with a validation error and does not submit a model update
