## Context

oictl currently implements resource-first command families for Open WebUI domains such as models, config, chats, automations, and providers. Tools and functions exist as Open WebUI domain nouns but are not yet exposed as implemented command families in oictl.

Open WebUI tools and functions can define two valve scopes: global `Valves` and per-user `UserValves`. The server exposes separate endpoints for current values, JSON schema specs, and updates for each scope. Those endpoints validate payloads against plugin-defined Pydantic models and enforce authorization based on the resource type, scope, and current user.

## Goals / Non-Goals

**Goals:**

- Add typed valve commands beneath the `tools` and `functions` resource families.
- Support reading current valve values, reading valve specs, and updating valve values for both global and user scopes.
- Reuse oictl's existing request, payload, output, authentication, and error handling patterns.
- Keep Open WebUI as the source of truth for valve schema resolution, payload validation, and authorization.

**Non-Goals:**

- Implement full tool or function lifecycle management beyond valve operations.
- Generate local forms or validate valve payloads against JSON schema in the CLI.
- Add declarative manifest support for valves in this change.
- Add pipeline valve operations; this change is limited to tools and functions.

## Decisions

1. Use nested resource commands: `oictl tools valves ...` and `oictl functions valves ...`.

   This keeps valve operations under the resource family that owns the plugin instance and avoids a top-level `valves` command that would need extra resource-type flags. User-scoped valves are nested as `valves user ...` to distinguish scope without creating a separate `user-valves` resource family.

2. Use `get`, `spec`, and `update` operations for each scope.

   `get` retrieves current values, `spec` retrieves the server-generated JSON schema, and `update` submits a JSON payload. This mirrors Open WebUI's endpoint model while staying concise and scriptable.

3. Forward payloads unchanged with `--data`, `--file`, or `--file -`.

   Valve payloads can vary per plugin. The CLI should not define static structs or strip fields locally; Open WebUI should perform model validation and return the accepted value set or error.

4. Share implementation shape between tools and functions.

   Both resources have the same valve endpoint suffixes. A small helper can construct `/api/v1/<resource>/id/<id>/valves...` paths and dispatch methods while each top-level command family remains independently discoverable in help.

## Risks / Trade-offs

- [Risk] Tools and functions will initially expose only valve subcommands, which may feel incomplete as command families. -> Mitigation: Help text should describe the limited scope clearly, and future lifecycle proposals can extend the same families.
- [Risk] Plugin-defined valve schemas are dynamic and may vary by user. -> Mitigation: Always request specs from Open WebUI at execution time and do not cache schema responses in the CLI.
- [Risk] Server endpoints have slightly different authorization rules for tools and functions. -> Mitigation: Do not duplicate permission logic locally; rely on server responses and preserve status/body diagnostics.
- [Risk] Update commands may send invalid or partial JSON payloads. -> Mitigation: Require a body for update operations and surface Open WebUI validation errors unchanged.
