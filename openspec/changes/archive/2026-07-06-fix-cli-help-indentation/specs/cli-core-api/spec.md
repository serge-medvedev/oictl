## ADDED Requirements

### Requirement: Help output indentation is consistent for affected command families
The CLI SHALL render help text for `tasks`, `tools`, `users`, `webhooks`, and manifest `sync` content with consistent section indentation and without tab-indented usage or flag rows.

#### Scenario: Command family help uses aligned rows
- **WHEN** the user runs help for `oictl tasks`, `oictl tools`, `oictl users`, or `oictl webhooks`
- **THEN** the help output exits with status 0
- **AND** usage, command, and example rows use the same space-based indentation convention within their section
- **AND** the help output still lists the documented commands for that family

#### Scenario: Manifest sync help rows align with manifest help sections
- **WHEN** the user runs `oictl manifests --help`
- **THEN** the help output exits with status 0
- **AND** the `sync` usage row and sync-specific flags use the same space-based indentation convention as the surrounding manifest help rows
- **AND** the help output still documents sync scope selection and destructive-action confirmation

#### Scenario: Help formatting changes do not alter command behavior
- **WHEN** the affected help text indentation is normalized
- **THEN** existing non-help invocations for `tasks`, `tools`, `users`, `webhooks`, and `manifests sync` keep their command names, flags, arguments, exit codes, request behavior, and response formatting
