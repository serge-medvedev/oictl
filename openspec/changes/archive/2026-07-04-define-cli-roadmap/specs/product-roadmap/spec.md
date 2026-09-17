## ADDED Requirements

### Requirement: Roadmap defines three product stages
The roadmap SHALL define exactly three product stages for `oictl`: Stage 1 Foundation Controls, Stage 2 Workspace Automation, and Stage 3 Operations Intelligence.

#### Scenario: Reader reviews roadmap stages
- **WHEN** a contributor reads the product roadmap specification
- **THEN** they can identify the three stage names in order: Stage 1 Foundation Controls, Stage 2 Workspace Automation, and Stage 3 Operations Intelligence

### Requirement: Stage 1 focuses on safe core control
Stage 1 Foundation Controls SHALL focus on safe, scriptable access to one Open WebUI instance, including the low-level API escape hatch, target/profile configuration, authentication context, model inventory, instance configuration, files, and knowledge resources.

#### Scenario: Stage 1 scope is evaluated
- **WHEN** a future proposal adds commands for `api`, `profiles`, `auth`, `models`, `config`, `files`, `knowledge`, or status checks
- **THEN** the proposal can classify those commands as Stage 1 Foundation Controls

#### Scenario: Stage 1 excludes advanced workflows
- **WHEN** a future proposal adds declarative multi-resource reconciliation, fleet management, analytics, SCIM provisioning, provider passthrough, or scheduled automation commands
- **THEN** the proposal does not classify those commands as Stage 1 Foundation Controls

### Requirement: Stage 2 focuses on repeatable workspace workflows
Stage 2 Workspace Automation SHALL focus on repeatable Open WebUI workspace workflows, especially declarative desired-state management for workspace resources and access grants through the `manifests` command family.

#### Scenario: Stage 2 scope is evaluated
- **WHEN** a future proposal adds `manifests diff`, `manifests apply`, `manifests sync`, or expands manifest support to prompts, tools, functions, pipelines, channels, automations, or other workspace resources
- **THEN** the proposal can classify those commands as Stage 2 Workspace Automation

#### Scenario: Stage 2 builds on Stage 1
- **WHEN** a future proposal depends on configured profiles or authentication context
- **THEN** the proposal treats those dependencies as Stage 1 prerequisites rather than redefining them in Stage 2

### Requirement: Stage 3 focuses on operations intelligence
Stage 3 Operations Intelligence SHALL focus on operating and integrating Open WebUI across advanced or sensitive surfaces, including chats, analytics and monitoring, automations, SCIM provisioning, provider passthrough, fleet profiles, audit and policy inspection, backup and restore workflows, migration support, terminal/orchestration operations, and CI-friendly health gates.

#### Scenario: Stage 3 scope is evaluated
- **WHEN** a future proposal adds commands for chats, analytics, automations, SCIM, providers, multiple instances, audits, monitoring, backups, restores, migrations, terminal orchestration, or CI health gates
- **THEN** the proposal can classify those commands as Stage 3 Operations Intelligence

#### Scenario: Stage 3 preserves earlier command contracts
- **WHEN** a Stage 3 proposal extends an existing Stage 1 or Stage 2 command family
- **THEN** it preserves the established command taxonomy unless the proposal explicitly modifies the taxonomy specification

### Requirement: Roadmap changes remain specification-only until implemented separately
The roadmap SHALL NOT require runtime CLI behavior changes until a later implementation proposal adds commands, handlers, API clients, or tests for a specific stage.

#### Scenario: Roadmap change is applied
- **WHEN** this roadmap change is accepted
- **THEN** the repository contains roadmap and taxonomy specifications without requiring changes to the executable CLI behavior
