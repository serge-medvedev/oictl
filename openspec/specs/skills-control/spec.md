# skills-control Specification

## Purpose
TBD - created by archiving change add-skills-control. Update Purpose after archive.
## Requirements
### Requirement: Skills command family
The CLI SHALL expose `oictl skills` as the command family for Open WebUI Skill resources.

#### Scenario: Help lists skills commands
- **WHEN** the user runs `oictl skills --help`
- **THEN** the help output lists skill list, get, create, update, access-update, toggle, delete, and export operations

### Requirement: Skills listing and search
The CLI SHALL provide a command to list and search Skills accessible to the authenticated user.

#### Scenario: Skills are listed
- **WHEN** the user runs `oictl skills list`
- **THEN** the CLI requests the Open WebUI Skills list endpoint and displays the returned Skills

#### Scenario: Skills are searched
- **WHEN** the user runs `oictl skills list --query review --page 2 --view-option shared`
- **THEN** the CLI sends the query, page, and view option filters to Open WebUI and displays the returned matching Skills

### Requirement: Skill inspection
The CLI SHALL provide a command to inspect one Skill by Skill identifier.

#### Scenario: Skill is retrieved
- **WHEN** the user runs `oictl skills get <skill-id>`
- **THEN** the CLI requests that Skill from Open WebUI and prints the response including access metadata returned by the server

### Requirement: Skill lifecycle mutation
The CLI SHALL provide commands to create, update, toggle, and delete Skills through Open WebUI.

#### Scenario: Skill is created from JSON
- **WHEN** the user runs `oictl skills create --file skill.json`
- **THEN** the CLI submits the JSON Skill payload to Open WebUI and prints the created Skill response

#### Scenario: Skill is updated from JSON
- **WHEN** the user runs `oictl skills update <skill-id> --file skill.json`
- **THEN** the CLI submits the JSON Skill payload to Open WebUI for the specified Skill and prints the updated Skill response

#### Scenario: Skill active state is toggled
- **WHEN** the user runs `oictl skills toggle <skill-id>`
- **THEN** the CLI requests the server toggle operation and prints the resulting Skill state

#### Scenario: Skill is deleted with confirmation
- **WHEN** the user runs `oictl skills delete <skill-id> --yes`
- **THEN** the CLI requests deletion of that Skill and reports the server result

#### Scenario: Skill delete omits confirmation
- **WHEN** the user runs `oictl skills delete <skill-id>` in a non-interactive context without `--yes`
- **THEN** the CLI exits non-zero before contacting Open WebUI and explains that confirmation is required

### Requirement: Skill access grant updates
The CLI SHALL provide a command to replace access grants for one Skill using Open WebUI's Skill access update API.

#### Scenario: Skill access grants are updated
- **WHEN** the user runs `oictl skills access-update <skill-id> --file access.json`
- **THEN** the CLI submits the access grant payload for that Skill and prints the Skill returned by Open WebUI

#### Scenario: Server filters unsupported grants
- **WHEN** Open WebUI accepts the access update but removes grants disallowed by sharing policy
- **THEN** the CLI prints the server-returned Skill so the accepted access grants are visible

### Requirement: Skills export
The CLI SHALL provide a command to export Skills returned by Open WebUI.

#### Scenario: Accessible Skills are exported as JSON
- **WHEN** the user runs `oictl skills export --out skills.json`
- **THEN** the CLI writes the JSON export response from Open WebUI to the requested file path

#### Scenario: One Skill is exported as JSON
- **WHEN** the user runs `oictl skills export <skill-id>`
- **THEN** the CLI retrieves that Skill and prints the JSON Skill response

### Requirement: Markdown Skill manifest support
The CLI SHALL support a bounded Markdown Skill manifest format for human-authored Skill create, update, and single-Skill export workflows.

#### Scenario: Skill is created from Markdown manifest
- **WHEN** the user runs `oictl skills create --manifest skill.md`
- **THEN** the CLI converts supported frontmatter and Markdown content into the native JSON Skill payload and submits it to Open WebUI

#### Scenario: Skill is updated from Markdown manifest
- **WHEN** the user runs `oictl skills update <skill-id> --manifest skill.md`
- **THEN** the CLI converts the manifest into a JSON Skill payload for the specified Skill and submits it to Open WebUI

#### Scenario: Skill is exported as Markdown manifest
- **WHEN** the user runs `oictl skills export <skill-id> --format manifest --out skill.md`
- **THEN** the CLI writes a Markdown file with supported frontmatter metadata and the Skill content body

#### Scenario: Complex Skill fields require JSON
- **WHEN** a user needs to set complex fields not supported by the bounded Markdown manifest format
- **THEN** the CLI requires the user to use JSON payload input rather than accepting an ambiguous manifest conversion

### Requirement: Skills authorization and server validation
The CLI SHALL rely on Open WebUI for Skill authorization, workspace permission, import/export permission, sharing policy, public-sharing policy, and content validation decisions.

#### Scenario: Skill action is unauthorized
- **WHEN** Open WebUI returns 401 or 403 for a Skills command
- **THEN** the CLI exits non-zero and prints the response status and detail without leaking credentials

#### Scenario: Skill payload is invalid
- **WHEN** Open WebUI rejects a Skill create or update payload with a validation error
- **THEN** the CLI exits non-zero and prints the server validation response

