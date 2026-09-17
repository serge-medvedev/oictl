## Context

`oictl` now has implemented foundation controls, declarative manifest workflows, and advanced Open WebUI surfaces. This umbrella change records the roadmap and taxonomy that those implementation changes converged on so future changes extend the CLI consistently instead of relying on stale pre-implementation assumptions.

Open WebUI exposes a broad set of operational domains, including models, users, groups, files, knowledge, tools, prompts, functions, pipelines, chats, channels, automations, terminals, config, analytics, SCIM, and provider passthrough. `oictl` needs a taxonomy that covers implemented domains while leaving room for later endpoint coverage.

## Goals / Non-Goals

**Goals:**

- Establish a 3-stage product roadmap for `oictl` that reflects completed implementation slices.
- Establish command taxonomy rules that future implementation proposals can follow without contradicting the current CLI surface.
- Keep roadmap and taxonomy independent of a specific Go CLI parsing library.
- Keep this change limited to OpenSpec documentation and specification artifacts.

**Non-Goals:**

- Add, rename, or remove runtime CLI commands in this docs/spec-only change.
- Implement additional Open WebUI API clients, authentication storage, config files, or command handlers.
- Claim complete API endpoint coverage for any stage.
- Replace Open WebUI's API or web UI with a separate source of truth.

## Decisions

1. Use staged product maturity rather than endpoint-by-endpoint delivery.

   The roadmap is divided into foundation, workspace automation, and operations stages. This gives future changes a clear sequencing model: first make `oictl` safe and useful for one instance, then expand into repeatable desired-state workflows, then add operational intelligence and integration-heavy surfaces.

   Alternative considered: define a flat backlog of resource commands. That would be easier to enumerate but would not communicate product priorities or readiness boundaries.

2. Use resource-first commands.

   Future commands should prefer `oictl <resource> <verb>` for resource operations, for example `oictl models list` or `oictl knowledge sync`. This keeps command discovery grouped by Open WebUI domain nouns and leaves room for resource-specific verbs. The implemented `api`, `auth`, `profiles`, and `manifests` families are accepted foundation/workflow command families rather than one-off aliases.

   Alternative considered: use verb-first commands such as `oictl list models`. That style is concise for simple CRUD but becomes less discoverable as resource-specific workflows grow.

3. Reserve top-level names narrowly and preserve implemented families.

   `help`, `version`, and `profiles` remain CLI/session concerns. Authentication operations are grouped under `auth` rather than top-level `login` or `logout`, and Open WebUI configuration control uses the implemented singular `config` family. `api` is a top-level foundation escape hatch, and `manifests` is the Stage 2 declarative workflow family.

   Alternative considered: reserve top-level `login`, `logout`, and `configs`. The implementation instead converged on `auth login`, `auth logout`, and `config` to keep authentication grouped and match the shipped command surface.

4. Treat destructive reconciliation as explicit.

   Taxonomy reserves `sync` for declarative reconciliation that may update or remove remote state, and distinguishes it from additive `import`. Future implementation proposals must make destructive behavior visible through command help, confirmation, dry-run behavior, or similarly explicit safeguards.

   Alternative considered: use `apply` for all declarative operations. `apply` is familiar from infrastructure tools, but Open WebUI documentation already uses import/sync language for model management, so this taxonomy preserves that distinction.

## Risks / Trade-offs

- Roadmap could become stale as Open WebUI APIs or implementation choices change -> Future proposals must update these capability specs when they introduce commands that no longer fit the staged roadmap or taxonomy.
- Resource-first taxonomy may produce longer commands for simple actions -> Consistent grouping is prioritized over shortest possible command spelling.
- Docs/spec-only change does not give users new behavior immediately -> This is intentional; implementation changes should be proposed separately against these specs.
