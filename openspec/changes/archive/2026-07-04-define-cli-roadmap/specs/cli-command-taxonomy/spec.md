## ADDED Requirements

### Requirement: Commands follow a resource-first taxonomy
Future resource operations SHALL use the form `oictl <resource> <verb>` unless the command is a reserved global/session command.

#### Scenario: Resource command is proposed
- **WHEN** a future proposal adds a command that operates on models
- **THEN** the command is expressed as a resource-first command such as `oictl models list` rather than a verb-first command such as `oictl list models`

#### Scenario: Resource-specific workflow is proposed
- **WHEN** a future proposal adds a workflow for knowledge collections
- **THEN** the command remains grouped under the `knowledge` resource family

### Requirement: Reserved global/session commands remain top-level
The taxonomy SHALL reserve top-level command space for CLI/session concerns and accepted foundation workflow families, including `help`, `version`, `completion`, `profiles`, `api`, `auth`, and `manifests`.

#### Scenario: Session command is proposed
- **WHEN** a future proposal adds authentication setup for the active Open WebUI instance
- **THEN** the command remains grouped under `oictl auth`, such as `oictl auth login`

#### Scenario: Resource command conflicts with reserved command
- **WHEN** a future proposal adds an Open WebUI resource command whose name conflicts with a reserved global/session command
- **THEN** the proposal selects a different resource family name or explicitly modifies this taxonomy specification

### Requirement: Resource families align with Open WebUI domain nouns
The taxonomy SHALL prefer Open WebUI domain nouns for resource families, including implemented families such as `models`, `config`, `files`, `knowledge`, `chats`, `analytics`, `automations`, `scim`, and `providers`, and future families such as `users`, `groups`, `prompts`, `tools`, `functions`, `pipelines`, `channels`, and `terminals`.

#### Scenario: Existing Open WebUI domain is exposed
- **WHEN** a future proposal exposes operations for Open WebUI prompts
- **THEN** it uses the `prompts` command family unless the proposal modifies this taxonomy specification

#### Scenario: Open WebUI configuration is exposed
- **WHEN** a future proposal extends Open WebUI configuration operations
- **THEN** it preserves the implemented singular `config` command family rather than introducing `configs`

#### Scenario: New Open WebUI domain is exposed
- **WHEN** a future proposal exposes an Open WebUI domain not listed in this taxonomy
- **THEN** it chooses the resource family name from the upstream Open WebUI domain noun where practical

### Requirement: Common verbs have consistent meanings
The taxonomy SHALL use common verbs consistently: `list` enumerates resources, `get` retrieves one resource, `create` creates one resource, `update` modifies one resource, `delete` removes one resource, `import` performs additive bulk input, `export` writes remote state to local output, `diff` previews declarative drift, `apply` creates or updates declared resources without pruning omitted remote resources, and `sync` reconciles remote state to a declared local source.

#### Scenario: Additive bulk operation is proposed
- **WHEN** a future proposal adds a bulk operation that creates or updates resources without deleting omitted remote resources
- **THEN** the command uses `import` rather than `sync`

#### Scenario: Declarative reconciliation is proposed
- **WHEN** a future proposal adds a bulk operation that may delete remote resources omitted from the local source
- **THEN** the command uses `sync` and documents the destructive reconciliation behavior

#### Scenario: Declarative non-pruning update is proposed
- **WHEN** a future proposal adds a bulk operation that creates or updates declared resources without pruning omitted remote resources
- **THEN** the command uses `apply` and exposes a non-mutating `diff` or dry-run mode where practical

### Requirement: Naming remains shell-friendly and stable
Command names, resource names, verbs, flags, and stage labels SHALL use lowercase ASCII words with hyphens for multi-word terms, and SHALL avoid aliases as primary documented names.

#### Scenario: Multi-word command concept is proposed
- **WHEN** a future proposal adds a command concept such as API keys or health gates
- **THEN** the documented command or flag name uses lowercase hyphenated spelling such as `api-keys` or `health-gates`

#### Scenario: Alias is proposed
- **WHEN** a future proposal adds a shorthand alias for a command
- **THEN** the full stable command name remains the primary documented command

### Requirement: Stage alignment is documented for command families
Each future proposal that adds command families SHALL identify the roadmap stage that owns the family or workflow.

#### Scenario: Command family is added
- **WHEN** a future proposal adds the `models` command family
- **THEN** the proposal identifies its roadmap stage, such as Stage 1 Foundation Controls

#### Scenario: Implemented command families are classified
- **WHEN** a contributor reviews implemented command families
- **THEN** `api`, `auth`, `profiles`, `models`, `config`, `files`, and `knowledge` align with Stage 1, `manifests` aligns with Stage 2, and `chats`, `analytics`, `automations`, `scim`, and `providers` align with Stage 3

#### Scenario: Command family spans stages
- **WHEN** a future proposal extends an existing command family with a later-stage workflow
- **THEN** the proposal identifies both the existing family and the later-stage workflow scope
