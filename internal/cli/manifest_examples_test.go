package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// R16: exercise the literal published examples, not a second hand-maintained
// approximation. Required fields below follow the original 0.11.0 ToolForm,
// FunctionForm, and PromptForm declarations in backend/open_webui/models/.
func TestPublishedManifestExamplesUseOriginalCreateForms(t *testing.T) {
	text, err := os.ReadFile("../../docs/manifests.md")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROMPT_COMMAND", "example-command")
	t.Setenv("TEAM_NAME", "example-team")
	blocks := regexp.MustCompile("(?s)```json\\n(.*?)\\n```").FindAllSubmatch(text, -1)
	seen := map[string]int{}
	for _, block := range blocks {
		var identity struct {
			Kind string `json:"kind"`
		}
		if json.Unmarshal(block[1], &identity) != nil {
			continue
		}
		if identity.Kind != "Tool" && identity.Kind != "Function" && identity.Kind != "Prompt" {
			continue
		}
		seen[identity.Kind]++
		t.Run(fmt.Sprintf("%s_%d", identity.Kind, seen[identity.Kind]), func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "tools"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "tools", "incident_lookup.py"), []byte("class Tools:\n    pass\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "example.json")
			if err := os.WriteFile(path, block[1], 0o600); err != nil {
				t.Fatal(err)
			}
			docs, err := loadManifestDocuments(manifestOptions{files: []string{path}}, manifestHandlers())
			if err != nil {
				t.Fatal(err)
			}
			var stored map[string]any
			posts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					posts++
					if !strings.HasSuffix(r.URL.Path, "/create") {
						t.Errorf("unexpected mutation %s", r.URL.Path)
					}
					if err := json.NewDecoder(r.Body).Decode(&stored); err != nil {
						t.Error(err)
					}
					fields := []string{"name", "content", "id"}
					if identity.Kind == "Prompt" {
						fields[2] = "command"
					}
					for _, field := range fields {
						if value, ok := stored[field].(string); !ok || value == "" {
							t.Errorf("%s create missing required string %s: %#v", identity.Kind, field, stored)
						}
					}
					if identity.Kind != "Prompt" {
						if _, ok := stored["meta"].(map[string]any); !ok {
							t.Errorf("missing required meta object: %#v", stored)
						}
					}
					if identity.Kind == "Prompt" {
						stored["id"] = "generated-prompt-id"
					}
					if identity.Kind == "Function" {
						stored["is_active"], stored["is_global"] = false, false
					}
				}
				_ = json.NewEncoder(w).Encode(stored)
			}))
			defer server.Close()
			var out, errOut bytes.Buffer
			app := New(&out, &errOut, "test")
			app.getenv = func(string) string { return "" }
			app.userConfigDir = func() (string, error) { return dir, nil }
			if _, err := manifestHandlers()[identity.Kind].Create(context.Background(), app, globalOptions{baseURL: server.URL, token: "fixture"}, docs[0]); err != nil {
				t.Fatal(err)
			}
			if posts != 1 {
				t.Fatalf("create mutations=%d", posts)
			}
		})
	}
	for _, kind := range []string{"Prompt", "Tool", "Function"} {
		if seen[kind] == 0 {
			t.Fatalf("no published %s examples found", kind)
		}
	}
}
