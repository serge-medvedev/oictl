# Declarative Resource Manifests

`oictl manifests` loads JSON and YAML manifests and plans resource changes before applying them to Open WebUI. Manifest files may use `.json`, `.yaml`, or `.yml` extensions.

## Commands

```sh
oictl manifests diff --file prompt.json
oictl manifests apply --directory manifests/
oictl manifests apply --dry-run --file prompt.json --output json
oictl manifests sync --scope prompts --directory manifests/ --yes
oictl manifests sync --scope models --directory manifests/ --yes
oictl manifests sync --scope channels --directory manifests/ --yes
```

`apply` creates or updates declared resources and does not delete omitted resources. `sync` reconciles the selected `--scope` and requires `--yes` when the plan contains deletes. Directory inputs load `.json`, `.yaml`, and `.yml` manifests deterministically.

Supported manifest kinds are `Knowledge`, `Prompt`, `Tool`, `ToolValve`, `Model`, `Skill`, `Function`, `FunctionValve`, `Group`, `Channel`, and `TerminalServerConnection`. Sync scopes accept singular and plural resource names for endpoint-backed owning resources, plus `all`; valve manifests and terminal server connections are not pruned by sync.

## Authoring Values

Manifest string values can reference environment variables with simple `${VAR}` placeholders. Placeholder names must be standard environment identifiers such as `TEAM_NAME`. `oictl` renders placeholders from the process environment after parsing JSON/YAML and before validation, diff, apply, or sync planning. If a referenced variable is unset, the command fails before contacting Open WebUI and reports the manifest path and variable name. Shell-style defaults and other expressions such as `${VAR:-default}` are not supported; set defaults in your shell, CI environment, or wrapper before invoking `oictl`.

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "Prompt",
  "metadata": { "name": "${PROMPT_COMMAND}" },
  "spec": {
    "command": "${PROMPT_COMMAND}",
    "content": "Use ${TEAM_NAME} runbooks."
  },
  "access_grants": [
    { "principal_type": "group", "principal_name": "${TEAM_NAME}", "permission": "read" }
  ]
}
```

`spec.content_file` can be used instead of `spec.content` for resources with large content bodies, including `Prompt`, `Tool`, `Skill`, and `Function`. The path may contain the same simple `${VAR}` placeholders. Relative paths are resolved from the directory containing the manifest file. `oictl` reads the file, assigns its text to `spec.content`, and removes `content_file` before planning or sending API payloads. Declaring both `spec.content` and `spec.content_file` is invalid.

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "Tool",
  "metadata": { "name": "incident_lookup" },
  "spec": {
    "name": "incident_lookup",
    "content_file": "tools/incident_lookup.py"
  },
  "access_grants": []
}
```

For manifest-managed resources that expose `spec.is_active`, active state is reconciled only when `is_active` is present in the manifest. If `spec.is_active` is omitted, `diff` and `apply` preserve the remote active state for existing resources.

Resource discovery follows all reported pages before matching, grant resolution, or pruning. Exact remote IDs take precedence over aliases; ambiguous names and multiple documents resolving to the same resource are rejected. A resolved resource is protected from pruning regardless of the alias used in its manifest.

Tool create forms default omitted `spec.id` to `metadata.name` and omitted `spec.meta` to `{}`. Function create forms also default omitted `spec.meta` to `{}`. Prompt forms use `spec.name` (not `title`); creation defaults it to `metadata.name` when omitted. Omitted fields remain unmanaged when updating existing resources. Required source content and essential field types are validated before mutation.

## Prompt

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "Prompt",
  "metadata": { "name": "summarize" },
  "spec": {
    "command": "summarize",
    "name": "Summarize",
    "content": "Summarize the selected text."
  },
  "access_grants": [
    { "principal_type": "user", "principal_email": "analyst@example.com", "permission": "read" },
    { "principal_type": "user", "principal_id": "*", "permission": "read" }
  ]
}
```

## Knowledge

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "Knowledge",
  "metadata": { "name": "ops-runbooks" },
  "spec": {
    "name": "ops-runbooks",
    "description": "Operational runbooks"
  },
  "access_grants": [
    { "principal_type": "group", "principal_name": "SRE", "permission": "write" }
  ]
}
```

## Tool

Tool manifests can be written directly as JSON/YAML or exported from Open WebUI as JSON with `oictl tools export <tool-id> --manifest --out tool.json` and `oictl tools export --manifest --directory manifests/tools`.

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "Tool",
  "metadata": { "name": "incident_lookup" },
  "spec": {
    "name": "incident_lookup",
    "content": "class Tools:\n    pass\n"
  },
  "access_grants": []
}
```

## Tool Valve

`ToolValve` manifests manage global valve values for an existing tool. `metadata.name` is the Open WebUI tool ID used in `/api/v1/tools/id/{id}/valves`; it is not a display-name lookup. The manifest `spec` object is sent as-is to the global valve update endpoint for both create and update plan actions. Per-user tool valves are intentionally out of scope for declarative manifests; use `oictl tools valves user ...` for imperative user-scoped operations.

Valve manifests are non-owning configuration resources: they do not create, delete, or prune the owning tool, and `sync --scope all` does not reset omitted valve manifests.

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "ToolValve",
  "metadata": { "name": "incident_lookup" },
  "spec": {
    "enabled": true,
    "api_key": "secret"
  }
}
```

## Model

