## ADDED Requirements

### Requirement: Task configuration inspection
The CLI SHALL provide a command to inspect Open WebUI task configuration through the tasks API.

#### Scenario: Task configuration is read
- **WHEN** the user runs `oictl tasks config get`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/tasks/config`
- **AND** the CLI prints the returned task configuration payload

#### Scenario: Task model fields are included in task config output
- **WHEN** Open WebUI returns task model settings in the task configuration payload
- **THEN** the CLI preserves fields such as `TASK_MODEL` and `TASK_MODEL_EXTERNAL` in its output

### Requirement: Task configuration update
The CLI SHALL provide a command to update Open WebUI task configuration through the tasks API.

#### Scenario: Task configuration is updated
- **WHEN** the user runs `oictl tasks config set --file tasks-config.json`
- **THEN** the CLI sends an authenticated `POST` request to `/api/v1/tasks/config/update` with the JSON file as the request body
- **AND** the CLI prints the updated task configuration payload returned by Open WebUI

#### Scenario: Task model configuration is updated as task config
- **WHEN** the request body includes `TASK_MODEL` or `TASK_MODEL_EXTERNAL`
- **THEN** the CLI sends those fields as part of the task configuration payload
- **AND** the change corresponds to Open WebUI config keys `task.model.default` and `task.model.external`

### Requirement: Task model configuration resource boundary
The CLI SHALL expose task model selection as task configuration and SHALL NOT introduce a separate task-model attachment resource for this change.

#### Scenario: Separate task model resource is not added
- **WHEN** task model configuration support is implemented
- **THEN** the documented command surface uses `oictl tasks config get` and `oictl tasks config set`
- **AND** it does not add `oictl task-model`, `oictl task-models`, or task-model attachment commands
