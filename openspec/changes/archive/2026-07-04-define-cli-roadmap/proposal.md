## Why

`oictl` needs a shared product direction that matches the command surface delivered by the initial implementation changes. Capturing the roadmap and command taxonomy gives future CLI work a stable target and reduces ad hoc command growth while preserving the decisions already made for Stage 1, Stage 2, and advanced surfaces.

## What Changes

- Define a 3-stage product roadmap for evolving `oictl` from foundation controls into workspace automation and operations-oriented surfaces.
- Define the top-level command taxonomy, including command families, naming conventions, stage alignment, and the as-built exceptions for foundation/session commands.
- Keep this change limited to OpenSpec documentation and specification artifacts; no runtime CLI behavior changes are included.
- No breaking changes.

## Capabilities

### New Capabilities

- `product-roadmap`: Defines the staged product direction, stage goals, and boundaries for `oictl`.
- `cli-command-taxonomy`: Defines the command family structure and naming rules for future `oictl` commands.

### Modified Capabilities

- None.

## Impact

- Affected documentation/specification: OpenSpec change artifacts only.
- Affected code: None.
- Affected APIs/dependencies/systems: None.
