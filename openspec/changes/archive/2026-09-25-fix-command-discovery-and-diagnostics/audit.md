# oictl value/help/version audit

Revision: `cad8b3e91162e2343e5d794a09d058b17c64387f`. Baseline source study and isolated real-binary probes. No live API access. This inventory describes observed baseline behavior, not the proposed implementation.

## Scope and exact totals

- **328 finite paths**: 264 action-only, 1 action-and-group (`analytics models`), 63 group-only (including root). Includes root aliases, 21 command families, intermediate groups, all fixed actions, non-JSON actions and profiles. Selector alternatives do not count as separate command paths.
- **5734 actual probes**, all individually recorded with argv, exit, stdout/stderr, loopback requests and config/run-directory content changes.
- **72 ordinary option names**: 7 global valued; 65 local = 51 valued, 13 booleans, 1 overloaded `manifest`. Full ownership is machine-readable. All 563 supported local valued flag/path pairs exercised without a value; all returned the legitimate nonzero requires-value error. All 65 local names also probed on an unsupported action, both bare and with a value.
- **177 paths** return `--help requires a value` for appended `--help` without selectors. Short-help outcomes across all 328 paths: `{'help': 148, 'version': 3, 'http-execution': 89, 'non-help-error': 86, 'local-side-effect': 1, 'non-help-success': 1}`. Three version aliases with appended help print version, counted separately rather than silently inventing precedence.
- Exhaustive finite token coverage is source-reviewed; it is not exhaustive enumeration of unbounded IDs, flag values, ordering permutations, generic API routes, or provider schemas.

## Isolation and execution evidence

- Build and probes ran under `bwrap --ro-bind / / --dev /dev --bind /tmp/oictl-value-audit-cad8b3e9 /tmp/oictl-value-audit-cad8b3e9 --unshare-net --clearenv`. Only the task directory was writable. HOME/XDG_CONFIG_HOME/TMPDIR were fake directories. Host network was unavailable; a loopback HTTP server inside the private namespace refused every request with HTTP 418. Synthetic non-credential markers enabled routing visibility without credentials.
- Fixture observed **594 requests** across all probe classes, never forwarded. Three probes changed seeded temporary profiles: `profiles delete -h`, `profiles delete help`, and `profiles set -h --base-url <loopback>`. Positional `help` is a legitimate profile name at the leaf, not a universally supported help alias.
- Build used cached dependencies with GOPROXY=off and CGO_ENABLED=0. No installs or downloads.

## Decisive parser causes and intended replacement

1. **Exactly two production emit sites**: `internal/cli/app.go:246` (`parseGlobalFlags`) and `:333` (`parseCommandFlags`). The former is legitimate when any of `--profile --base-url --url --token --output --timeout --out` is terminal with no value. The latter is legitimate only for known, supported valued flags. Preserve those nonzero diagnostics and genuine missing positional/body/confirmation/required-option errors.
2. **Unknown long options are assumed valued before ownership is checked.** `validateCommandFlags` calls the shared parser at `command_flags.go:19` before its allowlist check at `:25-49`; terminal `--audit-unknown` and known-but-unsupported `--data` therefore say requires-value. Supplying `=x` changes the same spelling into unsupported-option. Validate recognition/ownership before demanding an unknown or unsupported option value; remain nonzero. Boolean invalid-value semantics must remain nonzero.
3. **Help interception is incomplete.** `App.Run:53` only resolves `jsonInputReferences`; `isHelp` recognizes only the first remaining positional, and family validation only skips help immediately after the first family token. Non-JSON leaves and most intermediate groups send `--help` into scalar parsing. Resolve actual finite contextual help before validation/input/config/runtime work, without requiring selectors; reuse a small help-only route lookup/static text, not a schema or dispatch framework.
4. **`-h` is not a flag to the parser.** `splitFlag:287-295` recognizes only `--...`; naked `-h` is an operand. Strict handlers issue usage, permissive ones ignore it, and selector-taking handlers consume it as an ID. Actual probes reached HTTP mutation paths, profile reads and profile deletion. Fix both help spellings together before any side effects, including profile writes, logout cleanup and output-file handling.
5. **Top-level `--version` is broken despite a dispatch case.** Validation precedes `Run:125`; `--version` hits requires-value and never reaches version dispatch. `version` and `-v` work. Repair all three existing root spellings. Nested `--version` is not an established global version feature; reject unsupported nested spelling without execution rather than creating new semantics. Same concern applies to unrecognized short flags: current `-x` can become an ID or be ignored.
6. **Preserve value boundaries.** `--query --help`, `--data -h`, `--data=--help`, and global-valued-option arguments are literal values, not standalone help. These execution probes are controls, not defects. A later separate help flag must still be recognized. `--` currently is merely positional and is not an established end-of-options delimiter; do not accidentally promise new delimiter support.
7. **Overloaded `--manifest` is asymmetric.** The shared parser treats a terminal `--manifest` or one followed by `--...` as boolean; a following `-h` is consumed as its value. Tool export uses boolean mode; skill create/update consumes a path. Probes show `skills create --manifest -h` tries to open file `-h`, while `tools export --manifest -h` can execute a raw export. Make help scanning follow the actual selected action’s option meaning; preserve genuine valued-path omission as nonzero. No new manifest feature is needed.

