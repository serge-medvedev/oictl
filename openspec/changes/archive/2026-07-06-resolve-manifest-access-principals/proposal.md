## Why

Manifest access grants currently require raw Open WebUI principal IDs, which are hard to author, review, and keep stable across environments. Allowing grants to name users and groups by stable names, emails, or explicit refs makes manifests portable while preserving the existing ID-based API behavior.

## What Changes

- Extend manifest `access_grants` entries for `principal_type: user` and `principal_type: group` to accept a principal reference instead of only `principal_id`.
- Resolve user refs, user emails, group refs, group names, and existing IDs to Open WebUI principal IDs before diff, apply, and sync planning compares grants or sends grant update requests.
- Preserve `principal_id: "*"` as the public-user grant and leave existing ID-based manifests valid.
- Fail before mutation when a non-public principal reference is missing, ambiguous, or not compatible with the declared `principal_type`.
- Include tests and manifest documentation for ID, email/name, and explicit ref forms.

## Capabilities

### New Capabilities

### Modified Capabilities
- `access-grant-modeling`: Access grants can declare resolvable user and group principals by stable identifiers, and reconciliation operates on resolved principal IDs.
- `declarative-resource-management`: Manifest diff, apply, and sync workflows resolve access grant principals before planning or mutating resources.

## Impact

- Affected code: `internal/cli/manifests.go`, manifest workflow tests, and any helper code needed to query users/groups for resolution.
- Affected docs: `docs/manifests.md` access grant examples and reference text.
- Affected APIs: read-only Open WebUI user/group listing or lookup endpoints during manifest planning when non-ID principal references are present; existing grant update endpoints continue receiving `principal_id` values.
- No breaking changes are intended for existing manifests that already use `principal_id` or the public `*` grant.
