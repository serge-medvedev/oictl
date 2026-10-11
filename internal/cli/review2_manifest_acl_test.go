package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// Managed grants already matching the plan must still be checked after content
// actions; otherwise a server-side side effect could be reported as success.
func TestReview2ManifestACLVerifiedAfterContent(t *testing.T) {
	remote := map[string]any{"id": "sample", "command": "sample", "name": "Sample", "content": "old", "access_grants": []accessGrant{{PrincipalType: "group", PrincipalID: "desired", Permission: "read"}}}
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			remote["content"] = "new"
			remote["access_grants"] = []accessGrant{}
		}
		if r.URL.Path == "/api/v1/prompts/" {
			_ = json.NewEncoder(w).Encode([]any{remote})
		} else {
			_ = json.NewEncoder(w).Encode(remote)
		}
	})
	doc := manifestDocument{Kind: "Prompt", Metadata: manifestMetadata{Name: "sample"}, Spec: map[string]any{"content": "new"}, HasAccessGrants: true, AccessGrants: []accessGrant{{PrincipalType: "group", PrincipalID: "desired", Permission: "read"}}}
	plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, []manifestDocument{doc}, manifestHandlers(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := executeManifestPlan(context.Background(), app, globalOptions{}, plan, "apply"); err == nil {
		t.Fatal("final ACL mismatch reported success after content mutation")
	}
}

func review2ManifestApply(t *testing.T, app *App, docs []manifestDocument, prune map[string]bool) manifestPlan {
	t.Helper()
	plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), prune)
	if err != nil {
		t.Fatal(err)
	}
	if err := executeManifestPlan(context.Background(), app, globalOptions{}, plan, "sync"); err != nil {
		t.Fatal(err)
	}
	return plan
}

// Source-derived storage/filter fixture, not a live OWUI application:
// f8ae8a6 models/prompts.py:83-93,523-525; models/tools.py:94-100,377-381;
// routers/prompts.py:292-301; routers/tools.py:574-584;
// utils/access_control/__init__.py:237-300. Omitted/null ACL preserves storage;
// supplied grants are filtered and replace storage. Dedicated ACL writes replace.
func TestReview2ManifestACLFinalAuthority(t *testing.T) {
	for _, kind := range []string{"Prompt", "Tool"} {
		for _, managed := range []bool{true, false} {
			t.Run(kind+"/managed="+stringValue(managed), func(t *testing.T) {
				resource := strings.ToLower(kind) + "s"
				listPath := "/api/v1/" + resource + "/"
				if kind == "Tool" {
					listPath += "list"
				}
				old := []accessGrant{{PrincipalType: "group", PrincipalID: "old", Permission: "read"}}
				if !managed {
					old = []accessGrant{{PrincipalType: "user", PrincipalID: "*", Permission: "read"}}
				}
				remote := map[string]any{"id": "sample", "command": "sample", "name": "Sample", "content": "old", "meta": map[string]any{}, "access_grants": old}
				var actions []string
				contentACL := false
				app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPost {
						var form map[string]any
						_ = json.NewDecoder(r.Body).Decode(&form)
						if strings.HasSuffix(r.URL.Path, "/access/update") {
							actions = append(actions, "grants")
							remote["access_grants"] = form["access_grants"]
						} else {
							actions = append(actions, "content")
							if grants, supplied := form["access_grants"]; supplied {
								contentACL = true
								filtered := []accessGrant{}
								for _, grant := range accessGrantsFromValue(grants) {
									if grant.PrincipalID != "*" {
										filtered = append(filtered, grant)
									}
								}
								remote["access_grants"] = filtered
							}
							for _, key := range []string{"name", "content", "command", "meta"} {
								if v, ok := form[key]; ok {
									remote[key] = v
								}
							}
						}
					}
					if r.URL.Path == listPath {
						_ = json.NewEncoder(w).Encode([]any{remote})
					} else {
						_ = json.NewEncoder(w).Encode(remote)
					}
				})
				doc := manifestDocument{Kind: kind, Metadata: manifestMetadata{Name: "sample"}, Spec: map[string]any{"content": "new"}, HasAccessGrants: managed}
				if managed {
					doc.AccessGrants = []accessGrant{{PrincipalType: "group", PrincipalID: "new", Permission: "read"}}
				}
				review2ManifestApply(t, app, []manifestDocument{doc}, nil)
				want := old
				if managed {
					want = doc.AccessGrants
				}
				if !grantsEqual(accessGrantsFromValue(remote["access_grants"]), want) {
					t.Fatalf("final ACL=%v want=%v actions=%v", remote["access_grants"], want, actions)
				}
				if contentACL {
					t.Fatal("ordinary content payload resubmitted ACL")
				}
				if managed && !jsonEqual(actions, []string{"content", "grants"}) {
					t.Fatalf("unsafe order: %v", actions)
				}
				if remote["content"] != "new" {
					t.Fatalf("source not persisted: %v", remote)
				}
				before := len(actions)
				review2ManifestApply(t, app, []manifestDocument{doc}, nil)
				if len(actions) != before {
					t.Fatalf("repeat apply mutated: %v", actions)
				}
			})
		}
	}
}
