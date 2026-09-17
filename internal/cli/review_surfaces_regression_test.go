package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type reviewSurfaceOutputCase struct {
	name     string
	run      func(*App, string) int
	fileOnly bool
}

func reviewSurfaceOutputCases() []reviewSurfaceOutputCase {
	ctx := context.Background()
	flags := func(path string) commandFlags {
		return commandFlags{values: map[string]string{"out": path}, bools: map[string]bool{}}
	}
	cases := []reviewSurfaceOutputCase{
		{"local", func(a *App, p string) int { return a.writeLocalOutput([]byte(`{"secret":true}`), p) }, false},
		{"bulk-results", func(a *App, p string) int {
			return a.writeBulkUISettingsResults(globalOptions{}, flags(p), []bulkUISettingsResult{{UserID: "u", Status: "success"}})
		}, false},
		{"raw", func(a *App, p string) int {
			return a.executeRaw(ctx, globalOptions{}, requestSpec{path: "/raw", authRequired: true, outPath: p})
		}, false},
		{"raw-global-out", func(a *App, p string) int {
			return a.executeRaw(ctx, globalOptions{outPath: p}, requestSpec{path: "/raw", authRequired: true})
		}, false},
		{"stream", func(a *App, p string) int {
			return a.executeStream(ctx, globalOptions{}, requestSpec{path: "/raw", authRequired: true, outPath: p})
		}, false},
		{"stream-global-out", func(a *App, p string) int {
			return a.executeStream(ctx, globalOptions{outPath: p}, requestSpec{path: "/raw", authRequired: true})
		}, false},
		{"skill-markdown", func(a *App, p string) int {
			return a.exportSkillManifest(ctx, globalOptions{}, flags(p), []string{"s"})
		}, true},
		{"tool-directory", func(a *App, p string) int {
			f := flags("")
			f.values["directory"] = filepath.Dir(p)
			return a.exportToolManifests(ctx, globalOptions{}, f, nil)
		}, true},
		{"tool-single", func(a *App, p string) int {
			return a.exportToolManifests(ctx, globalOptions{}, flags(p), []string{"output"})
		}, false},
		{"webhook-url", func(a *App, p string) int {
			f := flags(p)
			f.bools["show-url"] = p == ""
			return a.revealChannelWebhookURL(ctx, globalOptions{}, f, "/webhooks", "w")
		}, false},
		{"webhook-ensure-url", func(a *App, p string) int {
			f := flags(p)
			f.values["name"] = "Hook"
			f.bools["show-url"] = p == ""
			return a.ensureChannelWebhook(ctx, globalOptions{}, f, "/webhooks", "c", "json")
		}, false},
	}
	for _, format := range []string{"json", "table"} {
		cases = append(cases,
			reviewSurfaceOutputCase{"structured-" + format, func(a *App, p string) int {
				return a.executeStructured(ctx, globalOptions{}, requestSpec{path: "/raw", authRequired: true, outPath: p}, format)
			}, false},
			reviewSurfaceOutputCase{"webhook-" + format, func(a *App, p string) int {
				return a.writeWebhookStructured(globalOptions{}, p, []byte(`{"id":"w","token":"secret"}`), format)
			}, false},
			reviewSurfaceOutputCase{"functions-" + format, func(a *App, p string) int { return a.listFunctions(ctx, globalOptions{}, flags(p), format) }, false},
		)
	}
	return cases
}

func reviewSurfaceOutputApp() (*App, *strings.Builder) {
	app, _, _ := newTestApp(map[string]string{"OPEN_WEBUI_URL": "https://fixture.invalid", "OPEN_WEBUI_API_KEY": "test"})
	errOut := &strings.Builder{}
	app.err = errOut
	app.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"id":"s","name":"Skill","content":"source","secret":true}`
		switch r.URL.Path {
		case "/webhooks":
			body = `[{"id":"w","name":"Hook","token":"secret"}]`
		case "/api/v1/functions/list":
			body = `[{"id":"f","name":"Function","type":"filter"}]`
		case "/api/v1/tools/export":
			body = `[{"id":"output","name":"Tool","content":"source","meta":{}}]`
		case "/api/v1/tools/id/output":
			body = `{"id":"output","name":"Tool","content":"source","meta":{}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	return app, errOut
}

