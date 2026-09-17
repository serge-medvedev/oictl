## 1. CLI Surface

- [x] 1.1 Add `api` to top-level command dispatch and help output.
- [x] 1.2 Add `oictl api --help` output documenting endpoint, `--url`, `--method`, `--header`, `--data`, `--data-file`, and `--api-key-header`.
- [x] 1.3 Parse `api` command flags and validate that exactly one endpoint path or URL argument is supplied.

## 2. Request Configuration

- [x] 2.1 Resolve the base URL from `--url` first and `OPEN_WEBUI_URL` second, with a clear error when neither is set.
- [x] 2.2 Resolve the API credential from `OPEN_WEBUI_API_KEY`, with a clear error when missing.
- [x] 2.3 Join base URLs and endpoint paths deterministically for leading/trailing slash combinations.
- [x] 2.4 Support `GET` as the default method and `--method` as an override.

## 3. HTTP Execution

- [x] 3.1 Add a small internal HTTP request executor reusable by future typed commands.
- [x] 3.2 Inject `Authorization: Bearer <credential>` by default.
- [x] 3.3 Inject the credential through `--api-key-header` when supplied and suppress automatic bearer injection in that mode.
- [x] 3.4 Support repeated `--header` values and reject malformed header input.
- [x] 3.5 Support inline request bodies from `--data` and file request bodies from `--data-file`.

## 4. Response And Error Handling

- [x] 4.1 Stream successful response bodies to stdout and exit 0 for 2xx responses.
- [x] 4.2 Write HTTP status and response body to stderr and exit non-zero for non-2xx responses.
- [x] 4.3 Write transport and validation errors to stderr and exit non-zero.
- [x] 4.4 Ensure CLI-generated output does not print credential values.

## 5. Tests And Documentation

- [x] 5.1 Add CLI tests for top-level help, `api --help`, missing URL, missing credential, and flag parsing.
- [x] 5.2 Add HTTP tests for URL joining, bearer auth, custom API key header auth, repeated headers, methods, body input, success output, HTTP error output, and transport failure handling.
- [x] 5.3 Update README usage with one `oictl api` example using `OPEN_WEBUI_URL` and `OPEN_WEBUI_API_KEY`.
- [x] 5.4 Run `go test ./...` and fix any failures.
