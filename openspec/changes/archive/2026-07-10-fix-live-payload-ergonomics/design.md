## Context

Live Open WebUI QA found three payload ergonomics issues where `oictl` builds payloads that are valid JSON but not aligned with current server expectations. The affected behavior is limited to command payload construction for chat tags, model mutations/import/sync, and channel member active-state updates.

The chat tag `--tag` payload is the observed bug. The omitted model `params` default and channel member `active` alias were found during the live matrix recheck and are included because they are the same class of outbound payload normalization issue, affect existing commands, and can be covered by focused tests without adding new user-facing concepts.

## Goals / Non-Goals

**Goals:**

- Send chat tag updates using the current Open WebUI `TagForm` field name, `name`.
- Ensure model create/import/sync outbound payloads contain `params: {}` when user input omits `params`.
- Accept `active` as a common alias for channel member active-state updates and normalize it to `is_active`.
- Preserve explicit `is_active` when a payload contains both `active` and `is_active`.

**Non-Goals:**

- Redesign chat, model, or channel command schemas beyond these payload compatibility fixes.
- Add client-side authorization or feature checks that duplicate Open WebUI decisions.
- Change response rendering, output formats, or stored profile behavior.

## Decisions

- Include the two live matrix recheck ergonomics fixes with the observed chat tag bug. A tag-only scope was rejected because it would leave known same-class live compatibility gaps while saving little implementation or review risk.
- Normalize payloads before sending requests to Open WebUI. This keeps command inputs ergonomic while preserving server-backed behavior and avoids adding response-side compatibility code.
- Default omitted model `params` to an empty object only for create/import/sync payloads. This targets the live server contract without changing unrelated update or manifest behavior beyond existing manifest defaults.
- Treat `is_active` as the canonical outbound channel member field. If both `active` and `is_active` are present, `is_active` wins and `active` is removed from the outbound payload.
- Keep chat tag construction explicit by emitting tag objects with `name`. This aligns `--tag` convenience flags with the server's current `TagForm` shape.

## Risks / Trade-offs

- Server versions with older tag shapes could differ from current Open WebUI. Mitigation: target the current live API contract found during QA and avoid adding speculative backward compatibility.
- Payload normalization can surprise users inspecting debug output if aliases are rewritten. Mitigation: document and test the canonical outbound field names.
- Model payloads with malformed non-object `params` remain possible if supplied explicitly. Mitigation: this change only defaults omitted values; broader validation is out of scope unless tests reveal existing validation gaps.
