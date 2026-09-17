package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestTerminalServerAccessGrantsGetDiffSetAndPreserveConfig(t *testing.T) {
	dir := t.TempDir()
	desired := dir + "/grants.json"
	if err := os.WriteFile(desired, []byte(`{"access_grants":[{"principal_type":"group","principal_id":"ops","permission":"read"}]}`), 0o600); err != nil {
		t.Fatalf("write desired grants: %v", err)
	}
	requests := []string{}
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.URL.Path != "/api/v1/configs/terminal_servers" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `{"OTHER":{"keep":true},"TERMINAL_SERVER_CONNECTIONS":[{"id":"shell-a","name":"Shell A","url":"https://a","config":{"mode":"pty","access_grants":[{"principal_type":"user","principal_id":"u1","permission":"read"}]}},{"id":"shell-b","name":"Shell B","config":{"untouched":true}}]}`)
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		posts++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode post body: %v", err)
		}
		if _, ok := body["OTHER"].(map[string]any); !ok {
			t.Fatalf("unknown top-level config was not preserved: %#v", body)
		}
		connections := body[terminalServerConnectionsKey].([]any)
		first := connections[0].(map[string]any)
		config := first["config"].(map[string]any)
		if first["url"] != "https://a" || config["mode"] != "pty" {
			t.Fatalf("target non-grant fields were not preserved: %#v", first)
		}
		grants := config["access_grants"].([]any)
		grant := grants[0].(map[string]any)
		if grant["principal_type"] != "group" || grant["principal_id"] != "ops" || grant["permission"] != "read" {
			t.Fatalf("access grants not replaced: %#v", grants)
		}
		second := connections[1].(map[string]any)
		if second["id"] != "shell-b" || second["config"].(map[string]any)["untouched"] != true {
			t.Fatalf("other connection was not preserved: %#v", second)
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-terminal"})
	if code := app.Run(context.Background(), []string{"config", "terminal-servers", "access-grants", "get", "shell-a"}); code != 0 || !strings.Contains(out.String(), `"principal_id":"u1"`) {
		t.Fatalf("get code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}

	out.Reset()
	errOut.Reset()
	before := len(requests)
	if code := app.Run(context.Background(), []string{"config", "terminal-servers", "access-grants", "diff", "Shell A", "--file", desired}); code != 0 {
		t.Fatalf("diff code=%d stderr=%q", code, errOut.String())
	}
	if posts != 0 || containsRequest(requests[before:], "POST ") || !strings.Contains(out.String(), `"grants_added"`) || !strings.Contains(out.String(), `"grants_removed"`) {
		t.Fatalf("diff output=%q posts=%d requests=%v", out.String(), posts, requests[before:])
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"config", "terminal-servers", "access-grants", "set", "shell-a", "--file", desired}); code != 0 || !strings.Contains(out.String(), `"access_grants"`) {
		t.Fatalf("set code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	if posts != 1 {
		t.Fatalf("posts = %d, want 1", posts)
	}
}

func TestTerminalServerConnectionLookupAndGrantParsing(t *testing.T) {
	connections := []map[string]any{
		{"id": "id-a", "name": "same", "config": map[string]any{"access_grants": []any{map[string]any{"principal_type": "user", "principal_id": "*", "permission": "read"}}}},
		{"id": "id-b", "name": "same"},
		{"id": "id-c", "name": "unique", "config": map[string]any{}},
	}
	if _, conn, err := lookupTerminalServerConnection(connections, "id-a"); err != nil || conn["id"] != "id-a" {
		t.Fatalf("lookup by id conn=%#v err=%v", conn, err)
	}
	if _, conn, err := lookupTerminalServerConnection(connections, "unique"); err != nil || conn["id"] != "id-c" {
		t.Fatalf("lookup by unique name conn=%#v err=%v", conn, err)
	}
	if _, _, err := lookupTerminalServerConnection(connections, "missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("not found error = %v", err)
	}
	if _, _, err := lookupTerminalServerConnection(connections, "same"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous error = %v", err)
	}
	if grants := terminalServerAccessGrants(connections[1]); len(grants) != 0 {
		t.Fatalf("missing config grants = %#v", grants)
	}
	if grants := terminalServerAccessGrants(connections[2]); len(grants) != 0 {
		t.Fatalf("empty access_grants = %#v", grants)
	}

	arrayGrants, err := decodeDesiredAccessGrants([]byte(`[{"principal_type":"user","principal_id":"*","permission":"read"},{"principal_type":"user","principal_id":"*","permission":"read"}]`))
	if err != nil || len(arrayGrants) != 1 || arrayGrants[0].PrincipalID != "*" {
		t.Fatalf("array grants = %#v err=%v", arrayGrants, err)
	}
	objectGrants, err := decodeDesiredAccessGrants([]byte(`{"access_grants":[]}`))
	if err != nil || len(objectGrants) != 0 {
		t.Fatalf("private grants = %#v err=%v", objectGrants, err)
	}
	if _, err := decodeDesiredAccessGrants([]byte(`{"access_grants":[{"principal_type":"role","principal_id":"admin","permission":"read"}]}`)); err == nil || !strings.Contains(err.Error(), "user and group") {
		t.Fatalf("invalid principal error = %v", err)
	}
	if _, err := decodeDesiredAccessGrants([]byte(`{"access_grants":[{"principal_type":"user","principal_id":"u1","permission":"admin"}]}`)); err == nil || !strings.Contains(err.Error(), "read and write") {
		t.Fatalf("invalid permission error = %v", err)
	}
}

func TestTerminalServerAccessGrantSetReportsServerFilteredResponse(t *testing.T) {
	desired := t.TempDir() + "/grants.json"
	if err := os.WriteFile(desired, []byte(`[{"principal_type":"user","principal_id":"*","permission":"read"}]`), 0o600); err != nil {
		t.Fatalf("write desired grants: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"TERMINAL_SERVER_CONNECTIONS":[{"id":"shell-a","name":"Shell A","config":{"access_grants":[]}}]}`)
		case http.MethodPost:
			_, _ = io.WriteString(w, `{"TERMINAL_SERVER_CONNECTIONS":[{"id":"shell-a","name":"Shell A","config":{"access_grants":[]}}]}`)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-terminal"})
	if code := app.Run(context.Background(), []string{"config", "terminal-servers", "access-grants", "set", "shell-a", "--file", desired}); code != 0 {
		t.Fatalf("set code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "server_grants_missing") || !strings.Contains(out.String(), `"principal_id":"*"`) {
		t.Fatalf("filtered output = %q", out.String())
	}
}

func newTestApp(env map[string]string) (*App, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	err := &bytes.Buffer{}
	app := New(out, err, "test")
	app.getenv = func(key string) string { return env[key] }
	return app, out, err
}

func TestTopLevelHelpListsAPICommand(t *testing.T) {
	app, out, errOut := newTestApp(nil)

	code := app.Run(context.Background(), []string{"--help"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "api") || !strings.Contains(out.String(), "Make an authenticated Open WebUI API request") {
		t.Fatalf("help output does not list api command:\n%s", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}

func TestAffectedHelpUsesSpaceIndentedRows(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "top level",
			args: []string{"--help"},
			want: []string{
				"  tasks        Manage Open WebUI task configuration",
				"  tools        Manage Open WebUI tools",
				"  users        Manage Open WebUI user settings",
				"  webhooks     Manage channel incoming webhooks and global event webhooks",
			},
		},
		{
			name: "tasks",
			args: []string{"tasks", "--help"},
			want: []string{
				"  oictl tasks config get",
				"  oictl tasks config set --file tasks-config.json",
				"  oictl tasks skills attach [--external] <skill-id-or-name>...",
				"TASK_MODEL maps to task.model.default",
			},
		},
		{
			name: "tools",
			args: []string{"tools", "--help"},
			want: []string{
				"  list          List tools",
				"  access-update Replace access grants from --data or --file JSON",
				"  oictl tools export --manifest --directory manifests/tools",
			},
		},
		{
			name: "users",
			args: []string{"users", "--help"},
			want: []string{
				"  settings get       Fetch the authenticated user's settings",
				"  ui-settings patch  Patch another user's settings.ui map as an admin",
				"Sensitive UI keys such as toolServers require --allow-sensitive-ui-keys.",
			},
		},
		{
			name: "webhooks",
			args: []string{"webhooks", "--help"},
			want: []string{
				"  channels list <channel-id>                              List incoming webhooks with token values redacted",
				"  events catalog                                           List event catalog entries",
				"  events delete <webhook-id> --yes                         Delete an event webhook",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, out, errOut := newTestApp(nil)

			code := app.Run(context.Background(), tt.args)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
			}
			assertHelpHasNoTabs(t, out.String())
			for _, want := range tt.want {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("help output missing %q:\n%s", want, out.String())
				}
			}
		})
	}
}

func assertHelpHasNoTabs(t *testing.T, output string) {
	t.Helper()
	for i, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "\t") {
			t.Fatalf("help output contains a tab on line %d: %q\n%s", i+1, line, output)
		}
	}
}

func TestAPIHelpDocumentsRequestFlags(t *testing.T) {
	app, out, errOut := newTestApp(nil)

	code := app.Run(context.Background(), []string{"api", "--help"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	for _, want := range []string{"<endpoint>", "--url", "--method", "--header", "--data", "--data-file", "--api-key-header", "OPEN_WEBUI_API_KEY"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("api help missing %q:\n%s", want, out.String())
		}
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}

func TestAPIValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		args []string
		want string
	}{
		{
			name: "missing endpoint",
			env:  map[string]string{"OPEN_WEBUI_URL": "http://example.test", "OPEN_WEBUI_API_KEY": "secret"},
			args: []string{"api"},
			want: "exactly one endpoint",
		},
		{
			name: "too many endpoints",
			env:  map[string]string{"OPEN_WEBUI_URL": "http://example.test", "OPEN_WEBUI_API_KEY": "secret"},
			args: []string{"api", "/one", "/two"},
			want: "exactly one endpoint",
		},
		{
			name: "missing url",
			env:  map[string]string{"OPEN_WEBUI_API_KEY": "secret"},
			args: []string{"api", "/api/models"},
			want: "Open WebUI URL is required",
		},
		{
			name: "missing credential",
			env:  map[string]string{"OPEN_WEBUI_URL": "http://example.test"},
			args: []string{"api", "/api/models"},
			want: "OPEN_WEBUI_API_KEY is required",
		},
		{
			name: "empty custom api key header",
			env:  map[string]string{"OPEN_WEBUI_URL": "http://example.test", "OPEN_WEBUI_API_KEY": "secret"},
			args: []string{"api", "--api-key-header", "", "/api/models"},
			want: "custom API key header name cannot be empty",
		},
		{
			name: "malformed header",
			env:  map[string]string{"OPEN_WEBUI_URL": "http://example.test", "OPEN_WEBUI_API_KEY": "secret"},
			args: []string{"api", "--header", "bad", "/api/models"},
			want: "malformed header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, out, errOut := newTestApp(tt.env)

			code := app.Run(context.Background(), tt.args)

			if code == 0 {
				t.Fatalf("exit code = 0, want non-zero")
			}
			if out.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", out.String())
			}
			if !strings.Contains(errOut.String(), tt.want) {
				t.Fatalf("stderr missing %q:\n%s", tt.want, errOut.String())
			}
			if strings.Contains(errOut.String(), "secret") {
				t.Fatalf("stderr leaked credential: %q", errOut.String())
			}
		})
	}
}

func TestResolveTargetJoinsBaseAndEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		flagURL  string
		envURL   string
		endpoint string
		want     string
	}{
		{name: "plain", envURL: "http://localhost:3000", endpoint: "api/models", want: "http://localhost:3000/api/models"},
		{name: "base trailing slash", envURL: "http://localhost:3000/", endpoint: "api/models", want: "http://localhost:3000/api/models"},
		{name: "endpoint leading slash", envURL: "http://localhost:3000", endpoint: "/api/models", want: "http://localhost:3000/api/models"},
		{name: "both slashes", envURL: "http://localhost:3000/", endpoint: "/api/models", want: "http://localhost:3000/api/models"},
		{name: "flag wins", flagURL: "http://flag.test", envURL: "http://env.test", endpoint: "/api/models", want: "http://flag.test/api/models"},
		{name: "absolute endpoint", envURL: "http://env.test", endpoint: "http://other.test/api/models", want: "http://other.test/api/models"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveTarget(tt.flagURL, tt.envURL, tt.endpoint)
			if err != nil {
				t.Fatalf("resolveTarget returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveTarget = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPISuccessUsesDefaultBearerAuthAndGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/models" {
			t.Fatalf("path = %q, want /api/models", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})

	code := app.Run(context.Background(), []string{"api", "/api/models"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
	if out.String() != `{"ok":true}` {
		t.Fatalf("stdout = %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}

func TestAPICustomAPIKeyHeaderSuppressesBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "sk-test" {
			t.Fatalf("x-api-key = %q, want credential", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty", got)
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})

	code := app.Run(context.Background(), []string{"api", "--api-key-header", "x-api-key", "/api/models"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
}

func TestAPIRequestConstructionSupportsHeadersMethodAndInlineBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != `{"name":"example"}` {
			t.Fatalf("body = %q", string(body))
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})

	code := app.Run(context.Background(), []string{
		"api",
		"--method", "post",
		"--header", "Accept:application/json",
		"--header", "Content-Type:application/json",
		"--data", `{"name":"example"}`,
		"/api/example",
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
}

func TestAPIRequestConstructionSupportsDataFile(t *testing.T) {
	payload := t.TempDir() + "/payload.json"
	if err := os.WriteFile(payload, []byte(`{"from":"file"}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != `{"from":"file"}` {
			t.Fatalf("body = %q", string(body))
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})

	code := app.Run(context.Background(), []string{"api", "--method", "POST", "--data-file", payload, "/api/example"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
}

func TestAPIHTTPErrorWritesStatusAndBodyToStderr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "unauthorized")
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})

	code := app.Run(context.Background(), []string{"api", "/api/models"})

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
	for _, want := range []string{"401 Unauthorized", "unauthorized"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, errOut.String())
		}
	}
	if strings.Contains(errOut.String(), "sk-test") {
		t.Fatalf("stderr leaked credential: %q", errOut.String())
	}
}

func TestAPITransportFailureWritesErrorToStderr(t *testing.T) {
	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": "http://example.test", "OPEN_WEBUI_API_KEY": "sk-test"})
	app.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("dial failed")
	})}

	code := app.Run(context.Background(), []string{"api", "/api/models"})

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "dial failed") {
		t.Fatalf("stderr missing transport error:\n%s", errOut.String())
	}
	if strings.Contains(errOut.String(), "sk-test") {
		t.Fatalf("stderr leaked credential: %q", errOut.String())
	}
}

func TestProfileTargetResolutionAndRedaction(t *testing.T) {
	app, out, errOut := newTestApp(nil)
	configDir := t.TempDir()
	app.userConfigDir = func() (string, error) { return configDir, nil }

	if code := app.Run(context.Background(), []string{"profiles", "set", "prod", "--base-url", "http://example.test", "--token", "sk-secret"}); code != 0 {
		t.Fatalf("profiles set exit code = %d; stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()

	if code := app.Run(context.Background(), []string{"profiles", "get", "prod"}); code != 0 {
		t.Fatalf("profiles get exit code = %d; stderr=%q", code, errOut.String())
	}
	if strings.Contains(out.String(), "sk-secret") || !strings.Contains(out.String(), "<redacted>") {
		t.Fatalf("profile output did not redact token: %q", out.String())
	}
}

func TestTypedCommandUsesGlobalTargetAndToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models/tags" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-global" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = io.WriteString(w, `["ops"]`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(nil)
	code := app.Run(context.Background(), []string{"--base-url", server.URL, "--token", "sk-global", "models", "tags"})

	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if out.String() != `["ops"]` {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestTasksConfigGetUsesAuthenticatedTasksConfigEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tasks/config" || r.Method != http.MethodGet {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-tasks" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = io.WriteString(w, `{"TASK_MODEL":"gpt-4.1","TASK_MODEL_EXTERNAL":"gpt-4.1-mini"}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-tasks"})
	if code := app.Run(context.Background(), []string{"tasks", "config", "get"}); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if out.String() != `{"TASK_MODEL":"gpt-4.1","TASK_MODEL_EXTERNAL":"gpt-4.1-mini"}` {
		t.Fatalf("stdout = %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}

func TestTasksConfigSetForwardsJSONFileToUpdateEndpoint(t *testing.T) {
	payload := t.TempDir() + "/tasks-config.json"
	if err := os.WriteFile(payload, []byte(`{"TASK_MODEL":"ops","TASK_MODEL_EXTERNAL":"ops-external"}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tasks/config/update" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-tasks" {
			t.Fatalf("Authorization = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != `{"TASK_MODEL":"ops","TASK_MODEL_EXTERNAL":"ops-external"}` {
			t.Fatalf("body = %q", string(body))
		}
		_, _ = io.WriteString(w, `{"TASK_MODEL":"ops","TASK_MODEL_EXTERNAL":"ops-external","updated":true}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-tasks"})
	if code := app.Run(context.Background(), []string{"tasks", "config", "set", "--file", payload}); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"updated":true`) {
		t.Fatalf("stdout = %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}

func TestTasksConfigInvalidUsage(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"tasks", "unknown"}, want: `unknown tasks command "unknown"`},
		{args: []string{"tasks", "config"}, want: "usage: oictl tasks config"},
		{args: []string{"tasks", "config", "unknown"}, want: `unknown tasks config operation "unknown"`},
		{args: []string{"tasks", "config", "set"}, want: "a task config payload is required"},
		{args: []string{"tasks", "skills"}, want: "usage: oictl tasks skills attach"},
		{args: []string{"tasks", "skills", "attach"}, want: "usage: oictl tasks skills attach"},
		{args: []string{"tasks", "skills", "remove", "review-helper"}, want: `unknown tasks skills command "remove"`},
	}
	for _, tt := range tests {
		app, out, errOut := newTestApp(nil)
		if code := app.Run(context.Background(), tt.args); code == 0 {
			t.Fatalf("%v exit code = 0, want non-zero", tt.args)
		}
		if out.Len() != 0 {
			t.Fatalf("%v stdout = %q, want empty", tt.args, out.String())
		}
		if !strings.Contains(errOut.String(), tt.want) {
			t.Fatalf("%v stderr missing %q:\n%s", tt.args, tt.want, errOut.String())
		}
	}
}

func TestTasksSkillsAttachUpdatesConfiguredTaskModel(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantModel string
		wantIDs   []string
	}{
		{name: "default model", args: []string{"tasks", "skills", "attach", "review-helper", "skill-ops"}, wantModel: "model-default", wantIDs: []string{"skill-existing", "skill-review", "skill-ops"}},
		{name: "external model", args: []string{"tasks", "skills", "attach", "--external", "review-helper"}, wantModel: "model-external", wantIDs: []string{"skill-existing", "skill-review"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer sk-tasks" {
					t.Fatalf("Authorization = %q", got)
				}
				switch r.Method + " " + r.URL.Path {
				case "GET /api/v1/tasks/config":
					_, _ = io.WriteString(w, `{"TASK_MODEL":"model-default","TASK_MODEL_EXTERNAL":"model-external"}`)
				case "GET /api/v1/models/model":
					if got := r.URL.Query().Get("id"); got != tt.wantModel {
						t.Fatalf("model id = %q, want %q", got, tt.wantModel)
					}
					_, _ = io.WriteString(w, `{"id":"`+tt.wantModel+`","base_model_id":"base-model","name":"Task Model","meta":{"description":"keep","skillIds":["skill-existing","skill-review"]},"params":{"temperature":0.2},"is_active":false,"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)
				case "GET /api/v1/skills/list":
					_, _ = io.WriteString(w, `[{"id":"skill-review","name":"review-helper"},{"id":"skill-ops","name":"Ops Helper"}]`)
				case "POST /api/v1/models/model/update":
					updated = true
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Fatalf("decode update: %v", err)
					}
					if payload["id"] != tt.wantModel || payload["name"] != "Task Model" || payload["base_model_id"] != "base-model" || payload["is_active"] != false {
						t.Fatalf("payload did not preserve model fields: %#v", payload)
					}
					meta, ok := payload["meta"].(map[string]any)
					if !ok || meta["description"] != "keep" {
						t.Fatalf("payload meta = %#v", payload["meta"])
					}
					gotIDs, ok := meta["skillIds"].([]any)
					if !ok || len(gotIDs) != len(tt.wantIDs) {
						t.Fatalf("skillIds = %#v, want %#v", meta["skillIds"], tt.wantIDs)
					}
					for i, want := range tt.wantIDs {
						if gotIDs[i] != want {
							t.Fatalf("skillIds[%d] = %#v, want %q in %#v", i, gotIDs[i], want, gotIDs)
						}
					}
					params, ok := payload["params"].(map[string]any)
					if !ok || params["temperature"] != 0.2 {
						t.Fatalf("payload params = %#v", payload["params"])
					}
					_, _ = io.WriteString(w, `{"id":"`+tt.wantModel+`","updated":true}`)
				default:
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
				}
			}))
			defer server.Close()

			app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-tasks"})
			if code := app.Run(context.Background(), tt.args); code != 0 {
				t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
			}
			if !updated {
				t.Fatal("model update was not submitted")
			}
			if !strings.Contains(out.String(), `"updated":true`) {
				t.Fatalf("stdout = %q", out.String())
			}
			if errOut.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", errOut.String())
			}
		})
	}
}

func TestTasksSkillsAttachValidationFailsBeforeMutation(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		config    string
		model     string
		skills    string
		wantError string
	}{
		{name: "missing task model config", args: []string{"tasks", "skills", "attach", "review-helper"}, config: `{"TASK_MODEL":""}`, wantError: "task config TASK_MODEL is empty"},
		{name: "unknown skill", args: []string{"tasks", "skills", "attach", "missing"}, config: `{"TASK_MODEL":"model-a"}`, model: `{"id":"model-a","name":"Task Model","meta":{"skillIds":[]},"params":{}}`, skills: `[{"id":"skill-a","name":"Review"}]`, wantError: `skill "missing" was not found`},
		{name: "ambiguous skill name", args: []string{"tasks", "skills", "attach", "Review"}, config: `{"TASK_MODEL":"model-a"}`, model: `{"id":"model-a","name":"Task Model","meta":{"skillIds":[]},"params":{}}`, skills: `[{"id":"skill-a","name":"Review"},{"id":"skill-b","name":"Review"}]`, wantError: `skill name "Review" is ambiguous`},
		{name: "missing model", args: []string{"tasks", "skills", "attach", "Review"}, config: `{"TASK_MODEL":"model-a"}`, model: `null`, wantError: `task model "model-a" was not found`},
		{name: "malformed skill ids", args: []string{"tasks", "skills", "attach", "Review"}, config: `{"TASK_MODEL":"model-a"}`, model: `{"id":"model-a","name":"Task Model","meta":{"skillIds":["skill-a",1]},"params":{}}`, skills: `[{"id":"skill-b","name":"Review"}]`, wantError: "task model meta.skillIds must be an array of strings"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method + " " + r.URL.Path {
				case "GET /api/v1/tasks/config":
					_, _ = io.WriteString(w, tt.config)
				case "GET /api/v1/models/model":
					if tt.model == "" {
						t.Fatalf("model endpoint should not be called")
					}
					_, _ = io.WriteString(w, tt.model)
				case "GET /api/v1/skills/list":
					if tt.skills == "" {
						t.Fatalf("skills endpoint should not be called")
					}
					_, _ = io.WriteString(w, tt.skills)
				case "POST /api/v1/models/model/update":
					updated = true
					_, _ = io.WriteString(w, `{"updated":true}`)
				default:
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
				}
			}))
			defer server.Close()

			app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-tasks"})
			if code := app.Run(context.Background(), tt.args); code == 0 {
				t.Fatalf("exit code = 0, want non-zero")
			}
			if updated {
				t.Fatal("model update was submitted after validation failure")
			}
			if out.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", out.String())
			}
			if !strings.Contains(errOut.String(), tt.wantError) {
				t.Fatalf("stderr missing %q:\n%s", tt.wantError, errOut.String())
			}
		})
	}
}

func TestUsersSettingsGetAndUpdateRequests(t *testing.T) {
	payload := t.TempDir() + "/settings.json"
	if err := os.WriteFile(payload, []byte(`{"ui":{"theme":"dark"},"profile":{"name":"Ops"}}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if got := r.Header.Get("Authorization"); got != "Bearer sk-users" {
			t.Fatalf("Authorization = %q", got)
		}
		switch r.URL.Path {
		case "/api/v1/users/user/settings":
			if r.Method != http.MethodGet {
				t.Fatalf("settings get method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"ui":{"theme":"light"}}`)
		case "/api/v1/users/user/settings/update":
			if r.Method != http.MethodPost {
				t.Fatalf("settings update method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"ui":{"theme":"dark"},"profile":{"name":"Ops"}}` {
				t.Fatalf("settings update body = %q", string(body))
			}
			_, _ = io.WriteString(w, `{"updated":true}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-users"})
	if code := app.Run(context.Background(), []string{"users", "settings", "get"}); code != 0 {
		t.Fatalf("get code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"theme":"light"`) {
		t.Fatalf("get stdout = %q", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "settings", "update", "--file", payload}); code != 0 {
		t.Fatalf("update code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"updated":true`) {
		t.Fatalf("update stdout = %q", out.String())
	}
	for _, want := range []string{"GET /api/v1/users/user/settings", "POST /api/v1/users/user/settings/update"} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}
}

func TestUsersUISettingsPatchRequestAndValidation(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.URL.EscapedPath() != "/api/v1/users/user%2Fone/settings/ui" || r.Method != http.MethodPatch {
			t.Fatalf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"theme":"dark"}` {
			t.Fatalf("patch body = %q", string(body))
		}
		_, _ = io.WriteString(w, `{"patched":true}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-users"})
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "patch", "--allow-ui-settings-extension", "user/one", "--data", `{"theme":"dark"}`}); code != 0 {
		t.Fatalf("patch code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"patched":true`) {
		t.Fatalf("patch stdout = %q", out.String())
	}

	for _, tt := range []struct {
		args []string
		want string
	}{
		{args: []string{"users", "ui-settings", "patch", "--allow-ui-settings-extension", "--data", `{"theme":"dark"}`}, want: "usage: oictl users ui-settings patch <user-id>"},
		{args: []string{"users", "ui-settings", "patch", "--allow-ui-settings-extension", "user-1"}, want: "a UI settings payload is required"},
		{args: []string{"users", "settings", "update"}, want: "a settings payload is required"},
		{args: []string{"users", "settings"}, want: "usage: oictl users settings <get|update>"},
	} {
		out.Reset()
		errOut.Reset()
		before := len(requests)
		if code := app.Run(context.Background(), tt.args); code == 0 {
			t.Fatalf("%v code=0, want non-zero", tt.args)
		}
		if out.Len() != 0 {
			t.Fatalf("%v stdout = %q, want empty", tt.args, out.String())
		}
		if !strings.Contains(errOut.String(), tt.want) {
			t.Fatalf("%v stderr missing %q:\n%s", tt.args, tt.want, errOut.String())
		}
		if len(requests) != before {
			t.Fatalf("%v contacted server", tt.args)
		}
	}
}

func TestUsersSettingsSensitiveKeyGuard(t *testing.T) {
	sent := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent++
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "tool-secret") {
			t.Fatalf("body missing allowed sensitive value: %q", string(body))
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-users"})
	if code := app.Run(context.Background(), []string{"users", "settings", "update", "--data", `{"ui":{"toolServers":{"x":"tool-secret"}}}`}); code == 0 {
		t.Fatalf("settings sensitive update code=0, want non-zero")
	}
	if sent != 0 {
		t.Fatalf("sensitive settings update contacted server")
	}
	if !strings.Contains(errOut.String(), "toolServers") || !strings.Contains(errOut.String(), "--allow-sensitive-ui-keys") {
		t.Fatalf("sensitive settings stderr = %q", errOut.String())
	}
	if strings.Contains(errOut.String(), "tool-secret") {
		t.Fatalf("sensitive value leaked: %q", errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "patch", "--allow-ui-settings-extension", "user-1", "--data", `{"toolServers":{"x":"tool-secret"}}`}); code == 0 {
		t.Fatalf("ui-settings sensitive patch code=0, want non-zero")
	}
	if sent != 0 {
		t.Fatalf("sensitive ui-settings patch contacted server")
	}
	if !strings.Contains(errOut.String(), "toolServers") || strings.Contains(errOut.String(), "tool-secret") {
		t.Fatalf("ui-settings sensitive stderr = %q", errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "patch", "--allow-ui-settings-extension", "user-1", "--allow-sensitive-ui-keys", "--data", `{"toolServers":{"x":"tool-secret"}}`}); code != 0 {
		t.Fatalf("allowed sensitive patch code=%d stderr=%q", code, errOut.String())
	}
	if sent != 1 {
		t.Fatalf("allowed sensitive patch sent = %d, want 1", sent)
	}
}

func TestUsersSettingsHTTPErrorStatusAndCredentialRedaction(t *testing.T) {
	responses := []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := responses[0]
		responses = responses[1:]
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"detail":"denied sk-users"}`)
	}))
	defer server.Close()

	commands := [][]string{
		{"users", "settings", "get"},
		{"users", "settings", "update", "--data", `{"ui":{"theme":"dark"}}`},
		{"users", "ui-settings", "patch", "--allow-ui-settings-extension", "missing-user", "--data", `{"theme":"dark"}`},
	}
	for i, args := range commands {
		app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-users"})
		if code := app.Run(context.Background(), args); code == 0 {
			t.Fatalf("%v code=0, want non-zero", args)
		}
		if out.Len() != 0 {
			t.Fatalf("%v stdout = %q, want empty", args, out.String())
		}
		if !strings.Contains(errOut.String(), http.StatusText([]int{401, 403, 404}[i])) || !strings.Contains(errOut.String(), "denied") {
			t.Fatalf("%v stderr missing status/detail: %q", args, errOut.String())
		}
		if strings.Contains(errOut.String(), "sk-users") {
			t.Fatalf("%v leaked credential: %q", args, errOut.String())
		}
	}
}

func TestUsersDirectoryListSearchRequestsHelpAndAuthorization(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.EscapedPath()+querySuffix(r.URL.RawQuery))
		if got := r.Header.Get("Authorization"); got != "Bearer sk-users" {
			t.Fatalf("Authorization = %q", got)
		}
		switch r.URL.EscapedPath() {
		case "/api/v1/users/":
			if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("query") != "alice" || r.URL.Query().Get("order_by") != "name" || r.URL.Query().Get("direction") != "asc" {
				t.Fatalf("list query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"users":[{"id":"user-1"}]}`)
		case "/api/v1/users/search":
			if r.URL.Query().Get("query") == "denied" {
				w.WriteHeader(http.StatusForbidden)
				_, _ = io.WriteString(w, `{"detail":"denied sk-users"}`)
				return
			}
			if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("query") != "alice" || r.URL.Query().Get("order_by") != "email" || r.URL.Query().Get("direction") != "desc" {
				t.Fatalf("search query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"users":[{"id":"user-2"}]}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-users"})
	if code := app.Run(context.Background(), []string{"users", "list", "--page", "2", "--query", "alice", "--order-by", "name", "--direction", "asc"}); code != 0 {
		t.Fatalf("list code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"user-1"`) {
		t.Fatalf("list stdout = %q", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "search", "--query", "alice", "--page", "1", "--order-by", "email", "--direction", "desc"}); code != 0 {
		t.Fatalf("search code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"user-2"`) {
		t.Fatalf("search stdout = %q", out.String())
	}
	for _, want := range []string{"GET /api/v1/users/?direction=asc&order_by=name&page=2&query=alice", "GET /api/v1/users/search?direction=desc&order_by=email&page=1&query=alice"} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "help"}); code != 0 {
		t.Fatalf("help code=%d stderr=%q", code, errOut.String())
	}
	for _, want := range []string{
		"users list", "users search", "users get <user-id>",
		"users create (--data JSON | --file path | --file -)",
		"users update <user-id> (--data JSON | --file path | --file -)",
		"users delete <user-id> (--yes | --confirm)",
		"users settings get", "users settings update", "users ui-settings patch", "users ui-settings bulk-patch",
		"--query text", "--page n", "--order-by field", "--direction asc|desc",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %q:\n%s", want, out.String())
		}
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "search", "--query", "denied"}); code == 0 {
		t.Fatalf("denied search code=0, want non-zero")
	}
	if out.Len() != 0 {
		t.Fatalf("denied stdout=%q, want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "403 Forbidden") || !strings.Contains(errOut.String(), "denied") || !strings.Contains(errOut.String(), "<redacted>") || strings.Contains(errOut.String(), "sk-users") {
		t.Fatalf("denied stderr = %q", errOut.String())
	}
}

