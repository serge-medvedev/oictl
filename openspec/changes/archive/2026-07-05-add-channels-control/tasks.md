## 1. Command Wiring

- [x] 1.1 Register `channels` in the top-level command dispatch and help output.
- [x] 1.2 Add `oictl channels --help` text covering channel, member, message, pin, and reaction operations.
- [x] 1.3 Add shared channel command parsing that preserves global `--output`, `--out`, target, profile, token, and timeout behavior.

## 2. Channel Records

- [x] 2.1 Implement `oictl channels list` with server query forwarding and structured output.
- [x] 2.2 Implement `oictl channels get <channel-id>` with path escaping and structured output.
- [x] 2.3 Implement `oictl channels create` using pass-through JSON from `--data`, `--file`, or stdin.
- [x] 2.4 Implement `oictl channels update <channel-id>` using pass-through JSON from `--data`, `--file`, or stdin.
- [x] 2.5 Implement `oictl channels delete <channel-id> --yes` with confirmation enforced before sending the request.

## 3. Members

- [x] 3.1 Implement `oictl channels members list <channel-id>` with `--page`, `--query`, `--order-by`, and `--direction` forwarding.
- [x] 3.2 Implement `oictl channels members add <channel-id>` with pass-through member payload support.
- [x] 3.3 Implement `oictl channels members remove <channel-id>` with pass-through member payload support.
- [x] 3.4 Implement `oictl channels members active <channel-id>` with pass-through active-state payload support and optional convenience flag synthesis if added.

## 4. Messages

- [x] 4.1 Implement `oictl channels messages list <channel-id>` with `--skip` and `--limit` forwarding.
- [x] 4.2 Implement `oictl channels messages post <channel-id>` using pass-through message payloads.
- [x] 4.3 Implement `oictl channels messages get <channel-id> <message-id>`.
- [x] 4.4 Implement `oictl channels messages data <channel-id> <message-id>`.
- [x] 4.5 Implement `oictl channels messages thread <channel-id> <message-id>` with `--skip` and `--limit` forwarding.
- [x] 4.6 Implement `oictl channels messages update <channel-id> <message-id>` using pass-through message payloads.
- [x] 4.7 Implement `oictl channels messages delete <channel-id> <message-id> --yes` with confirmation enforced before sending the request.

## 5. Pins And Reactions

- [x] 5.1 Implement `oictl channels pins list <channel-id>` with `--page` forwarding.
- [x] 5.2 Implement `oictl channels pins set <channel-id> <message-id>` using the server pin payload.
- [x] 5.3 Implement `oictl channels pins unset <channel-id> <message-id>` using the server unpin payload.
- [x] 5.4 Implement `oictl channels reactions add <channel-id> <message-id>` with `--name` convenience support and pass-through JSON override.
- [x] 5.5 Implement `oictl channels reactions remove <channel-id> <message-id>` with `--name` convenience support and pass-through JSON override.

## 6. Authorization And Errors

- [x] 6.1 Ensure all channel requests use authenticated Open WebUI target resolution and bearer token behavior.
- [x] 6.2 Ensure server 401, 403, 404, and feature-disabled responses exit non-zero and preserve redacted server error context.
- [x] 6.3 Ensure the CLI does not perform local membership, access-grant, author, or admin authorization decisions for channel commands.

## 7. Tests And Documentation

- [x] 7.1 Add command routing and help tests for `oictl channels` and nested subcommands.
- [x] 7.2 Add request construction tests for channel record operations, including payload passthrough and delete confirmation.
- [x] 7.3 Add request construction tests for member, message, pin, and reaction operations.
- [x] 7.4 Add error handling tests for server authorization failures and missing destructive confirmation.
- [x] 7.5 Update user-facing documentation with concise `oictl channels` examples.
