## MODIFIED Requirements

### Requirement: Resource families align with Open WebUI domain nouns
The taxonomy SHALL prefer Open WebUI domain nouns for resource families, including implemented families such as `models`, `config`, `files`, `knowledge`, `chats`, `analytics`, `automations`, `scim`, `providers`, `tools`, and `functions`, and future families such as `users`, `groups`, `prompts`, `pipelines`, `channels`, and `terminals`.

#### Scenario: Existing Open WebUI domain is exposed
- **WHEN** a future proposal exposes operations for Open WebUI prompts
- **THEN** it uses the `prompts` command family unless the proposal modifies this taxonomy specification

#### Scenario: Open WebUI configuration is exposed
- **WHEN** a future proposal extends Open WebUI configuration operations
- **THEN** it preserves the implemented singular `config` command family rather than introducing `configs`

#### Scenario: New Open WebUI domain is exposed
- **WHEN** a future proposal exposes an Open WebUI domain not listed in this taxonomy
- **THEN** it chooses the resource family name from the upstream Open WebUI domain noun where practical

#### Scenario: Tool valve workflow is exposed
- **WHEN** a proposal exposes valve operations for Open WebUI tools
- **THEN** the commands remain nested under the `tools` resource family, such as `oictl tools valves get <tool-id>`

#### Scenario: Function valve workflow is exposed
- **WHEN** a proposal exposes valve operations for Open WebUI functions
- **THEN** the commands remain nested under the `functions` resource family, such as `oictl functions valves get <function-id>`

### Requirement: Stage alignment is documented for command families
Each future proposal that adds command families SHALL identify the roadmap stage that owns the family or workflow.

#### Scenario: Command family is added
- **WHEN** a future proposal adds the `models` command family
- **THEN** the proposal identifies its roadmap stage, such as Stage 1 Foundation Controls

#### Scenario: Implemented command families are classified
- **WHEN** a contributor reviews implemented command families
- **THEN** `api`, `auth`, `profiles`, `models`, `config`, `files`, and `knowledge` align with Stage 1, `manifests` aligns with Stage 2, and `chats`, `analytics`, `automations`, `scim`, `providers`, `tools`, and `functions` align with Stage 3

#### Scenario: Command family spans stages
- **WHEN** a future proposal extends an existing command family with a later-stage workflow
- **THEN** the proposal identifies both the existing family and the later-stage workflow scope