type reviewSurfaceFailWriter struct{ short bool }

func (w reviewSurfaceFailWriter) Write(p []byte) (int, error) {
	if w.short {
		return len(p) / 2, nil
	}
	return 0, errors.New("sink failed")
}

func TestReviewSurfacesOutputFailures(t *testing.T) {
	cases := reviewSurfaceOutputCases()
	for _, compare := range []bool{false, true} {
		cases = append(cases, reviewSurfaceOutputCase{"webhook-verify" + map[bool]string{true: "-compare", false: ""}[compare], func(a *App, p string) int {
			f := commandFlags{values: map[string]string{"name": "Hook"}, bools: map[string]bool{"verify-url": true}}
			if compare {
				f.values["expected-url"] = "https://fixture.invalid/api/v1/channels/webhooks/w/secret"
			}
			return a.ensureChannelWebhook(context.Background(), globalOptions{}, f, "/webhooks", "c", "json")
		}, false})
	}
	for _, tc := range cases {
		if tc.fileOnly {
			continue
		}
		for _, short := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{true: "-short", false: "-error"}[short], func(t *testing.T) {
				app, errOut := reviewSurfaceOutputApp()
				app.out = reviewSurfaceFailWriter{short: short}
				if code := tc.run(app, ""); code == 0 || errOut.Len() == 0 {
					t.Fatalf("code=%d stderr=%s", code, errOut)
				}
			})
		}
	}
	for _, tc := range reviewSurfaceOutputCases() {
		t.Run(tc.name+"-file-error", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "output.json")
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			app, errOut := reviewSurfaceOutputApp()
			if code := tc.run(app, path); code == 0 || errOut.Len() == 0 {
				t.Fatalf("code=%d stderr=%s", code, errOut)
			}
		})
	}
}

type reviewSurfaceCloseFailure struct{ io.Reader }

func (reviewSurfaceCloseFailure) Close() error { return errors.New("response close failed") }
func TestReviewSurfacesStreamCloseFailure(t *testing.T) {
	app, errOut := reviewSurfaceOutputApp()
	app.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: reviewSurfaceCloseFailure{strings.NewReader("data")}, Header: make(http.Header)}, nil
	})}
	if code := app.executeStream(context.Background(), globalOptions{}, requestSpec{path: "/stream", authRequired: true}); code == 0 || !strings.Contains(errOut.String(), "close failed") {
		t.Fatalf("code=%d stderr=%s", code, errOut)
	}
}

func TestReviewSurfacesSecureOutputVariants(t *testing.T) {
	for _, tc := range reviewSurfaceOutputCases() {
		for _, existing := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{true: "-existing", false: "-new"}[existing], func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "output.json")
				if existing {
					if err := os.WriteFile(path, []byte("old secret"), 0644); err != nil {
						t.Fatal(err)
					}
					if err := os.Chmod(path, 0644); err != nil {
						t.Fatal(err)
					}
				}
				app, errOut := reviewSurfaceOutputApp()
				if code := tc.run(app, path); code != 0 {
					t.Fatalf("code=%d stderr=%s", code, errOut)
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != 0600 || len(data) == 0 || string(data) == "old secret" {
					t.Fatalf("mode=%o data=%q", info.Mode().Perm(), data)
				}
			})
		}
	}
}

