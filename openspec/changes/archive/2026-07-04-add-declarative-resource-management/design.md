## Context

`oictl` is planned as a thin Open WebUI control client with Stage 1 foundations for profiles, authentication, HTTP requests, structured input, output formats, and resource commands. Stage 2 Workspace Automation needs a higher-level workflow for repeatable desired state across resources without making shell scripts manually sequence every create, update, delete, and access update call.

Open WebUI resources do not all share one API shape, but several workspace domains expose stable identifiers, create/update/delete operations, and `access_grants` lists. The upstream access grant model uses `principal_type` values of `user` or `group`, `principal_id` values including `*` for public user access, and `permission` values of `read` or `write`.

## Goals / Non-Goals

**Goals:**

- Add a manifest command family for `oictl manifests diff`, `oictl manifests apply`, and `oictl manifests sync`.
- Define a versioned manifest envelope that can represent multiple Open WebUI resource kinds while preserving resource-specific `spec` payloads.
- Build a planning layer that loads manifests, resolves remote resources, compares desired and current state, and emits machine-readable plans.
- Apply access grants using Open WebUI's `access_grants` semantics rather than the older `access_control` shape.
- Make destructive reconciliation explicit through `sync`, previewable through `diff`, and gated by dry-run/confirmation flags during implementation.

**Non-Goals:**

- Do not replace Stage 1 resource commands; manifests orchestrate the same server APIs.
- Do not implement an offline Open WebUI schema validator that duplicates all server validation.
- Do not bypass Open WebUI authorization, sharing restrictions, or server-side grant filtering.
- Do not add role-based resource grants unless Open WebUI adds them to the `access_grants` API.
- Do not require complete Stage 2 domain coverage in the first implementation slice; unsupported manifest kinds must fail clearly.

## Decisions

1. Use `manifests` as the resource command family.

   Declarative operations will be exposed as `oictl manifests diff`, `oictl manifests apply`, and `oictl manifests sync`. This keeps the command tree resource-first and avoids reserving broad top-level verbs that could conflict with future taxonomy decisions. Alternative considered: top-level `oictl apply` and `oictl diff`, but that conflicts with the existing resource-first command specification.

2. Use a small versioned manifest envelope with resource-specific specs.

   Each manifest document will include `apiVersion`, `kind`, `metadata`, optional `access_grants`, and `spec`. The envelope gives `oictl` stable fields for identity, ownership metadata, and grants, while `spec` remains close to Open WebUI payloads for the target resource. Alternative considered: one custom schema per resource with fully modeled CLI fields, but that would duplicate upstream API models and slow support for new resource kinds.

3. Separate planning from execution.

   `diff`, `apply --dry-run`, and `sync --dry-run` will use the same planner as mutating commands. The planner will produce actions such as create, update, delete, replace-grants, unchanged, and unsupported, then the executor will run only approved mutation actions. Alternative considered: have each command perform ad hoc comparisons, but that risks drift between preview and execution behavior.

4. Treat `apply` as non-pruning and `sync` as pruning.

   `apply` will create or update resources declared in manifests and reconcile declared grants without deleting omitted remote resources. `sync` will reconcile a selected scope to the manifest set and may delete or detach remote resources omitted from that scope. This follows the existing taxonomy distinction between additive import/apply behavior and destructive sync behavior. Alternative considered: one `apply --prune` command, but a distinct `sync` verb makes destructive reconciliation more visible.

5. Use Open WebUI `access_grants` as the canonical grant representation.

   Manifests will use grant entries compatible with server payloads: `principal_type`, `principal_id`, and `permission`. `oictl` may normalize ordering and remove duplicate grants for planning, but the server remains authoritative and may filter disallowed grants. Alternative considered: introduce friendly grant aliases such as `public: true` or `users.read`, but translating aliases would obscure exact server behavior and create extra migration surface.

6. Start with explicit supported kinds.

   The implementation should register supported manifest kinds with handlers for identity lookup, read, create, update, delete, and grant update. Unsupported kinds must be validation errors rather than best-effort API calls. Alternative considered: use generic endpoint templates from manifest fields, but that would make validation and safety weaker.

## Risks / Trade-offs

- Open WebUI resource schemas evolve -> Keep `spec` close to server payloads and prefer passthrough JSON/YAML over over-modeled structs.
- Preview can differ from execution if server filters grants -> Re-fetch changed resources after mutation and report any server-side grant differences.
- `sync` can delete remote resources unintentionally -> Require an explicit scope and confirmation flag for destructive actions, and make dry-run output easy to inspect.
- Multiple manifests can declare the same identity -> Fail validation before contacting the server unless the duplicate documents are byte-for-byte identical and explicitly supported later.
- Access grants differ between database-backed resources and config-backed resources -> Model the grant list consistently in manifests, but route application through resource-specific handlers.
- YAML support requires a dependency -> Prefer JSON first if dependency policy is strict; add YAML only with a small, justified parser.

## Migration Plan

No server data migration is required. Existing Stage 1 commands and profile configuration remain valid. Rollback is removing the manifest commands; remote resources already changed by `apply` or `sync` remain in Open WebUI and can be restored by applying previous manifests or backups.

## Open Questions

- Which resource kinds should be included in the first Stage 2 slice: knowledge, prompts, tools, channels, automations, or another minimal set?
- Should YAML be required in the first implementation, or should JSON manifests ship first with YAML added later?
- Should `sync` require `--yes` for all mutations or only for delete/detach actions?