`metadata.name` is the Open WebUI model `id`. If `spec.id` is omitted, `oictl` sets it from `metadata.name`; if it is present, it must match. `spec.name` is required. Omitted `spec.meta` and `spec.params` default to empty objects.

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "Model",
  "metadata": { "name": "llama-ops" },
  "spec": {
    "base_model_id": "llama3.1:8b",
    "name": "Llama Ops",
    "meta": {
      "description": "Operations assistant"
    },
    "params": {
      "temperature": 0.2
    },
    "is_active": true
  },
  "access_grants": [
    { "principal_type": "group", "principal_id": "sre", "permission": "read" }
  ]
}
```

Use `--scope model` or `--scope models` to prune omitted models during sync. `--scope all` includes models together with knowledge, prompts, tools, skills, functions, groups, and channels.

Public access is represented as a `user` grant with `principal_id: "*"`. Private owner-only access is represented as an empty `access_grants` list.

Access grant principals can be authored with exactly one selector: `principal_id`, `principal_ref`, `principal_email`, or `principal_name`. `principal_id` is the raw Open WebUI ID and is used as-is. `principal_email` is valid only for `principal_type: "user"`. `principal_name` resolves exact user or group names within the declared `principal_type`. `principal_ref` resolves exact IDs, emails, or names for users and exact IDs or names for groups; optional `user:` and `group:` prefixes must match `principal_type`.

Reference fields are authoring-only. `oictl manifests diff`, dry-run `apply`, mutating `apply`, and `sync` resolve them to Open WebUI `principal_id` values before planning or updating grants. Resolution failures, ambiguous matches, and type mismatches fail before mutation. Manifests that use only raw `principal_id` values, including the public `*` grant, do not query users or groups.

Examples:

```json
{ "principal_type": "user", "principal_email": "analyst@example.com", "permission": "read" }
{ "principal_type": "user", "principal_name": "Analyst One", "permission": "read" }
{ "principal_type": "group", "principal_name": "SRE", "permission": "write" }
{ "principal_type": "group", "principal_ref": "group:SRE", "permission": "read" }
```

## Skill

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "Skill",
  "metadata": { "name": "review-helper" },
  "spec": {
    "id": "review-helper",
    "name": "Review Helper",
    "content": "Review the selected text for operational risks."
  },
  "access_grants": [
    { "principal_type": "group", "principal_id": "sre", "permission": "write" }
  ]
}
```

## Function

Function manifests manage Open WebUI function records. They do not support `access_grants`. Explicit `spec.is_active` and `spec.is_global` changes use the corresponding dedicated toggles with readback; omitted state remains unmanaged.

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "Function",
  "metadata": { "name": "ops_filter" },
  "spec": {
    "id": "ops_filter",
    "name": "Ops Filter",
    "content": "class Filter:\n    pass\n"
  }
}
```

## Function Valve

`FunctionValve` manifests manage global valve values for an existing function. `metadata.name` is the Open WebUI function ID used in `/api/v1/functions/id/{id}/valves`; it is not a display-name lookup. The manifest `spec` object is sent as-is to the global valve update endpoint for both create and update plan actions. Per-user function valves are intentionally out of scope for declarative manifests; use `oictl functions valves user ...` for imperative user-scoped operations.

Valve manifests do not support `access_grants`, do not create or delete the owning function, and are excluded from sync pruning scopes.

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "FunctionValve",
  "metadata": { "name": "ops_filter" },
  "spec": {
    "threshold": 0.8,
    "mode": "strict"
  }
}
```

## Group

Group manifests manage native Open WebUI groups. They do not support `access_grants`; use group membership commands for users.

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "Group",
  "metadata": { "name": "sre" },
  "spec": {
    "id": "sre",
    "name": "SRE"
  }
}
```

## Channel

Channel manifests reconcile access grants through channel create/update payloads and preserve omitted fields, including `is_private`, `data`, and `meta`. Discovery uses `/api/v1/channels/list`, the server's administrative inventory, rather than the caller's membership inventory.

Open WebUI generates channel IDs and lowercases names. An exact ID in `metadata.name` takes precedence; otherwise lookup requires a unique name match, accounting for lowercasing and the desired `spec.name`. The example below therefore resolves the stored name `ops room` on subsequent applies. To rename an existing channel unambiguously, use its remote ID as `metadata.name`.

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "Channel",
  "metadata": { "name": "ops-room" },
  "spec": {
    "name": "Ops Room",
    "description": "Operational coordination"
  },
  "access_grants": [
    { "principal_type": "user", "principal_id": "*", "permission": "read" }
  ]
}
```

## Terminal Server Connection

Terminal server connection manifests update entries inside `/api/v1/configs/terminal_servers`. `metadata.name` resolves by exact connection `id` first, then unique `name`. Top-level manifest `access_grants` are written to nested `config.access_grants`; omitting `access_grants` preserves existing grants, even when `spec.config` is replaced or set to null. Specs can contain connection secrets such as keys, so store manifests accordingly.

```json
{
  "apiVersion": "oictl.openwebui/v1",
  "kind": "TerminalServerConnection",
  "metadata": { "name": "ops-shell" },
  "spec": {
    "name": "Ops Shell",
    "url": "https://terminal.example.internal",
    "path": "/ws",
    "auth_type": "bearer"
  },
  "access_grants": [
    { "principal_type": "group", "principal_id": "sre", "permission": "read" }
  ]
}
```

`access_grants` are supported for `Knowledge`, `Prompt`, `Tool`, `Model`, `Skill`, `Channel`, and `TerminalServerConnection`. Declaring `access_grants` on `Function`, `FunctionValve`, `Group`, or `ToolValve` manifests fails validation before any remote mutation.