func TestUsersCRUDRequests(t *testing.T) {
	createPayload := []byte(`{"name":"New User","email":"new@example.test","password":"sensitive"}`)
	updatePayload := []byte(`{"name":"Updated User","role":"admin"}`)
	createFile := t.TempDir() + "/user.json"
	if err := os.WriteFile(createFile, createPayload, 0o600); err != nil {
		t.Fatalf("write create payload: %v", err)
	}
	if info, err := os.Stat(createFile); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("create payload mode: info=%v err=%v", info, err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.EscapedPath())
		if got := r.Header.Get("Authorization"); got != "Bearer sk-users-crud" {
			t.Fatalf("Authorization = %q", got)
		}
		switch r.Method + " " + r.URL.EscapedPath() {
		case "GET /api/v1/users/user%2Fone":
			_, _ = io.WriteString(w, `{"id":"user/one","name":"Operator","extra":{"preserved":true}}`)
		case "POST /api/v1/auths/add":
			body, _ := io.ReadAll(r.Body)
			if !bytes.Equal(body, createPayload) {
				t.Fatalf("create body = %q", body)
			}
			_, _ = io.WriteString(w, `{"id":"created","token":"issued-token"}`)
		case "POST /api/v1/users/user%2Ftwo/update":
			body, _ := io.ReadAll(r.Body)
			if !bytes.Equal(body, updatePayload) {
				t.Fatalf("update body = %q", body)
			}
			_, _ = io.WriteString(w, `{"id":"user/two","name":"Updated User","role":"admin"}`)
		case "DELETE /api/v1/users/user%2Fthree":
			_, _ = io.WriteString(w, `{"deleted":true,"id":"user/three"}`)
		default:
			t.Fatalf("request = %s %s", r.Method, r.URL.EscapedPath())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-users-crud"})
	if code := app.Run(context.Background(), []string{"users", "get", "user/one"}); code != 0 {
		t.Fatalf("get code=%d stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != `{"id":"user/one","name":"Operator","extra":{"preserved":true}}` {
		t.Fatalf("get stdout = %q", got)
	}
	if !containsRequest(requests, "GET /api/v1/users/user%2Fone") {
		t.Fatalf("get requests = %v", requests)
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "create", "--file", createFile}); code != 0 {
		t.Fatalf("create code=%d stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != `{"id":"created","token":"issued-token"}` {
		t.Fatalf("create stdout = %q", got)
	}
	if !containsRequest(requests, "POST /api/v1/auths/add") {
		t.Fatalf("create requests = %v", requests)
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "update", "user/two", "--data", string(updatePayload)}); code != 0 {
		t.Fatalf("update code=%d stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != `{"id":"user/two","name":"Updated User","role":"admin"}` {
		t.Fatalf("update stdout = %q", got)
	}
	if !containsRequest(requests, "POST /api/v1/users/user%2Ftwo/update") {
		t.Fatalf("update requests = %v", requests)
	}

	out.Reset()
	errOut.Reset()
	before := len(requests)
	if code := app.Run(context.Background(), []string{"users", "delete", "user/three"}); code == 0 {
		t.Fatal("unconfirmed delete code=0, want non-zero")
	}
	if len(requests) != before {
		t.Fatalf("unconfirmed delete sent requests: %v", requests[before:])
	}
	if !strings.Contains(errOut.String(), "requires --yes") {
		t.Fatalf("unconfirmed delete stderr = %q", errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "delete", "user/three", "--confirm"}); code != 0 {
		t.Fatalf("delete code=%d stderr=%q", code, errOut.String())
	}
	if got := out.String(); got != `{"deleted":true,"id":"user/three"}` {
		t.Fatalf("delete stdout = %q", got)
	}
	if !containsRequest(requests, "DELETE /api/v1/users/user%2Fthree") {
		t.Fatalf("delete requests = %v", requests)
	}
}

func TestUsersCRUDLocalValidationDoesNotContactServer(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "get missing id", args: []string{"users", "get"}, want: "usage: oictl users get <user-id>"},
		{name: "get extra id", args: []string{"users", "get", "one", "two"}, want: "usage: oictl users get <user-id>"},
		{name: "create missing payload", args: []string{"users", "create"}, want: "a user payload is required"},
		{name: "create positional", args: []string{"users", "create", "user", "--data", `{}`}, want: "usage: oictl users create"},
		{name: "update missing id", args: []string{"users", "update", "--data", `{}`}, want: "usage: oictl users update <user-id>"},
		{name: "update extra id", args: []string{"users", "update", "one", "two", "--data", `{}`}, want: "usage: oictl users update <user-id>"},
		{name: "update missing payload", args: []string{"users", "update", "one"}, want: "a user payload is required"},
		{name: "delete missing id", args: []string{"users", "delete", "--yes"}, want: "usage: oictl users delete <user-id>"},
		{name: "delete extra id", args: []string{"users", "delete", "one", "two", "--yes"}, want: "usage: oictl users delete <user-id>"},
		{name: "delete unconfirmed", args: []string{"users", "delete", "one"}, want: "destructive operation requires --yes"},
		{name: "unknown command", args: []string{"users", "unknown"}, want: `unknown users command "unknown"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-users-crud"})
			before := requests
			if code := app.Run(context.Background(), tt.args); code == 0 {
				t.Fatalf("code=0, want non-zero")
			}
			if out.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", out.String())
			}
			if !strings.Contains(errOut.String(), tt.want) {
				t.Fatalf("stderr missing %q: %q", tt.want, errOut.String())
			}
			if requests != before {
				t.Fatalf("requests=%d, want %d", requests, before)
			}
		})
	}
}

func TestUsersCRUDHTTPFailuresRedactConfiguredCredentials(t *testing.T) {
	statuses := []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound}
	request := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if request >= len(statuses) {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		w.WriteHeader(statuses[request])
		request++
		_, _ = fmt.Fprintf(w, `{"detail":"request denied for %s"}`, token)
	}))
	defer server.Close()

	tests := []struct {
		name       string
		args       []string
		credential string
		status     string
	}{
		{name: "validation api key", args: []string{"users", "get", "bad"}, credential: "sk-api-credential", status: "400 Bad Request"},
		{name: "authentication jwt", args: []string{"--token", "jwt-credential", "users", "create", "--data", `{}`}, credential: "jwt-credential", status: "401 Unauthorized"},
		{name: "authorization api key", args: []string{"users", "update", "denied", "--data", `{}`}, credential: "sk-api-credential", status: "403 Forbidden"},
		{name: "not found jwt", args: []string{"--token", "jwt-credential", "users", "delete", "missing", "--yes"}, credential: "jwt-credential", status: "404 Not Found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-api-credential"})
			if code := app.Run(context.Background(), tt.args); code == 0 {
				t.Fatal("code=0, want non-zero")
			}
			if out.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", out.String())
			}
			got := errOut.String()
			if !strings.Contains(got, tt.status) || !strings.Contains(got, "request denied") || !strings.Contains(got, "<redacted>") {
				t.Fatalf("stderr missing safe status/detail: %q", got)
			}
			if strings.Contains(got, tt.credential) {
				t.Fatalf("stderr leaked credential: %q", got)
			}
		})
	}
	if request != len(tests) {
		t.Fatalf("requests=%d, want %d", request, len(tests))
	}
}

func TestUsersUISettingsBulkPatchExplicitAndFileSuccess(t *testing.T) {
	payload := t.TempDir() + "/ui-settings.json"
	if err := os.WriteFile(payload, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	usersFile := t.TempDir() + "/users.txt"
	if err := os.WriteFile(usersFile, []byte("file-1\nfile-2\nfile-1\n\n"), 0o600); err != nil {
		t.Fatalf("write users file: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.EscapedPath())
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"theme":"dark"}` {
			t.Fatalf("body = %q", string(body))
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-bulk"})
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "user-1", "user-2", "--data", `{"theme":"dark"}`}); code != 0 {
		t.Fatalf("explicit bulk code=%d stderr=%q", code, errOut.String())
	}
	assertBulkStatuses(t, out.String(), []string{"user-1", "user-2"}, "success")

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--users-file", usersFile, "--file", payload}); code != 0 {
		t.Fatalf("file bulk code=%d stderr=%q", code, errOut.String())
	}
	assertBulkStatuses(t, out.String(), []string{"file-1", "file-2"}, "success")
	for _, want := range []string{"PATCH /api/v1/users/user-1/settings/ui", "PATCH /api/v1/users/user-2/settings/ui", "PATCH /api/v1/users/file-1/settings/ui", "PATCH /api/v1/users/file-2/settings/ui"} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}
	if countRequests(requests, "PATCH /api/v1/users/file-1/settings/ui") != 1 {
		t.Fatalf("duplicate file target was not de-duplicated: %v", requests)
	}
}

