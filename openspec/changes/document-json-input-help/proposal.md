## Why

Users can discover that a command accepts JSON but cannot discover the document they need to supply. For example, `users create` advertises pass-through JSON without fields or defaults, and `users create --help` currently fails instead of describing the input.

## What Changes

- Provide action-level `--help` and `-h` for every existing command that consumes JSON, including nested actions, optional request bodies, JSON-valued options, and JSON documents loaded from files/directories.
- Document the actual CLI input shape: root type, fields and nesting, required/optional/nullable values, meaningful defaults and constraints, accepted envelopes/alternatives, and a small valid example.
- Describe genuinely open or resource-dependent inputs honestly: explain the known structure and its variable parts, and name an existing schema-inspection command where available. Generic API/provider passthrough help explains that the selected endpoint owns the body contract.
- Make help available offline without positional IDs, a request body, configuration, credentials, input-file reads, or requests. Preserve existing top-level/family help and all non-help behavior.
- Use concise reference prose and neutral input-source examples. Omit unsolicited operational/security advice and recommendations between inline, file, and stdin input.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `cli-core-api`: Extend shared help requirements with complete JSON-input coverage, usable action-level help, source-grounded input references, and unchanged execution behavior.

## Impact

Expected implementation touches CLI help routing/content, focused help tests, and minimal README discovery guidance. It reuses existing dispatch, body loaders, flag validation, and upstream/local input contracts. There are no endpoint, request, response, configuration, dependency, installation, or deployment changes.

## Anti-overengineering Guardrails

Produce the smallest complete solution that satisfies the accepted task. Additional sophistication belongs in this implementation only when explicitly requested. Every edit must support help discovery, accurate input documentation, coverage tests, or preservation of existing behavior.

Use static text and small private helpers where useful. Do not introduce a CLI framework migration, general command/schema registry, JSON Schema engine, runtime OpenAPI fetching, code generation, schema export commands/flags, new validation/coercion/defaulting, compatibility negotiation, or new dependencies. Completeness means every JSON-input command is covered, not just representative families; genuinely variable input is documented as variable rather than replaced with an invented fixed schema.

## Delivery Boundary

This change's planning deliverable is a complete, validated, independently reviewed, locally committed OpenSpec proposal. Implementation tasks remain unchecked until separately authorized and exercised. Planning does not implement the CLI, synchronize canonical specs, archive the change, or publish commits.
