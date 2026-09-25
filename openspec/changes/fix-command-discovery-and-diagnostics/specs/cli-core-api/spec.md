## ADDED Requirements

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

## MODIFIED Requirements

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