func TestUsersUISettingsBulkPatchDryRunValidationAndPartialFailure(t *testing.T) {
	payload := t.TempDir() + "/ui-settings.json"
	if err := os.WriteFile(payload, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	usersFile := t.TempDir() + "/users.txt"
	if err := os.WriteFile(usersFile, []byte("user-2\nuser-3\nuser-2\n"), 0o600); err != nil {
		t.Fatalf("write users file: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.EscapedPath())
		if r.URL.EscapedPath() == "/api/v1/users/user-2/settings/ui" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"detail":"missing sk-bulk"}`)
			return
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-bulk"})
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "user-1", "user-1", "--user-id", " user-2 ", "--users-file", usersFile, "--file", payload, "--dry-run"}); code != 0 {
		t.Fatalf("dry-run code=%d stderr=%q", code, errOut.String())
	}
	assertBulkStatuses(t, out.String(), []string{"user-1", "user-2", "user-3"}, "planned")
	if len(requests) != 0 {
		t.Fatalf("dry-run sent requests: %v", requests)
	}

	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{name: "missing targets", args: []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--file", payload}, want: "at least one target user or discovery mode is required"},
		{name: "missing payload", args: []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "user-1"}, want: "a UI settings payload is required"},
		{name: "sensitive key", args: []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "user-1", "--data", `{"toolServers":{"x":"secret"}}`, "--dry-run"}, want: "toolServers"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out.Reset()
			errOut.Reset()
			before := len(requests)
			if code := app.Run(context.Background(), tt.args); code == 0 {
				t.Fatalf("code=0, want non-zero")
			}
			if out.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", out.String())
			}
			if !strings.Contains(errOut.String(), tt.want) {
				t.Fatalf("stderr missing %q:\n%s", tt.want, errOut.String())
			}
			if strings.Contains(errOut.String(), "secret") {
				t.Fatalf("stderr leaked sensitive value: %q", errOut.String())
			}
			if len(requests) != before {
				t.Fatalf("validation sent requests: %v", requests[before:])
			}
		})
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "user-1", "user-2", "--file", payload}); code == 0 {
		t.Fatalf("partial failure code=0, want non-zero")
	}
	if !strings.Contains(out.String(), `"status": "success"`) || !strings.Contains(out.String(), `"status": "failed"`) || !strings.Contains(out.String(), "404 Not Found") || !strings.Contains(out.String(), "missing") || !strings.Contains(out.String(), "<redacted>") || strings.Contains(out.String(), "sk-bulk") {
		t.Fatalf("partial failure output = %q", out.String())
	}
	if strings.Contains(errOut.String(), "sk-bulk") {
		t.Fatalf("partial failure stderr leaked credential: %q", errOut.String())
	}
	for _, want := range []string{"PATCH /api/v1/users/user-1/settings/ui", "PATCH /api/v1/users/user-2/settings/ui"} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing partial request %s in %v", want, requests)
		}
	}
}

func TestUsersUISettingsBulkPatchAllDryRunDiscovery(t *testing.T) {
	payload := t.TempDir() + "/ui-settings.json"
	if err := os.WriteFile(payload, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/users/" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = io.WriteString(w, `{"users":[{"id":"user-1"},{"id":"user-2"}],"total":4}`)
		case "2":
			_, _ = io.WriteString(w, `{"users":[{"id":"user-2"},{"id":"user-3"}],"total":4}`)
		default:
			t.Fatalf("unexpected page %q", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-bulk"})
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--all", "--file", payload, "--dry-run"}); code != 0 {
		t.Fatalf("all dry-run code=%d stderr=%q", code, errOut.String())
	}
	assertBulkStatuses(t, out.String(), []string{"user-1", "user-2", "user-3"}, "planned")
	assertBulkMetadata(t, out.String(), "all", "")
	if countRequests(requests, "GET /api/v1/users/?page=1") != 1 || countRequests(requests, "GET /api/v1/users/?page=2") != 1 {
		t.Fatalf("pagination requests = %v", requests)
	}
	for _, request := range requests {
		if strings.HasPrefix(request, "PATCH ") {
			t.Fatalf("dry-run sent patch request: %v", requests)
		}
	}
}

func TestUsersUISettingsBulkPatchQueryDryRunDiscovery(t *testing.T) {
	payload := t.TempDir() + "/ui-settings.json"
	if err := os.WriteFile(payload, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/users/" || r.URL.Query().Get("query") != "alice" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
		_, _ = io.WriteString(w, `{"users":[{"id":"user-a"}],"total":1}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-bulk"})
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--query", " alice ", "--file", payload, "--dry-run"}); code != 0 {
		t.Fatalf("query dry-run code=%d stderr=%q", code, errOut.String())
	}
	assertBulkStatuses(t, out.String(), []string{"user-a"}, "planned")
	assertBulkMetadata(t, out.String(), "query", "alice")
	if countRequests(requests, "GET /api/v1/users/?page=1&query=alice") != 1 {
		t.Fatalf("query requests = %v", requests)
	}
}

func TestUsersUISettingsBulkPatchDirectoryRequiresConfirmation(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-bulk"})
	for _, args := range [][]string{
		{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--all", "--data", `{"theme":"dark"}`},
		{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--query", "alice", "--data", `{"theme":"dark"}`},
	} {
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), args); code == 0 {
			t.Fatalf("code=0, want non-zero for %v", args)
		}
		if out.Len() != 0 || !strings.Contains(errOut.String(), "require --yes or --dry-run") {
			t.Fatalf("stdout=%q stderr=%q", out.String(), errOut.String())
		}
	}
	if len(requests) != 0 {
		t.Fatalf("unconfirmed directory patch sent requests: %v", requests)
	}
}

func TestUsersUISettingsBulkPatchConfirmedDirectoryRuns(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users/" && r.URL.Query().Get("query") == "":
			_, _ = io.WriteString(w, `{"users":[{"id":"all-1"},{"id":"all-2"}],"total":2}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users/" && r.URL.Query().Get("query") == "alice":
			_, _ = io.WriteString(w, `{"users":[{"id":"query-1"},{"id":"query-1"},{"id":"query-2"}],"total":3}`)
		case r.Method == http.MethodPatch:
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"theme":"dark"}` {
				t.Fatalf("body = %q", string(body))
			}
			_, _ = io.WriteString(w, `{"ok":true}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-bulk"})
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--all", "--data", `{"theme":"dark"}`, "--yes"}); code != 0 {
		t.Fatalf("confirmed all code=%d stderr=%q", code, errOut.String())
	}
	assertBulkStatuses(t, out.String(), []string{"all-1", "all-2"}, "success")
	assertBulkMetadata(t, out.String(), "all", "")

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--query", "alice", "--data", `{"theme":"dark"}`, "--confirm"}); code != 0 {
		t.Fatalf("confirmed query code=%d stderr=%q", code, errOut.String())
	}
	assertBulkStatuses(t, out.String(), []string{"query-1", "query-2"}, "success")
	assertBulkMetadata(t, out.String(), "query", "alice")
	for _, want := range []string{"PATCH /api/v1/users/all-1/settings/ui", "PATCH /api/v1/users/all-2/settings/ui", "PATCH /api/v1/users/query-1/settings/ui", "PATCH /api/v1/users/query-2/settings/ui"} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}
}

func TestUsersUISettingsBulkPatchDirectoryValidation(t *testing.T) {
	payload := t.TempDir() + "/ui-settings.json"
	if err := os.WriteFile(payload, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	usersFile := t.TempDir() + "/users.txt"
	if err := os.WriteFile(usersFile, []byte("user-1\n"), 0o600); err != nil {
		t.Fatalf("write users file: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		_, _ = io.WriteString(w, `{"users":[],"total":0}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-bulk"})
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{name: "all query", args: []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--all", "--query", "alice", "--file", payload, "--dry-run"}, want: "--all and --query cannot be combined"},
		{name: "positionals", args: []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "user-1", "--all", "--file", payload, "--dry-run"}, want: "directory discovery cannot be combined"},
		{name: "user id", args: []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--user-id", "user-1", "--all", "--file", payload, "--dry-run"}, want: "directory discovery cannot be combined"},
		{name: "users file", args: []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--users-file", usersFile, "--all", "--file", payload, "--dry-run"}, want: "directory discovery cannot be combined"},
		{name: "empty query", args: []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--query", "  ", "--file", payload, "--dry-run"}, want: "--query requires non-empty text"},
		{name: "sensitive key", args: []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--all", "--data", `{"toolServers":{"x":"secret"}}`, "--dry-run"}, want: "toolServers"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out.Reset()
			errOut.Reset()
			before := len(requests)
			if code := app.Run(context.Background(), tt.args); code == 0 {
				t.Fatalf("code=0, want non-zero")
			}
			if out.Len() != 0 || !strings.Contains(errOut.String(), tt.want) {
				t.Fatalf("stdout=%q stderr=%q want %q", out.String(), errOut.String(), tt.want)
			}
			if len(requests) != before {
				t.Fatalf("validation sent requests: %v", requests[before:])
			}
		})
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"users", "ui-settings", "bulk-patch", "--allow-ui-settings-extension", "--all", "--file", payload, "--dry-run"}); code == 0 {
		t.Fatalf("empty discovery code=0, want non-zero")
	}
	if out.Len() != 0 || !strings.Contains(errOut.String(), "no target user IDs were discovered") {
		t.Fatalf("stdout=%q stderr=%q", out.String(), errOut.String())
	}
	if len(requests) != 1 || requests[0] != "GET /api/v1/users/?page=1" {
		t.Fatalf("empty discovery requests = %v", requests)
	}
}

func assertBulkStatuses(t *testing.T, output string, userIDs []string, status string) {
	t.Helper()
	var results []bulkUISettingsResult
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("decode bulk results: %v\n%s", err, output)
	}
	if len(results) != len(userIDs) {
		t.Fatalf("result count = %d, want %d: %#v", len(results), len(userIDs), results)
	}
	for i, userID := range userIDs {
		if results[i].UserID != userID || results[i].Action != "patch_ui_settings" || results[i].Status != status {
			t.Fatalf("result[%d] = %#v, want user=%q status=%q", i, results[i], userID, status)
		}
	}
}

func assertBulkMetadata(t *testing.T, output string, targetSource string, query string) {
	t.Helper()
	var results []bulkUISettingsResult
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("decode bulk results: %v\n%s", err, output)
	}
	for i, result := range results {
		if result.TargetSource != targetSource || result.Query != query {
			t.Fatalf("result[%d] metadata = source %q query %q, want source %q query %q", i, result.TargetSource, result.Query, targetSource, query)
		}
	}
}

func countRequests(requests []string, want string) int {
	count := 0
	for _, request := range requests {
		if request == want {
			count++
		}
	}
	return count
}

func TestAuthLoginWritesSessionToGlobalOutFile(t *testing.T) {
	credentials := t.TempDir() + "/credentials.json"
	if err := os.WriteFile(credentials, []byte(`{"email":"user@example.test","password":"fixture-password"}`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	const session = `{"token":"fixture-login-token","token_type":"Bearer"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auths/signin" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, session)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(nil)
	outPath := t.TempDir() + "/session.json"
	if err := os.WriteFile(outPath, []byte("stale session"), 0o644); err != nil {
		t.Fatalf("pre-create output: %v", err)
	}
	if err := os.Chmod(outPath, 0o644); err != nil {
		t.Fatalf("set initial output mode: %v", err)
	}

	code := app.Run(context.Background(), []string{"--base-url", server.URL, "--out", outPath, "auth", "login", "--file", credentials})

	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(written) != session {
		t.Fatalf("output file = %q, want %q", string(written), session)
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("output mode = %o, want 600", got)
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}

func TestAuthLoginCanPersistTokenToProfile(t *testing.T) {
	credentials := t.TempDir() + "/credentials.json"
	if err := os.WriteFile(credentials, []byte(`{"email":"user@example.test","password":"pw"}`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auths/signin" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"token":"jwt-test","token_type":"Bearer"}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(nil)
	configDir := t.TempDir()
	app.userConfigDir = func() (string, error) { return configDir, nil }

	code := app.Run(context.Background(), []string{"--profile", "prod", "--base-url", server.URL, "auth", "login", "--file", credentials, "--save"})
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "jwt-test") {
		t.Fatalf("stdout missing session: %q", out.String())
	}
	profile, err := app.loadProfile("prod")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if profile.Token != "jwt-test" || profile.BaseURL != server.URL {
		t.Fatalf("profile = %+v", profile)
	}
}

func TestChannelsHelpAndUnknownCommand(t *testing.T) {
	app, out, errOut := newTestApp(nil)
	if code := app.Run(context.Background(), []string{"--help"}); code != 0 {
		t.Fatalf("top help code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "channels") {
		t.Fatalf("top help missing channels:\n%s", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"channels", "--help"}); code != 0 {
		t.Fatalf("channels help code=%d stderr=%q", code, errOut.String())
	}
	for _, want := range []string{"members list", "messages post", "pins set", "reactions add", "delete <channel-id> --yes"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("channels help missing %q:\n%s", want, out.String())
		}
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"channels", "unknown"}); code == 0 {
		t.Fatalf("unknown channels code=0, want non-zero")
	}
	if !strings.Contains(errOut.String(), `unknown channels command "unknown"`) || !strings.Contains(errOut.String(), "Usage:") {
		t.Fatalf("unknown stderr lacks context: %q", errOut.String())
	}
}

func TestChannelRecordRequestsPayloadsOutputAndConfirmation(t *testing.T) {
	payload := t.TempDir() + "/channel.json"
	if err := os.WriteFile(payload, []byte(`{"name":"Ops","type":"group"}`), 0o600); err != nil {
		t.Fatalf("write channel payload: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.EscapedPath()+querySuffix(r.URL.RawQuery))
		if got := r.Header.Get("Authorization"); got != "Bearer sk-channels" {
			t.Fatalf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		switch r.URL.EscapedPath() {
		case "/api/v1/channels/":
			if r.URL.RawQuery != "" {
				t.Fatalf("list query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `[{"id":"channel-a"}]`)
		case "/api/v1/channels/channel%2Fone":
			if r.Method == http.MethodGet {
				_, _ = io.WriteString(w, `{"id":"channel/one"}`)
				return
			}
			fallthrough
		case "/api/v1/channels/channel%2Fone/update":
			if r.Method != http.MethodPost || string(body) != `{"name":"Updated"}` {
				t.Fatalf("update request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"updated":true}`)
		case "/api/v1/channels/create":
			if r.Method != http.MethodPost || string(body) != `{"name":"Ops","type":"group"}` {
				t.Fatalf("create request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"channel-a"}`)
		case "/api/v1/channels/channel-a/delete":
			if r.Method != http.MethodDelete {
				t.Fatalf("delete method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `true`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-channels"})
	commands := [][]string{
		{"channels", "list", "--output", "json"},
		{"channels", "get", "channel/one", "--output", "json"},
		{"channels", "create", "--file", payload, "--output", "json"},
		{"channels", "update", "channel/one", "--data", `{"name":"Updated"}`, "--output", "json"},
		{"channels", "delete", "channel-a", "--yes", "--output", "json"},
	}
	for _, args := range commands {
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
	}
	for _, want := range []string{"GET /api/v1/channels/", "GET /api/v1/channels/channel%2Fone", "POST /api/v1/channels/create", "POST /api/v1/channels/channel%2Fone/update", "DELETE /api/v1/channels/channel-a/delete"} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}

	before := len(requests)
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"channels", "delete", "channel-a"}); code == 0 {
		t.Fatalf("delete without confirmation code=0, want non-zero")
	}
	if len(requests) != before || !strings.Contains(errOut.String(), "requires --yes") {
		t.Fatalf("delete confirmation stderr=%q requests=%v", errOut.String(), requests[before:])
	}
}

func TestChannelNestedRequestsPayloadsAndConfirmation(t *testing.T) {
	messagePayload := t.TempDir() + "/message.json"
	if err := os.WriteFile(messagePayload, []byte(`{"content":"hello"}`), 0o600); err != nil {
		t.Fatalf("write message payload: %v", err)
	}
	requests := []string{}
	activeBodies := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()
		requests = append(requests, r.Method+" "+path+querySuffix(r.URL.RawQuery))
		if got := r.Header.Get("Authorization"); got != "Bearer sk-channels" {
			t.Fatalf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		switch path {
		case "/api/v1/channels/channel-a/members":
			if r.URL.Query().Get("page") != "3" || r.URL.Query().Get("query") != "alice" || r.URL.Query().Get("order_by") != "name" || r.URL.Query().Get("direction") != "asc" {
				t.Fatalf("members query = %s", r.URL.RawQuery)
			}
		case "/api/v1/channels/channel-a/update/members/add":
			if string(body) != `{"user_ids":["user-1"]}` {
				t.Fatalf("members add body = %q", string(body))
			}
		case "/api/v1/channels/channel-a/update/members/remove":
			if string(body) != `{"user_ids":["user-1"]}` {
				t.Fatalf("members remove body = %q", string(body))
			}
		case "/api/v1/channels/channel-a/members/active":
			activeBodies = append(activeBodies, string(body))
		case "/api/v1/channels/channel-a/messages":
			if r.URL.Query().Get("skip") != "5" || r.URL.Query().Get("limit") != "10" {
				t.Fatalf("messages query = %s", r.URL.RawQuery)
			}
		case "/api/v1/channels/channel-a/messages/post":
			if string(body) != `{"content":"hello"}` {
				t.Fatalf("message post body = %q", string(body))
			}
		case "/api/v1/channels/channel-a/messages/msg-1":
			if r.Method != http.MethodGet {
				t.Fatalf("message get method = %s", r.Method)
			}
		case "/api/v1/channels/channel-a/messages/msg-1/data":
		case "/api/v1/channels/channel-a/messages/msg-1/thread":
			if r.URL.Query().Get("skip") != "1" || r.URL.Query().Get("limit") != "2" {
				t.Fatalf("thread query = %s", r.URL.RawQuery)
			}
		case "/api/v1/channels/channel-a/messages/msg-1/update":
			if string(body) != `{"content":"updated"}` {
				t.Fatalf("message update body = %q", string(body))
			}
		case "/api/v1/channels/channel-a/messages/msg-1/delete":
			if r.Method != http.MethodDelete {
				t.Fatalf("message delete method = %s", r.Method)
			}
		case "/api/v1/channels/channel-a/messages/pinned":
			if r.URL.Query().Get("page") != "4" {
				t.Fatalf("pins query = %s", r.URL.RawQuery)
			}
		case "/api/v1/channels/channel-a/messages/msg-1/pin":
			if string(body) != `{"is_pinned":true}` && string(body) != `{"is_pinned":false}` {
				t.Fatalf("pin body = %q", string(body))
			}
		case "/api/v1/channels/channel-a/messages/msg-1/reactions/add":
			if string(body) != `{"name":"thumbs-up"}` {
				t.Fatalf("reaction add body = %q", string(body))
			}
		case "/api/v1/channels/channel-a/messages/msg-1/reactions/remove":
			if string(body) != `{"name":"eyes"}` {
				t.Fatalf("reaction remove body = %q", string(body))
			}
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-channels"})
	commands := [][]string{
		{"channels", "members", "list", "channel-a", "--page", "3", "--query", "alice", "--order-by", "name", "--direction", "asc", "--output", "json"},
		{"channels", "members", "add", "channel-a", "--data", `{"user_ids":["user-1"]}`, "--output", "json"},
		{"channels", "members", "remove", "channel-a", "--data", `{"user_ids":["user-1"]}`, "--output", "json"},
		{"channels", "members", "active", "channel-a", "--data", `{"is_active":false}`, "--output", "json"},
		{"channels", "members", "active", "channel-a", "--data", `{"active":false}`, "--output", "json"},
		{"channels", "members", "active", "channel-a", "--data", `{"active":true,"is_active":false}`, "--output", "json"},
		{"channels", "messages", "list", "channel-a", "--skip", "5", "--limit", "10", "--output", "json"},
		{"channels", "messages", "post", "channel-a", "--file", messagePayload, "--output", "json"},
		{"channels", "messages", "get", "channel-a", "msg-1", "--output", "json"},
		{"channels", "messages", "data", "channel-a", "msg-1", "--output", "json"},
		{"channels", "messages", "thread", "channel-a", "msg-1", "--skip", "1", "--limit", "2", "--output", "json"},
		{"channels", "messages", "update", "channel-a", "msg-1", "--data", `{"content":"updated"}`, "--output", "json"},
		{"channels", "messages", "delete", "channel-a", "msg-1", "--yes", "--output", "json"},
		{"channels", "pins", "list", "channel-a", "--page", "4", "--output", "json"},
		{"channels", "pins", "set", "channel-a", "msg-1", "--output", "json"},
		{"channels", "pins", "unset", "channel-a", "msg-1", "--output", "json"},
		{"channels", "reactions", "add", "channel-a", "msg-1", "--name", "thumbs-up", "--output", "json"},
		{"channels", "reactions", "remove", "channel-a", "msg-1", "--data", `{"name":"eyes"}`, "--output", "json"},
	}
	for _, args := range commands {
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
	}

	before := len(requests)
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"channels", "messages", "delete", "channel-a", "msg-1"}); code == 0 {
		t.Fatalf("message delete without confirmation code=0, want non-zero")
	}
	if len(requests) != before || !strings.Contains(errOut.String(), "requires --yes") {
		t.Fatalf("message delete confirmation stderr=%q requests=%v", errOut.String(), requests[before:])
	}
	for _, want := range []string{"GET /api/v1/channels/channel-a/members?direction=asc&order_by=name&page=3&query=alice", "POST /api/v1/channels/channel-a/messages/msg-1/reactions/add", "POST /api/v1/channels/channel-a/messages/msg-1/pin"} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}
	if got, want := strings.Join(activeBodies, "\n"), `{"is_active":false}`+"\n"+`{"is_active":false}`+"\n"+`{"is_active":false}`; got != want {
		t.Fatalf("members active bodies = %q, want %q", got, want)
	}
}

