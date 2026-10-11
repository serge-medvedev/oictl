# Connection settings and profiles

[Project overview and quick start](../README.md) · [Workflow examples](workflows.md) · [API behavior and compatibility](api-compatibility.md)

## Connection settings

`oictl` connects to an existing Open WebUI instance with a bearer API key or JWT token. API keys must be enabled and allowed for your account; Open WebUI may also restrict which endpoints a key can access. See its [API key guide](https://docs.openwebui.com/features/authentication-access/api-keys).

Set connection details with environment variables:

```sh
export OPEN_WEBUI_URL='https://openwebui.example.com'
export OPEN_WEBUI_API_KEY='YOUR_API_KEY'
oictl auth me
```

Replace example values with your instance URL and token. Treat tokens as secrets; avoid storing them in shared scripts or shell history.

The base URL is resolved in this order:

1. `--base-url` (alias `--url`).
2. `OPEN_WEBUI_URL`.
3. `OPEN_WEBUI_BASE_URL`.
4. The selected profile's `base_url`.

The bearer token is resolved in this order:

1. `--token`.
2. `OPEN_WEBUI_API_KEY`.
3. The selected profile's `token`.

There is no built-in instance URL or credential. Flags override environment values, which override profile values.

## Named profiles

Use profiles for repeated access to one or more instances. With the connection variables above set:

```sh
oictl profiles set example --base-url "$OPEN_WEBUI_URL" --token "$OPEN_WEBUI_API_KEY"
oictl --profile example auth me
oictl profiles get example
```

Profiles are selected by `--profile`, then `OICTL_PROFILE`, then `default`. A saved default profile is used automatically. Environment connection values still override profile values, including when a profile is explicitly selected. Unset `OPEN_WEBUI_URL`, `OPEN_WEBUI_BASE_URL`, and `OPEN_WEBUI_API_KEY` if you want the selected profile to supply the connection.

Profiles are stored under `oictl/profiles/<name>.json` in the operating system's user configuration directory. On Linux this is normally `${XDG_CONFIG_HOME:-$HOME/.config}`. Profile files contain credentials in plaintext and are restricted to mode `0600`, including existing files before writing. Profile display redacts bearer and SCIM tokens.

### Login and logout

When your deployment supports email/password sign-in, prepare a protected `login.json` containing your credentials in this shape:

```json
{
  "email": "user@example.com",
  "password": "YOUR_PASSWORD"
}
```

Protect the file before adding real credentials. Login output includes the returned session token; direct it to a protected file rather than shared terminal logs:

```sh
chmod 600 login.json
oictl --profile example --base-url "$OPEN_WEBUI_URL" auth login --file login.json --save --out session.json
oictl --profile example auth me
```

`--save` persists the returned token; explicitly selecting `--profile` also enables persistence. Login persistence uses the same profile-selection order described above. An exported `OPEN_WEBUI_API_KEY` still overrides the newly saved token.

```sh
oictl --profile example auth logout
```

After remote logout succeeds, `logout` removes the selected profile's saved bearer token. It does not clear shell environment variables.

## Request input and offline help

Action help works without selectors, input files, connection configuration, or a server:

```sh
oictl users create --help
oictl channels messages update -h
oictl tools valves user update --help
oictl files upload --help
oictl manifests apply --help
```

JSON action help distinguishes CLI input from server defaults, replacement behavior, and resource-defined fields. A deployment can expose different fields.

For request-body actions, choose one source:

- `--data JSON`: literal JSON text.
- `--file PATH`: a body file.
- `--data-file PATH`: an alias for a body file.

Either body-file option reads stdin when the path is `-`. Manifest inputs are different: `manifests --file -` names a literal file, not stdin. See the [manifest guide](manifests.md) for file/directory envelopes and authoring rules.

Flags are validated for the selected action before requests. Boolean flags accept Go boolean spellings: `true`/`false`, `TRUE`/`FALSE`, `True`/`False`, `t`/`f`, `T`/`F`, and `1`/`0`. Invalid values fail locally.

`--dry-run` is supported for manifest apply/sync and extension-backed bulk UI settings patches. Other actions reject it; it is not a universal preview flag.

## Output and sensitive data

Output defaults vary by command: for example, channels, tools, skills, and manifest plans default to table output. Use `--output json` explicitly for structured results in scripts. Exports retain their documented formats, such as NDJSON for own chats. Select table output for interactive inspection, or save an export to a file:

```sh
oictl models list --output table
oictl models export --out models.json
```

JSON preserves complete server responses and may contain secrets. Export files can also contain personal data, source code, or sensitive configuration. Profile and export destinations are restricted to mode `0600` before writing, including existing files. Protect these files and exclude them from version control.

Webhook table output redacts channel webhook tokens, constructed incoming webhook URLs, and event destination URLs. This protection does not apply to complete JSON responses. See [webhook examples](workflows.md#channels-and-webhooks) for explicit URL export.

Output and persistence failures cause nonzero exit status. A false model-deletion result also causes a nonzero exit.

## Request timeouts

Ordinary requests have a 30-second deadline, including response reads. Override it with a positive duration:

```sh
oictl --timeout 2m models list
```

Explicit provider `--stream` requests have no default overall deadline. Use `--timeout` to bound a stream:

```sh
oictl --timeout 2m providers openai request --method POST /chat/completions --file request.json --stream
```

## SCIM credentials

SCIM provisioning uses a dedicated token supplied by `--scim-token`, `OPEN_WEBUI_SCIM_TOKEN`, or the selected profile's `scim_token`. It never falls back to the normal Open WebUI bearer token. Native `users` and `groups` commands use the normal token instead.

```sh
export OPEN_WEBUI_SCIM_TOKEN='YOUR_SCIM_TOKEN'
oictl scim users list --start-index 1 --count 50
```