## Replacement classification ceiling

- Explicit standalone help: local contextual syntax, exit 0, no profile/config/input/stdin/body/output file/HTTP effects, with and without representative selectors. Preserve working authored JSON input documentation, but implementation provenance is not needed in operator-facing help.
- No explicit help: preserve existing successful family discovery and meaningful nonzero missing-input semantics. Do not turn all arity failures into exit 0 or invent required arity from usage strings (e.g. `users search` currently accepts no query; several create commands currently allow no body; `skills export` has optional ID).
- Unknown command/option, known-but-unsupported option, malformed valued/boolean input: nonzero contextual diagnostic without execution. `help` in a selector position is not automatically a reserved word.
- Existing positional help: root `help`, every family’s first `help`, plus `tools valves help` / `functions valves help`. Root `help models list` and family `models help list` currently show root/family help respectively, not routed action help. Preserve these existing forms; arbitrary `... leaf help` is not a supported contract.

## Complete affected fixed-path list (`--help requires a value`)

- **analytics** (8): `analytics daily`, `analytics messages`, `analytics models`, `analytics models chats`, `analytics models overview`, `analytics summary`, `analytics tokens`, `analytics users`.
- **auth** (18): `auth admin-config`, `auth admin-config get`, `auth api-key`, `auth api-key create`, `auth api-key delete`, `auth api-key get`, `auth ldap-config`, `auth ldap-config get`, `auth ldap-server-config`, `auth ldap-server-config get`, `auth logout`, `auth me`, `auth oauth-config`, `auth oauth-config get`, `auth password`, `auth profile`, `auth timezone`, `auth timezone set`.
- **automations** (7): `automations delete`, `automations get`, `automations list`, `automations run`, `automations runs`, `automations runs list`, `automations toggle`.
- **channels** (16): `channels delete`, `channels get`, `channels list`, `channels members`, `channels members list`, `channels messages`, `channels messages data`, `channels messages delete`, `channels messages get`, `channels messages list`, `channels messages thread`, `channels pins`, `channels pins list`, `channels pins set`, `channels pins unset`, `channels reactions`.
- **chats** (10): `chats archive`, `chats delete`, `chats export`, `chats get`, `chats list`, `chats pin`, `chats search`, `chats share`, `chats tags`, `chats tags get`.
- **config** (18): `config banners`, `config banners get`, `config code-execution`, `config code-execution get`, `config connections`, `config connections get`, `config export`, `config models`, `config models defaults`, `config models get`, `config namespace`, `config oauth-client`, `config terminal-servers`, `config terminal-servers access-grants`, `config terminal-servers access-grants get`, `config terminal-servers get`, `config tool-servers`, `config tool-servers get`.
- **files** (12): `files content`, `files count`, `files data-content`, `files delete`, `files delete-all`, `files get`, `files html-content`, `files list`, `files named-content`, `files rename`, `files search`, `files status`.
- **functions** (7): `functions delete`, `functions export`, `functions get`, `functions list`, `functions load-url`, `functions toggle`, `functions toggle-global`.
- **groups** (8): `groups delete`, `groups export`, `groups get`, `groups info`, `groups list`, `groups preview`, `groups users`, `groups users list`.
- **knowledge** (9): `knowledge dirs`, `knowledge files`, `knowledge files add`, `knowledge files list`, `knowledge files pending`, `knowledge list`, `knowledge search`, `knowledge search-files`, `knowledge sync`.
- **models** (7): `models base`, `models base-tags`, `models delete-all`, `models export`, `models get`, `models list`, `models tags`.
- **profiles** (3): `profiles delete`, `profiles get`, `profiles set`.
- **providers** (11): `providers ollama`, `providers ollama config`, `providers ollama config get`, `providers ollama models`, `providers ollama ps`, `providers ollama tags`, `providers ollama version`, `providers openai`, `providers openai config`, `providers openai config get`, `providers openai models`.
- **scim** (11): `scim groups`, `scim groups delete`, `scim groups get`, `scim groups list`, `scim resource-types`, `scim schemas`, `scim service-provider-config`, `scim users`, `scim users delete`, `scim users get`, `scim users list`.
- **skills** (5): `skills delete`, `skills export`, `skills get`, `skills list`, `skills toggle`.
- **tasks** (4): `tasks config`, `tasks config get`, `tasks skills`, `tasks skills attach`.
- **tools** (4): `tools delete`, `tools export`, `tools get`, `tools list`.
- **users** (7): `users delete`, `users get`, `users list`, `users search`, `users settings`, `users settings get`, `users ui-settings`.
- **webhooks** (12): `webhooks channels`, `webhooks channels delete`, `webhooks channels get`, `webhooks channels list`, `webhooks channels url`, `webhooks events`, `webhooks events catalog`, `webhooks events delete`, `webhooks events disable`, `webhooks events enable`, `webhooks events get`, `webhooks events list`.

