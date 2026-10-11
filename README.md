# oictl

`oictl` is a command-line client for [Open WebUI](https://github.com/open-webui/open-webui). It lets operators administer an instance, export resources, and automate configuration changes from a terminal or script.

Use it to:

- Inspect and manage models, knowledge bases, files, prompts, tools, functions, and skills.
- Administer users, groups, channels, and webhooks.
- Preview and apply declarative JSON/YAML resource manifests, including access grants.
- Export resources and configuration, inspect analytics, and work with automations, SCIM provisioning, and provider APIs through Open WebUI.

Commands connect to an existing Open WebUI instance. What you can access depends on your account permissions and the APIs available in that deployment.

## Build from source

Requires [Go](https://go.dev/dl/) 1.26.4 or later. Clone the repository and build:

```sh
git clone https://github.com/serge-medvedev/oictl.git
cd oictl
go build -o bin/oictl ./cmd/oictl
export PATH="$PWD/bin:$PATH"
oictl --help
```

The `PATH` setting makes the built binary available in the current shell.

## Quick start

You need your Open WebUI instance URL and an API key or JWT token. API keys are available under **Settings > Account** when enabled by your administrator and permitted for your account; see Open WebUI's [API key guide](https://docs.openwebui.com/features/authentication-access/api-keys). Administrative commands require the corresponding server permissions.

Set the URL and token in your shell, replacing the example values:

```sh
export OPEN_WEBUI_URL='https://openwebui.example.com'
export OPEN_WEBUI_API_KEY='YOUR_API_KEY'
```

Verify your identity and list available models:

```sh
oictl auth me
oictl models list --output table
```

Output defaults vary by command. Use `--output json` explicitly for structured results in scripts, or `--output table` for a readable view. Avoid putting real credentials in shared scripts or shell history. For repeated access or multiple instances, see [connection settings and profiles](docs/configuration.md).

## Typical workflows

Inspect resources and save exports:

```sh
oictl knowledge list
oictl models export --out models.json
oictl config export --out config.json
```

For declarative changes, write a resource manifest using the [manifest guide](docs/manifests.md), inspect the plan, then apply it:

```sh
oictl manifests diff --file prompt.json
oictl manifests apply --file prompt.json
```

`apply` creates or updates declared resources without deleting omitted resources. `sync` can also delete resources absent from the selected scope; inspect its plan with `--dry-run` before confirming deletions.

See [workflow examples](docs/workflows.md) for resource updates, exports, channels, webhooks, and integrations.

## Safety and compatibility

- Open WebUI API availability and permissions vary by deployment. Some commands require newer APIs or explicitly acknowledged deployment-specific extensions; see [API behavior and compatibility](docs/api-compatibility.md).
- JSON responses and exported files can contain credentials, personal data, or other sensitive values. Profile and export files are written with mode `0600`; protect them and keep them out of version control.
- Tool and function source runs as Python in Open WebUI. Deploy source only from trusted authors.
- Preview support is action-specific. Check command help before relying on `--dry-run`; unsupported actions reject it.

## Documentation and help

Command help works offline, without credentials or a server. Append `--help` or `-h` at any command level; action help includes flags, inputs, and JSON examples where applicable:

```sh
oictl users --help
oictl users create --help
oictl manifests apply --help
```

- [Connection settings and profiles](docs/configuration.md): authentication, configuration precedence, input/output, and timeouts.
- [Workflow examples](docs/workflows.md): commands grouped by task.
- [Declarative resource manifests](docs/manifests.md): authoring, access grants, diff, apply, and sync.
- [API behavior and compatibility](docs/api-compatibility.md): resource-specific contracts and deployment constraints.

Report bugs or request features through [GitHub issues](https://github.com/serge-medvedev/oictl/issues). Include the `oictl version` output, your Open WebUI version, and a reproduction with credentials and private data removed.

## Development

From the repository root:

```sh
go test ./...
go vet ./...
```

Contributions should follow the [repository instructions](AGENTS.md) and use Conventional Commits.

## License

[MIT](LICENSE).
