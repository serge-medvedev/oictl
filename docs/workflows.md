# Workflow examples

[Project overview and quick start](../README.md) · [Connection settings and profiles](configuration.md) · [API behavior and compatibility](api-compatibility.md)

These examples assume a built `oictl` binary and configured connection. Replace resource IDs with those returned by your instance. Files such as `function.json`, `message.json`, and `request.json` are inputs you prepare; use the corresponding action's `--help` for required fields and examples.

Inspection/export blocks below read server state, though exports write local files. Blocks labeled **Changes server state** contain mutations; they are alternatives for individual tasks, not scripts to run from top to bottom. Server permissions apply to every operation.

## Inspect and export resources

```sh
oictl models list --output json
oictl models export --out models.json
oictl config export --out config.json
oictl knowledge list
oictl knowledge export knowledge-id --out knowledge.zip
oictl tools list
```

For an API route without a dedicated command:

```sh
oictl api /api/models
```

Exports may contain sensitive data. Resource-specific export coverage differs; see [export formats](api-compatibility.md#export-formats).

## Upload a file

**Changes server state.** Upload a local document without requesting processing:

```sh
oictl files upload ./document.pdf --process false
```

See `oictl files upload --help` for processing and metadata options, and [file upload behavior](api-compatibility.md#file-uploads) for MIME handling.

## Manage functions and filters

Filters are `functions` records whose returned `type` is `filter`:

```sh
oictl functions list --type filter
oictl functions get function-id
oictl functions export --include-valves --out functions.json
```

**Changes server state.** Create, update, activate, or synchronize function source:

```sh
oictl functions create --file function.json
oictl functions update function-id --file function.json
oictl functions load-url https://example.invalid/function.py
oictl functions sync --file functions.json --yes
oictl functions toggle function-id
oictl functions toggle-global function-id
```

The URL is a placeholder for a trusted source. Function source is arbitrary Python loaded by Open WebUI. Only create, update, load from URL, or sync trusted source. Sync requires confirmation; [inventory input rules](api-compatibility.md#model-and-function-inventories) distinguish it from declarative manifest sync.

**Deletes a resource:**

```sh
oictl functions delete function-id --yes
```

## Manage tools and skills

Inspect skills and export Tool manifests or a skill's root Markdown:

```sh
oictl skills list --query review
oictl tools export tool-id --manifest --out tool-manifest.json
oictl tools export --manifest --directory manifests/tools
oictl skills export skill-id --format manifest --out skill.md
```

The Markdown skill export contains only the root file. To retain supporting files, use the [JSON package export](api-compatibility.md#export-formats).

**Changes server state.** Create resources from prepared inputs:

```sh
oictl tools create --file tool-create.json
oictl skills create --file skill.json
oictl skills create --manifest skill.md
```

An exported Tool manifest is a declarative envelope, not a native `tools create` payload. Prepare `tool-create.json` from `oictl tools create --help` for native creation. To deploy the exported manifest instead, preview and apply it:

```sh
oictl manifests diff --file tool-manifest.json
# Changes server state:
oictl manifests apply --file tool-manifest.json
```

Tool source executes in Open WebUI; use trusted Python. Skill Markdown input is distinct from the [declarative Skill resource envelope](manifests.md#skill). See [skill input and revision rules](api-compatibility.md#skill-input-and-revisions) before updating packages or files.

## Configure task models and attach skills

Inspect task configuration:

```sh
oictl tasks config get
```

**Changes server state.** Update settings or attach existing skills:

```sh
oictl tasks config set --file tasks-config.json
oictl tasks skills attach review-helper ops-helper
oictl tasks skills attach --external review-helper
```

Supply existing skill IDs or names. `--external` targets `TASK_MODEL_EXTERNAL`; otherwise the attachment targets `TASK_MODEL`. See [task model configuration](api-compatibility.md#task-model-configuration) for the underlying configuration keys.

## Administer users and groups

Native `users` and `groups` use the normal Open WebUI token:

```sh
oictl groups list --share true
oictl users get user-id
```

**Changes server state.** Create or update users/groups and add group membership:

```sh
chmod 600 user-create.json
oictl users create --file user-create.json
oictl users update user-id --file user-update.json
oictl groups create --file group.json
oictl groups update group-id --file group.json
oictl groups users add group-id user-id
```

User payloads can contain passwords. Protect files before adding credentials; inline values and generated output are also sensitive. Native administration is distinct from [SCIM provisioning](configuration.md#scim-credentials).

**Deletes resources:**

```sh
oictl users delete user-id --yes
oictl groups delete group-id --yes
```

## Channels and webhooks

Inspect channels, members, and webhook types:

```sh
oictl channels list
oictl channels members list channel-id --query alice
oictl webhooks channels list channel-id --output table
oictl webhooks events catalog
```

**Changes server state.** Create a channel, post/pin a message, add a reaction, or manage webhooks:

```sh
oictl channels create --file channel.json
oictl channels messages post channel-id --file message.json
oictl channels pins set channel-id message-id
oictl channels reactions add channel-id message-id --name thumbs-up
oictl webhooks channels create channel-id --name ci-bot
oictl webhooks events create --file event-webhook.json
oictl webhooks events disable webhook-id
```

Export an incoming webhook URL without printing it:

```sh
oictl webhooks channels url channel-id webhook-id --out webhook-url.txt
```

Use `--show-url` only when you intend to print the full incoming URL. Webhook table output redacts tokens and URLs, but JSON responses preserve secret-bearing values. See [channel and webhook API behavior](api-compatibility.md#channels-and-webhooks).

## Apply declarative configuration

Use the [manifest guide](manifests.md) to prepare JSON/YAML resource files. Inspect changes without mutation:

```sh
oictl manifests diff --file prompt.json
oictl manifests apply --dry-run --file prompt.json --output json
oictl manifests sync --scope prompts --directory manifests/ --dry-run
```

**Changes server state.** Apply creates or updates declared resources. Sync also deletes omitted resources in the selected scope when confirmed:

```sh
oictl manifests apply --directory manifests/
oictl manifests sync --scope prompts --directory manifests/ --yes
```

`--scope all` extends reconciliation to every supported pruning scope. Use it only when the input directory describes the intended resource set for all those scopes. Manifest grants are supported for knowledge, prompts, tools, models, skills, channels, and terminal server connections; functions and groups reject grants before mutation.

## Chats, analytics, and automations

Read/export your own chats and inspect administrative analytics:

```sh
oictl chats list --page 1
oictl chats export --out chats.ndjson
oictl analytics models --start-date 1700000000 --end-date 1700600000
```

Explicitly export every user's chats with an administrator credential:

```sh
oictl chats export --all-users --out all-chats.json
```

The ordinary export is NDJSON; the administrative export is database-wide JSON. Protect both. Pagination and analytics filters vary by action; see [pagination and analytics](api-compatibility.md#pagination-and-analytics).

**Changes server state.** Create a scheduled automation from a prepared payload:

```sh
oictl automations create --file automation.json
```

## SCIM and provider integrations

SCIM requires a [dedicated provisioning token](configuration.md#scim-credentials):

```sh
oictl scim users list --start-index 1 --count 50
```

Inspect provider tags through Open WebUI:

```sh
oictl providers ollama tags
```

Send a completion request from a prepared JSON body, bounding the stream explicitly:

```sh
oictl --timeout 2m providers openai request --method POST /chat/completions --file request.json --stream
```

Provider requests can invoke models and incur usage. They route through the configured Open WebUI instance and reject external provider URLs or `Authorization` header overrides.
