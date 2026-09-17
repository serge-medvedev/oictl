## Why

`oictl` needs a small authenticated HTTP foundation before higher-level Open WebUI resource commands can be built safely. A low-level `api` command gives operators and contributors an immediate scriptable escape hatch while establishing shared URL, credential, request, response, and error-handling behavior for Stage 1 Foundation Controls.

## What Changes

- Add a low-level `oictl api` command for authenticated HTTP requests to an Open WebUI instance.
- Support a configured or environment-provided base URL and API credential without exposing secrets in normal output.
- Send credentials using `Authorization: Bearer <token>` by default and support an explicit custom API key header for reverse-proxy deployments.
- Support common request controls needed by scripts: method selection, path/URL handling, headers, request body input, response output, and non-2xx failure behavior.
- Keep the command intentionally low-level; typed resource commands such as `models list` remain future Stage 1 work.
- No breaking changes.

## Capabilities

### New Capabilities

- `api-command`: Defines the authenticated low-level API command contract for direct Open WebUI HTTP requests.

### Modified Capabilities

- None.

## Impact

- Affected code: CLI command parsing and help, HTTP client/request execution, configuration/environment loading, credential handling, and tests.
- Affected APIs: Outbound calls to Open WebUI HTTP API endpoints; no new server API is introduced.
- Affected dependencies: May add a standard library-only implementation or minimal CLI/test dependencies if needed by the implementation.
- Affected systems: Local operator shell environments and scripts using `oictl` against one Open WebUI instance.