## Full finite help matrix

Legend: H=help success; V=version success; R=requires-value; E=other non-help error; X=HTTP execution; W=temporary local write/delete; S=non-help success. `—` means no representative selector applicable. Rows include unaffected paths so closure is inspectable. Exact help argv, exit and outcome reside in `inventory.json`.

| Fixed path | Bare | --help | -h | selectors + --help | selectors + -h | positional help |
|---|---|---|---|---|---|---|
| `(root)` | H | H | H | — | — | H |
| `--help` | H | H | H | — | — | H |
| `--version` | R | V | V | — | — | V |
| `-h` | H | H | H | — | — | H |
| `-v` | V | V | V | — | — | V |
| `analytics` | H | H | H | — | — | H |
| `analytics daily` | X | R | X | — | — | X |
| `analytics messages` | X | R | X | — | — | X |
| `analytics models` | X | R | E | — | — | E |
| `analytics models chats` | E | R | X | R | E | X |
| `analytics models overview` | E | R | X | R | E | X |
| `analytics summary` | X | R | X | — | — | X |
| `analytics tokens` | X | R | X | — | — | X |
| `analytics users` | X | R | X | — | — | X |
| `api` | E | H | H | H | H | H |
| `auth` | H | H | H | — | — | H |
| `auth admin-config` | E | R | E | — | — | E |
| `auth admin-config get` | X | R | E | — | — | E |
| `auth admin-config set` | X | H | H | — | — | E |
| `auth api-key` | E | R | E | — | — | E |
| `auth api-key create` | X | R | E | — | — | E |
| `auth api-key delete` | X | R | E | — | — | E |
| `auth api-key get` | X | R | E | — | — | E |
| `auth ldap-config` | E | R | E | — | — | E |
| `auth ldap-config get` | X | R | E | — | — | E |
| `auth ldap-config set` | X | H | H | — | — | E |
| `auth ldap-server-config` | E | R | E | — | — | E |
| `auth ldap-server-config get` | X | R | E | — | — | E |
| `auth ldap-server-config set` | X | H | H | — | — | E |
| `auth login` | X | H | H | — | — | X |
| `auth logout` | X | R | X | — | — | X |
| `auth me` | X | R | X | — | — | X |
| `auth oauth-config` | E | R | E | — | — | E |
| `auth oauth-config get` | X | R | E | — | — | E |
| `auth oauth-config set` | X | H | H | — | — | E |
| `auth password` | E | R | E | — | — | E |
| `auth password update` | X | H | H | — | — | E |
| `auth profile` | E | R | E | — | — | E |
| `auth profile update` | X | H | H | — | — | E |
| `auth timezone` | E | R | E | — | — | E |
| `auth timezone set` | E | R | X | R | E | X |
| `automations` | H | H | H | — | — | H |
| `automations create` | E | H | H | — | — | E |
| `automations delete` | E | R | E | R | E | E |
| `automations get` | E | R | X | R | E | X |
| `automations list` | X | R | X | — | — | X |
| `automations run` | E | R | X | R | E | X |
| `automations runs` | E | R | E | — | — | E |
| `automations runs list` | E | R | X | R | E | X |
| `automations toggle` | E | R | X | R | E | X |
| `automations update` | E | H | H | H | H | E |
| `channels` | H | H | H | — | — | H |
| `channels create` | E | H | H | — | — | E |
| `channels delete` | E | R | E | R | E | E |
| `channels get` | E | R | X | R | E | X |
| `channels list` | X | R | E | — | — | E |
| `channels members` | E | R | E | — | — | E |
| `channels members active` | E | H | H | H | H | E |
| `channels members add` | E | H | H | H | H | E |
| `channels members list` | E | R | X | R | E | X |
| `channels members remove` | E | H | H | H | H | E |
| `channels messages` | E | R | E | — | — | E |
| `channels messages data` | E | R | E | R | E | E |
| `channels messages delete` | E | R | E | R | E | E |
| `channels messages get` | E | R | E | R | E | E |
| `channels messages list` | E | R | X | R | E | X |
| `channels messages post` | E | H | H | H | H | E |
| `channels messages thread` | E | R | E | R | E | E |
| `channels messages update` | E | H | H | H | H | E |
| `channels pins` | E | R | E | — | — | E |
| `channels pins list` | E | R | X | R | E | X |
| `channels pins set` | E | R | E | R | E | E |
| `channels pins unset` | E | R | E | R | E | E |
| `channels reactions` | E | R | E | — | — | E |
| `channels reactions add` | E | H | H | H | H | E |
| `channels reactions remove` | E | H | H | H | H | E |
| `channels update` | E | H | H | H | H | E |
| `chats` | H | H | H | — | — | H |
| `chats archive` | E | R | X | R | E | X |
| `chats compact` | E | H | H | H | H | X |
| `chats delete` | E | R | E | R | E | E |
| `chats export` | X | R | X | — | — | X |
| `chats get` | E | R | X | R | E | X |
| `chats import` | E | H | H | — | — | E |
| `chats list` | X | R | X | — | — | X |
| `chats pin` | E | R | X | R | E | X |
| `chats search` | X | R | X | — | — | X |
| `chats share` | E | R | X | R | E | X |
| `chats tags` | E | R | E | — | — | E |
| `chats tags delete` | E | H | H | H | H | E |
| `chats tags get` | E | R | X | R | X | X |
| `chats tags set` | E | H | H | H | H | E |
| `config` | H | H | H | — | — | H |
| `config banners` | E | R | E | — | — | E |
| `config banners get` | X | R | E | — | — | E |
| `config banners set` | X | H | H | — | — | E |
| `config code-execution` | E | R | E | — | — | E |
| `config code-execution get` | X | R | E | — | — | E |
| `config code-execution set` | X | H | H | — | — | E |
| `config connections` | E | R | E | — | — | E |
| `config connections get` | X | R | E | — | — | E |
| `config connections set` | X | H | H | — | — | E |
| `config export` | X | R | X | — | — | X |
| `config import` | X | H | H | — | — | X |
| `config models` | E | R | E | — | — | E |
| `config models defaults` | X | R | E | — | — | E |
| `config models get` | X | R | E | — | — | E |
| `config models set` | X | H | H | — | — | E |
| `config namespace` | E | R | X | R | E | X |
| `config oauth-client` | E | R | E | — | — | E |
| `config oauth-client register` | X | H | H | — | — | E |
| `config suggestions` | X | H | H | — | — | X |
| `config terminal-servers` | E | R | E | — | — | E |
| `config terminal-servers access-grants` | E | R | E | — | — | E |
| `config terminal-servers access-grants diff` | E | H | H | H | H | E |
| `config terminal-servers access-grants get` | E | R | X | R | E | X |
| `config terminal-servers access-grants set` | E | H | H | H | H | E |
| `config terminal-servers get` | X | R | E | — | — | E |
| `config terminal-servers lifecycle` | X | H | H | — | — | E |
| `config terminal-servers policy` | X | H | H | — | — | E |
| `config terminal-servers refresh` | X | H | H | — | — | E |
| `config terminal-servers set` | X | H | H | — | — | E |
| `config terminal-servers verify` | X | H | H | — | — | E |
| `config tool-servers` | E | R | E | — | — | E |
| `config tool-servers get` | X | R | E | — | — | E |
| `config tool-servers set` | X | H | H | — | — | E |
| `config tool-servers verify` | X | H | H | — | — | E |
| `files` | H | H | H | — | — | H |
| `files content` | E | R | X | R | E | X |
| `files count` | X | R | X | — | — | X |
| `files data-content` | E | R | X | R | E | X |
| `files delete` | E | R | X | R | E | X |
| `files delete-all` | X | R | X | — | — | X |
| `files get` | E | R | X | R | E | X |
| `files html-content` | E | R | X | R | E | X |
| `files list` | X | R | X | — | — | X |
| `files named-content` | E | R | E | R | E | E |
| `files rename` | E | R | E | R | E | E |
| `files search` | X | R | X | — | — | X |
| `files status` | E | R | X | R | E | X |
| `files update-content` | E | H | H | H | H | X |
| `files upload` | E | H | H | H | H | E |
| `functions` | H | H | H | — | — | H |
| `functions create` | E | H | H | — | — | E |
| `functions delete` | E | R | E | R | E | E |
| `functions export` | X | R | X | — | — | X |
| `functions get` | E | R | X | R | E | X |
| `functions list` | X | R | X | — | — | X |
| `functions load-url` | E | R | X | R | E | X |
| `functions sync` | E | H | H | — | — | E |
| `functions toggle` | E | R | X | R | E | X |
| `functions toggle-global` | E | R | X | R | E | X |
| `functions update` | E | H | H | H | H | E |
| `functions valves` | H | H | H | — | — | H |
| `functions valves get` | E | H | H | H | H | X |
| `functions valves spec` | E | H | H | H | H | X |
| `functions valves update` | E | H | H | H | H | E |
| `functions valves user` | E | H | H | — | — | E |
| `functions valves user get` | E | H | H | H | H | X |
| `functions valves user spec` | E | H | H | H | H | X |
| `functions valves user update` | E | H | H | H | H | E |
| `groups` | H | H | H | — | — | H |
| `groups create` | E | H | H | — | — | E |
| `groups delete` | E | R | E | R | E | E |
| `groups export` | E | R | X | R | E | X |
| `groups get` | E | R | X | R | E | X |
| `groups info` | E | R | X | R | E | X |
| `groups list` | X | R | X | — | — | X |
| `groups preview` | E | R | X | R | E | X |
| `groups update` | E | H | H | H | H | E |
| `groups users` | E | R | E | — | — | E |
| `groups users add` | E | H | H | H | H | E |
| `groups users list` | E | R | X | R | E | X |
| `groups users remove` | E | H | H | H | H | E |
| `help` | H | H | H | — | — | H |
| `knowledge` | H | H | H | — | — | H |
| `knowledge access-update` | E | H | H | H | H | X |
| `knowledge create` | X | H | H | — | — | X |
| `knowledge delete` | E | H | H | H | H | X |
| `knowledge dirs` | E | R | E | — | — | E |
| `knowledge dirs create` | E | H | H | H | H | X |
| `knowledge dirs delete` | E | H | H | H | H | E |
| `knowledge dirs update` | E | H | H | H | H | E |
| `knowledge export` | E | H | H | H | H | X |
| `knowledge files` | E | R | E | — | — | E |
| `knowledge files add` | E | R | E | R | E | E |
| `knowledge files batch-add` | E | H | H | H | H | X |
| `knowledge files list` | E | R | X | R | X | X |
| `knowledge files move` | E | H | H | H | H | X |
| `knowledge files pending` | E | R | X | R | X | X |
| `knowledge files remove` | E | H | H | H | H | X |
| `knowledge files update` | E | H | H | H | H | X |
| `knowledge get` | E | H | H | H | H | X |
| `knowledge list` | X | R | X | — | — | X |
| `knowledge metadata-reindex` | X | H | H | — | — | X |
| `knowledge reindex` | X | H | H | — | — | X |
| `knowledge reset` | E | H | H | H | H | X |
| `knowledge search` | X | R | X | — | — | X |
| `knowledge search-files` | X | R | X | — | — | X |
| `knowledge sync` | E | R | E | — | — | E |
| `knowledge sync cleanup` | E | H | H | H | H | X |
| `knowledge sync diff` | E | H | H | H | H | X |
| `knowledge update` | E | H | H | H | H | X |
| `manifests` | H | H | H | — | — | H |
| `manifests apply` | E | H | H | — | — | E |
| `manifests diff` | E | H | H | — | — | E |
| `manifests sync` | E | H | H | — | — | E |
| `models` | H | H | H | — | — | H |
| `models access-update` | E | H | H | H | H | X |
| `models base` | X | R | X | — | — | X |
| `models base-tags` | X | R | X | — | — | X |
| `models create` | X | H | H | — | — | X |
| `models delete` | E | H | H | H | H | X |
| `models delete-all` | X | R | X | — | — | X |
| `models export` | X | R | X | — | — | X |
| `models get` | E | R | X | R | E | X |
| `models import` | E | H | H | — | — | E |
| `models list` | X | R | X | — | — | X |
| `models sync` | E | H | H | — | — | E |
| `models tags` | X | R | X | — | — | X |
| `models toggle` | E | H | H | H | H | X |
| `models update` | E | H | H | H | H | X |
| `profiles` | H | H | H | — | — | H |
| `profiles delete` | E | R | W | R | E | W |
| `profiles get` | E | R | S | R | E | S |
| `profiles set` | E | R | E | R | E | E |
| `providers` | H | H | H | — | — | H |
| `providers ollama` | E | R | E | — | — | E |
| `providers ollama config` | E | R | E | — | — | E |
| `providers ollama config get` | X | R | E | — | — | E |
| `providers ollama config set` | E | H | H | — | — | E |
| `providers ollama config update` | E | H | H | — | — | E |
| `providers ollama models` | X | R | X | — | — | X |
| `providers ollama ps` | X | R | X | — | — | X |
| `providers ollama request` | E | H | H | H | H | X |
| `providers ollama tags` | X | R | X | — | — | X |
| `providers ollama verify` | X | H | H | — | — | X |
| `providers ollama version` | X | R | X | — | — | X |
| `providers openai` | E | R | E | — | — | E |
| `providers openai config` | E | R | E | — | — | E |
| `providers openai config get` | X | R | E | — | — | E |
| `providers openai config set` | E | H | H | — | — | E |
| `providers openai config update` | E | H | H | — | — | E |
| `providers openai models` | X | R | X | — | — | X |
| `providers openai request` | E | H | H | H | H | X |
| `providers openai verify` | X | H | H | — | — | X |
| `scim` | H | H | H | — | — | H |
| `scim groups` | E | R | E | — | — | E |
| `scim groups create` | E | H | H | — | — | E |
| `scim groups delete` | E | R | E | R | E | E |
| `scim groups get` | E | R | X | R | E | X |
| `scim groups list` | X | R | X | — | — | X |
| `scim groups patch` | E | H | H | H | H | E |
| `scim groups replace` | E | H | H | H | H | E |
| `scim resource-types` | X | R | X | — | — | X |
| `scim schemas` | X | R | X | — | — | X |
| `scim service-provider-config` | X | R | X | — | — | X |
| `scim users` | E | R | E | — | — | E |
| `scim users create` | E | H | H | — | — | E |
| `scim users delete` | E | R | E | R | E | E |
| `scim users get` | E | R | X | R | E | X |
| `scim users list` | X | R | X | — | — | X |
| `scim users patch` | E | H | H | H | H | E |
| `scim users replace` | E | H | H | H | H | E |
| `skills` | H | H | H | — | — | H |
| `skills access-update` | E | H | H | H | H | E |
| `skills create` | E | H | H | — | — | E |
| `skills delete` | E | R | E | R | E | E |
| `skills export` | X | R | X | R | E | X |
| `skills get` | E | R | X | R | E | X |
| `skills list` | X | R | X | — | — | X |
| `skills toggle` | E | R | X | R | E | X |
| `skills update` | E | H | H | H | H | E |
| `tasks` | H | H | H | — | — | H |
| `tasks config` | E | R | E | — | — | E |
| `tasks config get` | X | R | E | — | — | E |
| `tasks config set` | E | H | H | — | — | E |
| `tasks skills` | E | R | E | — | — | E |
| `tasks skills attach` | E | R | X | R | X | X |
| `tools` | H | H | H | — | — | H |
| `tools access-update` | E | H | H | H | H | E |
| `tools create` | E | H | H | — | — | E |
| `tools delete` | E | R | X | R | E | X |
| `tools export` | X | R | E | — | — | E |
| `tools get` | E | R | X | R | E | X |
| `tools list` | X | R | X | — | — | X |
| `tools load-url` | E | H | H | — | — | E |
| `tools update` | E | H | H | H | H | E |
| `tools valves` | H | H | H | — | — | H |
| `tools valves get` | E | H | H | H | H | X |
| `tools valves spec` | E | H | H | H | H | X |
| `tools valves update` | E | H | H | H | H | E |
| `tools valves user` | E | H | H | — | — | E |
| `tools valves user get` | E | H | H | H | H | X |
| `tools valves user spec` | E | H | H | H | H | X |
| `tools valves user update` | E | H | H | H | H | E |
| `users` | H | H | H | — | — | H |
| `users create` | E | H | H | — | — | E |
| `users delete` | E | R | E | R | E | E |
| `users get` | E | R | X | R | E | X |
| `users list` | X | R | E | — | — | E |
| `users search` | X | R | E | — | — | E |
| `users settings` | E | R | E | — | — | E |
| `users settings get` | X | R | E | — | — | E |
| `users settings update` | E | H | H | — | — | E |
| `users ui-settings` | E | R | E | — | — | E |
| `users ui-settings bulk-patch` | E | H | H | H | H | E |
| `users ui-settings patch` | E | H | H | H | H | E |
| `users update` | E | H | H | H | H | E |
| `version` | V | V | V | — | — | V |
| `webhooks` | H | H | H | — | — | H |
| `webhooks channels` | E | R | E | — | — | E |
| `webhooks channels create` | E | H | H | H | H | E |
| `webhooks channels delete` | E | R | E | R | E | E |
| `webhooks channels ensure` | E | H | H | H | H | E |
| `webhooks channels get` | E | R | E | R | E | E |
| `webhooks channels list` | E | R | X | R | E | X |
| `webhooks channels update` | E | H | H | H | H | E |
| `webhooks channels url` | E | R | E | R | E | E |
| `webhooks events` | E | R | E | — | — | E |
| `webhooks events catalog` | X | R | E | — | — | E |
| `webhooks events create` | E | H | H | — | — | E |
| `webhooks events delete` | E | R | E | R | E | E |
| `webhooks events disable` | E | R | X | R | E | X |
| `webhooks events enable` | E | R | X | R | E | X |
| `webhooks events get` | E | R | X | R | E | X |
| `webhooks events list` | X | R | E | — | — | E |
| `webhooks events update` | E | H | H | H | H | E |

