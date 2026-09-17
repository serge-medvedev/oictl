## Context

The CLI help output is implemented with static text literals in the command dispatch files. Most command families use a simple two-space indentation convention under sections such as `Usage:`, `Commands:`, and `Flags:`, but some newer help blocks contain inconsistent spacing or tab-indented lines.

## Goals / Non-Goals

**Goals:**

- Make the affected help output visually consistent and easy to scan.
- Keep the change limited to presentation text and tests.
- Verify `tasks`, `tools`, `users`, `webhooks`, and manifest `sync` help content still appears.

**Non-Goals:**

- Change command parsing, flags, aliases, exit codes, HTTP requests, or output formats for non-help command execution.
- Introduce a new help rendering framework.
- Reword command semantics beyond what is necessary to align indentation.

## Decisions

- Update existing static help strings in place instead of introducing shared formatting helpers. This is the smallest safe fix because the inconsistency is localized to a handful of literals.
- Treat tab-indented help lines as defects and replace them with the existing two-space convention used elsewhere in the CLI.
- Add focused help-output assertions for the affected command families rather than broad golden files. This keeps tests stable while still guarding against the specific regression.

## Risks / Trade-offs

- Help text assertions can become brittle if they compare entire output blocks. Mitigation: assert representative aligned lines and absence of tab-indented help rows rather than whole-file snapshots.
- Manual string edits can accidentally remove documented commands. Mitigation: keep existing command names and add tests that selected documented commands remain present.
