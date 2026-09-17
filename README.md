# oictl

Control CLI utility for Open WebUI.

## Build from source

Requires Go 1.26.4 or later. From the repository root:

```sh
go build -o bin/oictl ./cmd/oictl
export PATH="$PWD/bin:$PATH"
```

The examples below use the built `oictl` binary. The `PATH` setting applies to the current shell.

## Usage

```sh
oictl --help
```

Set connection details with environment variables:

```sh
export OPEN_WEBUI_URL=http://localhost:3000
export OPEN_WEBUI_API_KEY=sk-your-key
```

`OPEN_WEBUI_BASE_URL` is also accepted as a base URL fallback.

Make an authenticated Open WebUI API request:

```sh
oictl api /api/models
```

Use named profiles for repeated access to one or more instances:

```sh
oictl profiles set local --base-url http://localhost:3000 --token sk-your-key
oictl --profile local auth me
```

Common control commands:

```sh
oictl models list --output json
oictl models export --out models.json
oictl config export --out config.json
oictl files upload ./document.pdf --process false
oictl knowledge list
oictl knowledge export <knowledge-id> --out knowledge.zip
oictl functions list --type filter
oictl functions get <function-id>
oictl functions create --file function.json
oictl functions update <function-id> --file function.json
oictl functions delete <function-id> --yes
oictl functions export --include-valves --out functions.json
oictl functions load-url https://example.invalid/function.py
oictl functions sync --file functions.json --yes
oictl functions toggle <function-id>
oictl functions toggle-global <function-id>
oictl skills list --query review
oictl skills create --file skill.json
oictl skills create --manifest skill.md
oictl skills export <skill-id> --format manifest --out skill.md
oictl tasks config get
oictl tasks config set --file tasks-config.json
oictl tasks skills attach review-helper ops-helper
oictl tasks skills attach --external review-helper
oictl tools list
oictl tools create --file tool.json
oictl tools export <tool-id> --manifest --out tool.json
oictl tools export --manifest --directory manifests/tools
oictl groups list --share true
oictl groups create --file group.json
oictl groups update <group-id> --file group.json
oictl groups delete <group-id> --yes
oictl groups users add <group-id> <user-id>
chmod 600 user-create.json
oictl users create --file user-create.json
oictl users get <user-id>
oictl users update <user-id> --file user-update.json
oictl users delete <user-id> --yes
oictl channels list
oictl channels create --file channel.json
oictl channels members list <channel-id> --query alice
oictl channels messages post <channel-id> --file message.json
oictl channels pins set <channel-id> <message-id>
oictl channels reactions add <channel-id> <message-id> --name thumbs-up
oictl webhooks channels list <channel-id>
oictl webhooks channels create <channel-id> --name ci-bot
oictl webhooks channels url <channel-id> <webhook-id> --out webhook-url.txt
oictl webhooks events catalog
oictl webhooks events create --file event-webhook.json
oictl webhooks events disable <webhook-id>
```

Open WebUI filters are managed through `functions` records whose returned `type` is `filter`. Function source is arbitrary Python loaded by Open WebUI; only create, update, load from URL, or sync functions from trusted sources.

Task model settings are managed as task configuration through `oictl tasks config`. In the Open WebUI task config payload, `TASK_MODEL` maps to config key `task.model.default` and `TASK_MODEL_EXTERNAL` maps to `task.model.external`; `oictl` does not provide a separate `task-model` or `task-models` resource. To attach existing skills to the configured task model, use `oictl tasks skills attach <skill-id-or-name>...`; add `--external` to target `TASK_MODEL_EXTERNAL` instead of `TASK_MODEL`.

Webhook table output redacts channel webhook tokens, constructed incoming webhook URLs, and event destination URLs by default. JSON output preserves complete server responses for automation and may include secret-bearing values. Use `webhooks channels url --show-url` only when you intend to print the full incoming webhook URL, or `--out path` to write it without printing it to the terminal.