## Full ordinary-option inventory

- Globals, all valued: `--profile`, `--base-url`, `--url`, `--token`, `--output`, `--timeout`, `--out`.
- Local valued: `--api-key-header`, `--archived`, `--chat-id`, `--content`, `--count`, `--data`, `--data-file`, `--days`, `--dir`, `--direction`, `--directory`, `--directory-id`, `--end-date`, `--expected-url`, `--expected-url-env`, `--file`, `--filename`, `--filter`, `--folder-id`, `--format`, `--granularity`, `--group-id`, `--header`, `--include-content`, `--limit`, `--metadata`, `--method`, `--model`, `--model-id`, `--name`, `--order-by`, `--page`, `--process`, `--process-in-background`, `--profile-image-url`, `--q`, `--query`, `--scim-token`, `--scope`, `--share`, `--skip`, `--source`, `--start-date`, `--start-index`, `--startIndex`, `--status`, `--tag`, `--type`, `--user-id`, `--users-file`, `--view-option`.
- Local booleans: `--all`, `--all-users`, `--allow-sensitive-ui-keys`, `--allow-ui-settings-extension`, `--confirm`, `--dry-run`, `--external`, `--include-valves`, `--save`, `--show-url`, `--stream`, `--verify-url`, `--yes`.
- Dual: `--manifest` (tool-export boolean / skill-input path).
- All local names have supported owner lists in `inventory.json`; command aliases and option aliases are recorded there. No unaccounted production flag consumers remain.

