package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestWebhooksHelp(t *testing.T) {
	app, out, errOut := newTestApp(nil)
	if code := app.Run(context.Background(), []string{"--help"}); code != 0 {
		t.Fatalf("top help code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "webhooks") {
		t.Fatalf("top help missing webhooks:\n%s", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "--help"}); code != 0 {
		t.Fatalf("webhooks help code=%d stderr=%q", code, errOut.String())
	}
	for _, want := range []string{"channels list", "channels ensure", "--expected-url", "--expected-url-env", "channels url", "events catalog", "events enable", "destination URLs"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("webhooks help missing %q:\n%s", want, out.String())
		}
	}
}

func TestChannelWebhookEnsureReusesCreatesAndDetectsDuplicates(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.EscapedPath())
		if got := r.Header.Get("Authorization"); got != "Bearer sk-ensure" {
			t.Fatalf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		switch r.URL.EscapedPath() {
		case "/api/v1/channels/existing/webhooks":
			if r.Method != http.MethodGet {
				t.Fatalf("existing method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `[{"id":"wh-existing","name":"ci-bot","token":"existing-token","url":"https://example.test/api/v1/channels/webhooks/wh-existing/existing-token"},{"id":"wh-other","name":"other"}]`)
		case "/api/v1/channels/existing/webhooks/create":
			t.Fatalf("ensure existing should not create")
		case "/api/v1/channels/missing/webhooks":
			if r.Method != http.MethodGet {
				t.Fatalf("missing list method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `[{"id":"wh-other","name":"other"}]`)
		case "/api/v1/channels/missing/webhooks/create":
			if r.Method != http.MethodPost || string(body) != `{"name":"ci-bot","profile_image_url":"https://images.example/bot.png"}` {
				t.Fatalf("create request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"wh-new","name":"ci-bot","token":"new-token","url":"https://example.test/api/v1/channels/webhooks/wh-new/new-token"}`)
		case "/api/v1/channels/duplicate/webhooks":
			if r.Method != http.MethodGet {
				t.Fatalf("duplicate list method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `[{"id":"wh-a","name":"ci-bot","token":"dup-token-a","url":"https://example.test/api/v1/channels/webhooks/wh-a/dup-token-a"},{"id":"wh-b","name":"ci-bot","token":"dup-token-b","url":"https://example.test/api/v1/channels/webhooks/wh-b/dup-token-b"}]`)
		case "/api/v1/channels/duplicate/webhooks/create":
			t.Fatalf("ensure duplicate should not create")
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-ensure"})
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "existing", "--name", " ci-bot "}); code != 0 {
		t.Fatalf("ensure existing code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "wh-existing") || strings.Contains(out.String(), "existing-token") || strings.Contains(out.String(), "/api/v1/channels/webhooks/wh-existing/existing-token") {
		t.Fatalf("ensure existing output = %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "missing", "--name", "ci-bot", "--profile-image-url", "https://images.example/bot.png"}); code != 0 {
		t.Fatalf("ensure missing code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "wh-new") || strings.Contains(out.String(), "new-token") || strings.Contains(out.String(), "/api/v1/channels/webhooks/wh-new/new-token") {
		t.Fatalf("ensure create output = %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "duplicate", "--name", "ci-bot"}); code == 0 || !strings.Contains(errOut.String(), "ambiguous") {
		t.Fatalf("ensure duplicate code=%d stderr=%q", code, errOut.String())
	}
	for _, leaked := range []string{"dup-token-a", "dup-token-b", "/api/v1/channels/webhooks/wh-a/dup-token-a", "/api/v1/channels/webhooks/wh-b/dup-token-b"} {
		if strings.Contains(errOut.String(), leaked) {
			t.Fatalf("duplicate error leaked %q: %q", leaked, errOut.String())
		}
	}

	before := len(requests)
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "missing"}); code == 0 || !strings.Contains(errOut.String(), "--name is required") {
		t.Fatalf("ensure missing name code=%d stderr=%q", code, errOut.String())
	}
	if len(requests) != before {
		t.Fatalf("ensure missing name contacted server: %v", requests[before:])
	}
}

func TestChannelWebhookEnsureURLModes(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.EscapedPath())
		if got := r.Header.Get("Authorization"); got != "Bearer sk-url" {
			t.Fatalf("Authorization = %q", got)
		}
		switch r.URL.EscapedPath() {
		case "/api/v1/channels/channel-url/webhooks":
			if r.Method != http.MethodGet {
				t.Fatalf("url list method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `[{"id":"wh-url","name":"ci-bot","token":"channel-secret-token","url":"https://example.test/api/v1/channels/webhooks/wh-url/channel-secret-token"}]`)
		case "/api/v1/channels/channel-url/webhooks/create":
			t.Fatalf("ensure url should not create")
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	env := map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-url"}
	app, out, errOut := newTestApp(env)
	wantURL := server.URL + "/api/v1/channels/webhooks/wh-url/channel-secret-token\n"
	wantURLValue := strings.TrimSuffix(wantURL, "\n")
	assertSecretSafe := func(label string) {
		t.Helper()
		combined := out.String() + errOut.String()
		for _, leaked := range []string{"channel-secret-token", wantURLValue} {
			if strings.Contains(combined, leaked) {
				t.Fatalf("%s leaked %q: stdout=%q stderr=%q", label, leaked, out.String(), errOut.String())
			}
		}
	}
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--show-url"}); code != 0 {
		t.Fatalf("ensure show-url code=%d stderr=%q", code, errOut.String())
	}
	if out.String() != wantURL {
		t.Fatalf("ensure show-url = %q, want %q", out.String(), wantURL)
	}

	out.Reset()
	errOut.Reset()
	outFile := t.TempDir() + "/webhook-url.txt"
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--out", outFile}); code != 0 {
		t.Fatalf("ensure out code=%d stderr=%q", code, errOut.String())
	}
	if out.Len() != 0 || strings.Contains(errOut.String(), "channel-secret-token") {
		t.Fatalf("ensure out leaked stdout=%q stderr=%q", out.String(), errOut.String())
	}
	written, err := os.ReadFile(outFile)
	if err != nil || string(written) != wantURL {
		t.Fatalf("written url = %q err=%v", string(written), err)
	}

	out.Reset()
	errOut.Reset()
	jsonOutFile := t.TempDir() + "/webhook-url.txt"
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--out", jsonOutFile, "--output", "json"}); code != 0 {
		t.Fatalf("ensure out json code=%d stderr=%q", code, errOut.String())
	}
	written, err = os.ReadFile(jsonOutFile)
	if err != nil || string(written) != wantURL {
		t.Fatalf("json written url = %q err=%v", string(written), err)
	}
	var jsonOutput map[string]any
	if err := json.Unmarshal([]byte(out.String()), &jsonOutput); err != nil {
		t.Fatalf("json stdout invalid: %v output=%q", err, out.String())
	}
	if jsonOutput["id"] != "wh-url" || jsonOutput["name"] != "ci-bot" || jsonOutput["token"] != nil || jsonOutput["url"] != nil || strings.Contains(out.String(), "channel-secret-token") || strings.Contains(out.String(), wantURLValue) || strings.Contains(out.String(), "<redacted>") {
		t.Fatalf("json stdout unsafe or incomplete: %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	tableOutFile := t.TempDir() + "/webhook-url.txt"
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--out", tableOutFile, "--output", "table"}); code != 0 {
		t.Fatalf("ensure out table code=%d stderr=%q", code, errOut.String())
	}
	written, err = os.ReadFile(tableOutFile)
	if err != nil || string(written) != wantURL {
		t.Fatalf("table written url = %q err=%v", string(written), err)
	}
	if !strings.Contains(out.String(), "ID") || !strings.Contains(out.String(), "wh-url") || !strings.Contains(out.String(), "ci-bot") || strings.Contains(out.String(), "channel-secret-token") || strings.Contains(out.String(), wantURLValue) || strings.Contains(out.String(), "<redacted>") {
		t.Fatalf("table stdout unsafe or incomplete: %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--verify-url"}); code != 0 {
		t.Fatalf("ensure verify code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "verified") || strings.Contains(out.String(), "channel-secret-token") || strings.Contains(out.String(), "/api/v1/channels/webhooks/wh-url/channel-secret-token") {
		t.Fatalf("ensure verify output = %q", out.String())
	}

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "expected URL implies verify", args: []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--expected-url", wantURLValue}},
		{name: "verify with expected URL", args: []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--verify-url", "--expected-url", wantURLValue}},
		{name: "expected URL env implies verify", args: []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--expected-url-env", "WEBHOOK_URL"}},
		{name: "verify with expected URL env", args: []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--verify-url", "--expected-url-env", "WEBHOOK_URL"}},
	} {
		env["WEBHOOK_URL"] = wantURLValue
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), tc.args); code != 0 {
			t.Fatalf("%s code=%d stderr=%q", tc.name, code, errOut.String())
		}
		if !strings.Contains(out.String(), "matched expected URL") {
			t.Fatalf("%s output = %q", tc.name, out.String())
		}
		assertSecretSafe(tc.name)
	}

	out.Reset()
	errOut.Reset()
	mismatchURL := "https://example.invalid/api/v1/channels/webhooks/wh-url/other-secret-token"
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--expected-url", mismatchURL}); code == 0 || !strings.Contains(errOut.String(), "did not match expected URL") {
		t.Fatalf("ensure expected URL mismatch code=%d stderr=%q", code, errOut.String())
	}
	for _, leaked := range []string{"channel-secret-token", wantURLValue, mismatchURL} {
		if strings.Contains(out.String()+errOut.String(), leaked) {
			t.Fatalf("expected URL mismatch leaked %q: stdout=%q stderr=%q", leaked, out.String(), errOut.String())
		}
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot"}); code != 0 {
		t.Fatalf("ensure default code=%d stderr=%q", code, errOut.String())
	}
	if strings.Contains(out.String(), "channel-secret-token") || strings.Contains(out.String(), "/api/v1/channels/webhooks/wh-url/channel-secret-token") {
		t.Fatalf("ensure default leaked output = %q", out.String())
	}

	before := len(requests)
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--show-url", "--verify-url"}); code == 0 || !strings.Contains(errOut.String(), "use only one") {
		t.Fatalf("ensure URL exclusivity code=%d stderr=%q", code, errOut.String())
	}
	if len(requests) != before {
		t.Fatalf("exclusive URL modes contacted server: %v", requests[before:])
	}

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "show with expected URL", args: []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--show-url", "--expected-url", wantURLValue}, want: "use only one"},
		{name: "out with expected URL", args: []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--out", t.TempDir() + "/url.txt", "--expected-url", wantURLValue}, want: "use only one"},
		{name: "conflicting expected sources", args: []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--verify-url", "--expected-url", wantURLValue, "--expected-url-env", "WEBHOOK_URL"}, want: "use only one of --expected-url or --expected-url-env"},
		{name: "missing expected URL env", args: []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--expected-url-env", "MISSING_WEBHOOK_URL"}, want: "environment variable MISSING_WEBHOOK_URL is required"},
		{name: "empty expected URL env name", args: []string{"webhooks", "channels", "ensure", "channel-url", "--name", "ci-bot", "--expected-url-env", ""}, want: "non-empty environment variable name"},
	} {
		before = len(requests)
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), tc.args); code == 0 || !strings.Contains(errOut.String(), tc.want) {
			t.Fatalf("%s code=%d stderr=%q", tc.name, code, errOut.String())
		}
		if len(requests) != before {
			t.Fatalf("%s contacted server: %v", tc.name, requests[before:])
		}
		assertSecretSafe(tc.name)
	}
}

func TestChannelWebhookCommandsRedactionAndURLReveal(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.EscapedPath())
		if got := r.Header.Get("Authorization"); got != "Bearer sk-webhook" {
			t.Fatalf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		switch r.URL.EscapedPath() {
		case "/api/v1/channels/channel-a/webhooks":
			if r.Method != http.MethodGet {
				t.Fatalf("list method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `[{"id":"wh-1","name":"CI","profile_image_url":"https://images.example/bot.png","token":"channel-secret-token","url":"https://example.test/api/v1/channels/webhooks/wh-1/channel-secret-token"}]`)
		case "/api/v1/channels/channel-a/webhooks/create":
			if r.Method != http.MethodPost || string(body) != `{"name":"CI","profile_image_url":"https://images.example/bot.png"}` {
				t.Fatalf("create request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"wh-1","name":"CI","token":"channel-secret-token"}`)
		case "/api/v1/channels/channel-a/webhooks/wh-1/update":
			if r.Method != http.MethodPost || string(body) != `{"name":"Deploy","profile_image_url":"https://images.example/bot.png"}` {
				t.Fatalf("update request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"wh-1","name":"Deploy","token":"channel-secret-token"}`)
		case "/api/v1/channels/channel-a/webhooks/wh-1/delete":
			if r.Method != http.MethodDelete {
				t.Fatalf("delete method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `true`)
		case "/api/v1/channels/denied/webhooks":
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"detail":"no sk-webhook","token":"channel-secret-token","url":"https://example.test/api/v1/channels/webhooks/wh-1/channel-secret-token"}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-webhook"})
	for _, args := range [][]string{
		{"webhooks", "channels", "list", "channel-a"},
		{"webhooks", "channels", "get", "channel-a", "wh-1"},
		{"webhooks", "channels", "create", "channel-a", "--name", "CI", "--profile-image-url", "https://images.example/bot.png"},
		{"webhooks", "channels", "update", "channel-a", "wh-1", "--name", "Deploy"},
	} {
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
		if strings.Contains(out.String(), "channel-secret-token") || strings.Contains(out.String(), "/api/v1/channels/webhooks/wh-1/channel-secret-token") {
			t.Fatalf("default output leaked webhook secret for %v: %q", args, out.String())
		}
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "create", "channel-a", "--name", "CI", "--profile-image-url", "https://images.example/bot.png", "--output", "json"}); code != 0 {
		t.Fatalf("json create code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "channel-secret-token") {
		t.Fatalf("json output should preserve server response: %q", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "url", "channel-a", "wh-1", "--show-url"}); code != 0 {
		t.Fatalf("url reveal code=%d stderr=%q", code, errOut.String())
	}
	wantURL := server.URL + "/api/v1/channels/webhooks/wh-1/channel-secret-token\n"
	if out.String() != wantURL {
		t.Fatalf("url reveal = %q, want %q", out.String(), wantURL)
	}

	out.Reset()
	errOut.Reset()
	outFile := t.TempDir() + "/webhook-url.txt"
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "url", "channel-a", "wh-1", "--out", outFile}); code != 0 {
		t.Fatalf("url out code=%d stderr=%q", code, errOut.String())
	}
	if out.Len() != 0 || strings.Contains(errOut.String(), "channel-secret-token") {
		t.Fatalf("url out leaked stdout=%q stderr=%q", out.String(), errOut.String())
	}
	written, err := os.ReadFile(outFile)
	if err != nil || string(written) != wantURL {
		t.Fatalf("written url = %q err=%v", string(written), err)
	}

	before := len(requests)
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "delete", "channel-a", "wh-1"}); code == 0 {
		t.Fatalf("delete without --yes code=0")
	}
	if len(requests) != before || !strings.Contains(errOut.String(), "requires --yes") {
		t.Fatalf("delete confirmation stderr=%q requests=%v", errOut.String(), requests[before:])
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "delete", "channel-a", "wh-1", "--yes"}); code != 0 {
		t.Fatalf("delete code=%d stderr=%q", code, errOut.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "list", "denied"}); code == 0 || !strings.Contains(errOut.String(), "403 Forbidden") {
		t.Fatalf("forbidden code=%d stderr=%q", code, errOut.String())
	}
	for _, leaked := range []string{"sk-webhook", "channel-secret-token", "/api/v1/channels/webhooks/wh-1/channel-secret-token"} {
		if strings.Contains(errOut.String(), leaked) {
			t.Fatalf("forbidden error leaked %q: %q", leaked, errOut.String())
		}
	}
}

func TestEventWebhookCommandsLifecycleAndRedaction(t *testing.T) {
	payload := t.TempDir() + "/event-webhook.json"
	if err := os.WriteFile(payload, []byte(`{"name":"Audit","url":"https://hooks.example/secret","enabled":true}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.EscapedPath())
		if got := r.Header.Get("Authorization"); got != "Bearer sk-events" {
			t.Fatalf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		switch r.URL.EscapedPath() {
		case "/api/events":
			if r.Method != http.MethodGet {
				t.Fatalf("catalog method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `[{"name":"chat.created","description":"Chat created"}]`)
		case "/api/events/webhooks":
			if r.Method == http.MethodGet {
				_, _ = io.WriteString(w, `[{"id":"ev-1","name":"Audit","enabled":true,"url":"https://hooks.example/secret","events":["chat.created"]}]`)
				return
			}
			if r.Method != http.MethodPost || string(body) != `{"name":"Audit","url":"https://hooks.example/secret","enabled":true}` {
				t.Fatalf("create request = %s body=%q", r.Method, string(body))
			}
			_, _ = io.WriteString(w, `{"id":"ev-1","name":"Audit","url":"https://hooks.example/secret","enabled":true}`)
		case "/api/events/webhooks/ev-1":
			if r.Method == http.MethodDelete {
				_, _ = io.WriteString(w, `true`)
				return
			}
			if r.Method != http.MethodPut {
				t.Fatalf("event update method = %s", r.Method)
			}
			var decoded map[string]any
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decode update body %q: %v", string(body), err)
			}
			if _, ok := decoded["enabled"].(bool); !ok {
				t.Fatalf("enabled not preserved/set in body: %#v", decoded)
			}
			if decoded["name"] != "Audit" || decoded["url"] != "https://hooks.example/secret" {
				t.Fatalf("fields not preserved in body: %#v", decoded)
			}
			_ = json.NewEncoder(w).Encode(decoded)
		case "/api/events/webhooks/bad":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"detail":"invalid url","url":"https://hooks.example/secret"}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk-events"})
	for _, args := range [][]string{
		{"webhooks", "events", "catalog"},
		{"webhooks", "events", "list"},
		{"webhooks", "events", "get", "ev-1"},
		{"webhooks", "events", "create", "--file", payload},
		{"webhooks", "events", "enable", "ev-1"},
		{"webhooks", "events", "disable", "ev-1"},
	} {
		out.Reset()
		errOut.Reset()
		if code := app.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
		if strings.Contains(out.String(), "https://hooks.example/secret") {
			t.Fatalf("default output leaked event URL for %v: %q", args, out.String())
		}
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "events", "get", "ev-1", "--output", "json"}); code != 0 {
		t.Fatalf("event json get code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "https://hooks.example/secret") {
		t.Fatalf("json output should preserve destination URL: %q", out.String())
	}

	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "events", "update", "bad", "--data", `{"url":"not-a-url"}`}); code == 0 || !strings.Contains(errOut.String(), "400 Bad Request") {
		t.Fatalf("invalid payload code=%d stderr=%q", code, errOut.String())
	}
	if strings.Contains(errOut.String(), "https://hooks.example/secret") {
		t.Fatalf("validation error leaked destination URL: %q", errOut.String())
	}
	denied := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-events" {
			t.Fatalf("Authorization = %q", got)
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"detail":"no sk-events","url":"https://hooks.example/secret"}`)
	}))
	defer denied.Close()
	deniedApp, _, deniedErr := newTestApp(map[string]string{"OPEN_WEBUI_URL": denied.URL, "OPEN_WEBUI_API_KEY": "sk-events"})
	if code := deniedApp.Run(context.Background(), []string{"webhooks", "events", "list"}); code == 0 || !strings.Contains(deniedErr.String(), "403 Forbidden") {
		t.Fatalf("denied code=%d stderr=%q", code, deniedErr.String())
	}
	if strings.Contains(deniedErr.String(), "sk-events") || strings.Contains(deniedErr.String(), "https://hooks.example/secret") {
		t.Fatalf("authorization error leaked secret: %q", deniedErr.String())
	}

	before := len(requests)
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "events", "delete", "ev-1"}); code == 0 || !strings.Contains(errOut.String(), "requires --yes") {
		t.Fatalf("delete without --yes code=%d stderr=%q", code, errOut.String())
	}
	if len(requests) != before {
		t.Fatalf("delete without --yes contacted server")
	}
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"webhooks", "events", "delete", "ev-1", "--yes"}); code != 0 {
		t.Fatalf("delete code=%d stderr=%q", code, errOut.String())
	}
}
