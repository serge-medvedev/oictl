## Purpose
Define shared CLI dispatch, target resolution, profiles, HTTP request execution, input loading, and output behavior for Open WebUI control commands.

## Requirements

### Requirement: Domain command dispatch
The CLI SHALL dispatch commands using a domain/action command tree and SHALL keep existing `help` and `version` behavior available.

#### Scenario: Existing help remains available
- **WHEN** the user runs `oictl help` or `oictl --help`
- **THEN** the CLI exits with status 0 and prints top-level usage including Stage 1 domains

#### Scenario: Unknown command fails predictably
- **WHEN** the user runs an unknown command
- **THEN** the CLI exits non-zero and prints the unknown command plus relevant usage to stderr

### Requirement: Command-specific flag validation
The CLI SHALL reject unsupported flags for the selected action before discovery or mutation. Unsupported `--dry-run` SHALL fail even when supplied with a false value. Confirmation booleans SHALL use strict boolean parsing; malformed values SHALL fail locally.

#### Scenario: Unsupported safety intent is rejected
- **WHEN** the user runs `oictl users delete user-1 --yes --dry-run`
- **THEN** the CLI exits non-zero without sending any request and names the unsupported flag

### Requirement: Target resolution
The CLI SHALL resolve the Open WebUI target from explicit global flags first, environment variables second, and the selected named profile third.

#### Scenario: Explicit target overrides profile
- **WHEN** a command is run with `--base-url` and `--token` while a profile is selected
- **THEN** the request uses the explicit base URL and token values

#### Scenario: Missing target is rejected
- **WHEN** a command that requires Open WebUI access has no resolvable base URL
- **THEN** the CLI exits non-zero and explains how to provide a base URL

### Requirement: Profile storage
The CLI SHALL support named profiles containing at least a base URL and optional bearer token, stored under the user's config directory with restrictive permissions when secrets are present.

#### Scenario: Profile is used for API command
- **WHEN** the user runs an API command with `--profile prod`
- **THEN** the CLI loads the `prod` profile and uses its target settings for the request

#### Scenario: Token is not exposed
- **WHEN** the CLI prints profile information or an error
- **THEN** stored bearer token values are not printed in full

### Requirement: Authenticated HTTP requests
The CLI SHALL send HTTP requests to the resolved Open WebUI base URL, include bearer authentication when a token is available, enforce a timeout, and support JSON, multipart, and raw response bodies.

#### Scenario: Bearer token is sent
- **WHEN** a token is resolved for an API command
- **THEN** the CLI sends an `Authorization: Bearer <token>` header

#### Scenario: Server error is surfaced
- **WHEN** Open WebUI returns a non-2xx response
- **THEN** the CLI exits non-zero and prints the HTTP status plus any response detail without leaking tokens

### Requirement: Structured input
The CLI SHALL allow complex request bodies to be supplied as inline JSON or from a JSON file for create, update, import, and configuration commands.

#### Scenario: JSON file input is submitted
- **WHEN** the user runs a mutation command with `--file payload.json`
- **THEN** the CLI reads the file as JSON and submits it as the request body

### Requirement: Output formats
The CLI SHALL support `--output json` for all structured responses and MAY provide compact table output for list-style responses.

#### Scenario: JSON output is requested
- **WHEN** a command returns structured data and the user passes `--output json`
- **THEN** the CLI prints valid JSON representing the response

#### Scenario: Raw output is written
- **WHEN** a content or export command is run with `--out path`
- **THEN** the CLI writes the raw response body to the specified path and does not table-format it

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

### Requirement: JSON input shape is discoverable from each consuming command
The CLI SHALL provide command-specific input documentation through `--help` and `-h` for every implemented command that consumes user-supplied JSON. Coverage SHALL include required and optional bodies, JSON-valued options, JSON file/directory inputs, nested actions, and existing aliases. Documentation SHALL describe the document accepted by the CLI, including any existing shorthand or envelope, rather than confusing response records or transformed wire bodies with user input. Family help SHALL make the action-level help convention discoverable.

#### Scenario: User discovers a creation document
- **WHEN** the user runs `oictl users create --help`
- **THEN** the help describes a JSON object with required string fields `name`, `email`, and `password`, optional nullable string fields `role` and `profile_image_url`, and their documented defaults `"pending"` and `"/user.png"`
- **AND** it contains a syntactically valid example document and the supported input-source syntax

#### Scenario: Optional and ancillary JSON are covered
- **WHEN** the user requests help for a command with an optional JSON body or an ancillary JSON option such as `files upload --metadata`
- **THEN** its help identifies that input's expected shape and whether it is optional
- **AND** ancillary metadata is distinguished from the uploaded file contents

#### Scenario: Local JSON documents are covered
- **WHEN** the user requests help for `manifests diff`, `manifests apply`, or `manifests sync`
- **THEN** the help describes the JSON document envelope and the supported kind-specific `spec` structures, required fields, and conditional creation/update semantics
- **AND** accepted file/directory input and any supported multi-document forms are distinguished from an API request body

