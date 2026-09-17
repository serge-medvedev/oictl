## Why

Open WebUI channels are a separate collaboration surface from chats, but oictl does not yet provide typed commands for channels, members, channel messages, pins, or reactions. Adding these commands gives operators and automation scripts a stable CLI surface while preserving Open WebUI's server-side feature gates, membership checks, and access-grant authorization.

## What Changes

- Add an `oictl channels` command family for listing, creating, retrieving, updating, and deleting Open WebUI channels.
- Add member commands for listing members, adding and removing channel members, and updating the caller's active member state where Open WebUI supports it.
- Add message commands for listing, posting, retrieving, updating, threading, and deleting channel messages.
- Add pin commands for listing pinned messages and pinning or unpinning a channel message.
- Add reaction commands for adding and removing reactions on channel messages.
- Preserve Open WebUI authorization by forwarding requests to server endpoints using the resolved bearer token and surfacing 401/403/404 responses without client-side bypass or local access emulation.
- Pass channel and message mutation payloads through as JSON from `--data`, `--file`, or stdin rather than inventing an oictl-specific schema.
- Require explicit confirmation for destructive channel and message deletion operations.

## Capabilities

### New Capabilities
- `channels-control`: Typed control commands for Open WebUI channels, members, messages, pins, and reactions.

### Modified Capabilities

## Impact

- Affected CLI surfaces: `oictl channels ...` command dispatch, help output, request construction, and shared structured output handling.
- Affected Open WebUI APIs: `/api/v1/channels` endpoints for channels, members, messages, pins, threads, and reactions.
- Authorization impact: no new local authorization model; all effective permissions remain enforced by Open WebUI.
- Testing impact: command routing, payload forwarding, confirmation handling, and HTTP request/response behavior need unit coverage with test servers or fake transports.
