package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// Executes examples extracted from the built CLI's published help, not copies of
// help constants. Fixtures assert the independent source-derived wire contracts.
// This is local CLI acceptance, not live Open WebUI/deployment acceptance.
func TestJSONHelpPublishedExamplesRealCLI(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "oictl")
	build := exec.Command("go", "build", "-o", bin, "./cmd/oictl")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	env := []string{"HOME=" + dir, "XDG_CONFIG_HOME=" + filepath.Join(dir, "config")}
	run := func(t *testing.T, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Env = env
		var out, errOut bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errOut
		if err := cmd.Run(); err != nil {
			t.Fatalf("%v: %v stdout=%s stderr=%s", args, err, out.String(), errOut.String())
		}
		return out.String()
	}
	example := func(key string, idx int) string {
		t.Helper()
		out := run(t, append(strings.Fields(key), "--help")...)
		_, section, ok := strings.Cut(out, "Examples:\n")
		if !ok {
			t.Fatal("missing examples")
		}
		if strings.Contains(out, "\nSource:") {
			t.Fatal("published help contains a source-provenance footer")
		}
		lines := strings.Split(strings.TrimSpace(section), "\n")
		if idx >= len(lines) {
			t.Fatal("missing example")
		}
		return strings.TrimSpace(lines[idx])
	}
	cases := []struct {
		key                string
		args               []string
		method, path, want string
	}{
		{"users create", []string{"users", "create"}, "POST", "/api/v1/auths/add", `{"name":"Example User","email":"user@example.com","password":"example-password"}`},
		{"config terminal-servers set", []string{"config", "terminal-servers", "set"}, "POST", "/api/v1/configs/terminal_servers", `{"TERMINAL_SERVER_CONNECTIONS":[{"id":"shell-a","name":"Shell","url":"https://terminal.example.com","config":{"access_grants":[]}}]}`},
		{"models import", []string{"models", "import"}, "POST", "/api/v1/models/import", `{"models":[{"id":"example-model","name":"Example Model","meta":{},"params":{}}]}`},
		{"knowledge files batch-add", []string{"knowledge", "files", "batch-add", "knowledge-a"}, "POST", "/api/v1/knowledge/knowledge-a/files/batch/add", `[{"file_id":"file-a","directory_id":null}]`},
		{"automations create", []string{"automations", "create"}, "POST", "/api/v1/automations/create", `{"name":"Example schedule","data":{"prompt":"Summarize the day","model_id":"example-model","rrule":"FREQ=DAILY;BYHOUR=9;BYMINUTE=0;BYSECOND=0","terminal":{"server_id":"shell-a","cwd":"/tmp"}},"is_active":false}`},
		{"scim users create", []string{"scim", "users", "create", "--scim-token", "example-scim-token"}, "POST", "/api/v1/scim/v2/Users", `{"userName":"user@example.com","displayName":"Example User","emails":[{"value":"user@example.com","primary":true}],"active":true}`},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			payload := example(tc.key, 0)
			var mu sync.Mutex
			requests := 0
			var observed string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests++
				b, _ := io.ReadAll(r.Body)
				observed = string(b)
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("wire %s %s; expected %s %s", r.Method, r.URL.Path, tc.method, tc.path)
				}
				assertJSONHelpEqual(t, tc.want, observed)
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{}`)
			}))
			defer server.Close()
			args := append([]string{"--base-url", server.URL, "--token", "example-token", "--output", "json"}, tc.args...)
			run(t, append(args, "--data", payload)...)
			mu.Lock()
			defer mu.Unlock()
			if requests != 1 {
				t.Errorf("requests=%d", requests)
			}
			t.Logf("published example accepted: %s %s (%d request)", tc.method, tc.path, requests)
		})
	}
	t.Run("multipart metadata", func(t *testing.T) {
		payload := example("files upload", 0)
		file := filepath.Join(dir, "example.txt")
		os.WriteFile(file, []byte("file bytes"), 0600)
		requests := 0
		var mu sync.Mutex
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			requests++
			if r.Method != "POST" || r.URL.Path != "/api/v1/files/" {
				t.Errorf("unexpected upload %s %s", r.Method, r.URL.Path)
			}
			if err := r.ParseMultipartForm(1024 * 1024); err != nil {
				t.Error(err)
				return
			}
			defer r.MultipartForm.RemoveAll()
			assertJSONHelpEqual(t, `{"file_hash":"example-checksum","language":"en"}`, r.FormValue("metadata"))
			f, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
				return
			}
			defer f.Close()
			b, _ := io.ReadAll(f)
			if string(b) != "file bytes" {
				t.Errorf("file=%q", b)
			}
			io.WriteString(w, `{"id":"file-a"}`)
		}))
		defer server.Close()
		run(t, "--base-url", server.URL, "--token", "example-token", "files", "upload", file, "--metadata", payload)
		mu.Lock()
		defer mu.Unlock()
		if requests != 1 {
			t.Errorf("requests=%d", requests)
		}
	})
	t.Run("manifest apply", func(t *testing.T) {
		payload := example("manifests apply", 5)
		path := filepath.Join(dir, "model.json")
		os.WriteFile(path, []byte(payload), 0600)
		var mu sync.Mutex
		requests := []string{}
		created := false
		state := `{"id":"example-model","name":"Example Model","meta":{},"params":{},"is_active":true,"access_grants":[]}`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			requests = append(requests, r.Method+" "+r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			switch r.Method + " " + r.URL.Path {
			case "GET /api/v1/models/list":
				if created {
					fmt.Fprintf(w, `{"items":[%s],"total":1}`, state)
				} else {
					io.WriteString(w, `{"items":[],"total":0}`)
				}
			case "POST /api/v1/models/create":
				b, _ := io.ReadAll(r.Body)
				assertJSONHelpEqual(t, `{"id":"example-model","name":"Example Model","meta":{},"params":{}}`, string(b))
				created = true
				io.WriteString(w, state)
			case "GET /api/v1/models/model":
				if r.URL.Query().Get("id") != "example-model" {
					t.Error("wrong resolved ID")
				}
				io.WriteString(w, state)
			default:
				t.Errorf("unexpected manifest request %s %s", r.Method, r.URL.Path)
				http.Error(w, "unexpected", 400)
			}
		}))
		defer server.Close()
		run(t, "--base-url", server.URL, "--token", "example-token", "manifests", "apply", "--file", path)
		mu.Lock()
		defer mu.Unlock()
		if !created {
			t.Error("manifest example did not create")
		}
		mutations := 0
		for _, r := range requests {
			if strings.HasPrefix(r, "POST ") {
				mutations++
			}
		}
		if mutations != 1 {
			t.Errorf("mutations=%d requests=%v", mutations, requests)
		}
		t.Logf("published manifest execution: %v", requests)
	})
	// Real executable stdin remains open without data. Help must not block on reads.
	t.Run("offline stdin and out", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		outPath := filepath.Join(dir, "not-created.json")
		cmd := exec.CommandContext(ctx, bin, "users", "create", "--file", "-", "--out", outPath, "--help")
		cmd.Env = env
		pipe, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		defer pipe.Close()
		if b, err := cmd.CombinedOutput(); err != nil || !bytes.Contains(b, []byte("Input:")) {
			t.Fatalf("inert help %v: %s", err, b)
		}
		if _, err := os.Stat(outPath); !os.IsNotExist(err) {
			t.Errorf("help created --out: %v", err)
		}
	})
}

func assertJSONHelpEqual(t *testing.T, want, got string) {
	t.Helper()
	var a, b any
	if err := json.Unmarshal([]byte(want), &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(got), &b); err != nil {
		t.Errorf("bad JSON %q: %v", got, err)
		return
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("wire JSON\n got: %s\nwant: %s", got, want)
	}
}
