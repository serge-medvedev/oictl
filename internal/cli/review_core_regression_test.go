package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// All requests are intercepted locally; no production credentials or endpoints.
func coreReviewApp(t *testing.T, response string) (*App, *int) {
	t.Helper()
	app, _, _ := newTestApp(map[string]string{"OPEN_WEBUI_URL": "http://example.invalid", "OPEN_WEBUI_API_KEY": "synthetic"})
	dir := t.TempDir()
	app.userConfigDir = func() (string, error) { return dir, nil }
	calls := new(int)
	app.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		*calls++
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})}
	return app, calls
}

type coreReviewFailWriter struct{ short bool }

func (w coreReviewFailWriter) Write(p []byte) (int, error) {
	if w.short {
		return len(p) / 2, nil
	}
	return 0, io.ErrClosedPipe
}

func TestReviewCoreR29R34TaskOutput(t *testing.T) {
	for _, mode := range []string{"file", "short", "error"} {
		t.Run(mode, func(t *testing.T) {
			app, _ := coreReviewApp(t, `true`)
			app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"id":"model","name":"Model","meta":{},"params":{}}`
				if r.URL.Path == "/api/v1/tasks/config" {
					body = `{"TASK_MODEL":"model"}`
				}
				if r.URL.Path == "/api/v1/skills/list" {
					body = `[{"id":"skill","name":"Skill"}]`
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})
			args := []string{"tasks", "skills", "attach", "skill"}
			path := filepath.Join(t.TempDir(), "out.json")
			if mode == "file" {
				if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--out", path)
			} else {
				app.out = coreReviewFailWriter{short: mode == "short"}
			}
			code := app.Run(context.Background(), args)
			if mode != "file" {
				if code == 0 {
					t.Fatal("output failure ignored")
				}
				return
			}
			if code != 0 {
				t.Fatalf("code=%d", code)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("permissions=%o", info.Mode().Perm())
			}
			body, err := os.ReadFile(path)
			if err != nil || !json.Valid(body) {
				t.Errorf("readback=%q err=%v", body, err)
			}
		})
	}
}

func TestReviewCoreR34OutputFailures(t *testing.T) {
	for _, args := range [][]string{{"api", "/api/models"}, {"profiles", "get", "work"}, {"models", "delete", "a"}} {
		for _, short := range []bool{false, true} {
			t.Run(strings.Join(args, "_")+"/"+strconv.FormatBool(short), func(t *testing.T) {
				app, _ := coreReviewApp(t, `true`)
				app.out = coreReviewFailWriter{short: short}
				if code := app.Run(context.Background(), args); code == 0 {
					t.Fatal("output failure reported success")
				}
			})
		}
	}
	app, _ := coreReviewApp(t, `true`)
	if code := app.Run(context.Background(), []string{"api", "/api/models", "--out", t.TempDir()}); code == 0 {
		t.Fatal("directory output reported success")
	}
}

func TestReviewCoreR33OrdinaryRequestDeadline(t *testing.T) {
	for _, timeout := range []string{"", "2s", "0s", "-1s", "bad"} {
		t.Run(timeout, func(t *testing.T) {
			app, calls := coreReviewApp(t, `true`)
			transport := app.httpClient.Transport
			valid := timeout == "" || timeout == "2s"
			app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				deadline, ok := r.Context().Deadline()
				want := 30 * time.Second
				if timeout == "2s" {
					want = 2 * time.Second
				}
				remaining := time.Until(deadline)
				if !ok || remaining > want || remaining < want-time.Second {
					t.Errorf("deadline present=%v remaining=%v want=%v", ok, remaining, want)
				}
				return transport.RoundTrip(r)
			})
			args := []string{"api", "/api/models"}
			if timeout != "" {
				args = append(args, "--timeout", timeout)
			}
			code := app.Run(context.Background(), args)
			if (code == 0) != valid || (*calls == 1) != valid {
				t.Errorf("code=%d calls=%d", code, *calls)
			}
		})
	}
}

func TestReviewCoreR33StalledHeadersAndBody(t *testing.T) {
	for _, flush := range []bool{false, true} {
		t.Run(strconv.FormatBool(flush), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if flush {
					w.WriteHeader(200)
					w.(http.Flusher).Flush()
				}
				<-r.Context().Done()
			}))
			defer server.Close()
			app, _ := coreReviewApp(t, `true`)
			app.httpClient = server.Client()
			if code := app.Run(context.Background(), []string{"--base-url", server.URL, "--timeout", "20ms", "api", "/api/models"}); code == 0 {
				t.Fatal("stalled response succeeded")
			}
		})
	}
}

func TestReviewCoreLogoutCleansTokenDespiteOutputFailure(t *testing.T) {
	app, _ := coreReviewApp(t, `true`)
	if err := app.saveProfile("work", profileConfig{BaseURL: "http://example.invalid", Token: "old"}); err != nil {
		t.Fatal(err)
	}
	app.out = coreReviewFailWriter{}
	if code := app.Run(context.Background(), []string{"--profile", "work", "auth", "logout"}); code == 0 {
		t.Fatal("output failure reported success")
	}
	profile, err := app.loadProfile("work")
	if err != nil || profile.Token != "" {
		t.Fatalf("token was not removed: token=%q err=%v", profile.Token, err)
	}
}

func TestReviewCoreR31AuthPersistenceFailures(t *testing.T) {
	for _, action := range []string{"login", "logout"} {
		t.Run(action, func(t *testing.T) {
			app, calls := coreReviewApp(t, `{"token":"new"}`)
			if err := app.saveProfile("work", profileConfig{BaseURL: "http://example.invalid", Token: "old"}); err != nil {
				t.Fatal(err)
			}
			path, _ := app.profilePath("work")
			transport := app.httpClient.Transport
			app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
				return transport.RoundTrip(r)
			})
			var diagnostics strings.Builder
			app.err = &diagnostics
			args := []string{"--profile", "work", "auth", action}
			if action == "login" {
				args = append(args, "--save")
			}
			if code := app.Run(context.Background(), args); code == 0 || *calls != 1 || diagnostics.Len() == 0 {
				t.Fatalf("code=%d calls=%d stderr=%q", code, *calls, diagnostics.String())
			}
		})
	}
	for _, body := range []string{`{`, `null`, `{}`, `{"token":3}`} {
		t.Run("session/"+body, func(t *testing.T) {
			app, _ := coreReviewApp(t, body)
			if code := app.Run(context.Background(), []string{"auth", "login", "--save"}); code == 0 {
				t.Fatal("invalid session saved successfully")
			}
		})
	}
}

func TestReviewCoreR30EffectiveProfilePersistence(t *testing.T) {
	for _, tc := range []struct{ name, env, flag, want string }{{"default", "", "", "default"}, {"environment", "work", "", "work"}, {"explicit", "work", "chosen", "chosen"}} {
		t.Run(tc.name, func(t *testing.T) {
			app, _ := coreReviewApp(t, `{"token":"new-token"}`)
			app.getenv = func(key string) string {
				if key == "OICTL_PROFILE" {
					return tc.env
				}
				return ""
			}
			args := []string{"auth", "login", "--save", "--base-url", "http://example.invalid", "--data", `{}`}
			if tc.flag != "" {
				args = append(args, "--profile", tc.flag)
			}
			if code := app.Run(context.Background(), args); code != 0 {
				t.Fatalf("login code=%d", code)
			}
			path, err := app.profilePath(tc.want)
			if err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(body), "new-token") {
				t.Errorf("saved profile=%q err=%v", body, err)
			}
			target, err := app.resolveTarget(globalOptions{profile: tc.flag}, true)
			if err != nil || target.token != "new-token" {
				t.Errorf("active target=%+v err=%v", target, err)
			}
			args = []string{"auth", "logout"}
			if tc.flag != "" {
				args = append(args, "--profile", tc.flag)
			}
			if code := app.Run(context.Background(), args); code != 0 {
				t.Errorf("logout code=%d", code)
			}
			body, err = os.ReadFile(path)
			if err != nil || strings.Contains(string(body), "new-token") {
				t.Errorf("logout readback=%q err=%v", body, err)
			}
		})
	}
}

func TestReviewCoreR05TaskSkillCompleteDiscovery(t *testing.T) {
	cases := []struct {
		name, ref, first, second, want string
		fail                           bool
	}{
		{"page_two", "second", `{"id":"first","name":"First"}`, `{"id":"second","name":"Second"}`, "second", false},
		{"exact_id_precedence", "second", `{"id":"first","name":"second"}`, `{"id":"second","name":"Second"}`, "second", false},
		{"cross_page_ambiguity", "same", `{"id":"first","name":"same"}`, `{"id":"second","name":"same"}`, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, _ := coreReviewApp(t, `true`)
			pages, posts := 0, 0
			app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{}`
				switch r.URL.Path {
				case "/api/v1/tasks/config":
					body = `{"TASK_MODEL":"model"}`
				case "/api/v1/models/model":
					body = `{"id":"model","name":"Model","meta":{},"params":{}}`
				case "/api/v1/skills/list":
					pages++
					item := tc.first
					if r.URL.Query().Get("page") == "2" {
						item = tc.second
					}
					body = `{"items":[` + item + `],"total":2}`
				case "/api/v1/models/model/update":
					posts++
					var model map[string]any
					if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
						t.Fatal(err)
					}
					ids := model["meta"].(map[string]any)["skillIds"].([]any)
					if len(ids) != 1 || ids[0] != tc.want {
						t.Errorf("attached=%v want=%s", ids, tc.want)
					}
				default:
					t.Errorf("unexpected path=%s", r.URL.Path)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})
			code := app.Run(context.Background(), []string{"tasks", "skills", "attach", tc.ref})
			if (code != 0) != tc.fail || pages != 2 || (posts == 0) != tc.fail {
				t.Fatalf("code=%d pages=%d posts=%d", code, pages, posts)
			}
		})
	}
}

