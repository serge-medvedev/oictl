## Why

Extensive live Open WebUI QA found payload shape mismatches between common `oictl` usage and the current server API contracts. The directly observed failure was `chats tags set --tag` sending a tag shape the server rejects. The live matrix recheck also identified two same-class payload ergonomics gaps for omitted model `params` and channel member active-state aliases. These fixes make the CLI send accepted payloads for tags, model parameters, and channel member activation without forcing users to know server-internal field names or defaults.

## What Changes

- Make `chats tags --tag` send the current Open WebUI `TagForm` payload shape with `name`.
- Make `models create`, `models import`, and `models sync` tolerate omitted parameters by defaulting `params` to `{}` in outbound payloads.
- Make `channels members active` accept `active` as a user-facing alias by normalizing it to `is_active`.
- Preserve explicit `is_active` values when both `active` and `is_active` are present.

## Scope Decision

Keep all three payload ergonomics fixes in this change. A tag-only change would repair the observed failure but leave two live-rechecked compatibility issues with the same root cause: CLI payload construction exposing server-specific shape details. The added scope is still narrow because it touches only outbound payload normalization for existing commands and adds focused request-body tests.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `chats-control`: Tag mutation payloads must use the server's `TagForm {name}` shape.
- `models-control`: Model create/import/sync payloads must include an object `params` value when the user omits parameters.
- `channels-control`: Channel member active updates must accept `active` as an alias for `is_active` while preserving explicit `is_active`.

## Impact

- Affected CLI command payload construction for chats, models, and channels.
- Affected integration behavior against live Open WebUI APIs.
- No new dependencies or breaking CLI changes are expected.
