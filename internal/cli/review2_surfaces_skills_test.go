package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Source-derived contract fixture: skills.py export_skills:164-206 returns a
// package object for one selected ID. insert_new_skill:171-185 stores supplied
// files, falling back to a root-only file only when files/data are absent.
func TestReview2SurfacesR15SingleSkillPackageRecreation(t *testing.T) {
	packageJSON := `{"id":"skill /α","name":"Package","description":"Keep all files","meta":{"tags":["x"]},"is_active":false,"files":[{"path":"SKILL.md","content":"# Instructions\nSee scripts/helper.py\n"},{"path":"scripts/helper.py","content":"print(\"fixture\")\n"}]}`
	for _, file := range []bool{false, true} {
		t.Run(map[bool]string{false: "stdout", true: "file"}[file], func(t *testing.T) {
			var exported, restored map[string]any
			if err := json.Unmarshal([]byte(packageJSON), &exported); err != nil {
				t.Fatal(err)
			}
			gets, posts := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					posts++
					if r.URL.Path != "/api/v1/skills/create" {
						t.Errorf("create path=%s", r.URL.Path)
					}
					var form map[string]any
					if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
						t.Error(err)
					}
					// The original persistence method gives files/data precedence over content.
					files, ok := form["files"]
					if !ok {
						files = []any{map[string]any{"path": "SKILL.md", "content": form["content"]}}
					}
					restored = map[string]any{"files": files, "name": form["name"], "meta": form["meta"], "is_active": form["is_active"]}
					json.NewEncoder(w).Encode(restored)
					return
				}
				gets++
				if r.URL.Path == "/api/v1/skills/export" {
					if q := r.URL.Query(); q.Get("ids") != "skill /α" || len(q) != 1 {
						t.Errorf("export query=%s", r.URL.RawQuery)
					}
					io.WriteString(w, packageJSON)
				} else {
					// Actual detail shape excludes package files, reproducing the old loss.
					io.WriteString(w, `{"id":"skill /α","name":"Package","content":"# Instructions\nSee scripts/helper.py\n"}`)
				}
			}))
			defer server.Close()
			app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "synthetic"})
			args := []string{"skills", "export", "skill /α"}
			path := filepath.Join(t.TempDir(), "skill.json")
			if file {
				args = append(args, "--out", path)
			}
			if code := app.Run(context.Background(), args); code != 0 {
				t.Fatalf("export code=%d stderr=%s", code, errOut)
			}
			data := []byte(out.String())
			if file {
				if out.Len() != 0 {
					t.Fatalf("unexpected stdout=%s", out)
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
			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, exported) {
				t.Fatalf("incomplete export=%s", data)
			}
			out.Reset()
			if code := app.Run(context.Background(), []string{"skills", "create", "--data", string(data), "--output", "json"}); code != 0 {
				t.Fatalf("recreate code=%d stderr=%s", code, errOut)
			}
			if gets != 1 || posts != 1 || !reflect.DeepEqual(restored["files"], exported["files"]) || restored["is_active"] != false {
				t.Fatalf("gets=%d posts=%d restored=%v", gets, posts, restored)
			}
		})
	}
}