func TestReviewSurfacesExplicitExtensionAndStrictBooleans(t *testing.T) {
	for _, action := range []string{"patch", "bulk-patch"} {
		for _, preview := range []bool{false, true} {
			if action == "patch" && preview {
				continue
			}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/users/other/settings/ui" {
					t.Errorf("wrong target: %s %s", r.Method, r.URL.Path)
				}
				io.WriteString(w, `{}`)
			}))
			app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
			args := []string{"users", "ui-settings", action, "other", "--allow-ui-settings-extension", "--data", `{"theme":"dark"}`}
			if preview {
				args = append(args, "--dry-run")
			}
			code := app.Run(context.Background(), args)
			wantCalls := 1
			if preview {
				wantCalls = 0
			}
			if code != 0 || calls != wantCalls {
				t.Errorf("args=%v code=%d calls=%d stderr=%s", args, code, calls, errOut)
			}
			server.Close()
		}
	}
	for _, value := range []string{"FALSE", "False", "false", "0", "nope"} {
		for _, args := range [][]string{
			{"functions", "sync", "--yes=" + value, "--data", `[]`},
			{"users", "ui-settings", "patch", "other", "--allow-ui-settings-extension=" + value, "--data", `{}`},
		} {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; io.WriteString(w, `{}`) }))
			app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
			if code := app.Run(context.Background(), args); code == 0 || calls != 0 {
				t.Errorf("args=%v code=%d calls=%d stderr=%s", args, code, calls, errOut)
			}
			server.Close()
		}
	}
}

func TestReviewSurfacesChatsRejectConflictingScopes(t *testing.T) {
	for _, flags := range [][]string{{"--user-id", "other", "--archived", "true"}, {"--user-id", "other", "--folder-id", "folder"}, {"--folder-id", "folder", "--archived", "true"}, {"--user-id", "other", "--archived", "false"}} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; io.WriteString(w, `[]`) }))
		app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
		code := app.Run(context.Background(), append([]string{"chats", "list"}, flags...))
		if code == 0 || calls != 0 {
			t.Errorf("flags=%v code=%d calls=%d stderr=%s", flags, code, calls, errOut)
		}
		server.Close()
	}
}

func TestReviewSurfacesChatsExportScope(t *testing.T) {
	for _, tc := range []struct {
		name           string
		flags          []string
		path, response string
	}{
		{"own NDJSON", nil, "/api/v1/chats/all", "{\"id\":\"a\"}\n{\"id\":\"b\"}\n"},
		{"own single NDJSON", nil, "/api/v1/chats/all", "{\"id\":\"a\"}\n"},
		{"explicit admin", []string{"--all-users"}, "/api/v1/chats/all/db", `[{"id":"other"}]`},
		{"explicit chat", []string{"--chat-id", "chat"}, "/api/v1/chats/chat", `{"id":"chat","chat":{"messages":[]}}`},
	} {
		for _, file := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{true: " file", false: " stdout"}[file], func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "GET" || r.URL.Path != tc.path {
						t.Errorf("route %s %s want %s", r.Method, r.URL.Path, tc.path)
					}
					io.WriteString(w, tc.response)
				}))
				defer server.Close()
				app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
				args := append([]string{"chats", "export"}, tc.flags...)
				path := t.TempDir() + "/export"
				if file {
					args = append(args, "--out", path)
				}
				code := app.Run(context.Background(), args)
				got := out.String()
				if file {
					data, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					got = string(data)
				}
				if code != 0 || got != tc.response {
					t.Fatalf("code=%d got=%q want=%q stderr=%s", code, got, tc.response, errOut)
				}
			})
		}
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; io.WriteString(w, `{}`) }))
	defer server.Close()
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
	if code := app.Run(context.Background(), []string{"chats", "export", "--all-users", "--chat-id", "chat"}); code == 0 || calls != 0 {
		t.Fatalf("conflicting export code=%d calls=%d stderr=%s", code, calls, errOut)
	}
}

