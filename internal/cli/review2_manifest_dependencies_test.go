package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// f8ae8a6 routers/functions.py:214-220,471-489 and routers/tools.py:407-413,744-753:
// create lowercases valid IDs; valve routes address the resulting owner ID.
func TestReview2ManifestCanonicalValveDependencies(t *testing.T) {
	for _, kind := range []string{"Tool", "Function"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			writeManifest(t, dir+"/o.json", `{"apiVersion":"oictl.openwebui/v1","kind":"`+kind+`","metadata":{"name":"OpsFilter"},"spec":{"id":"OpsFilter","name":"Owner","content":"source"}}`)
			writeManifest(t, dir+"/v.json", `{"apiVersion":"oictl.openwebui/v1","kind":"`+kind+`Valve","metadata":{"name":"OpsFilter"},"spec":{"enabled":true}}`)
			resource := strings.ToLower(kind) + "s"
			var remote map[string]any
			var valves map[string]any
			mutations, wrongIDs := 0, 0
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				prefix := "/api/v1/" + resource
				if strings.HasSuffix(r.URL.Path, "/list") {
					items := []any{}
					if remote != nil {
						items = append(items, remote)
					}
					_ = json.NewEncoder(w).Encode(items)
					return
				}
				if r.URL.Path == prefix+"/create" {
					mutations++
					_ = json.NewDecoder(r.Body).Decode(&remote)
					remote["id"] = strings.ToLower(stringValue(remote["id"]))
					remote["is_active"] = false
					remote["is_global"] = false
				} else if strings.Contains(r.URL.Path, "/id/") {
					if !strings.HasPrefix(r.URL.Path, prefix+"/id/opsfilter") {
						wrongIDs++
						http.NotFound(w, r)
						return
					}
					if strings.Contains(r.URL.Path, "/valves") {
						if remote == nil {
							http.NotFound(w, r)
							return
						}
						if r.Method == http.MethodPost {
							mutations++
							_ = json.NewDecoder(r.Body).Decode(&valves)
						}
						_ = json.NewEncoder(w).Encode(valves)
						return
					}
				}
				_ = json.NewEncoder(w).Encode(remote)
			})
			args := []string{"manifests", "apply", "--directory", dir}
			for i := 0; i < 2; i++ {
				if code := app.Run(context.Background(), args); code != 0 {
					t.Fatalf("code=%d mutations=%d wrongIDs=%d error=%v", code, mutations, wrongIDs, app.err)
				}
			}
			if mutations != 2 || wrongIDs != 0 || valves["enabled"] != true {
				t.Fatalf("dependencies do not converge: mutations=%d wrongIDs=%d valves=%v", mutations, wrongIDs, valves)
			}
		})
	}
}

func TestReview2ManifestOwnerExactIDPrecedence(t *testing.T) {
	for _, kind := range []string{"Tool", "Function"} {
		t.Run(kind, func(t *testing.T) {
			exact := map[string]any{"id": "exact", "name": "Exact", "content": "source", "meta": map[string]any{}}
			other := map[string]any{"id": "other", "name": "Other", "content": "source", "meta": map[string]any{}}
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/list") {
					_ = json.NewEncoder(w).Encode([]any{other, exact})
					return
				}
				if strings.Contains(r.URL.Path, "/id/exact") {
					_ = json.NewEncoder(w).Encode(exact)
				} else {
					_ = json.NewEncoder(w).Encode(other)
				}
			})
			state, err := manifestHandlers()[kind].Lookup(context.Background(), app, globalOptions{}, manifestDocument{Kind: kind, Metadata: manifestMetadata{Name: "exact"}, Spec: map[string]any{"id": "other"}})
			if err != nil || state == nil || state.ID != "exact" {
				t.Fatalf("metadata exact ID lost precedence: state=%+v err=%v", state, err)
			}
		})
	}
}

func TestReview2ManifestCanonicalValveCollisionPreflight(t *testing.T) {
	for _, kind := range []string{"Tool", "Function"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			writeManifest(t, dir+"/a.json", `{"apiVersion":"oictl.openwebui/v1","kind":"Channel","metadata":{"name":"earlier"},"spec":{"name":"earlier"}}`)
			writeManifest(t, dir+"/o.json", `{"apiVersion":"oictl.openwebui/v1","kind":"`+kind+`","metadata":{"name":"OpsFilter"},"spec":{"content":"source"}}`)
			writeManifest(t, dir+"/v1.json", `{"apiVersion":"oictl.openwebui/v1","kind":"`+kind+`Valve","metadata":{"name":"OpsFilter"},"spec":{"enabled":true}}`)
			writeManifest(t, dir+"/v2.json", `{"apiVersion":"oictl.openwebui/v1","kind":"`+kind+`Valve","metadata":{"name":"opsfilter"},"spec":{"enabled":false}}`)
			mutations := 0
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					mutations++
					http.Error(w, "unexpected mutation", 400)
					return
				}
				_ = json.NewEncoder(w).Encode([]any{})
			})
			if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code == 0 || mutations != 0 || !strings.Contains(stringValue(app.err), "canonical") {
				t.Fatalf("collision escaped preflight: code=%d mutations=%d error=%v", code, mutations, app.err)
			}
		})
	}
}

func TestReview2ManifestCanonicalOwnerCollision(t *testing.T) {
	for _, kind := range []string{"Tool", "Function"} {
		t.Run(kind, func(t *testing.T) {
			mutations := 0
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					mutations++
				}
				_ = json.NewEncoder(w).Encode([]any{})
			})
			docs := []manifestDocument{{Kind: kind, Metadata: manifestMetadata{Name: "First"}, Spec: map[string]any{"id": "OpsFilter", "content": "source"}}, {Kind: kind, Metadata: manifestMetadata{Name: "Second"}, Spec: map[string]any{"id": "opsfilter", "content": "source"}}}
			_, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
			if err == nil || mutations != 0 {
				t.Fatalf("canonical collision accepted: err=%v mutations=%d", err, mutations)
			}
		})
	}
}
