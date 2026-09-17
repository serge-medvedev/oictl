## 1. Command Routing and Help

- [x] 1.1 Add `skills` to top-level command routing and global help output.
- [x] 1.2 Add `runSkills` command help covering `list`, `get`, `create`, `update`, `access-update`, `toggle`, `delete`, and `export`.
- [x] 1.3 Add table rendering support for Skills list responses if existing row extraction does not render `items` responses adequately.

## 2. JSON Skills API Operations

- [x] 2.1 Implement `oictl skills list` using `/api/v1/skills/list` with `--query`, `--view-option`, and `--page` filters.
- [x] 2.2 Implement `oictl skills get <skill-id>` using `/api/v1/skills/id/{id}`.
- [x] 2.3 Implement `oictl skills create --file|--data` using `/api/v1/skills/create`.
- [x] 2.4 Implement `oictl skills update <skill-id> --file|--data` using `/api/v1/skills/id/{id}/update`.
- [x] 2.5 Implement `oictl skills access-update <skill-id> --file|--data` using `/api/v1/skills/id/{id}/access/update`.
- [x] 2.6 Implement `oictl skills toggle <skill-id>` using `/api/v1/skills/id/{id}/toggle`.
- [x] 2.7 Implement `oictl skills delete <skill-id> --yes` using `/api/v1/skills/id/{id}/delete` and reject deletion without confirmation before making an HTTP request.
- [x] 2.8 Implement `oictl skills export [skill-id]` so bulk export uses `/api/v1/skills/export` and single-skill export uses `/api/v1/skills/id/{id}`.

## 3. Markdown Skill Manifest Support

- [x] 3.1 Implement `--manifest` input for `skills create` and `skills update` that converts supported Markdown frontmatter plus body content into native Skill JSON.
- [x] 3.2 Support only the documented bounded frontmatter subset: scalar `id`, `name`, `description`, optional `is_active`, and simple `tags`; require JSON for complex fields such as `access_grants`.
- [x] 3.3 Implement `oictl skills export <skill-id> --format manifest --out <path>` that writes supported frontmatter and Markdown content for one Skill.
- [x] 3.4 Add clear errors for conflicting payload inputs such as `--manifest` with `--file` or `--data`.

## 4. Tests and Documentation

- [x] 4.1 Add CLI tests for skills help, top-level routing, and unknown subcommand behavior.
- [x] 4.2 Add request mapping tests for list filters, get, create, update, access-update, toggle, delete, and export endpoints.
- [x] 4.3 Add tests proving delete without `--yes` fails before sending a request.
- [x] 4.4 Add tests for Markdown manifest create/update conversion, single-skill manifest export, and conflict validation.
- [x] 4.5 Update user-facing docs or README command examples to mention `skills` commands and JSON/manifest payload options.

## 5. Verification

- [x] 5.1 Run `go test ./...`.
- [x] 5.2 Optionally smoke-test against a configured Open WebUI instance for list/get/export and one non-destructive manifest conversion path.
