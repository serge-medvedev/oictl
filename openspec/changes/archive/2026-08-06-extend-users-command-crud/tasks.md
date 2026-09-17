## 1. Help And Discovery

- [x] 1.1 Extend the focused users help test for all existing and new commands, JSON input syntax, and deletion confirmation; record the focused RED result.
- [x] 1.2 Update users help text minimally and record the focused GREEN result.

## 2. Get And Create

- [x] 2.1 Add a focused `httptest` contract for authenticated get with escaped ID and complete response output; record RED, implement get, and record GREEN.
- [x] 2.2 Extend the focused contract for create from a protected JSON file with byte-preserved body and complete response output; record RED, implement create, and record GREEN.

## 3. Update And Delete

- [x] 3.1 Extend the focused contract for update with escaped ID, byte-preserved partial JSON, and complete response output; record RED, implement update, and record GREEN.
- [x] 3.2 Add no-request validation for unconfirmed delete and a confirmed delete request contract; record RED, implement delete, and record GREEN.

## 4. Validation And Failures

- [x] 4.1 Add table-driven no-request tests for CRUD arity, required payloads, confirmation, and unknown subcommands.
- [x] 4.2 Add representative 400, 401, 403, and not-found server failure tests for non-zero exit, visible safe status/detail, and configured credential redaction.

## 5. Documentation And Verification

- [x] 5.1 Add compact README examples for native user get/create/update/delete, sensitive JSON handling, and separation from SCIM users.
- [x] 5.2 Run formatting, focused and full Go tests, build the CLI, inspect built users help, strictly validate the exact change, run `git diff --check`, and inspect the exact candidate for unrelated changes.