Declarative resource workflows:

```sh
oictl manifests diff --file prompt.json
oictl manifests apply --directory manifests/
oictl manifests apply --dry-run --file prompt.json --output json
oictl manifests sync --scope prompts --directory manifests/ --yes
oictl manifests sync --scope all --directory manifests/ --yes
```

See `docs/manifests.md` for JSON/YAML manifest examples and authoring rules covering knowledge, prompts, tools, models, skills, functions, groups, channels, terminal server connections, public grants, group grants, private access, apply, diff, and sync. Manifest `access_grants` are supported for knowledge, prompts, tools, models, skills, channels, and terminal server connections; functions and groups reject grants before mutation.

Advanced administration and integration surfaces:

```sh
oictl chats list --page 1
oictl chats export --out chats.ndjson
# Explicit administrative export of every user's chats:
oictl chats export --all-users --out all-chats.json
oictl analytics models --start-date 1700000000 --end-date 1700600000
oictl automations create --file automation.json
OPEN_WEBUI_SCIM_TOKEN=scim-secret oictl scim users list --start-index 1 --count 50
oictl providers openai request --method POST /chat/completions --file request.json --stream
oictl providers ollama tags
```

Native `groups` and `users` commands manage Open WebUI resources with the normal API token. User create and update JSON may contain passwords; treat inline values, files, and generated output as sensitive, and protect payload files appropriately. Native `oictl users` administration is distinct from `oictl scim users`: SCIM commands retain their provisioning protocol and require a dedicated SCIM token from `--scim-token`, `OPEN_WEBUI_SCIM_TOKEN`, or a profile `scim_token`; they do not fall back to normal Open WebUI API tokens. Provider commands route through the configured Open WebUI instance and reject external provider URLs or `Authorization` header overrides.

## Safety and API compatibility

- Flags are validated for the selected action before requests. `--dry-run` is supported for manifest apply/sync and extension-backed bulk UI settings patches; other actions reject it instead of silently mutating. Boolean flags accept Go boolean spellings (`true`/`false`, `TRUE`/`FALSE`, `True`/`False`, `t`/`f`, `T`/`F`, `1`/`0`); invalid values fail locally.
- Ordinary requests have a 30-second deadline, including response reads. Override it with a positive `--timeout`, such as `--timeout 2m`. Explicit provider `--stream` requests have no default overall deadline; use `--timeout` to bound the stream.
- Profile selection is `--profile`, then `OICTL_PROFILE`, then `default`. Login persistence and logout cleanup use that same profile. A saved default profile is used automatically; environment connection values still override profile values. Profile display redacts both bearer and SCIM tokens.
- Model import/sync accepts `{"models":[...]}` or an array shorthand that is wrapped in that envelope; omitted `params` defaults inside each model. Function sync likewise accepts `{"functions":[...]}` or an array. Missing/wrong-shaped inventories are rejected. An intentionally empty model sync requires `--yes`; all function syncs require confirmation.
- `channels list` returns the caller's complete accessible collection. It does not support page, query, or sort flags. Channel manifests use the administration inventory, subject to server authorization.
- `chats export` exports only the authenticated user's chats and preserves the server's NDJSON format. `--all-users` explicitly selects the administrative database export. `skills export` defaults to lossless JSON on stdout as well as `--out`.
- Cross-user `users ui-settings patch` and `bulk-patch` require a deployment-specific `PATCH /api/v1/users/{id}/settings/ui` extension that the original Open WebUI API does not provide. Supply `--allow-ui-settings-extension` only for a deployment that implements it, including when previewing with `--dry-run`. Ordinary `users settings get/update` uses the original current-user API.
- Channel webhook creation requires a name. Flag-based webhook updates preserve omitted current metadata; raw JSON updates remain deliberate replacement payloads.
- Profile and export destinations are restricted to mode `0600` before writing, including existing files. Output, persistence, and false model-deletion results cause nonzero exits on failure.
