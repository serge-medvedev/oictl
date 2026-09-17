## Context

Open WebUI stores terminal server connections in the `terminal_server.connections` config row and exposes them through `/api/v1/configs/terminal_servers` as `TERMINAL_SERVER_CONNECTIONS`. Each connection is a config object with identity fields such as `id` and `name`, connection details such as `url`, `path`, `key`, `auth_type`, and a free-form `config` object. Runtime terminal access uses `config.access_grants` inside the connection object to decide which verified users can see and proxy to the connection.

`oictl` already has JSON passthrough get/set commands for terminal server config and shared access grant validation/diff helpers for manifests. The missing layer is a focused operator workflow that reads or replaces grants for one terminal server connection without requiring manual editing of the whole config payload.

## Goals / Non-Goals

**Goals:**

- Add first-class CLI workflows to get, set, and diff access grants for one terminal server connection.
- Use the same `access_grants` tuple model, normalization, validation, and diff semantics already used by manifest-managed resources.
- Preserve all non-target terminal server connections and all non-grant fields on the target connection when replacing grants.
- Optionally allow terminal server connections to be managed through manifests as config-backed resources.
- Keep Open WebUI authorization and sharing behavior authoritative.

**Non-Goals:**

- Do not add new Open WebUI APIs or assume a per-terminal-server access endpoint exists.
- Do not alter terminal server verify, policy, lifecycle, refresh, or proxy behavior.
- Do not migrate legacy `access_control` shapes in `oictl`; the CLI should operate on `config.access_grants` and surface server state as returned.
- Do not make manifest `sync` prune terminal server connections in this change.

## Decisions

1. Address terminal server connections as entries in `TERMINAL_SERVER_CONNECTIONS`.

   The CLI will fetch `/api/v1/configs/terminal_servers`, locate a connection by exact `id` first and then by unique `name`, and mutate the in-memory collection before posting the full config payload back. This matches the Open WebUI config API contract. Alternative considered: invent a synthetic per-connection endpoint in the CLI abstraction, but that would hide the actual update semantics and increase the risk of dropping unrelated config fields.

2. Store grant state at `connection.config.access_grants`.

   Open WebUI's terminal access check reads grants from the nested `config` object, not from the top-level connection. Grant get/set/diff commands should therefore read and replace only `config.access_grants`, creating `config` when needed and preserving all other `config` keys. Alternative considered: use a top-level `access_grants` field for consistency with other resources, but that would not affect terminal access in Open WebUI.

3. Add an `access-grants` subcommand under `config terminal-servers`.

   The proposed shape is `oictl config terminal-servers access-grants get|set|diff <connection-id-or-name>`, with `set` and `diff` accepting `--file` or existing JSON input mechanisms. This keeps grant operations discoverable near the existing terminal server config commands while avoiding overloading the raw `set` operation. Alternative considered: add top-level `terminal-servers access-update`, but terminal servers are currently part of config control, not their own command family.

4. Accept desired grants as either a raw array or an object with `access_grants`.

   A raw array is convenient for focused grant files, while the object form aligns with existing manifest/resource shapes. Both forms normalize to the same internal grant list before validation and diffing. Alternative considered: require only full terminal server config documents, but that would preserve the manual-editing problem this change is meant to solve.

5. Model manifest support as a config-backed resource adapter.

   If implemented, `TerminalServerConnection` manifests will use `metadata.name` to resolve the connection by `id` or unique `name`, `spec` for managed connection fields, and optional top-level `access_grants` that map to nested `spec.config.access_grants` during apply. Apply will create or replace one connection inside the collection and preserve undeclared connections. Alternative considered: require users to manage the entire `TERMINAL_SERVER_CONNECTIONS` array as a config manifest, but that would make grant-only diffs noisy and risky.

## Risks / Trade-offs

- Full config replacement can overwrite concurrent admin changes → Mitigation: fetch immediately before mutation, replace only the target connection in the returned collection, and include tests that unrelated connections and fields are preserved.
- Connection names may not be unique → Mitigation: prefer exact `id` matches and fail clearly on ambiguous name matches.
- Terminal server keys are sensitive and may appear in full config responses → Mitigation: grant commands should output only grants and diff summaries by default; manifest docs should warn users that connection specs may contain secrets.
- Server may normalize, reject, or filter grants differently from the CLI → Mitigation: validate before mutation, then compare returned grants with desired grants and surface server-side differences.
- Config-backed resources do not support safe pruning semantics → Mitigation: exclude terminal server connection manifests from destructive `sync` pruning in this change.

## Migration Plan

No server or data migration is required. Existing terminal server config commands continue to work. Rollback is removing the new CLI workflows; any grants written remain normal Open WebUI terminal server connection config and can still be edited through the raw config API or UI.

## Open Questions

- Should the manifest kind be named `TerminalServerConnection` or a shorter `TerminalServer`? The design assumes `TerminalServerConnection` because the underlying config object represents a connection, not a running terminal server.
