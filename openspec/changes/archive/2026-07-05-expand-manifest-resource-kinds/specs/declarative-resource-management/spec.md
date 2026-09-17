## ADDED Requirements

### Requirement: Manifests support additional workspace resource kinds
The CLI SHALL support `Model`, `Skill`, `Function`, `Group`, and `Channel` manifests in addition to `Knowledge`, `Prompt`, and `Tool`.

#### Scenario: Model manifest is planned
- **WHEN** the user runs `oictl manifests diff --file model.json` and the manifest declares `kind: Model`
- **THEN** the CLI validates the manifest and plans create, update, replace-grants, unchanged, or delete actions using the Open WebUI model resource state

#### Scenario: Skill manifest is planned
- **WHEN** the user runs `oictl manifests diff --file skill.json` and the manifest declares `kind: Skill`
- **THEN** the CLI validates the manifest and plans create, update, replace-grants, unchanged, or delete actions using the Open WebUI skill resource state

#### Scenario: Function manifest is planned
- **WHEN** the user runs `oictl manifests diff --file function.json` and the manifest declares `kind: Function`
- **THEN** the CLI validates the manifest and plans create, update, unchanged, or delete actions using the Open WebUI function resource state

#### Scenario: Group manifest is planned
- **WHEN** the user runs `oictl manifests diff --file group.json` and the manifest declares `kind: Group`
- **THEN** the CLI validates the manifest and plans create, update, unchanged, or delete actions using the Open WebUI group resource state

#### Scenario: Channel manifest is planned
- **WHEN** the user runs `oictl manifests diff --file channel.json` and the manifest declares `kind: Channel`
- **THEN** the CLI validates the manifest and plans create, update, replace-grants, unchanged, or delete actions using the Open WebUI channel resource state

### Requirement: Sync scopes include all supported manifest kinds
The CLI SHALL accept sync scopes for every supported manifest kind and SHALL include all supported kinds when `--scope all` is selected.

#### Scenario: Model sync scope is selected
- **WHEN** the user runs `oictl manifests sync --scope models --file manifests/ --dry-run`
- **THEN** the CLI reconciles remote `Model` resources against the manifest set without pruning other supported kinds

#### Scenario: All sync scope includes new kinds
- **WHEN** the user runs `oictl manifests sync --scope all --file manifests/ --dry-run`
- **THEN** the CLI includes `Knowledge`, `Prompt`, `Tool`, `Model`, `Skill`, `Function`, `Group`, and `Channel` resources in the sync plan

#### Scenario: Singular sync scope is accepted
- **WHEN** the user runs `oictl manifests sync --scope channel --file manifests/ --dry-run`
- **THEN** the CLI treats the scope as the `Channel` resource kind

### Requirement: Resource-specific manifest handlers preserve existing workflow semantics
The CLI SHALL apply resource-specific Open WebUI endpoint shapes while preserving existing manifest diff, apply, sync, dry-run, JSON output, and duplicate identity behavior.

#### Scenario: Apply does not prune omitted new-kind resources
- **WHEN** the user runs `oictl manifests apply --file model.json` and another remote model is omitted from the manifest set
- **THEN** the CLI leaves the omitted remote model unchanged

#### Scenario: Sync plans omitted new-kind resources for deletion
- **WHEN** the user runs `oictl manifests sync --scope skills --file manifests/ --dry-run` and a remote skill in scope is absent from the manifests
- **THEN** the CLI includes a delete action for that skill in the sync plan

#### Scenario: Duplicate identity is rejected across new kinds
- **WHEN** two input files declare the same `kind` and `metadata.name` for `Model`, `Skill`, `Function`, `Group`, or `Channel`
- **THEN** the CLI exits non-zero before contacting Open WebUI and reports the duplicate manifest identity