func TestReviewSurfacesSkillExportJSON(t *testing.T) {
	for _, one := range []bool{false, true} {
		for _, file := range []bool{false, true} {
			record := `{"id":"s","user_id":"u","name":"Skill","updated_at":1,"content":"full source","meta":{"tags":["x"]},"access_grants":[]}`
			response := record
			if !one {
				response = "[" + record + "]"
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, response) }))
			app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
			args := []string{"skills", "export"}
			if one {
				args = append(args, "s")
			}
			path := t.TempDir() + "/skill.json"
			if file {
				args = append(args, "--out", path)
			}
			code := app.Run(context.Background(), args)
			got := out.String()
			if file {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				got = string(data)
			}
			if code != 0 || got != response || !json.Valid([]byte(got)) {
				t.Errorf("one=%v file=%v code=%d got=%s stderr=%s", one, file, code, got, errOut)
			}
			server.Close()
		}
	}
}

func TestReviewSurfacesWebhookReplacementContract(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		wantName  string
		wantImage any
		valid     bool
		fetch     bool
	}{
		{name: "create requires name", args: []string{"create", "c"}},
		{name: "create blank name", args: []string{"create", "c", "--name", " "}},
		{name: "create image only", args: []string{"create", "c", "--profile-image-url", "new"}},
		{name: "create raw missing name", args: []string{"create", "c", "--data", `{}`}},
		{name: "create named", args: []string{"create", "c", "--name", "New"}, wantName: "New", valid: true},
		{name: "rename preserves image", args: []string{"update", "c", "w", "--name", "New"}, wantName: "New", wantImage: "old-image", valid: true, fetch: true},
		{name: "image preserves name", args: []string{"update", "c", "w", "--profile-image-url", "new-image"}, wantName: "Old", wantImage: "new-image", valid: true, fetch: true},
		{name: "clear image explicitly", args: []string{"update", "c", "w", "--profile-image-url", ""}, wantName: "Old", wantImage: "", valid: true, fetch: true},
		{name: "raw replacement does not merge", args: []string{"update", "c", "w", "--data", `{"name":"Raw"}`}, wantName: "Raw", valid: true},
		{name: "update requires input", args: []string{"update", "c", "w"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gets, posts := 0, 0
			// Model the original replacement form: omitted image becomes null.
			stored := map[string]any{"name": "Old", "profile_image_url": "old-image"}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					gets++
					io.WriteString(w, `[{"id":"w","name":"Old","profile_image_url":"old-image","token":"secret"}]`)
					return
				}
				posts++
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if name, ok := body["name"].(string); !ok || strings.TrimSpace(name) == "" {
					w.WriteHeader(422)
					return
				}
				stored["name"], stored["profile_image_url"] = body["name"], body["profile_image_url"]
				json.NewEncoder(w).Encode(stored)
			}))
			defer server.Close()
			app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
			code := app.Run(context.Background(), append([]string{"webhooks", "channels"}, tc.args...))
			if !tc.valid {
				if code == 0 || gets+posts != 0 {
					t.Fatalf("invalid code=%d gets=%d posts=%d stderr=%s", code, gets, posts, errOut)
				}
				return
			}
			wantGets := 0
			if tc.fetch {
				wantGets = 1
			}
			if code != 0 || gets != wantGets || posts != 1 || stored["name"] != tc.wantName || stored["profile_image_url"] != tc.wantImage {
				t.Fatalf("code=%d gets=%d posts=%d stored=%v stderr=%s", code, gets, posts, stored, errOut)
			}
		})
	}
}

