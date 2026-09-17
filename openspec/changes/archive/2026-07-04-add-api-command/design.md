## Context

`oictl` currently has a small hand-rolled CLI skeleton with `help` and `version`. Open WebUI exposes HTTP API endpoints authenticated by API keys or JWTs, normally via `Authorization: Bearer <token>`, with a custom API key header available for reverse-proxy deployments that consume `Authorization`.

This change is Stage 1 foundation work. It should establish reusable request/auth behavior and provide an operator escape hatch, without committing to high-level resource commands or a full profile/configuration system yet.

## Goals / Non-Goals

**Goals:**

- Add `oictl api` as a scriptable low-level authenticated HTTP command.
- Resolve Open WebUI base URL and credential from explicit flags first, then environment variables.
- Support the authentication modes documented by Open WebUI: bearer auth by default and an explicit custom API key header mode.
- Preserve response bodies and status-oriented exit codes so shell scripts can use the command predictably.
- Keep the implementation small enough to fit the existing CLI structure and standard library.

**Non-Goals:**

- Add persistent profile/config commands.
- Add interactive login, API key creation, or browser-based authentication.
- Add typed resource commands such as `models list`, `users get`, or `groups create`.
- Reimplement the full `curl` surface area.

## Decisions

### Use a top-level `api` command

`oictl api` is a foundation/escape-hatch command rather than a resource family. This keeps it separate from future resource-first commands such as `oictl models list` while allowing contributors to exercise Open WebUI endpoints immediately.

Alternative considered: require all Stage 1 work to begin with typed commands. That would delay reusable HTTP/auth behavior and force early decisions about every resource surface.

### Use flags and environment variables before persistent config

The command should accept `--url` and read `OPEN_WEBUI_URL` when the flag is omitted. It should accept credentials through environment variables such as `OPEN_WEBUI_API_KEY`, with a flag available only when needed for non-interactive automation. Explicit flags win over environment values.

Alternative considered: introduce profiles and persistent config in the same change. That is broader than the requested low-level command and would make rollback/testing harder.

### Default to bearer authentication with explicit custom header override

By default, requests should include `Authorization: Bearer <token>`. If a caller supplies a custom API key header name, the command should send the credential in that header instead and not also add the bearer authorization header. This matches Open WebUI deployments where a reverse proxy owns `Authorization`.

Alternative considered: always send both headers. That can leak credentials to intermediaries unnecessarily and makes authentication failures harder to reason about.

### Keep HTTP execution in a small internal client boundary

Request construction, base URL joining, auth header injection, body handling, and response status mapping should live behind a small internal function or type instead of being embedded directly in the command switch. This gives future typed commands a single place to reuse and test HTTP behavior.

Alternative considered: implement everything inline in `Run`. That is initially shorter but makes future Stage 1 commands duplicate authentication and error handling.

### Preserve server output by default

Successful responses should write the response body to stdout. Failed HTTP responses should write the response body to stderr when present and return a non-zero exit code. The command should not pretty-print, redact, or transform arbitrary JSON by default because it is a low-level transport command.

Alternative considered: always parse and format JSON. That is nicer for humans but risks changing payloads and hides exact server responses needed by scripts.

## Risks / Trade-offs

- Secret exposure through command-line flags -> Prefer environment variables in documentation and tests; avoid printing credentials in help-expanded errors or diagnostics.
- Low-level command can bypass safer future typed UX -> Document it as an escape hatch and keep typed resource commands as separate future work.
- Path joining can produce surprising URLs -> Require deterministic handling for base URLs with or without trailing slashes and paths with or without leading slashes.
- Non-2xx status mapping loses exact HTTP status in exit code -> Print the status line to stderr and use a stable non-zero exit code for scripts.
- Custom headers can override important defaults -> Define user-supplied header precedence and test auth/header behavior explicitly.
