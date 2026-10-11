package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// f8ae8a6 models/skills.py:73-97; routers/skills.py:80-134,295-324:
// inventory excludes content, detail contains it; root content is not a package.
func TestReview2ManifestSkillSummaryHydration(t *testing.T) {
	for _, selector := range []string{"skill-id", "Display"} {
		t.Run(selector, func(t *testing.T) {
			remote := map[string]any{"id": "skill-id", "name": "Display", "description": "Keep", "content": "source", "meta": map[string]any{}, "is_active": false}
			mutations, details := 0, 0
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					mutations++
					var form map[string]any
					_ = json.NewDecoder(r.Body).Decode(&form)
					for k, v := range form {
						remote[k] = v
					}
				}
				if strings.HasSuffix(r.URL.Path, "/list") {
					summary := copySpec(remote)
					delete(summary, "content")
					_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{summary}, "total": 1})
				} else {
					details++
					_ = json.NewEncoder(w).Encode(remote)
				}
			})
			doc := manifestDocument{Kind: "Skill", Metadata: manifestMetadata{Name: selector}, Spec: map[string]any{"content": "source"}}
			review2ManifestApply(t, app, []manifestDocument{doc}, nil)
			review2ManifestApply(t, app, []manifestDocument{doc}, nil)
			if mutations != 0 || details != 2 {
				t.Fatalf("false source drift: mutations=%d details=%d", mutations, details)
			}
			doc.Spec = map[string]any{"name": "Renamed"}
			review2ManifestApply(t, app, []manifestDocument{doc}, nil)
			if remote["content"] != "source" || remote["is_active"] != false || remote["description"] != "Keep" {
				t.Fatalf("partial Skill lost fields: %v", remote)
			}
		})
	}
}
