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
The CLI SHALL present help using consistent space-based formatting and neutral descriptions of supported inputs. Help SHALL focus on purpose, syntax, arguments, options, fields, constraints, defaults, omission semantics, and examples. It SHALL omit unsolicited operational or security advice and recommendations to prefer a particular input source. Rendered help SHALL omit implementation provenance and irrelevant Open WebUI source/API references, including source footers, source-code links, upstream revision hashes, Python module paths, class/function locators, and backend routes the operator does not supply. Maintainer evidence SHALL remain in source comments or tests. Necessary API method/path examples for passthrough commands and existing operator-facing schema-inspection commands SHALL remain available.

#### Scenario: Secret-bearing input is described neutrally
- **WHEN** the user views the `users create` input reference
- **THEN** it documents the password field and supported input sources
- **AND** it contains no shell-history/process-list warning or recommendation to use a protected file or stdin

#### Scenario: Input help contains no source-research footer
- **WHEN** the user views help for `users create` or any other action
- **THEN** the output contains useful usage/input information without a `Source: Open WebUI ...` footer, upstream commit hash, or Python source reference

#### Scenario: Useful operator-facing references survive cleanup
- **WHEN** the user requests help for a valve update or generic API request
- **THEN** the existing valve-schema inspection recipe or applicable method/path usage remains documented
- **AND** unrelated source-code/provenance references are omitted

### Requirement: Input documentation preserves execution contracts
Adding help and improving informational/option-error dispatch SHALL preserve command names, aliases, flags, positional grammar, supported option values, body loading, local validation/normalization, request methods/paths/bodies, authorization, confirmation requirements, and successful operation output/exit behavior. Existing top-level/family help SHALL remain available. The explicitly corrected help/version routing and contextual unknown/unsupported/missing-value diagnostics SHALL replace their prior erroneous outcomes. Documenting an upstream schema SHALL not introduce client-side validation, coercion, or defaulting.

#### Scenario: Existing non-help behavior remains unchanged
- **WHEN** existing valid and invalid JSON-input invocations are exercised without help flags
- **THEN** they retain their prior requests, local rejection and confirmation requirements, and successful output behavior
- **AND** only the explicitly corrected option-error classification and contextual diagnostics change

#### Scenario: Existing help remains reachable
- **WHEN** the user requests top-level or family help through an already-supported invocation
- **THEN** it retains the relevant command overview and exits zero
- **AND** the family overview points users to action-level input documentation

### Requirement: Informational help covers the complete supported command tree
The CLI SHALL recognize standalone `--help` and `-h` options on every supported command and intermediate group, independently of whether it consumes JSON. Leaf help SHALL describe the selected action, usage, required positional arguments, relevant options, and existing JSON-input reference where applicable. Group help SHALL list its supported child actions and show how to request their help. Both forms SHALL work with selectors omitted or supplied in their actual command positions, print to stdout, and exit zero. Existing top-level version spellings with appended help SHALL retain version output rather than become action-help commands. Existing top-level and already-supported positional help spellings SHALL remain available; arbitrary command argument values SHALL not become new help aliases.

#### Scenario: Non-JSON leaf help is useful
- **WHEN** the user runs `oictl users get --help` or `oictl users get user-a -h`
- **THEN** the CLI shows the get-user action's usage including its user-ID argument and relevant options
- **AND** it exits zero rather than reporting a missing value or executing the lookup

#### Scenario: Intermediate group help lists local actions
- **WHEN** the user runs `oictl config terminal-servers --help` or `oictl channels messages -h`
- **THEN** the CLI shows the selected group's children and local usage without requiring a deeper action or selectors

#### Scenario: JSON help retains its input detail
- **WHEN** the user requests help for an existing JSON-input command
- **THEN** its fields, types, defaults, constraints, and examples remain available through the same help flags

#### Scenario: Unknown command is not successful informational discovery
- **WHEN** a help request contains an unsupported command or subcommand rather than a known command path
- **THEN** the CLI exits nonzero with the unknown command identified and the nearest supported usage
- **AND** it performs no operation

