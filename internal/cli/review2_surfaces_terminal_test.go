package cli

import (
	"context"
	"encoding/json"

	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReview2SurfacesR17TerminalGrantOutput(t *testing.T) {
	for _, action := range []string{"get", "diff", "set"} {
		for _, sink := range []string{"stdout", "new", "existing", "directory", "error-writer", "short-writer"} {
			t.Run(action+"/"+sink, func(t *testing.T) {
				grant := map[string]any{"principal_type": "group", "principal_id": "group-a", "permission": "read"}
				config := map[string]any{"TERMINAL_SERVER_CONNECTIONS": []any{map[string]any{"id": "terminal-a", "name": "Terminal", "config": map[string]any{"access_grants": []any{grant}, "keep": true}}}}
				gets, posts := 0, 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != terminalServersConfigPath {
						t.Errorf("path=%s", r.URL.Path)
					}
					if r.Method == "POST" {
						posts++
						if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
							t.Error(err)
						}
					} else {
						gets++
					}
					json.NewEncoder(w).Encode(config)
				}))
				defer server.Close()
				app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "synthetic"})
				args := []string{"config", "terminal-servers", "access-grants", action, "terminal-a"}
				if action != "get" {
					args = append(args, "--data", `[{"principal_type":"group","principal_id":"group-a","permission":"read"}]`)
				}
				path := filepath.Join(t.TempDir(), "grants.json")
				switch sink {
				case "new", "existing", "directory":
					args = append(args, "--out", path)
					if sink == "existing" {
						if err := os.WriteFile(path, []byte("old bytes"), 0644); err != nil {
							t.Fatal(err)
						}
						if err := os.Chmod(path, 0644); err != nil {
							t.Fatal(err)
						}
					}
					if sink == "directory" {
						if err := os.Mkdir(path, 0700); err != nil {
							t.Fatal(err)
						}
					}
				case "error-writer":
					app.out = reviewSurfaceFailWriter{}
				case "short-writer":
					app.out = reviewSurfaceFailWriter{short: true}
				}
				code := app.Run(context.Background(), args)
				wantPosts := 0
				if action == "set" {
					wantPosts = 1
				}
				if gets != 1 || posts != wantPosts {
					t.Fatalf("gets=%d posts=%d; no mutation replay permitted", gets, posts)
				}
				if sink == "directory" || strings.HasSuffix(sink, "writer") {
					if code == 0 || errOut.Len() == 0 {
						t.Fatalf("failed sink code=%d stderr=%s", code, errOut)
					}
					return
				}
				if code != 0 {
					t.Fatalf("code=%d stderr=%s", code, errOut)
				}
				data := []byte(out.String())
				if sink != "stdout" {
					if out.Len() != 0 {
						t.Fatalf("file output leaked stdout=%s", out)
					}
					var err error
					data, err = os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					info, err := os.Stat(path)
					if err != nil || info.Mode().Perm() != 0600 {
						t.Fatalf("mode=%v err=%v", info, err)
					}
				}
				var got any
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatalf("invalid output=%q: %v", data, err)
				}
				var want any
				switch action {
				case "get":
					want = []any{grant}
				case "diff":
					want = map[string]any{"grants_added": nil, "grants_removed": nil}
				case "set":
					want = map[string]any{"connection": "terminal-a", "access_grants": []any{grant}}
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got=%v want=%v", got, want)
				}
			})
		}
	}
}

func TestReview2SurfacesR17EncodingFailure(t *testing.T) {
	app, out, errOut := newTestApp(nil)
	if code := app.writeTerminalServerGrantOutput(globalOptions{}, commandFlags{}, make(chan int)); code == 0 || out.Len() != 0 || errOut.Len() == 0 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, out, errOut)
	}
}