func TestChannelsServerErrorsPreserveRedactedContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"detail":"channels disabled sk-channels"}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-channels"})
	if code := app.Run(context.Background(), []string{"channels", "members", "list", "channel-a", "--output", "json"}); code == 0 {
		t.Fatalf("server error code=0, want non-zero")
	}
	if out.Len() != 0 {
		t.Fatalf("stdout=%q, want empty", out.String())
	}
	if !strings.Contains(errOut.String(), "403 Forbidden") || !strings.Contains(errOut.String(), "channels disabled") || !strings.Contains(errOut.String(), "<redacted>") || strings.Contains(errOut.String(), "sk-channels") {
		t.Fatalf("stderr did not preserve redacted context: %q", errOut.String())
	}
}

func querySuffix(raw string) string {
	if raw == "" {
		return ""
	}
	return "?" + raw
}

func TestModelGetMapsToModelEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models/model" || r.URL.Query().Get("id") != "model-a" {
			t.Fatalf("request = %s", r.URL.String())
		}
		_, _ = io.WriteString(w, `{"id":"model-a"}`)
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})
	if code := app.Run(context.Background(), []string{"models", "get", "model-a"}); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
}

func TestImperativeModelCreateAndUpdateEndpointsArePreserved(t *testing.T) {
	payload := t.TempDir() + "/model.json"
	if err := os.WriteFile(payload, []byte(`{"name":"Llama Ops"}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	importPayload := t.TempDir() + "/models-import.json"
	if err := os.WriteFile(importPayload, []byte(`[{"id":"imported","name":"Imported"}]`), 0o600); err != nil {
		t.Fatalf("write import payload: %v", err)
	}
	syncPayload := t.TempDir() + "/models-sync.json"
	if err := os.WriteFile(syncPayload, []byte(`[{"id":"synced","name":"Synced"}]`), 0o600); err != nil {
		t.Fatalf("write sync payload: %v", err)
	}
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.Method+" "+r.URL.Path] = true
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/api/v1/models/create":
			if r.Method != http.MethodPost || string(body) != `{"name":"Llama Ops","params":{}}` {
				t.Fatalf("create request = %s %s body=%q", r.Method, r.URL.Path, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"created"}`)
		case "/api/v1/models/import":
			if r.Method != http.MethodPost || string(body) != `{"models":[{"id":"imported","name":"Imported","params":{}}]}` {
				t.Fatalf("import request = %s %s body=%q", r.Method, r.URL.Path, string(body))
			}
			_, _ = io.WriteString(w, `[{"id":"imported"}]`)
		case "/api/v1/models/sync":
			if r.Method != http.MethodPost || string(body) != `{"models":[{"id":"synced","name":"Synced","params":{}}]}` {
				t.Fatalf("sync request = %s %s body=%q", r.Method, r.URL.Path, string(body))
			}
			_, _ = io.WriteString(w, `[{"id":"synced"}]`)
		case "/api/v1/models/model/update":
			if r.Method != http.MethodPost || !strings.Contains(string(body), `"id":"model-a"`) || !strings.Contains(string(body), `"name":"Llama Ops"`) {
				t.Fatalf("update request = %s %s body=%q", r.Method, r.URL.Path, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"model-a"}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})
	if code := app.Run(context.Background(), []string{"models", "create", "--file", payload}); code != 0 {
		t.Fatalf("create exit code = %d; stderr=%q", code, errOut.String())
	}
	if code := app.Run(context.Background(), []string{"models", "import", "--file", importPayload}); code != 0 {
		t.Fatalf("import exit code = %d; stderr=%q", code, errOut.String())
	}
	if code := app.Run(context.Background(), []string{"models", "sync", "--file", syncPayload}); code != 0 {
		t.Fatalf("sync exit code = %d; stderr=%q", code, errOut.String())
	}
	if code := app.Run(context.Background(), []string{"models", "update", "model-a", "--file", payload}); code != 0 {
		t.Fatalf("update exit code = %d; stderr=%q", code, errOut.String())
	}
	for _, want := range []string{"POST /api/v1/models/create", "POST /api/v1/models/import", "POST /api/v1/models/sync", "POST /api/v1/models/model/update"} {
		if !seen[want] {
			t.Fatalf("missing %s in %#v", want, seen)
		}
	}
}

func TestConfigSetForwardsJSONFile(t *testing.T) {
	payload := t.TempDir() + "/connections.json"
	if err := os.WriteFile(payload, []byte(`{"ENABLE_DIRECT_CONNECTIONS":true}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/configs/connections" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"ENABLE_DIRECT_CONNECTIONS":true}` {
			t.Fatalf("body = %q", string(body))
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})
	if code := app.Run(context.Background(), []string{"config", "connections", "set", "--file", payload}); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
}

func TestModelsExportWritesOutFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models/export" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `[{"id":"a"}]`)
	}))
	defer server.Close()
	outPath := t.TempDir() + "/models.json"

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})
	if code := app.Run(context.Background(), []string{"models", "export", "--out", outPath}); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
	body, err := os.ReadFile(outPath)
	if err != nil || string(body) != `[{"id":"a"}]` {
		t.Fatalf("output file = %q, err=%v", string(body), err)
	}
}

func TestModelDeleteReclassifiesOpenWebUINotFound401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/models/model/delete" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"detail":"We could not find what you're looking for :/"}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-models"})
	if code := app.Run(context.Background(), []string{"models", "delete", "missing-model", "--output", "json"}); code == 0 {
		t.Fatalf("delete code=0 stdout=%q stderr=%q", out.String(), errOut.String())
	}
	if out.Len() != 0 || !strings.Contains(errOut.String(), `model "missing-model" not found`) || strings.Contains(errOut.String(), "401 Unauthorized") || strings.Contains(errOut.String(), "We could not find") {
		t.Fatalf("delete stderr=%q stdout=%q", errOut.String(), out.String())
	}
}

func TestModelDeleteReclassifiesLiteralNotFound401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/models/model/delete" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"detail":"Model not found"}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-models"})
	if code := app.Run(context.Background(), []string{"models", "delete", "missing-model"}); code == 0 {
		t.Fatalf("delete code=0 stdout=%q stderr=%q", out.String(), errOut.String())
	}
	if out.Len() != 0 || !strings.Contains(errOut.String(), `model "missing-model" not found`) || strings.Contains(errOut.String(), "401 Unauthorized") {
		t.Fatalf("delete stderr=%q stdout=%q", errOut.String(), out.String())
	}
}

func TestModelDeletePreservesAuthorization401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/models/model/delete" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"detail":"Not authenticated sk-models"}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-models"})
	if code := app.Run(context.Background(), []string{"models", "delete", "model-a"}); code == 0 {
		t.Fatalf("delete code=0 stdout=%q stderr=%q", out.String(), errOut.String())
	}
	if out.Len() != 0 || !strings.Contains(errOut.String(), "401 Unauthorized") || !strings.Contains(errOut.String(), "<redacted>") || strings.Contains(errOut.String(), "sk-models") || strings.Contains(errOut.String(), "not found") {
		t.Fatalf("delete stderr=%q stdout=%q", errOut.String(), out.String())
	}
}

func TestFilesUploadUsesMultipartAndQueryFlags(t *testing.T) {
	upload := t.TempDir() + "/doc.txt"
	if err := os.WriteFile(upload, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write upload: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/files/" || r.URL.Query().Get("process") != "false" {
			t.Fatalf("request = %s", r.URL.String())
		}
		if err := r.ParseMultipartForm(1024); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if r.MultipartForm.Value["metadata"][0] != `{"kind":"test"}` {
			t.Fatalf("metadata = %v", r.MultipartForm.Value["metadata"])
		}
		_, _ = io.WriteString(w, `{"id":"file-a"}`)
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})
	if code := app.Run(context.Background(), []string{"files", "upload", upload, "--process", "false", "--metadata", `{"kind":"test"}`}); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
}

func TestKnowledgeFileAddMapsToMembershipEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/knowledge/kb-a/file/add" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"file_id":"file-a"}` {
			t.Fatalf("body = %q", string(body))
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})
	if code := app.Run(context.Background(), []string{"knowledge", "files", "add", "kb-a", "file-a"}); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
}

func TestKnowledgeExportWritesZipOutFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/knowledge/kb-a/export" || r.Method != http.MethodGet {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write([]byte("PK\x05\x06binary-export"))
	}))
	defer server.Close()
	outPath := t.TempDir() + "/knowledge.zip"

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-test"})
	if code := app.Run(context.Background(), []string{"knowledge", "export", "kb-a", "--out", outPath}); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
	body, err := os.ReadFile(outPath)
	if err != nil || string(body) != "PK\x05\x06binary-export" {
		t.Fatalf("output file = %q, err=%v", string(body), err)
	}
}

func TestAdvancedSurfaceHelpListsCommands(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{args: []string{"--help"}, want: []string{"chats", "analytics", "automations", "scim", "providers", "functions", "groups", "skills", "tasks", "tools"}},
		{args: []string{"chats", "--help"}, want: []string{"list", "search", "export", "compact"}},
		{args: []string{"analytics", "--help"}, want: []string{"models", "users", "messages", "tokens"}},
		{args: []string{"automations", "--help"}, want: []string{"create", "toggle", "runs"}},
		{args: []string{"scim", "--help"}, want: []string{"service-provider-config", "resource-types", "users", "groups"}},
		{args: []string{"providers", "--help"}, want: []string{"openai", "ollama", "Open WebUI"}},
		{args: []string{"functions", "--help"}, want: []string{"list", "get", "create", "update", "delete", "export", "load-url", "sync", "toggle", "toggle-global", "valves", "type is \"filter\"", "arbitrary Python"}},
		{args: []string{"groups", "--help"}, want: []string{"native Open WebUI groups", "list", "create", "get", "info", "export", "update", "delete", "preview", "users list", "users add", "users remove", "oictl scim groups"}},
		{args: []string{"skills", "--help"}, want: []string{"list", "get", "create", "update", "access-update", "toggle", "delete", "export", "--manifest"}},
		{args: []string{"tasks", "--help"}, want: []string{"tasks config get", "tasks config set", "tasks skills attach", "TASK_MODEL", "task.model.default", "no separate task-model"}},
		{args: []string{"tools", "--help"}, want: []string{"list", "get", "create", "update", "delete", "export", "load-url", "access-update", "valves", "--manifest", "--directory"}},
	}

	for _, tt := range tests {
		app, out, errOut := newTestApp(nil)
		if code := app.Run(context.Background(), tt.args); code != 0 {
			t.Fatalf("%v exit code = %d; stderr=%q", tt.args, code, errOut.String())
		}
		for _, want := range tt.want {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("%v help missing %q:\n%s", tt.args, want, out.String())
			}
		}
	}
}

