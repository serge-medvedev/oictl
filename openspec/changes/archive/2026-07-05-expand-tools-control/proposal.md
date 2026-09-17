## Why

Open WebUI tools are already supported by `oictl` declarative manifests, but operators do not have an imperative `tools` command family for direct inspection and lifecycle control. Adding these commands closes that gap and gives scripts a simple resource-first interface for day-to-day tool administration while preserving manifest compatibility.

## What Changes

- Add an imperative `oictl tools` command family for listing, inspecting, creating, updating, deleting, exporting, loading from URL, and updating access grants for Open WebUI tools.
- Support JSON passthrough input for create, update, load-url, and access-update payloads using the existing `--data` and `--file` conventions.
- Support `tools export` output that can be consumed directly by the existing `Tool` declarative manifest workflow where practical.
- Preserve Open WebUI authorization and validation semantics by surfacing server errors without reinterpreting tool execution or sharing policy behavior.
- Do not add tool execution, tool runtime testing, function management, or prompt management in this change.

## Capabilities

### New Capabilities

- `tools-control`: Imperative CLI control of Open WebUI tool records, including list/get/create/update/delete/export/load-url/access-update and manifest-aligned export behavior.

### Modified Capabilities

None.

## Impact

- Affected code: `internal/cli` command dispatch, help output, shared JSON input/output paths, and command tests.
- Affected APIs: Open WebUI tool endpoints under `/api/v1/tools`, including list, get by id, create, update by id, delete by id, access update, export, and URL loading endpoints where exposed by the server.
- Affected user interface: New `oictl tools` resource family with resource-first subcommands and existing global flags.
- Dependencies: No new dependency is expected; implementation should reuse the existing internal API client, JSON input loader, output handling, and manifest resource helpers where appropriate.