func TestReviewCoreR29ProfileFilePermissions(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(strconv.FormatBool(existing), func(t *testing.T) {
			app, _ := coreReviewApp(t, `true`)
			dir := t.TempDir()
			app.userConfigDir = func() (string, error) { return dir, nil }
			path, err := app.profilePath("work")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if existing {
				if err := os.WriteFile(path, []byte(`old`), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := app.saveProfile("work", profileConfig{Token: "new-secret"}); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("permissions=%o", info.Mode().Perm())
			}
			body, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(body), "new-secret") {
				t.Fatalf("readback=%q err=%v", body, err)
			}
		})
	}
}

func TestReviewCoreR28ModelDeleteBooleanResult(t *testing.T) {
	for _, result := range []string{`true`, `false`, `null`, `{}`} {
		t.Run(result, func(t *testing.T) {
			app, calls := coreReviewApp(t, result)
			var out strings.Builder
			app.out = &out
			code := app.Run(context.Background(), []string{"models", "delete", "a"})
			if (code == 0) != (result == `true`) || *calls != 1 {
				t.Errorf("code=%d requests=%d", code, *calls)
			}
			if (result == `true` || result == `false`) && out.String() != result {
				t.Errorf("output=%q want=%q", out.String(), result)
			}
		})
	}
}