## Dynamic boundaries / confidence limits

- api endpoint/full URL, optional positional HTTP method (seven recognized standard spellings; case insensitive), arbitrary --method value
- providers openai/ollama request relative path; supported HTTP method finite, path unbounded
- knowledge sync <action> <knowledge-id>: diff/cleanup documented; implementation interpolates arbitrary action without validating it
- resource IDs/names, profile names, namespace names, files and option values are operands, not finite command tokens
- permissive handlers may ignore extra operands; these are parser accidents, not additional command paths
- `providers openai tags/version/ps` are explicitly rejected branches, not supported commands. They were separately probed and must not silently become features.
- `api` positional method variants GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS were individually help-probed with and without endpoint. Method values passed via `--method` are not finite command paths.
- `knowledge sync audit-operation audit-id-1` actually reached POST `/api/v1/knowledge/audit-id-1/sync/audit-operation`; arbitrary sync action acceptance is an existing dynamic boundary, not a missing finite command.
- Refusing fixture deliberately stops at the first request: no claim of end-to-end remote success, subsequent request sequences, server-side validation, or live acceptance. Filesystem snapshots cover content/add/remove in fake home/config/run directories; mount isolation separately prevents writes outside the task directory.

## Retained inventory

`inventory.json` retains complete command paths, aliases, option ownership, source anchors, exact help probe arguments/exit/outcomes and baseline classifications. The matrix above and this compact inventory are maintained planning evidence; full raw stdout/request logs and disposable audit tooling remain outside the repository. Counts are observations at the named baseline, not a permanent fixed command-count requirement.
