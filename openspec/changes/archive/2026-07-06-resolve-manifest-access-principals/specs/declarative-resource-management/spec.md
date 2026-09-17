## ADDED Requirements

### Requirement: Manifest workflows resolve access grant principals before planning
The CLI SHALL resolve reference-based access grant principals for all loaded manifests before producing a diff, dry-run apply, apply, or sync plan.

#### Scenario: Diff resolves grant principal references
- **WHEN** the user runs `oictl manifests diff --file resource.json` and the manifest declares an access grant by user email or group name
- **THEN** the CLI resolves the grant principal to an Open WebUI principal ID before comparing desired state with current resource state

#### Scenario: Apply dry run resolves without mutation
- **WHEN** the user runs `oictl manifests apply --dry-run --file resource.json` and the manifest declares an access grant by principal reference
- **THEN** the CLI resolves the principal and prints the planned action without sending create, update, delete, or grant update requests

#### Scenario: Apply sends resolved grant IDs
- **WHEN** the user runs `oictl manifests apply --file resource.json` and the manifest declares an access grant by principal reference
- **THEN** the CLI sends the grant update request with the resolved `principal_id` value

#### Scenario: Sync resolves before destructive confirmation
- **WHEN** the user runs `oictl manifests sync --scope prompts --file manifests/` and the manifests declare access grants by principal reference
- **THEN** the CLI resolves the grant principals before printing the sync plan and before requiring destructive action confirmation

#### Scenario: Resolution failure prevents mutation
- **WHEN** the user runs `oictl manifests apply --file resource.json` and an access grant principal reference cannot be resolved uniquely
- **THEN** the CLI exits non-zero before sending create, update, delete, or grant update requests