func TestReviewCoreR32StrictBooleanFlags(t *testing.T) {
	for _, name := range []string{"save", "yes", "dry-run", "confirm", "stream", "include-valves", "allow-sensitive-ui-keys", "show-url", "verify-url", "external", "all", "all-users", "allow-ui-settings-extension"} {
		for _, value := range []string{"false", "FALSE", "False", "0", "true", "TRUE", "True", "1", "typo", ""} {
			t.Run(name+"/"+value, func(t *testing.T) {
				flags, _, err := parseCommandFlags([]string{"--" + name + "=" + value})
				want, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					if err == nil {
						t.Fatal("malformed boolean accepted")
					}
					return
				}
				if err != nil || flags.bools[name] != want {
					t.Fatalf("got=%v error=%v want=%v", flags.bools, err, want)
				}
			})
		}
	}
}

func TestReviewCoreR19FilePaginationFilters(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		query string
	}{
		{"files", []string{"files", "list", "--page", "2", "--content", "false"}, "content=false&page=2"},
		{"root", []string{"knowledge", "files", "list", "k", "--page", "2", "--include-content", "true", "--limit", "50", "--directory-id=", "--q", "report"}, "directory_id=&include_content=true&limit=50&page=2&query=report"},
		{"knowledge", []string{"knowledge", "files", "list", "k", "--page", "2", "--query", "report", "--order-by", "filename", "--direction", "asc", "--view-option", "all"}, "direction=asc&order_by=filename&page=2&query=report&view_option=all"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, _ := coreReviewApp(t, `true`)
			app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.RawQuery != tc.query {
					t.Errorf("query=%q want %q", r.URL.RawQuery, tc.query)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"items":[],"total":0}`)), Header: make(http.Header)}, nil
			})
			if code := app.Run(context.Background(), tc.args); code != 0 {
				t.Fatalf("code=%d", code)
			}
		})
	}
}

func TestReviewCoreR17RenameFilename(t *testing.T) {
	app, calls := coreReviewApp(t, `true`)
	if code := app.Run(context.Background(), []string{"files", "rename", "file-a"}); code == 0 || *calls != 0 {
		t.Errorf("missing name: code=%d requests=%d", code, *calls)
	}
	app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["filename"] != "renamed.pdf" || len(body) != 1 {
			t.Errorf("wrong rename form: %#v", body)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`true`)), Header: make(http.Header)}, nil
	})
	if code := app.Run(context.Background(), []string{"files", "rename", "file-a", "--name", "renamed.pdf"}); code != 0 {
		t.Fatalf("code=%d", code)
	}
}

func TestReviewCoreR07ProfileRedactsAllCredentials(t *testing.T) {
	app, _ := coreReviewApp(t, `true`)
	dir := t.TempDir()
	app.userConfigDir = func() (string, error) { return dir, nil }
	if err := app.saveProfile("work", profileConfig{BaseURL: "http://example.invalid", Token: "ordinary-secret", SCIMToken: "provisioning-secret"}); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	app.out = &out
	if code := app.Run(context.Background(), []string{"profiles", "get", "work"}); code != 0 {
		t.Fatalf("code=%d", code)
	}
	if strings.Contains(out.String(), "ordinary-secret") || strings.Contains(out.String(), "provisioning-secret") || !strings.Contains(out.String(), "redacted") {
		t.Fatalf("credential disclosure: %s", out.String())
	}
}

func TestReviewCoreR20ModelBulkEnvelopes(t *testing.T) {
	for _, action := range []string{"import", "sync"} {
		for _, payload := range []string{`[{"id":"a","name":"A","meta":{}}]`, `{"models":[{"id":"a","name":"A","meta":{}},{"id":"b","params":{"temperature":0.2}}]}`} {
			t.Run(action+"/"+payload, func(t *testing.T) {
				app, _ := coreReviewApp(t, `true`)
				app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
					var envelope map[string]any
					if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
						t.Errorf("expected models envelope: %v", err)
					}
					if _, ok := envelope["params"]; ok {
						t.Error("params belongs to models, not envelope")
					}
					items, ok := envelope["models"].([]any)
					if !ok {
						t.Errorf("missing inventory: %#v", envelope)
					}
					for _, item := range items {
						if _, ok := item.(map[string]any)["params"].(map[string]any); !ok {
							t.Errorf("missing model params: %#v", item)
						}
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`true`)), Header: make(http.Header)}, nil
				})
				if code := app.Run(context.Background(), []string{"models", action, "--data", payload}); code != 0 {
					t.Fatalf("code=%d", code)
				}
			})
		}
	}
}

func TestReviewCoreR02ModelSyncInventorySafety(t *testing.T) {
	for _, payload := range []string{`{}`, `{"id":"a"}`, `{"models":null}`, `{"models":{}}`, `null`, `3`, `{`, `{"models":[null]}`, `{"models":[]}`, `[]`} {
		t.Run(payload, func(t *testing.T) {
			app, calls := coreReviewApp(t, `true`)
			if code := app.Run(context.Background(), []string{"models", "sync", "--data", payload}); code == 0 || *calls != 0 {
				t.Fatalf("code=%d requests=%d", code, *calls)
			}
		})
	}
	for _, payload := range []string{`{"models":[]}`, `[]`, `{"models":[{"id":"a","params":{}}]}`} {
		t.Run("confirmed/"+payload, func(t *testing.T) {
			app, calls := coreReviewApp(t, `true`)
			if code := app.Run(context.Background(), []string{"models", "sync", "--yes", "--data", payload}); code != 0 || *calls != 1 {
				t.Fatalf("code=%d requests=%d", code, *calls)
			}
		})
	}
}

func TestReviewCoreRequestBodySourceConflicts(t *testing.T) {
	for _, args := range [][]string{{"--data", `{}`, "--file", "missing"}, {"--data", `{}`, "--data-file", "missing"}, {"--file", "missing", "--data-file", "other"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			flags, _, err := parseCommandFlags(args)
			if err != nil {
				t.Fatal(err)
			}
			_, err = requestBody(flags)
			if err == nil || !strings.Contains(err.Error(), "only one") {
				t.Fatalf("expected source conflict, got %v", err)
			}
		})
	}
}

func TestReviewCoreR35CreatePayloadShapes(t *testing.T) {
	for _, payload := range []string{`null`, `42`, `"text"`, `[]`, `{`} {
		t.Run(payload, func(t *testing.T) {
			app, calls := coreReviewApp(t, `true`)
			if code := app.Run(context.Background(), []string{"models", "create", "--data", payload}); code == 0 || *calls != 0 {
				t.Fatalf("code=%d requests=%d", code, *calls)
			}
		})
	}
}

func TestReviewCoreR35ModelPayloadShapes(t *testing.T) {
	for _, action := range []string{"update", "toggle", "access-update", "delete"} {
		for _, payload := range []string{`null`, `42`, `"text"`, `[]`, `{`} {
			t.Run(action+"/"+payload, func(t *testing.T) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("payload panicked: %v", r)
					}
				}()
				app, calls := coreReviewApp(t, `true`)
				if code := app.Run(context.Background(), []string{"models", action, "A", "--data", payload}); code == 0 || *calls != 0 {
					t.Fatalf("code=%d requests=%d", code, *calls)
				}
			})
		}
	}
}

func TestReviewCoreR03ModelTargetConflicts(t *testing.T) {
	for _, action := range []string{"update", "toggle", "access-update", "delete"} {
		t.Run(action, func(t *testing.T) {
			app, calls := coreReviewApp(t, `true`)
			if code := app.Run(context.Background(), []string{"models", action, "A", "--data", `{"id":"B"}`}); code == 0 || *calls != 0 {
				t.Fatalf("code=%d requests=%d; conflicting target must fail before HTTP", code, *calls)
			}
		})
	}
}