### Requirement: Input references describe fixed and variable structure accurately
For a fixed contract, input help SHALL identify root types, field names and types, requiredness, nullability, relevant defaults, meaningful constraints, nested structure, and accepted alternative forms. Omission semantics for update, import, sync, and replacement operations SHALL be described where they affect the supplied document. Known fields SHALL not be hidden behind a generic label such as "JSON payload" or "object". Each distinct supported input shape SHALL have a compact valid JSON example; genuinely free-form parts SHALL be explicitly labeled, and such examples SHALL be labeled illustrative rather than universal contracts.

For runtime-defined inputs, help SHALL describe the known envelope and variable portions and identify their owner. Where an existing command exposes the resource's schema, help SHALL show that command without executing it. Generic endpoint-selecting passthrough help SHALL explain that the method/path or provider endpoint determines the body contract, rather than claiming one fixed schema. Documentation SHALL be grounded in the existing CLI transformations and an identified upstream form/router or local contract; it SHALL distinguish documented defaults from values inserted by the CLI and deployment-specific extensions from standard upstream behavior.

#### Scenario: Nested fields and alternatives are actionable
- **WHEN** a documented input contains nested records, arrays, or an accepted envelope/shorthand alternative
- **THEN** the help describes their fields or element shapes and the relevant requiredness/default semantics
- **AND** examples demonstrate the supported shapes without presenting output-only fields as required input

#### Scenario: Valve fields depend on the selected resource
- **WHEN** the user requests help for `tools valves update` or `functions valves user update`
- **THEN** the help describes a valve-values object whose keys and value types are defined by that tool or function
- **AND** it shows the matching existing `valves spec` or `valves user spec` invocation with an ID placeholder
- **AND** it does not claim a universal set of valve fields

#### Scenario: Passthrough has no fixed resource schema
- **WHEN** the user requests help for `api` or a provider `request` action
- **THEN** the help describes the supported body sources and endpoint-owned body shape
- **AND** any example names the method/path it illustrates without implying it fits all endpoints

#### Scenario: Input has replacement or CLI normalization semantics
- **WHEN** help describes an update or import/sync input that resets omitted values or is normalized before transmission
- **THEN** the help describes the actual omission behavior or accepted pre-normalization input
- **AND** defaults applied by the server are distinguished from defaults inserted by the CLI

### Requirement: Action-level help is offline and inert
For every JSON-input action, recognized `--help` and `-h` flags SHALL print the relevant usage and input reference to stdout and exit zero without requiring IDs, paths, a body, configuration, or credentials. Help SHALL be resolved before required-input/confirmation checks, input-file/directory reads, stdin consumption, target/profile resolution, or HTTP requests, and SHALL not write output files. Help SHALL also work with the action's positional selectors present, including nested actions taking multiple selectors. Help recognition SHALL respect option-value boundaries and SHALL not mistake JSON string content or a flag's value for a help request.

#### Scenario: Nested help requires no execution prerequisites
- **WHEN** the user runs `oictl tools valves user update --help` without a tool ID, body, or target configuration
- **THEN** the CLI prints the user-valve update input reference to stdout and exits zero
- **AND** it performs no input reads, profile resolution, writes, or HTTP requests

#### Scenario: Help does not read requested input or write a response file
- **WHEN** the user requests help for a recognized JSON-input action while also supplying a nonexistent `--file`, an unreadable manifest directory, stdin input, or an `--out` path
- **THEN** help is printed without opening those input/output paths, consuming stdin, or executing the action

#### Scenario: Selectors may be supplied or omitted for help
- **WHEN** a JSON-input action is requested with `--help` or `-h`, with or without its positional selectors
- **THEN** the CLI resolves the same action-specific reference using that action's existing grammar

#### Scenario: A body value is not a help flag
- **WHEN** a non-help invocation supplies JSON containing `"--help"`, `"-h"`, or `"help"`, or supplies such a token as an option's value
- **THEN** normal command parsing and execution rules apply rather than routing to help

### Requirement: Help remains concise reference documentation
The CLI SHALL present input references using consistent space-based formatting and neutral descriptions of supported inline, file, and stdin sources. Help SHALL focus on syntax, fields, constraints, defaults, omission semantics, and examples. It SHALL omit unsolicited operational or security advice and recommendations to prefer a particular input source.

#### Scenario: Secret-bearing input is described neutrally
- **WHEN** the user views the `users create` input reference
- **THEN** it documents the password field and supported input sources
- **AND** it contains no shell-history/process-list warning or recommendation to use a protected file or stdin

### Requirement: Input documentation preserves execution contracts
Adding JSON-input help SHALL preserve existing top-level and family help, command names, aliases, flags, positional grammar, non-help flag validation, body-loading behavior, local validation/normalization, request methods/paths/bodies, authorization, confirmation requirements, output formats, and exit behavior. Documenting an upstream schema SHALL not turn pass-through input into a new client-side validation, coercion, or defaulting layer.

#### Scenario: Existing non-help behavior remains unchanged
- **WHEN** existing valid and invalid JSON-input invocations are exercised without help flags
- **THEN** they retain their prior requests, local rejection behavior, confirmation requirements, and output behavior

#### Scenario: Existing help remains reachable
- **WHEN** the user requests top-level or family help through an already-supported invocation
- **THEN** it retains the relevant command overview and exits zero
- **AND** the family overview points users to action-level input documentation
