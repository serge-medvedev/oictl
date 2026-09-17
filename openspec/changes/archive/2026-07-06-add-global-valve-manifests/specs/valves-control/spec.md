## ADDED Requirements

### Requirement: Declarative workflows use global valve endpoints
Declarative `ToolValve` and `FunctionValve` manifest workflows SHALL inspect and update only global valve values through the existing Open WebUI global valve endpoints for the owning resource family, using the manifest `metadata.name` value as the owner ID path segment.

#### Scenario: Tool valve manifest reads global valves
- **WHEN** a manifest workflow plans `ToolValve/weather_tool`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/tools/id/weather_tool/valves`

#### Scenario: Tool valve manifest updates global valves
- **WHEN** a manifest apply updates `ToolValve/weather_tool`
- **THEN** the CLI sends the manifest `spec` object in an authenticated `POST` request to `/api/v1/tools/id/weather_tool/valves/update`

#### Scenario: Function valve manifest reads global valves
- **WHEN** a manifest workflow plans `FunctionValve/fn_a`
- **THEN** the CLI sends an authenticated `GET` request to `/api/v1/functions/id/fn_a/valves`

#### Scenario: Function valve manifest updates global valves
- **WHEN** a manifest apply updates `FunctionValve/fn_a`
- **THEN** the CLI sends the manifest `spec` object in an authenticated `POST` request to `/api/v1/functions/id/fn_a/valves/update`

### Requirement: Declarative valve workflows preserve server validation
Declarative valve workflows SHALL rely on Open WebUI for owner existence checks, authorization, schema validation, defaulting, and accepted valve value normalization.

#### Scenario: Server rejects declarative valve update
- **WHEN** Open WebUI returns a validation error for a `ToolValve` or `FunctionValve` manifest apply
- **THEN** the CLI exits non-zero and prints the response status and body to stderr

#### Scenario: Server returns normalized valve values
- **WHEN** Open WebUI accepts a declarative valve update and returns normalized global valve values
- **THEN** the CLI treats the returned values as the post-apply resource state without locally synthesizing defaults

### Requirement: Declarative valve workflows exclude user-scoped valves
Declarative valve workflows SHALL NOT read from or write to user-scoped valve endpoints.

#### Scenario: Tool valve manifest does not use user endpoint
- **WHEN** a manifest workflow plans or applies `ToolValve/weather_tool`
- **THEN** the CLI does not send requests to `/api/v1/tools/id/weather_tool/valves/user` or `/api/v1/tools/id/weather_tool/valves/user/update`

#### Scenario: Function valve manifest does not use user endpoint
- **WHEN** a manifest workflow plans or applies `FunctionValve/fn_a`
- **THEN** the CLI does not send requests to `/api/v1/functions/id/fn_a/valves/user` or `/api/v1/functions/id/fn_a/valves/user/update`