### Requirement: Top-level version spellings bypass action option validation
The CLI SHALL handle top-level `version`, `--version`, and `-v` as equivalent requests for its version, writing the same version line to stdout and exiting zero without requiring an option value or connection settings. Version recognition SHALL occur only in the top-level informational-command position, not by scanning arbitrary argument values for version-like strings.

#### Scenario: Long version option works
- **WHEN** the user runs `oictl --version`
- **THEN** the CLI prints the same version line as `oictl version` and `oictl -v`
- **AND** it does not report `--version requires a value`

#### Scenario: Connection options do not make version remote
- **WHEN** the user supplies syntactically complete global options alongside a top-level version request
- **THEN** the CLI prints the version without resolving configuration, contacting a server, or writing an output file

### Requirement: Informational requests respect argument boundaries and remain inert
Recognized help/version requests SHALL be handled before operation admission, target/profile resolution, file/directory reads, stdin consumption, output-file creation, or network access. Option values, including `--help`, `-h`, `--version`, `-v`, or `help`, SHALL remain values where the selected action's option grammar consumes them, except for the explicitly corrected tool-export boolean-manifest boundary. Valid boolean options SHALL remain valueless when used without an explicit boolean value. A missing-valued option followed by a help-looking token that occupies its value position SHALL not be reinterpreted as an informational request.

#### Scenario: Overloaded manifest option follows the selected action
- **WHEN** the user runs `oictl tools export --manifest -h` or `oictl tools export --manifest --help`
- **THEN** the CLI treats `--manifest` as the export boolean and prints inert export help
- **WHEN** the user runs `oictl skills create --manifest -h`
- **THEN** `-h` remains the manifest path value, not a help request
- **AND** a terminal `skills create --manifest` remains a nonzero missing-path error with relevant help

#### Scenario: Local mutation help leaves local state unchanged
- **WHEN** the user runs `oictl profiles delete example --help`
- **THEN** help is printed without reading, changing, or deleting stored profiles

#### Scenario: Destructive remote help performs no request
- **WHEN** the user requests help for a delete/sync action without confirmation, with nonexistent input paths or an output destination
- **THEN** the CLI prints help without admitting or executing the action, opening inputs, consuming stdin, or writing that destination

#### Scenario: Help and version strings inside input remain data
- **WHEN** a supported value-bearing option receives a help/version-like string as its value
- **THEN** ordinary value validation/execution semantics apply unless a separate standalone help flag is present
- **AND** the CLI does not report successful help/version in place of the requested operation

### Requirement: Option errors explain the actual invocation defect
For an ordinary invocation, the CLI SHALL distinguish a known value-bearing option missing its value from an unknown option and an option unsupported by the selected action. All three SHALL exit nonzero, identify the offending option without echoing secret values, and give relevant usage or a directly usable help invocation. The CLI SHALL not claim that an unknown or unsupported option merely needs a value. Genuine missing values SHALL remain errors; adding help SHALL not supply guessed defaults, perform an operation, or convert malformed input into success. Existing valid option values and boolean behavior SHALL be preserved.

#### Scenario: A real missing value stays an error with context
- **WHEN** the user runs `oictl --profile` or `oictl users create --data`
- **THEN** the CLI reports the missing value, exits nonzero, and shows top-level or action-specific usage/help respectively
- **AND** it performs no request or local mutation

#### Scenario: Unknown option is classified before value arity
- **WHEN** the user appends an unknown option such as `--unknown-option` to a supported action, with or without a following value
- **THEN** the CLI reports an unknown option and relevant help, not that the option requires a value

#### Scenario: Known but unsupported option is rejected accurately
- **WHEN** the user supplies `--data` to `users list`, or an unsupported `--dry-run` to `users delete`
- **THEN** the CLI reports that the option is unsupported by the selected action and exits nonzero before any request
- **AND** a supplied false boolean does not bypass the unsupported-option check