func TestReviewSurfacesSkillMarkdownQuotedRoundTrip(t *testing.T) {
	original := map[string]any{"id": "s", "name": "Quoted \"name\" & <tag>", "description": "line one\nline two\\path\t雪", "content": "# Body\n", "meta": map[string]any{"tags": []string{"a,b", "x & y", `a\b`, "quote\"", "line\nbreak"}}}
	body, _ := json.Marshal(original)
	for i := 0; i < 2; i++ {
		markdown, err := skillManifestFromJSON(body)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parseSkillManifest(markdown)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(parsed, original) {
			t.Fatalf("roundtrip %d got=%#v want=%#v markdown=%s", i, parsed, original, markdown)
		}
		body, _ = json.Marshal(parsed)
	}
	for _, tc := range []struct {
		raw, want string
		bad       bool
	}{
		{`'it''s a \\path'`, `it's a \\path`, false}, {`"a\n\u0026"`, "a\n&", false}, {`"bad\q"`, "", true}, {`'unterminated`, "", true}, {`'bad'quote'`, "", true},
	} {
		got, err := parseManifestScalar(tc.raw)
		if (err != nil) != tc.bad || !tc.bad && got != tc.want {
			t.Errorf("scalar %q got=%q err=%v", tc.raw, got, err)
		}
	}
}

func TestReviewSurfacesUISettingsExtensionGate(t *testing.T) {
	for _, args := range [][]string{{"patch", "other"}, {"bulk-patch", "other"}, {"bulk-patch", "other", "--dry-run"}, {"bulk-patch", "--all", "--dry-run"}} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			io.WriteString(w, `{"users":[{"id":"other"}],"total":1}`)
		}))
		app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
		cmd := append([]string{"users", "ui-settings"}, args...)
		cmd = append(cmd, "--data", `{"theme":"dark"}`)
		code := app.Run(context.Background(), cmd)
		if code == 0 || calls != 0 || !strings.Contains(errOut.String(), "--allow-ui-settings-extension") || !strings.Contains(errOut.String(), "original API") {
			t.Errorf("cmd=%v code=%d calls=%d stderr=%s", cmd, code, calls, errOut)
		}
		server.Close()
	}
}

func TestReviewSurfacesChatTagDelete(t *testing.T) {
	for _, flags := range [][]string{{"--tag", "ops"}, {"--data", `{"name":"ops"}`}, nil, {"--tag", "a", "--tag", "b"}} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			var body map[string]string
			err := json.NewDecoder(r.Body).Decode(&body)
			if r.Method != "DELETE" || r.URL.Path != "/api/v1/chats/chat/tags" || err != nil || body["name"] != "ops" {
				t.Errorf("request %s %s body=%v err=%v", r.Method, r.URL.Path, body, err)
			}
			io.WriteString(w, `true`)
		}))
		app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
		code := app.Run(context.Background(), append([]string{"chats", "tags", "delete", "chat"}, flags...))
		valid := len(flags) == 2
		if valid && (code != 0 || calls != 1) || !valid && (code == 0 || calls != 0) {
			t.Errorf("flags=%v code=%d calls=%d stderr=%s", flags, code, calls, errOut)
		}
		server.Close()
	}
}

func TestReviewSurfacesFunctionSyncEnvelope(t *testing.T) {
	for _, input := range []string{`{}`, `{"id":"f"}`, `{"models":[]}`, `{"functions":null}`, `{"functions":{}}`, `{"functions":"bad"}`, `null`, `true`, `[`, `[] trailing`} {
		t.Run(input, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; io.WriteString(w, `{}`) }))
			defer server.Close()
			app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
			code := app.Run(context.Background(), []string{"functions", "sync", "--yes", "--data", input})
			if code == 0 || calls != 0 {
				t.Fatalf("unsafe envelope code=%d calls=%d stderr=%s", code, calls, errOut)
			}
		})
	}
	for _, input := range []string{`[]`, `{"functions":[]}`, `[{"id":"f"}]`, `{"functions":[{"id":"f"}],"extension":true}`} {
		body, err := functionSyncBody(commandFlags{values: map[string]string{"data": input}})
		if err != nil {
			t.Fatal(err)
		}
		var envelope map[string]json.RawMessage
		if err := json.NewDecoder(body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		var inventory []any
		if err := json.Unmarshal(envelope["functions"], &inventory); err != nil || inventory == nil {
			t.Fatalf("inventory=%s err=%v", envelope["functions"], err)
		}
	}
}
