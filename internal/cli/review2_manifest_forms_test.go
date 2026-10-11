package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

// f8ae8a6 models/groups.py:125-153,545-575 and models/knowledge.py:165-168,786-796:
// required strings are full-form replacements; omitted grants preserve storage.
func TestReview2ManifestRequiredPartialUpdates(t *testing.T) {
	for _, kind := range []string{"Group", "Knowledge"} {
		for _, field := range []string{"name", "description"} {
			t.Run(kind+"/"+field, func(t *testing.T) {
				resource := strings.ToLower(kind) + "s"
				if kind == "Knowledge" {
					resource = "knowledge"
				}
				remote := map[string]any{"id": "record-uuid", "name": "Existing name", "description": "Existing description", "permissions": map[string]any{"keep": true}, "access_grants": []accessGrant{{PrincipalType: "user", PrincipalID: "*", Permission: "read"}}}
				mutations := 0
				app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPost {
						mutations++
						var form map[string]any
						_ = json.NewDecoder(r.Body).Decode(&form)
						for _, required := range []string{"name", "description"} {
							if _, ok := form[required].(string); !ok {
								http.Error(w, "missing "+required, 422)
								return
							}
						}
						if _, supplied := form["access_grants"]; supplied {
							t.Error("unmanaged grants submitted")
						}
						remote["name"], remote["description"] = form["name"], form["description"]
					}
					if r.URL.Path == "/api/v1/"+resource+"/" {
						_ = json.NewEncoder(w).Encode([]any{remote})
					} else {
						_ = json.NewEncoder(w).Encode(remote)
					}
				})
				doc := manifestDocument{Kind: kind, Metadata: manifestMetadata{Name: "record-uuid"}, Spec: map[string]any{field: "Changed"}}
				review2ManifestApply(t, app, []manifestDocument{doc}, nil)
				omitted := "name"
				if field == "name" {
					omitted = "description"
				}
				if remote[omitted] != "Existing "+omitted {
					t.Fatalf("omitted %s overwritten: %v", omitted, remote)
				}
				review2ManifestApply(t, app, []manifestDocument{doc}, nil)
				if mutations != 1 {
					t.Fatalf("nonconvergent partial update: %d", mutations)
				}
			})
		}
	}
}

func TestReview2ManifestMissingDescriptionWholePlan(t *testing.T) {
	for _, kind := range []string{"Group", "Knowledge"} {
		for _, deferred := range []bool{false, true} {
			t.Run(kind+"/deferred="+stringValue(deferred), func(t *testing.T) {
				dir := t.TempDir()
				writeManifest(t, dir+"/a.json", `{"apiVersion":"oictl.openwebui/v1","kind":"Channel","metadata":{"name":"earlier"},"spec":{"name":"earlier"}}`)
				writeManifest(t, dir+"/z.json", `{"apiVersion":"oictl.openwebui/v1","kind":"`+kind+`","metadata":{"name":"invalid"},"spec":{"name":"Invalid"}}`)
				if deferred {
					writeManifest(t, dir+"/t.json", `{"apiVersion":"oictl.openwebui/v1","kind":"Tool","metadata":{"name":"owner"},"spec":{"content":"class Tools: pass"}}`)
					writeManifest(t, dir+"/v.json", `{"apiVersion":"oictl.openwebui/v1","kind":"ToolValve","metadata":{"name":"owner"},"spec":{}}`)
				}
				mutations := 0
				app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodGet {
						mutations++
						http.Error(w, "missing description", 422)
						return
					}
					_ = json.NewEncoder(w).Encode([]any{})
				})
				code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir})
				if code == 0 || mutations != 0 || !strings.Contains(stringValue(app.err), "spec.description") {
					t.Fatalf("code=%d mutations=%d error=%v", code, mutations, app.err)
				}
			})
		}
	}
}

func TestReview2ManifestPublishedGroupForm(t *testing.T) {
	text, err := os.ReadFile("../../docs/manifests.md")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, block := range regexp.MustCompile("(?s)```json\\n(.*?)\\n```").FindAllSubmatch(text, -1) {
		var doc manifestDocument
		if json.Unmarshal(block[1], &doc) != nil || doc.Kind != "Group" {
			continue
		}
		count++
		// Validate the literal document against required GroupForm declarations.
		for _, field := range []string{"name", "description"} {
			if _, ok := doc.Spec[field].(string); !ok {
				t.Fatalf("published Group missing %s", field)
			}
		}
		if _, err := (endpointHandler{kind: "Group"}).manifestPayload(doc, nil); err != nil {
			t.Fatal(err)
		}
	}
	if count != 1 {
		t.Fatalf("Group examples=%d", count)
	}
}