func TestNativeGroupsRequestsPayloadsOutputAndAuth(t *testing.T) {
	dir := t.TempDir()
	payload := dir + "/group.json"
	if err := os.WriteFile(payload, []byte(`{"id":"group-a","name":"Group A"}`), 0o600); err != nil {
		t.Fatalf("write group payload: %v", err)
	}
	exportOut := dir + "/group-export.json"
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		if got := r.Header.Get("Authorization"); got != "Bearer sk-groups" {
			t.Fatalf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/api/v1/groups/":
			if r.Method != http.MethodGet || r.URL.Query().Get("share") != "true" {
				t.Fatalf("list request = %s %s", r.Method, r.URL.String())
			}
			_, _ = io.WriteString(w, `[{"id":"group-a","name":"Group A"}]`)
		case "/api/v1/groups/create":
			if r.Method != http.MethodPost || string(body) != `{"id":"group-a","name":"Group A"}` {
				t.Fatalf("create request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"group-a"}`)
		case "/api/v1/groups/id/group-a":
			if r.Method != http.MethodGet {
				t.Fatalf("get method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"id":"group-a","name":"Group A"}`)
		case "/api/v1/groups/id/group-a/info":
			if r.Method != http.MethodGet {
				t.Fatalf("info method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"id":"group-a","users":2}`)
		case "/api/v1/groups/id/group-a/export":
			if r.Method != http.MethodGet {
				t.Fatalf("export method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"id":"group-a","exported":true}`)
		case "/api/v1/groups/id/group-a/update":
			if r.Method != http.MethodPost || string(body) != `{"name":"Updated"}` {
				t.Fatalf("update request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"group-a","name":"Updated"}`)
		case "/api/v1/groups/id/group-a/delete":
			if r.Method != http.MethodDelete {
				t.Fatalf("delete method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `true`)
		case "/api/v1/groups/id/group-a/preview":
			if r.Method != http.MethodGet {
				t.Fatalf("preview method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"resources":[{"id":"r1"}]}`)
		case "/api/v1/groups/id/group-a/users":
			if r.Method != http.MethodPost {
				t.Fatalf("users list method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `[{"id":"user-1","name":"User One"}]`)
		case "/api/v1/groups/id/group-a/users/add":
			if r.Method != http.MethodPost || string(body) != `{"user_ids":["user-1","user-2"]}` {
				t.Fatalf("users add request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"added":2}`)
		case "/api/v1/groups/id/group-a/users/remove":
			if r.Method != http.MethodPost || string(body) != `{"user_ids":["user-1","user-2"]}` {
				t.Fatalf("users remove request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"removed":2}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-groups"})
	commands := [][]string{
		{"groups", "list", "--share", "true"},
		{"groups", "create", "--file", payload},
		{"groups", "get", "group-a"},
		{"groups", "info", "group-a"},
		{"groups", "export", "group-a", "--out", exportOut},
		{"groups", "update", "group-a", "--data", `{"name":"Updated"}`},
		{"groups", "delete", "group-a", "--confirm"},
		{"groups", "preview", "group-a"},
		{"groups", "users", "list", "group-a"},
		{"groups", "users", "add", "group-a", "user-1", "user-2"},
		{"groups", "users", "remove", "group-a", "user-1", "user-2"},
	}
	for _, args := range commands {
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
	}
	if body, err := os.ReadFile(exportOut); err != nil || string(body) != `{"id":"group-a","exported":true}` {
		t.Fatalf("export file = %q, err=%v", string(body), err)
	}
	for _, want := range []string{
		"GET /api/v1/groups/?share=true", "POST /api/v1/groups/create", "GET /api/v1/groups/id/group-a", "GET /api/v1/groups/id/group-a/info", "GET /api/v1/groups/id/group-a/export", "POST /api/v1/groups/id/group-a/update", "DELETE /api/v1/groups/id/group-a/delete", "GET /api/v1/groups/id/group-a/preview", "POST /api/v1/groups/id/group-a/users", "POST /api/v1/groups/id/group-a/users/add", "POST /api/v1/groups/id/group-a/users/remove",
	} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}
}

func TestNativeGroupsValidationErrorsDoNotContactServerAndRedactToken(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path == "/api/v1/groups/id/missing" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"detail":"missing sk-groups"}`)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"groups", "unknown"}, want: `unknown groups command "unknown"`},
		{args: []string{"groups", "create"}, want: "a group payload is required"},
		{args: []string{"groups", "update", "group-a"}, want: "a group payload is required"},
		{args: []string{"groups", "delete", "group-a"}, want: "requires --yes"},
		{args: []string{"groups", "users", "add", "group-a"}, want: "at least one user id is required"},
		{args: []string{"groups", "users", "remove", "group-a"}, want: "at least one user id is required"},
	}
	for _, tt := range tests {
		app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-groups"})
		before := requests
		if code := app.Run(context.Background(), tt.args); code == 0 {
			t.Fatalf("%v exit code = 0, want non-zero", tt.args)
		}
		if out.Len() != 0 {
			t.Fatalf("%v stdout = %q, want empty", tt.args, out.String())
		}
		if !strings.Contains(errOut.String(), tt.want) {
			t.Fatalf("%v stderr missing %q:\n%s", tt.args, tt.want, errOut.String())
		}
		if requests != before {
			t.Fatalf("%v contacted server", tt.args)
		}
	}

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-groups"})
	if code := app.Run(context.Background(), []string{"groups", "get", "missing", "--output", "json"}); code == 0 || !strings.Contains(errOut.String(), "404 Not Found") || !strings.Contains(errOut.String(), "<redacted>") || strings.Contains(errOut.String(), "sk-groups") {
		t.Fatalf("missing get code=%d stderr=%q", code, errOut.String())
	}
}

func TestNativeGroupsUseNormalTokenAndSCIMGroupsRemainSeparate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/groups/":
			if got := r.Header.Get("Authorization"); got != "Bearer normal-token" {
				t.Fatalf("native Authorization = %q", got)
			}
			_, _ = io.WriteString(w, `[]`)
		case "/api/v1/scim/v2/Groups":
			if got := r.Header.Get("Authorization"); got != "Bearer scim-token" {
				t.Fatalf("SCIM Authorization = %q", got)
			}
			_, _ = io.WriteString(w, `{"Resources":[]}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "normal-token"})
	if code := app.Run(context.Background(), []string{"groups", "list", "--output", "json"}); code != 0 {
		t.Fatalf("native groups code=%d stderr=%q", code, errOut.String())
	}
	errOut.Reset()
	app.getenv = func(key string) string {
		switch key {
		case "OPEN_WEBUI_URL":
			return server.URL
		case "OPEN_WEBUI_API_KEY":
			return "normal-token"
		case "OPEN_WEBUI_SCIM_TOKEN":
			return "scim-token"
		default:
			return ""
		}
	}
	if code := app.Run(context.Background(), []string{"scim", "groups", "list"}); code != 0 {
		t.Fatalf("scim groups code=%d stderr=%q", code, errOut.String())
	}
}

func TestFunctionsListFilterGetFormattingAndErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-functions" {
			t.Fatalf("Authorization = %q", got)
		}
		switch r.URL.Path {
		case "/api/v1/functions/list":
			_, _ = io.WriteString(w, `[{"id":"fn_a","name":"Action","type":"action","is_active":true,"is_global":false,"created_at":1,"updated_at":2,"content":"print('a')"},{"id":"filter_a","name":"Filter","type":"filter","is_active":false,"is_global":true,"created_at":3,"updated_at":4,"content":"print('f')","meta":{"toggle":true}}]`)
		case "/api/v1/functions/id/filter_a":
			_, _ = io.WriteString(w, `{"id":"filter_a","name":"Filter","type":"filter","content":"class Filter: pass","meta":{"toggle":true}}`)
		case "/api/v1/functions/id/missing":
			http.Error(w, `{"detail":"not found"}`, http.StatusNotFound)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-functions"})
	if code := app.Run(context.Background(), []string{"functions", "list"}); code != 0 {
		t.Fatalf("list code=%d stderr=%q", code, errOut.String())
	}
	for _, want := range []string{"ID", "NAME", "TYPE", "ACTIVE", "GLOBAL", "UPDATED_AT", "CREATED_AT", "fn_a", "filter_a"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list table missing %q:\n%s", want, out.String())
		}
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "list", "--type", "filter", "--output", "json"}); code != 0 {
		t.Fatalf("filter list code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"id":"filter_a"`) || strings.Contains(out.String(), `"id":"fn_a"`) {
		t.Fatalf("filtered json = %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "get", "filter_a"}); code != 0 {
		t.Fatalf("get code=%d stderr=%q", code, errOut.String())
	}
	for _, want := range []string{`"content":"class Filter: pass"`, `"meta":{"toggle":true}`, `"type":"filter"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("get json missing %q: %q", want, out.String())
		}
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "get", "missing"}); code == 0 || !strings.Contains(errOut.String(), "404 Not Found") || !strings.Contains(errOut.String(), "not found") {
		t.Fatalf("missing get code=%d stderr=%q", code, errOut.String())
	}
}

func TestFunctionsLifecycleRequestsPayloadsOutputAndConfirmation(t *testing.T) {
	payload := t.TempDir() + "/function.json"
	if err := os.WriteFile(payload, []byte(`{"id":"filter_a","name":"Filter","content":"class Filter: pass","meta":{}}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/api/v1/functions/create":
			if string(body) == `{"id":"bad"}` {
				http.Error(w, `{"detail":"Error loading function"}`, http.StatusBadRequest)
				return
			}
			if string(body) != `{"id":"filter_a","name":"Filter","content":"class Filter: pass","meta":{}}` {
				t.Fatalf("create body = %q", string(body))
			}
			_, _ = io.WriteString(w, `{"id":"filter_a","type":"filter","is_active":true}`)
		case "/api/v1/functions/id/filter_a/update":
			if string(body) != `{"name":"Updated","content":"class Filter: pass"}` {
				t.Fatalf("update body = %q", string(body))
			}
			_, _ = io.WriteString(w, `{"id":"filter_a","type":"filter","name":"Updated"}`)
		case "/api/v1/functions/id/filter_a/delete":
			if r.Method != http.MethodDelete {
				t.Fatalf("delete method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `true`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-functions"})
	if code := app.Run(context.Background(), []string{"functions", "create", "--file", payload}); code != 0 {
		t.Fatalf("create code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"type":"filter"`) {
		t.Fatalf("create output = %q", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "update", "filter_a", "--data", `{"name":"Updated","content":"class Filter: pass"}`}); code != 0 {
		t.Fatalf("update code=%d stderr=%q", code, errOut.String())
	}
	errOut.Reset()
	before := len(requests)
	if code := app.Run(context.Background(), []string{"functions", "delete", "filter_a"}); code == 0 || !strings.Contains(errOut.String(), "requires --yes") {
		t.Fatalf("delete without confirmation code=%d stderr=%q", code, errOut.String())
	}
	if len(requests) != before {
		t.Fatalf("delete without --yes contacted server: %v", requests[before:])
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "delete", "filter_a", "--yes"}); code != 0 {
		t.Fatalf("delete code=%d stderr=%q", code, errOut.String())
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "create", "--data", `{"id":"bad"}`}); code == 0 || !strings.Contains(errOut.String(), "400 Bad Request") || !strings.Contains(errOut.String(), "Error loading function") {
		t.Fatalf("validation code=%d stderr=%q", code, errOut.String())
	}
}

func TestFunctionsExportLoadURLSyncAndConfirmation(t *testing.T) {
	dir := t.TempDir()
	outPath := dir + "/functions.json"
	syncPath := dir + "/sync.json"
	if err := os.WriteFile(syncPath, []byte(`[{"id":"fn_a","content":"print(1)"}]`), 0o600); err != nil {
		t.Fatalf("write sync payload: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/api/v1/functions/export":
			if r.URL.Query().Get("include_valves") != "true" {
				t.Fatalf("export query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `[{"id":"fn_a","valves":{"x":1}}]`)
		case "/api/v1/functions/load/url":
			if string(body) == `{"url":"https://example.invalid/bad.py"}` {
				http.Error(w, `{"detail":"Failed to fetch the function"}`, http.StatusBadRequest)
				return
			}
			if string(body) != `{"url":"https://example.invalid/function.py"}` {
				t.Fatalf("load-url body = %q", string(body))
			}
			_, _ = io.WriteString(w, `{"name":"function","content":"print(1)"}`)
		case "/api/v1/functions/sync":
			if string(body) != `{"functions":[{"id":"fn_a","content":"print(1)"}]}` {
				t.Fatalf("sync body = %q", string(body))
			}
			_, _ = io.WriteString(w, `[{"id":"fn_a","type":"action"}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-functions"})
	if code := app.Run(context.Background(), []string{"functions", "export", "--include-valves", "--out", outPath}); code != 0 {
		t.Fatalf("export code=%d stderr=%q", code, errOut.String())
	}
	if body, err := os.ReadFile(outPath); err != nil || string(body) != `[{"id":"fn_a","valves":{"x":1}}]` {
		t.Fatalf("export file = %q, err=%v", string(body), err)
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "load-url", "https://example.invalid/function.py"}); code != 0 {
		t.Fatalf("load-url code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"content":"print(1)"`) {
		t.Fatalf("load-url output = %q", out.String())
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "load-url", "https://example.invalid/bad.py"}); code == 0 || !strings.Contains(errOut.String(), "400 Bad Request") || !strings.Contains(errOut.String(), "Failed to fetch the function") {
		t.Fatalf("load-url error code=%d stderr=%q", code, errOut.String())
	}
	errOut.Reset()
	before := len(requests)
	if code := app.Run(context.Background(), []string{"functions", "sync", "--file", syncPath}); code == 0 || !strings.Contains(errOut.String(), "can remove remote functions") {
		t.Fatalf("sync without confirmation code=%d stderr=%q", code, errOut.String())
	}
	if len(requests) != before {
		t.Fatalf("sync without --yes contacted server: %v", requests[before:])
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "sync", "--file", syncPath, "--yes"}); code != 0 {
		t.Fatalf("sync code=%d stderr=%q", code, errOut.String())
	}
}

func TestFunctionsTogglePathsAndForbiddenErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/functions/id/fn_a/toggle":
			_, _ = io.WriteString(w, `{"id":"fn_a","is_active":false}`)
		case "/api/v1/functions/id/fn_a/toggle/global":
			_, _ = io.WriteString(w, `{"id":"fn_a","is_global":true}`)
		case "/api/v1/functions/id/denied/toggle":
			http.Error(w, `{"detail":"forbidden"}`, http.StatusForbidden)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-functions"})
	if code := app.Run(context.Background(), []string{"functions", "toggle", "fn_a"}); code != 0 || !strings.Contains(out.String(), `"is_active":false`) {
		t.Fatalf("toggle code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "toggle-global", "fn_a"}); code != 0 || !strings.Contains(out.String(), `"is_global":true`) {
		t.Fatalf("toggle-global code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"functions", "toggle", "denied"}); code == 0 || !strings.Contains(errOut.String(), "403 Forbidden") || !strings.Contains(errOut.String(), "forbidden") {
		t.Fatalf("forbidden code=%d stderr=%q", code, errOut.String())
	}
}

func TestSkillsUnknownSubcommand(t *testing.T) {
	app, out, errOut := newTestApp(nil)

	code := app.Run(context.Background(), []string{"skills", "nope"})

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
	if !strings.Contains(errOut.String(), `unknown skills command "nope"`) {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestSkillsRequestsPayloadsOutputAndConfirmation(t *testing.T) {
	jsonPayload := t.TempDir() + "/skill.json"
	if err := os.WriteFile(jsonPayload, []byte(`{"id":"skill-a","name":"Skill A","content":"Do work"}`), 0o600); err != nil {
		t.Fatalf("write skill payload: %v", err)
	}
	accessPayload := t.TempDir() + "/access.json"
	if err := os.WriteFile(accessPayload, []byte(`{"access_grants":[]}`), 0o600); err != nil {
		t.Fatalf("write access payload: %v", err)
	}
	bulkOut := t.TempDir() + "/skills.json"
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		if got := r.Header.Get("Authorization"); got != "Bearer sk-skills" {
			t.Fatalf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/api/v1/skills/list":
			if r.URL.Query().Get("query") != "review" || r.URL.Query().Get("view_option") != "shared" || r.URL.Query().Get("page") != "2" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"items":[{"id":"skill-a","name":"Review","is_active":true}],"total":1}`)
		case "/api/v1/skills/id/skill-a":
			_, _ = io.WriteString(w, `{"id":"skill-a","name":"Review"}`)
		case "/api/v1/skills/create":
			if string(body) != `{"id":"skill-a","name":"Skill A","content":"Do work"}` {
				t.Fatalf("create body = %q", string(body))
			}
			_, _ = io.WriteString(w, `{"id":"skill-a"}`)
		case "/api/v1/skills/id/skill-a/update":
			if string(body) != `{"name":"Updated","content":"Better"}` {
				t.Fatalf("update body = %q", string(body))
			}
			_, _ = io.WriteString(w, `{"id":"skill-a"}`)
		case "/api/v1/skills/id/skill-a/access/update":
			if string(body) != `{"access_grants":[]}` {
				t.Fatalf("access body = %q", string(body))
			}
			_, _ = io.WriteString(w, `{"id":"skill-a","access_grants":[]}`)
		case "/api/v1/skills/id/skill-a/toggle":
			if r.Method != http.MethodPost {
				t.Fatalf("toggle method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"id":"skill-a","is_active":false}`)
		case "/api/v1/skills/id/skill-a/delete":
			if r.Method != http.MethodDelete {
				t.Fatalf("delete method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `true`)
		case "/api/v1/skills/export":
			_, _ = io.WriteString(w, `[{"id":"skill-a"}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-skills"})
	if code := app.Run(context.Background(), []string{"skills", "list", "--query", "review", "--view-option", "shared", "--page", "2"}); code != 0 {
		t.Fatalf("list code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "ID") || !strings.Contains(out.String(), "skill-a") {
		t.Fatalf("list table = %q", out.String())
	}
	for _, args := range [][]string{
		{"skills", "get", "skill-a"},
		{"skills", "create", "--file", jsonPayload},
		{"skills", "update", "skill-a", "--data", `{"name":"Updated","content":"Better"}`},
		{"skills", "access-update", "skill-a", "--file", accessPayload},
		{"skills", "toggle", "skill-a"},
		{"skills", "export", "skill-a"},
		{"skills", "export", "--out", bulkOut},
	} {
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
	}
	if body, err := os.ReadFile(bulkOut); err != nil || string(body) != `[{"id":"skill-a"}]` {
		t.Fatalf("bulk export file = %q, err=%v", string(body), err)
	}
	errOut.Reset()
	before := len(requests)
	if code := app.Run(context.Background(), []string{"skills", "delete", "skill-a"}); code == 0 || !strings.Contains(errOut.String(), "requires --yes") {
		t.Fatalf("delete without confirmation code=%d stderr=%q", code, errOut.String())
	}
	if len(requests) != before {
		t.Fatalf("delete without --yes contacted server: %v", requests[before:])
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"skills", "delete", "skill-a", "--yes"}); code != 0 {
		t.Fatalf("delete code=%d stderr=%q", code, errOut.String())
	}
}

func TestSkillsMarkdownManifestConversionAndExport(t *testing.T) {
	dir := t.TempDir()
	manifest := dir + "/skill.md"
	if err := os.WriteFile(manifest, []byte("---\nid: skill-a\nname: Review Bot\ndescription: Helps review\nis_active: false\ntags:\n  - ops\n  - review\n---\nUse this skill for reviews.\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	outPath := dir + "/exported.md"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/api/v1/skills/create", "/api/v1/skills/id/skill-a/update":
			for _, want := range []string{`"id":"skill-a"`, `"name":"Review Bot"`, `"description":"Helps review"`, `"is_active":false`, `"content":"Use this skill for reviews.\n"`, `"tags":["ops","review"]`} {
				if !strings.Contains(string(body), want) {
					t.Fatalf("manifest body missing %s: %q", want, string(body))
				}
			}
			_, _ = io.WriteString(w, `{"id":"skill-a"}`)
		case "/api/v1/skills/id/skill-a":
			_, _ = io.WriteString(w, `{"id":"skill-a","name":"Review Bot","description":"Helps review","content":"Use this skill for reviews.\n","is_active":false,"meta":{"tags":["ops","review"]}}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-skills"})
	if code := app.Run(context.Background(), []string{"skills", "create", "--manifest", manifest}); code != 0 {
		t.Fatalf("create manifest code=%d stderr=%q", code, errOut.String())
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"skills", "update", "skill-a", "--manifest", manifest}); code != 0 {
		t.Fatalf("update manifest code=%d stderr=%q", code, errOut.String())
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"skills", "export", "skill-a", "--format", "manifest", "--out", outPath}); code != 0 {
		t.Fatalf("export manifest code=%d stderr=%q", code, errOut.String())
	}
	body, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	for _, want := range []string{`id: "skill-a"`, `name: "Review Bot"`, `description: "Helps review"`, `is_active: false`, `  - "ops"`, "Use this skill for reviews."} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("export missing %q:\n%s", want, string(body))
		}
	}
	errOut.Reset()
	before := requests
	if code := app.Run(context.Background(), []string{"skills", "create", "--manifest", manifest, "--data", `{}`}); code == 0 || !strings.Contains(errOut.String(), "use only one of --manifest") {
		t.Fatalf("conflict code=%d stderr=%q", code, errOut.String())
	}
	if requests != before {
		t.Fatalf("conflict contacted server")
	}
	bad := dir + "/bad.md"
	if err := os.WriteFile(bad, []byte("---\naccess_grants: []\n---\nbody\n"), 0o600); err != nil {
		t.Fatalf("write bad manifest: %v", err)
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"skills", "create", "--manifest", bad}); code == 0 || !strings.Contains(errOut.String(), "access_grants") {
		t.Fatalf("bad manifest code=%d stderr=%q", code, errOut.String())
	}
}

func TestToolsRequestsPayloadsRawExportAndForbiddenErrors(t *testing.T) {
	dir := t.TempDir()
	payload := dir + "/tool.json"
	if err := os.WriteFile(payload, []byte(`{"id":"weather_tool","name":"Weather","content":"class Tools: pass","meta":{"description":"Weather"}}`), 0o600); err != nil {
		t.Fatalf("write tool payload: %v", err)
	}
	accessPayload := dir + "/access.json"
	if err := os.WriteFile(accessPayload, []byte(`{"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`), 0o600); err != nil {
		t.Fatalf("write access payload: %v", err)
	}
	rawOut := dir + "/tools.json"
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		if got := r.Header.Get("Authorization"); got != "Bearer sk-tools" {
			t.Fatalf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/api/v1/tools/list":
			if r.Method != http.MethodGet {
				t.Fatalf("list method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `[{"id":"weather_tool","name":"Weather"}]`)
		case "/api/v1/tools/id/weather_tool":
			if r.Method != http.MethodGet {
				t.Fatalf("get method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"id":"weather_tool","name":"Weather"}`)
		case "/api/v1/tools/create":
			if r.Method != http.MethodPost || string(body) != `{"id":"weather_tool","name":"Weather","content":"class Tools: pass","meta":{"description":"Weather"}}` {
				t.Fatalf("create request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"weather_tool"}`)
		case "/api/v1/tools/id/weather_tool/update":
			if r.Method != http.MethodPost || string(body) != `{"name":"Updated","content":"class Tools: pass"}` {
				t.Fatalf("update request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"weather_tool","name":"Updated"}`)
		case "/api/v1/tools/id/weather_tool/access/update":
			if r.Method != http.MethodPost || string(body) != `{"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}` {
				t.Fatalf("access request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"weather_tool","access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)
		case "/api/v1/tools/load/url":
			if r.Method != http.MethodPost || string(body) != `{"url":"https://example.invalid/weather.py"}` {
				t.Fatalf("load-url request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"name":"Weather","content":"class Tools: pass"}`)
		case "/api/v1/tools/id/weather_tool/delete":
			if r.Method != http.MethodDelete {
				t.Fatalf("delete method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `true`)
		case "/api/v1/tools/export":
			if r.Method != http.MethodGet {
				t.Fatalf("export method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `[{"id":"weather_tool"}]`)
		case "/api/v1/tools/id/denied":
			http.Error(w, `{"detail":"forbidden"}`, http.StatusForbidden)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-tools"})
	for _, args := range [][]string{
		{"tools", "list"},
		{"tools", "get", "weather_tool"},
		{"tools", "create", "--file", payload},
		{"tools", "update", "weather_tool", "--data", `{"name":"Updated","content":"class Tools: pass"}`},
		{"tools", "access-update", "weather_tool", "--file", accessPayload},
		{"tools", "load-url", "--data", `{"url":"https://example.invalid/weather.py"}`},
		{"tools", "delete", "weather_tool"},
		{"tools", "export", "--out", rawOut},
	} {
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
	}
	if body, err := os.ReadFile(rawOut); err != nil || string(body) != `[{"id":"weather_tool"}]` {
		t.Fatalf("raw export file = %q, err=%v", string(body), err)
	}
	for _, want := range []string{"GET /api/v1/tools/list", "GET /api/v1/tools/id/weather_tool", "POST /api/v1/tools/create", "POST /api/v1/tools/id/weather_tool/update", "POST /api/v1/tools/id/weather_tool/access/update", "POST /api/v1/tools/load/url", "DELETE /api/v1/tools/id/weather_tool/delete", "GET /api/v1/tools/export"} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"tools", "get", "denied"}); code == 0 || !strings.Contains(errOut.String(), "403 Forbidden") || !strings.Contains(errOut.String(), "forbidden") {
		t.Fatalf("forbidden code=%d stderr=%q", code, errOut.String())
	}
}

func TestToolsManifestExportShapeFilesAndLoading(t *testing.T) {
	dir := t.TempDir()
	singleOut := dir + "/weather.json"
	manifestDir := dir + "/manifests"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tools/id/weather_tool":
			_, _ = io.WriteString(w, `{"id":"weather_tool","name":"Weather","content":"class Tools: pass","meta":{"description":"Weather"},"user_id":"u1","specs":[{"name":"ignored"}],"created_at":1,"updated_at":2,"write_access":true,"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)
		case "/api/v1/tools/export":
			_, _ = io.WriteString(w, `[{"id":"z_tool","name":"Z","content":"z","access_grants":[]},{"id":"weather/tool","name":"Weather","content":"class Tools: pass","meta":{"description":"Weather"},"created_at":1}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-tools"})
	if code := app.Run(context.Background(), []string{"tools", "export", "weather_tool", "--manifest", "--out", singleOut}); code != 0 {
		t.Fatalf("single export code=%d stderr=%q", code, errOut.String())
	}
	doc, err := readManifestDocument(singleOut)
	if err != nil {
		t.Fatalf("read single manifest: %v", err)
	}
	if doc.APIVersion != manifestAPIVersion || doc.Kind != "Tool" || doc.Metadata.Name != "weather_tool" {
		t.Fatalf("manifest identity = %#v", doc)
	}
	for _, want := range []string{"id", "name", "content", "meta"} {
		if _, ok := doc.Spec[want]; !ok {
			t.Fatalf("manifest spec missing %q: %#v", want, doc.Spec)
		}
	}
	for _, generated := range []string{"user_id", "specs", "created_at", "updated_at", "write_access"} {
		if _, ok := doc.Spec[generated]; ok {
			t.Fatalf("manifest spec included generated field %q: %#v", generated, doc.Spec)
		}
	}
	if !doc.HasAccessGrants || len(doc.AccessGrants) != 1 || doc.AccessGrants[0].PrincipalID != "*" {
		t.Fatalf("access grants = %+v has=%t", doc.AccessGrants, doc.HasAccessGrants)
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"tools", "export", "--manifest", "--directory", manifestDir}); code != 0 {
		t.Fatalf("directory export code=%d stderr=%q", code, errOut.String())
	}
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		t.Fatalf("read manifest dir: %v", err)
	}
	if len(entries) != 2 || entries[0].Name() != "weather_tool.json" || entries[1].Name() != "z_tool.json" {
		names := []string{}
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("manifest files = %v", names)
	}
	docs, err := loadManifestDocuments(manifestOptions{directories: []string{manifestDir}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load exported manifests: %v", err)
	}
	if len(docs) != 2 || docs[0].Kind != "Tool" || docs[1].Kind != "Tool" {
		t.Fatalf("loaded docs = %+v", docs)
	}
}

func TestToolsValidationErrorsDoNotContactServer(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"tools", "unknown"}, want: `unknown tools command "unknown"`},
		{args: []string{"tools", "get"}, want: "tool id is required"},
		{args: []string{"tools", "delete"}, want: "tool id is required"},
		{args: []string{"tools", "create"}, want: "a JSON payload is required"},
		{args: []string{"tools", "update", "weather_tool"}, want: "a JSON payload is required"},
		{args: []string{"tools", "access-update", "weather_tool"}, want: "a JSON payload is required"},
		{args: []string{"tools", "load-url"}, want: "a JSON payload is required"},
		{args: []string{"tools", "export", "--manifest"}, want: "--directory is required"},
	}
	for _, tt := range tests {
		app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-tools"})
		before := requests
		if code := app.Run(context.Background(), tt.args); code == 0 {
			t.Fatalf("%v exit code = 0, want non-zero", tt.args)
		}
		if out.Len() != 0 {
			t.Fatalf("%v stdout = %q, want empty", tt.args, out.String())
		}
		if !strings.Contains(errOut.String(), tt.want) {
			t.Fatalf("%v stderr missing %q:\n%s", tt.args, tt.want, errOut.String())
		}
		if requests != before {
			t.Fatalf("%v contacted server", tt.args)
		}
	}
}

func TestValveHelpListsGlobalAndUserOperations(t *testing.T) {
	tests := [][]string{
		{"tools", "valves", "--help"},
		{"functions", "valves", "--help"},
	}
	for _, args := range tests {
		app, out, errOut := newTestApp(nil)
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v exit code = %d; stderr=%q", args, code, errOut.String())
		}
		for _, want := range []string{"get", "spec", "update", "user get", "user spec", "user update", "--data", "--file -"} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("%v help missing %q:\n%s", args, want, out.String())
			}
		}
		if errOut.Len() != 0 {
			t.Fatalf("%v stderr = %q, want empty", args, errOut.String())
		}
	}
}

func TestValveRequestsMapToToolAndFunctionEndpoints(t *testing.T) {
	dir := t.TempDir()
	outPath := dir + "/valves.json"
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		if got := r.Header.Get("Authorization"); got != "Bearer sk-valves" {
			t.Fatalf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case "/api/v1/tools/id/weather_tool/valves", "/api/v1/tools/id/weather_tool/valves/spec", "/api/v1/tools/id/weather_tool/valves/user", "/api/v1/tools/id/weather_tool/valves/user/spec",
			"/api/v1/functions/id/fn_a/valves", "/api/v1/functions/id/fn_a/valves/spec", "/api/v1/functions/id/fn_a/valves/user", "/api/v1/functions/id/fn_a/valves/user/spec":
			if r.Method != http.MethodGet {
				t.Fatalf("%s method = %s, want GET", r.URL.Path, r.Method)
			}
			_, _ = io.WriteString(w, `{"path":"`+r.URL.Path+`"}`)
		case "/api/v1/tools/id/weather_tool/valves/update", "/api/v1/tools/id/weather_tool/valves/user/update":
			if r.Method != http.MethodPost || string(body) != `{"enabled":true}` {
				t.Fatalf("tool update request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"accepted":true}`)
		case "/api/v1/functions/id/fn_a/valves/update", "/api/v1/functions/id/fn_a/valves/user/update":
			if r.Method != http.MethodPost || string(body) != `{"level":2}` {
				t.Fatalf("function update request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"accepted":true}`)
		case "/api/v1/tools/id/denied/valves":
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"detail":"bad sk-valves"}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-valves"})
	commands := [][]string{
		{"tools", "valves", "get", "weather_tool"},
		{"tools", "valves", "spec", "weather_tool"},
		{"tools", "valves", "update", "weather_tool", "--data", `{"enabled":true}`},
		{"tools", "valves", "user", "get", "weather_tool"},
		{"tools", "valves", "user", "spec", "weather_tool"},
		{"tools", "valves", "user", "update", "weather_tool", "--data", `{"enabled":true}`},
		{"functions", "valves", "get", "fn_a"},
		{"functions", "valves", "spec", "fn_a"},
		{"functions", "valves", "update", "fn_a", "--data", `{"level":2}`},
		{"functions", "valves", "user", "get", "fn_a"},
		{"functions", "valves", "user", "spec", "fn_a", "--out", outPath},
		{"functions", "valves", "user", "update", "fn_a", "--data", `{"level":2}`},
	}
	for _, args := range commands {
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
	}
	if body, err := os.ReadFile(outPath); err != nil || !strings.Contains(string(body), "/api/v1/functions/id/fn_a/valves/user/spec") {
		t.Fatalf("out file = %q, err=%v", string(body), err)
	}
	for _, want := range []string{
		"GET /api/v1/tools/id/weather_tool/valves", "GET /api/v1/tools/id/weather_tool/valves/spec", "POST /api/v1/tools/id/weather_tool/valves/update", "GET /api/v1/tools/id/weather_tool/valves/user", "GET /api/v1/tools/id/weather_tool/valves/user/spec", "POST /api/v1/tools/id/weather_tool/valves/user/update",
		"GET /api/v1/functions/id/fn_a/valves", "GET /api/v1/functions/id/fn_a/valves/spec", "POST /api/v1/functions/id/fn_a/valves/update", "GET /api/v1/functions/id/fn_a/valves/user", "GET /api/v1/functions/id/fn_a/valves/user/spec", "POST /api/v1/functions/id/fn_a/valves/user/update",
	} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"tools", "valves", "get", "denied"}); code == 0 || !strings.Contains(errOut.String(), "403 Forbidden") || !strings.Contains(errOut.String(), "<redacted>") || strings.Contains(errOut.String(), "sk-valves") {
		t.Fatalf("denied code=%d stderr=%q", code, errOut.String())
	}
}

func TestValveValidationErrorsDoNotContactServer(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"tools", "valves", "bogus", "weather_tool"}, want: `unknown tools valves command "bogus"`},
		{args: []string{"tools", "valves", "get"}, want: "usage: oictl tools valves get <tool-id>"},
		{args: []string{"tools", "valves", "user"}, want: "usage: oictl tools valves user <get|spec|update> <tool-id>"},
		{args: []string{"functions", "valves", "user", "bogus", "fn_a"}, want: `unknown functions valves command "bogus"`},
		{args: []string{"functions", "valves", "update", "fn_a"}, want: "a valve payload is required"},
		{args: []string{"functions", "valves", "user", "update", "fn_a"}, want: "a valve payload is required"},
	}
	for _, tt := range tests {
		app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-secret"})
		before := requests
		if code := app.Run(context.Background(), tt.args); code == 0 {
			t.Fatalf("%v exit code = 0, want non-zero", tt.args)
		}
		if out.Len() != 0 {
			t.Fatalf("%v stdout = %q, want empty", tt.args, out.String())
		}
		if !strings.Contains(errOut.String(), tt.want) {
			t.Fatalf("%v stderr missing %q:\n%s", tt.args, tt.want, errOut.String())
		}
		if strings.Contains(errOut.String(), "sk-secret") {
			t.Fatalf("%v stderr leaked credential: %q", tt.args, errOut.String())
		}
		if requests != before {
			t.Fatalf("%v contacted server", tt.args)
		}
	}
}

func TestValveUpdateSupportsStdinPayload(t *testing.T) {
	stdin, err := os.CreateTemp(t.TempDir(), "stdin-*.json")
	if err != nil {
		t.Fatalf("create stdin file: %v", err)
	}
	if _, err := stdin.WriteString(`{"from":"stdin"}`); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	if _, err := stdin.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek stdin: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = stdin
	defer func() { os.Stdin = oldStdin }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/functions/id/fn_a/valves/user/update" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"from":"stdin"}` {
			t.Fatalf("body = %q", string(body))
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-valves"})
	if code := app.Run(context.Background(), []string{"functions", "valves", "user", "update", "fn_a", "--file", "-"}); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if out.String() != `{"ok":true}` {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestChatsRequestsPayloadsOutputAndConfirmation(t *testing.T) {
	payload := t.TempDir() + "/chats.json"
	if err := os.WriteFile(payload, []byte(`[{"id":"chat-import"}]`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		if got := r.Header.Get("Authorization"); got != "Bearer sk-chat" {
			t.Fatalf("Authorization = %q", got)
		}
		switch r.URL.Path {
		case "/api/v1/chats/list":
			if r.URL.Query().Get("page") != "1" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `[{"id":"c1","title":"Ops Review"}]`)
		case "/api/v1/chats/search":
			if r.URL.Query().Get("query") != "tag:ops" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `[{"id":"c2","title":"Incident"}]`)
		case "/api/v1/chats/import":
			body, _ := io.ReadAll(r.Body)
			if string(body) != `[{"id":"chat-import"}]` {
				t.Fatalf("body = %q", string(body))
			}
			_, _ = io.WriteString(w, `{"imported":1}`)
		case "/api/v1/chats/c1/tags":
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"name":"ops"}` {
				t.Fatalf("tags body = %q", string(body))
			}
			_, _ = io.WriteString(w, `[{"name":"ops"}]`)
		case "/api/v1/chats/c1":
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `true`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-chat"})
	if code := app.Run(context.Background(), []string{"chats", "list", "--page", "1"}); code != 0 {
		t.Fatalf("list code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "TITLE") || !strings.Contains(out.String(), "Ops Review") {
		t.Fatalf("table output = %q", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"chats", "search", "--query", "tag:ops", "--output", "json"}); code != 0 || !strings.Contains(out.String(), "Incident") {
		t.Fatalf("search code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"chats", "import", "--file", payload}); code != 0 {
		t.Fatalf("import code=%d stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"chats", "tags", "set", "c1", "--tag", "ops"}); code != 0 {
		t.Fatalf("tags code=%d stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	before := len(requests)
	if code := app.Run(context.Background(), []string{"chats", "tags", "set", "c1", "--tag", "ops", "--tag", "review"}); code == 0 || !strings.Contains(errOut.String(), "exactly one --tag") {
		t.Fatalf("multiple tags code=%d stderr=%q", code, errOut.String())
	}
	if len(requests) != before {
		t.Fatalf("multiple tags contacted server: %v", requests[before:])
	}
	out.Reset()
	errOut.Reset()
	before = len(requests)
	if code := app.Run(context.Background(), []string{"chats", "delete", "c1"}); code == 0 || !strings.Contains(errOut.String(), "requires --yes") {
		t.Fatalf("delete without confirmation code=%d stderr=%q", code, errOut.String())
	}
	if len(requests) != before {
		t.Fatalf("delete without --yes contacted server: %v", requests[before:])
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"chats", "delete", "c1", "--yes"}); code != 0 {
		t.Fatalf("delete code=%d stderr=%q", code, errOut.String())
	}
}

func TestAnalyticsFiltersAndForbiddenRedaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/analytics/models":
			if r.URL.Query().Get("start_date") != "1700000000" || r.URL.Query().Get("group_id") != "ops" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"models":[{"model_id":"m1","count":2}]}`)
		case "/api/v1/analytics/messages":
			if r.URL.Query().Get("model_id") != "m1" || r.URL.Query().Get("limit") != "25" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `[]`)
		case "/api/v1/analytics/summary":
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"detail":"bad sk-admin"}`)
		default:
			t.Fatalf("unexpected request %s", r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-admin"})
	if code := app.Run(context.Background(), []string{"analytics", "models", "--start-date", "1700000000", "--group-id", "ops"}); code != 0 {
		t.Fatalf("models code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "MODEL_ID") || !strings.Contains(out.String(), "m1") {
		t.Fatalf("analytics table = %q", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"analytics", "messages", "--model-id", "m1", "--limit", "25", "--output", "json"}); code != 0 || out.String() != `[]` {
		t.Fatalf("messages code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"analytics", "summary"}); code == 0 || !strings.Contains(errOut.String(), "403 Forbidden") {
		t.Fatalf("summary code=%d stderr=%q", code, errOut.String())
	}
	if strings.Contains(errOut.String(), "sk-admin") || !strings.Contains(errOut.String(), "<redacted>") {
		t.Fatalf("stderr did not redact token: %q", errOut.String())
	}
}

func TestAutomationsLifecycleRequestsAndConfirmation(t *testing.T) {
	payload := t.TempDir() + "/automation.json"
	if err := os.WriteFile(payload, []byte(`{"title":"nightly","schedule":{"rrule":"bad"}}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.String())
		switch r.URL.Path {
		case "/api/v1/automations/list":
			if r.URL.Query().Get("status") != "active" || r.URL.Query().Get("page") != "1" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `[{"id":"a1","status":"active"}]`)
		case "/api/v1/automations/create":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"rrule":"bad"`) {
				t.Fatalf("body = %q", string(body))
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"detail":"invalid RRULE"}`)
		case "/api/v1/automations/a1/toggle", "/api/v1/automations/a1/run":
			_, _ = io.WriteString(w, `{"id":"a1","status":"inactive"}`)
		case "/api/v1/automations/a1/runs":
			_, _ = io.WriteString(w, `[{"id":"r1","status":"ok"}]`)
		case "/api/v1/automations/a1/delete":
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `true`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-auto"})
	if code := app.Run(context.Background(), []string{"automations", "list", "--page", "1", "--status", "active"}); code != 0 {
		t.Fatalf("list code=%d stderr=%q", code, errOut.String())
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"automations", "create", "--file", payload}); code == 0 || !strings.Contains(errOut.String(), "invalid RRULE") {
		t.Fatalf("create code=%d stderr=%q", code, errOut.String())
	}
	errOut.Reset()
	for _, args := range [][]string{{"automations", "toggle", "a1"}, {"automations", "run", "a1"}, {"automations", "runs", "list", "a1"}} {
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
		errOut.Reset()
	}
	before := len(requests)
	if code := app.Run(context.Background(), []string{"automations", "delete", "a1"}); code == 0 || !strings.Contains(errOut.String(), "requires --yes") {
		t.Fatalf("delete no yes code=%d stderr=%q", code, errOut.String())
	}
	if len(requests) != before {
		t.Fatalf("delete without --yes contacted server")
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"automations", "delete", "a1", "--yes"}); code != 0 {
		t.Fatalf("delete code=%d stderr=%q", code, errOut.String())
	}
}

func TestSCIMUsesDedicatedTokenAndPreservesResponses(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.Header.Get("Authorization"); got != "Bearer scim-secret" {
			t.Fatalf("Authorization = %q", got)
		}
		switch r.URL.Path {
		case "/api/v1/scim/v2/Users":
			if r.URL.Query().Get("startIndex") != "1" || r.URL.Query().Get("count") != "50" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"Resources":[{"id":"u1","userName":"user@example.test"}]}`)
		case "/api/v1/scim/v2/Users/missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"schemas":["urn:ietf:params:scim:api:messages:2.0:Error"],"detail":"missing scim-secret"}`)
		case "/api/v1/scim/v2/Groups/g1":
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "normal-token"})
	if code := app.Run(context.Background(), []string{"scim", "users", "list"}); code == 0 || !strings.Contains(errOut.String(), "SCIM token is required") {
		t.Fatalf("missing scim token code=%d stderr=%q", code, errOut.String())
	}
	if requests != 0 {
		t.Fatalf("normal API token was used for SCIM")
	}
	errOut.Reset()
	app.getenv = func(key string) string {
		if key == "OPEN_WEBUI_URL" {
			return server.URL
		}
		if key == "OPEN_WEBUI_SCIM_TOKEN" {
			return "scim-secret"
		}
		return ""
	}
	if code := app.Run(context.Background(), []string{"scim", "users", "list", "--start-index", "1", "--count", "50"}); code != 0 {
		t.Fatalf("scim list code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Resources") {
		t.Fatalf("scim stdout=%q", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"scim", "users", "get", "missing"}); code == 0 || !strings.Contains(errOut.String(), "404 Not Found") {
		t.Fatalf("scim error code=%d stderr=%q", code, errOut.String())
	}
	if strings.Contains(errOut.String(), "scim-secret") || !strings.Contains(errOut.String(), "<redacted>") {
		t.Fatalf("SCIM token leaked: %q", errOut.String())
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"scim", "groups", "delete", "g1", "--yes"}); code != 0 {
		t.Fatalf("scim delete code=%d stderr=%q", code, errOut.String())
	}
}

func TestProviderPassthroughGuardsRawAndStreaming(t *testing.T) {
	payload := t.TempDir() + "/request.json"
	if err := os.WriteFile(payload, []byte(`{"model":"m1","stream":true}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-provider" {
			t.Fatalf("Authorization = %q", got)
		}
		switch r.URL.Path {
		case "/openai/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"m1"}]}`)
		case "/openai/chat/completions":
			body, _ := io.ReadAll(r.Body)
			if r.Method != http.MethodPost || string(body) != `{"model":"m1","stream":true}` {
				t.Fatalf("request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, "chunk1\n")
			_, _ = io.WriteString(w, "chunk2\n")
		case "/ollama/api/tags":
			_, _ = io.WriteString(w, `{"models":[{"name":"llama"}]}`)
		case "/openai/config/update":
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"detail":"bad sk-provider"}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-provider"})
	if code := app.Run(context.Background(), []string{"providers", "openai", "models"}); code != 0 || !strings.Contains(out.String(), "m1") {
		t.Fatalf("models code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"providers", "ollama", "tags"}); code != 0 || !strings.Contains(out.String(), "llama") {
		t.Fatalf("tags code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"providers", "openai", "request", "--method", "POST", "--file", payload, "--stream", "/chat/completions"}); code != 0 || !strings.Contains(out.String(), "chunk1\nchunk2") {
		t.Fatalf("stream code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"providers", "openai", "request", "https://api.openai.com/v1/models"}); code == 0 || !strings.Contains(errOut.String(), "routed through Open WebUI") {
		t.Fatalf("external url code=%d stderr=%q", code, errOut.String())
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"providers", "openai", "request", "--header", "Authorization: Bearer upstream", "/models"}); code == 0 || !strings.Contains(errOut.String(), "Authorization header") || strings.Contains(errOut.String(), "upstream") {
		t.Fatalf("authorization override code=%d stderr=%q", code, errOut.String())
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"providers", "openai", "config", "update", "--data", `{}`}); code == 0 || !strings.Contains(errOut.String(), "403 Forbidden") {
		t.Fatalf("config update code=%d stderr=%q", code, errOut.String())
	}
	if strings.Contains(errOut.String(), "sk-provider") || !strings.Contains(errOut.String(), "<redacted>") {
		t.Fatalf("provider token leaked: %q", errOut.String())
	}
}
