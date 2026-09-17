package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func reviewManifestApp(t *testing.T, handler http.HandlerFunc) *App {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	app, _, _ := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "test"})
	return app
}

func TestReviewManifestChannelCanonicalCreateCollision(t *testing.T) {
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode([]any{}) })
	docs := []manifestDocument{{Kind: "Channel", Metadata: manifestMetadata{Name: "ops-room"}, Spec: map[string]any{"name": "Ops Room"}}, {Kind: "Channel", Metadata: manifestMetadata{Name: "other-key"}, Spec: map[string]any{"name": "ops room"}}}
	if _, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil); err == nil {
		t.Fatal("two creates for the same canonical channel accepted")
	}
}

func TestReviewManifestExactTargetPrunesOnlyUnresolvedIDs(t *testing.T) {
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{map[string]any{"id": "a", "name": "b"}, map[string]any{"id": "b", "name": "other"}})
	})
	docs := []manifestDocument{{Kind: "Knowledge", Metadata: manifestMetadata{Name: "b"}, Spec: map[string]any{}}}
	plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), map[string]bool{"Knowledge": true})
	if err != nil {
		t.Fatal(err)
	}
	deleted := []string{}
	for _, action := range plan.Actions {
		if action.Action == "delete" {
			deleted = append(deleted, action.ID)
		}
	}
	if !jsonEqual(deleted, []string{"a"}) {
		t.Fatalf("prune confused IDs and aliases: %+v", plan.Actions)
	}
}

func TestReviewManifestDuplicateTargetsAcrossDependencyPhases(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir+"/a.json", `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"uuid"},"spec":{"content":"changed"}}`)
	writeManifest(t, dir+"/b.json", `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"Display"},"spec":{"content":"changed"},"access_grants":[{"principal_type":"group","principal_name":"new-group","permission":"read"}]}`)
	writeManifest(t, dir+"/g.json", `{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"new-group"},"spec":{"name":"new-group"}}`)
	remote := map[string]any{"id": "uuid", "command": "slash", "name": "Display", "content": "old"}
	groups := []any{}
	mutations := 0
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mutations++
			var form map[string]any
			_ = json.NewDecoder(r.Body).Decode(&form)
			if r.URL.Path == "/api/v1/groups/create" {
				form["id"] = "group-uuid"
				groups = append(groups, form)
				_ = json.NewEncoder(w).Encode(form)
				return
			}
			for k, v := range form {
				remote[k] = v
			}
		}
		switch r.URL.Path {
		case "/api/v1/prompts/":
			_ = json.NewEncoder(w).Encode([]any{remote})
		case "/api/v1/groups/":
			_ = json.NewEncoder(w).Encode(groups)
		default:
			if strings.HasPrefix(r.URL.Path, "/api/v1/groups/") && len(groups) > 0 {
				_ = json.NewEncoder(w).Encode(groups[0])
			} else {
				_ = json.NewEncoder(w).Encode(remote)
			}
		}
	})
	code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir})
	if code == 0 || mutations != 0 {
		t.Fatalf("duplicate target crossed phase boundary: code=%d mutations=%d", code, mutations)
	}
}

func TestReviewManifestSourceMetadataDefaultsConverge(t *testing.T) {
	for _, kind := range []string{"Tool", "Function"} {
		t.Run(kind, func(t *testing.T) {
			meta := map[string]any{"description": nil, "manifest": map[string]any{"title": "generated from source"}}
			if kind == "Tool" {
				meta["has_user_valves"] = false
			}
			remote := map[string]any{"id": "sample", "name": "sample", "content": "source", "meta": meta}
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/list") {
					summary := copySpec(remote)
					summary["content"] = nil
					_ = json.NewEncoder(w).Encode([]any{summary})
				} else {
					_ = json.NewEncoder(w).Encode(remote)
				}
			})
			docs := []manifestDocument{{Kind: kind, Metadata: manifestMetadata{Name: "sample"}, Spec: map[string]any{"name": "sample", "content": "source", "meta": map[string]any{}}}}
			plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Actions) != 1 || plan.Actions[0].Action != "unchanged" {
				t.Fatalf("schema defaults cause perpetual update: %+v", plan.Actions)
			}
		})
	}
}

func TestReviewManifestModelDeleteConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name, reply string
		remains     bool
		wantError   bool
	}{{"false", "false", true, true}, {"true but remains", "true", true, true}, {"confirmed", "true", false, false}, {"invalid", "{}", true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					_, _ = w.Write([]byte(tc.reply))
					return
				}
				items := []any{}
				if tc.remains {
					items = append(items, map[string]any{"id": "model"})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "total": len(items)})
			})
			err := (modelHandler{}).Delete(context.Background(), app, globalOptions{}, resourceState{ID: "model"})
			if (err != nil) != tc.wantError {
				t.Fatalf("err=%v wantError=%v", err, tc.wantError)
			}
		})
	}
}

func TestReviewManifestEssentialFormsLifecycle(t *testing.T) {
	for _, kind := range []string{"Tool", "Function", "Prompt"} {
		t.Run(kind, func(t *testing.T) {
			var remote map[string]any
			mutations := 0
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					var form map[string]any
					_ = json.NewDecoder(r.Body).Decode(&form)
					required := []string{"name", "content"}
					if kind == "Prompt" {
						required = append(required, "command")
					} else {
						required = append(required, "id", "meta")
					}
					for _, key := range required {
						if _, ok := form[key]; !ok {
							http.Error(w, "missing required "+key, 422)
							return
						}
					}
					mutations++
					remote = form
					if kind == "Prompt" {
						remote["id"] = "prompt-uuid"
					} else {
						meta := map[string]any{"description": nil, "manifest": map[string]any{}}
						if kind == "Tool" {
							meta["has_user_valves"] = false
						}
						for k, v := range form["meta"].(map[string]any) {
							meta[k] = v
						}
						remote["meta"] = meta
					}
					if kind == "Function" {
						remote["is_active"] = false
						remote["is_global"] = false
					}
				}
				if strings.HasSuffix(r.URL.Path, "/list") || r.URL.Path == "/api/v1/prompts/" {
					items := []any{}
					if remote != nil {
						summary := copySpec(remote)
						if kind == "Tool" {
							summary["content"] = nil
						}
						items = append(items, summary)
					}
					_ = json.NewEncoder(w).Encode(items)
				} else {
					_ = json.NewEncoder(w).Encode(remote)
				}
			})
			spec := map[string]any{"name": "sample", "content": "source"}
			if kind == "Prompt" {
				delete(spec, "name")
				spec["command"] = "sample"
			}
			if kind == "Function" {
				spec["id"] = "sample"
			}
			docs := []manifestDocument{{Kind: kind, APIVersion: manifestAPIVersion, Metadata: manifestMetadata{Name: "sample"}, Spec: spec}}
			if err := validateManifestDocument(&docs[0], manifestHandlers()); err != nil {
				t.Fatal(err)
			}
			apply := func() {
				t.Helper()
				plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
				if err != nil {
					t.Fatal(err)
				}
				if err := executeManifestPlan(context.Background(), app, globalOptions{}, plan, "apply"); err != nil {
					t.Fatal(err)
				}
			}
			apply()
			apply()
			if mutations != 1 {
				t.Fatalf("second create/apply mutated: %d", mutations)
			}
			docs[0].Spec = map[string]any{"content": "changed"}
			apply()
			apply()
			if mutations != 2 {
				t.Fatalf("partial source update did not converge: %d", mutations)
			}
			if remote["name"] != "sample" {
				t.Fatalf("required name not preserved: %+v", remote)
			}
			docs[0].Spec = map[string]any{"name": "Renamed"}
			apply()
			apply()
			if mutations != 3 || remote["name"] != "Renamed" || remote["content"] != "changed" {
				t.Fatalf("omitted source was not preserved/convergent: %+v mutations=%d", remote, mutations)
			}
		})
	}
}

func TestReviewManifestMissingCreateContentPreflight(t *testing.T) {
	for _, kind := range []string{"Tool", "Function", "Prompt"} {
		for _, dependency := range []string{"none", "group", "valve"} {
			if dependency == "valve" && kind == "Prompt" {
				continue
			}
			for _, dryRun := range []bool{true, false} {
				t.Run(kind+"/"+dependency+"/dry-run="+stringValue(dryRun), func(t *testing.T) {
					dir := t.TempDir()
					writeManifest(t, dir+"/a.json", `{"apiVersion":"oictl.openwebui/v1","kind":"Channel","metadata":{"name":"earlier"},"spec":{"name":"earlier"}}`)
					doc := manifestDocument{APIVersion: manifestAPIVersion, Kind: kind, Metadata: manifestMetadata{Name: "sample"}, Spec: map[string]any{"name": "sample"}}
					if dependency == "group" {
						writeManifest(t, dir+"/g.json", `{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"new-group"},"spec":{"name":"new-group"}}`)
						grant := []accessGrant{{PrincipalType: "group", PrincipalName: "new-group", Permission: "read"}}
						if kind == "Function" {
							// Functions do not support grants; defer a companion Channel.
							writeManifest(t, dir+"/b.json", `{"apiVersion":"oictl.openwebui/v1","kind":"Channel","metadata":{"name":"later"},"spec":{"name":"later"},"access_grants":[{"principal_type":"group","principal_name":"new-group","permission":"read"}]}`)
						} else {
							doc.AccessGrants = grant
						}
					}
					if dependency == "valve" {
						writeManifest(t, dir+"/v.json", `{"apiVersion":"oictl.openwebui/v1","kind":"`+kind+`Valve","metadata":{"name":"sample"},"spec":{}}`)
					}
					body, err := json.Marshal(doc)
					if err != nil {
						t.Fatal(err)
					}
					writeManifest(t, dir+"/z.json", string(body))
					mutations := 0
					stored := map[string]map[string]any{}
					app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodGet {
							mutations++
							var form map[string]any
							_ = json.NewDecoder(r.Body).Decode(&form)
							switch r.URL.Path {
							case "/api/v1/channels/create":
								form["id"] = "channel-uuid"
								stored["channels"] = form
							case "/api/v1/groups/create":
								form["id"] = "group-uuid"
								stored["groups"] = form
							default:
								t.Errorf("unexpected mutation: %s %s", r.Method, r.URL.Path)
							}
							_ = json.NewEncoder(w).Encode(form)
							return
						}
						switch r.URL.Path {
						case "/api/v1/channels/list", "/api/v1/groups/", "/api/v1/tools/list", "/api/v1/functions/list", "/api/v1/prompts/":
							items := []any{}
							if item := stored[strings.Split(r.URL.Path, "/")[3]]; item != nil {
								items = append(items, item)
							}
							_ = json.NewEncoder(w).Encode(items)
						case "/api/v1/channels/channel-uuid":
							_ = json.NewEncoder(w).Encode(stored["channels"])
						case "/api/v1/groups/id/group-uuid":
							_ = json.NewEncoder(w).Encode(stored["groups"])
						case "/api/v1/tools/id/sample/valves", "/api/v1/functions/id/sample/valves":
							_, _ = w.Write([]byte("null"))
						default:
							t.Errorf("unexpected read: %s", r.URL.Path)
							http.NotFound(w, r)
						}
					})
					args := []string{"manifests", "apply", "--directory", dir}
					if dryRun {
						args = append(args, "--dry-run")
					}
					code := app.Run(context.Background(), args)
					if code == 0 || mutations != 0 {
						t.Fatalf("missing create content escaped whole-plan preflight: code=%d mutations=%d", code, mutations)
					}
					// Dry-run does not resolve same-set groups that do not yet exist.
					if !(dryRun && dependency == "group") && !strings.Contains(stringValue(app.err), "spec.content") {
						t.Fatalf("expected content validation error, got %v", app.err)
					}
				})
			}
		}
	}
}

func TestReviewManifestEssentialFieldsValidation(t *testing.T) {
	for _, tc := range []struct {
		kind, field string
		value       any
	}{{"Tool", "id", 9}, {"Tool", "meta", nil}, {"Function", "meta", []any{}}, {"Prompt", "name", false}, {"Function", "is_global", "true"}, {"Skill", "is_active", nil}} {
		t.Run(tc.kind+"/"+tc.field, func(t *testing.T) {
			doc := manifestDocument{APIVersion: manifestAPIVersion, Kind: tc.kind, Metadata: manifestMetadata{Name: "sample"}, Spec: map[string]any{tc.field: tc.value}}
			if err := validateManifestDocument(&doc, manifestHandlers()); err == nil {
				t.Fatal("invalid supplied field accepted")
			}
		})
	}
}

func TestReviewManifestFunctionGlobalLifecycle(t *testing.T) {
	for _, stuck := range []bool{false, true} {
		t.Run(stringValue(stuck), func(t *testing.T) {
			remote := map[string]any{"id": "fn", "name": "Function", "content": "source", "meta": map[string]any{}, "is_active": false, "is_global": false}
			toggles, mutations := 0, 0
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					mutations++
					if strings.HasSuffix(r.URL.Path, "/toggle/global") {
						toggles++
						if !stuck {
							remote["is_global"] = !remote["is_global"].(bool)
						}
					} else {
						var form map[string]any
						_ = json.NewDecoder(r.Body).Decode(&form)
						for _, key := range []string{"id", "name", "content", "meta"} {
							if value, ok := form[key]; ok {
								remote[key] = value
							}
						}
					}
				}
				if strings.HasSuffix(r.URL.Path, "/list") {
					_ = json.NewEncoder(w).Encode([]any{remote})
				} else {
					_ = json.NewEncoder(w).Encode(remote)
				}
			})
			docs := []manifestDocument{{Kind: "Function", Metadata: manifestMetadata{Name: "fn"}, Spec: map[string]any{"name": "Function", "content": "source", "is_global": true}}}
			plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
			if err != nil {
				t.Fatal(err)
			}
			err = executeManifestPlan(context.Background(), app, globalOptions{}, plan, "apply")
			if stuck {
				if err == nil {
					t.Fatal("nonconvergent global toggle reported success")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if toggles != 1 || remote["is_global"] != true || remote["is_active"] != false {
				t.Fatalf("state=%+v toggles=%d", remote, toggles)
			}
			before := mutations
			plan, err = buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err = executeManifestPlan(context.Background(), app, globalOptions{}, plan, "apply"); err != nil {
				t.Fatal(err)
			}
			if mutations != before {
				t.Fatalf("second apply mutated %d -> %d", before, mutations)
			}
		})
	}
}

func TestReviewManifestToolDeferredContent(t *testing.T) {
	remote := map[string]any{"id": "tool", "name": "Tool", "content": "old", "meta": map[string]any{}}
	details, mutations := 0, 0
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mutations++
			_ = json.NewDecoder(r.Body).Decode(&remote)
		}
		if r.URL.Path == "/api/v1/tools/list" {
			summary := copySpec(remote)
			summary["content"] = nil
			_ = json.NewEncoder(w).Encode([]any{summary})
		} else {
			details++
			_ = json.NewEncoder(w).Encode(remote)
		}
	})
	docs := []manifestDocument{{Kind: "Tool", Metadata: manifestMetadata{Name: "tool"}, Spec: map[string]any{"id": "tool", "name": "Tool", "content": "new", "meta": map[string]any{}}}}
	for apply := 0; apply < 2; apply++ {
		plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := executeManifestPlan(context.Background(), app, globalOptions{}, plan, "apply"); err != nil {
			t.Fatal(err)
		}
	}
	if mutations != 1 || details < 2 {
		t.Fatalf("mutations=%d details=%d", mutations, details)
	}
}

func TestReviewManifestChannelLifecycle(t *testing.T) {
	var remote map[string]any
	mutations := 0
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mutations++
			var form map[string]any
			_ = json.NewDecoder(r.Body).Decode(&form)
			if remote == nil {
				remote = map[string]any{"id": "channel-uuid"}
			}
			// ChannelForm is replacement-style, including nullable unmanaged fields.
			for _, key := range []string{"name", "description", "is_private", "data", "meta"} {
				remote[key] = form[key]
			}
			if r.URL.Path == "/api/v1/channels/create" {
				remote["name"] = strings.ToLower(stringValue(remote["name"]))
			}
			if value, ok := form["access_grants"]; ok {
				remote["access_grants"] = value
			}
		}
		switch r.URL.Path {
		case "/api/v1/channels/":
			_ = json.NewEncoder(w).Encode([]any{}) // Not a member.
		case "/api/v1/channels/list":
			items := []any{}
			if remote != nil {
				items = append(items, remote)
			}
			_ = json.NewEncoder(w).Encode(items)
		default:
			_ = json.NewEncoder(w).Encode(remote)
		}
	})
	docs := []manifestDocument{{Kind: "Channel", Metadata: manifestMetadata{Name: "ops-room"}, Spec: map[string]any{"name": "Ops Room", "description": "original", "is_private": true, "data": map[string]any{"keep": true}, "meta": map[string]any{"image": "keep"}}}}
	apply := func() {
		t.Helper()
		plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := executeManifestPlan(context.Background(), app, globalOptions{}, plan, "apply"); err != nil {
			t.Fatal(err)
		}
	}
	apply()
	apply()
	if mutations != 1 {
		t.Fatalf("second apply did not converge: mutations=%d", mutations)
	}
	if remote["name"] != "ops room" {
		t.Fatalf("creation did not lowercase the channel name: %+v", remote)
	}
	docs[0].Spec = map[string]any{"name": "Ops Room", "description": "changed"}
	apply()
	apply()
	if remote["name"] != "Ops Room" {
		t.Fatalf("update did not preserve the channel name verbatim: %+v", remote)
	}
	if mutations != 2 || remote["is_private"] != true || !jsonEqual(remote["data"], map[string]any{"keep": true}) || !jsonEqual(remote["meta"], map[string]any{"image": "keep"}) {
		t.Fatalf("partial update lost state: %+v mutations=%d", remote, mutations)
	}
	docs[0].HasAccessGrants = true
	docs[0].AccessGrants = []accessGrant{{PrincipalType: "group", PrincipalID: "ops", Permission: "read"}}
	apply()
	apply()
	if mutations != 3 || remote["is_private"] != true || remote["description"] != "changed" {
		t.Fatalf("grant update lost state: %+v mutations=%d", remote, mutations)
	}
}

func TestReviewManifestOmittedActivePreserved(t *testing.T) {
	for _, kind := range []string{"Model", "Skill"} {
		t.Run(kind, func(t *testing.T) {
			remote := map[string]any{"id": "sample", "name": "Old", "content": "source", "meta": map[string]any{}, "params": map[string]any{}, "is_active": false}
			mutations := 0
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					mutations++
					var form map[string]any
					_ = json.NewDecoder(r.Body).Decode(&form)
					remote["is_active"] = true
					for key, value := range form {
						remote[key] = value
					}
				}
				if strings.HasSuffix(r.URL.Path, "/list") {
					_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{remote}, "total": 1})
				} else {
					_ = json.NewEncoder(w).Encode(remote)
				}
			})
			docs := []manifestDocument{{Kind: kind, Metadata: manifestMetadata{Name: "sample"}, Spec: map[string]any{"id": "sample", "name": "New"}}}
			for apply := 0; apply < 2; apply++ {
				plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
				if err != nil {
					t.Fatal(err)
				}
				if err := executeManifestPlan(context.Background(), app, globalOptions{}, plan, "apply"); err != nil {
					t.Fatal(err)
				}
			}
			if remote["is_active"] != false || mutations != 1 {
				t.Fatalf("stored=%+v mutations=%d", remote, mutations)
			}
			docs[0].Spec["is_active"] = true
			plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := executeManifestPlan(context.Background(), app, globalOptions{}, plan, "apply"); err != nil {
				t.Fatal(err)
			}
			if remote["is_active"] != true {
				t.Fatalf("explicit activation not applied: %+v", remote)
			}
		})
	}
}

func TestReviewManifestTerminalPreservesNestedACL(t *testing.T) {
	for _, config := range []any{map[string]any{"keep": false}, nil} {
		t.Run(stringValue(config), func(t *testing.T) {
			remote := map[string]any{"id": "term", "name": "term", "url": "http://terminal", "config": map[string]any{"keep": true, "access_grants": []accessGrant{{PrincipalType: "group", PrincipalID: "ops", Permission: "read"}}}}
			stored := map[string]any{"TERMINAL_SERVER_CONNECTIONS": []any{remote}, "ENABLE_TERMINAL_SERVER": true}
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					_ = json.NewDecoder(r.Body).Decode(&stored)
				}
				_ = json.NewEncoder(w).Encode(stored)
			})
			handler := terminalServerConnectionHandler{}
			current := handler.stateFromConnection(remote)
			desired := manifestDocument{Kind: "TerminalServerConnection", Metadata: manifestMetadata{Name: "term"}, Spec: map[string]any{"config": config}}
			updated, err := handler.Update(context.Background(), app, globalOptions{}, current, desired)
			if err != nil {
				t.Fatal(err)
			}
			if !grantsEqual(updated.AccessGrants, current.AccessGrants) {
				t.Fatalf("ACL lost: %+v", updated)
			}
			plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, []manifestDocument{desired}, manifestHandlers(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Actions) != 1 || plan.Actions[0].Action != "unchanged" {
				t.Fatalf("second terminal apply did not converge: %+v", plan.Actions)
			}
		})
	}
}

func TestReviewManifestCompleteInventories(t *testing.T) {
	for _, kind := range []string{"Knowledge", "Skill", "Model"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				id := "first"
				if r.URL.Query().Get("page") == "2" {
					id = "wanted"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{map[string]any{"id": id, "name": id}}, "total": 2})
			})
			state, err := manifestHandlers()[kind].Lookup(context.Background(), app, globalOptions{}, manifestDocument{Metadata: manifestMetadata{Name: "wanted"}})
			if err != nil || state == nil || state.ID != "wanted" || calls != 2 {
				t.Fatalf("state=%+v calls=%d err=%v", state, calls, err)
			}
		})
	}
	t.Run("incomplete inventory", func(t *testing.T) {
		calls := 0
		app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{map[string]any{"id": "same"}}, "total": 2})
		})
		_, err := manifestHandlers()["Knowledge"].List(context.Background(), app, globalOptions{})
		if err == nil || calls != 2 {
			t.Fatalf("incomplete list accepted calls=%d err=%v", calls, err)
		}
	})
	t.Run("user ambiguity and UUID precedence", func(t *testing.T) {
		uuid := "5f28fbfe-e317-43ad-b032-c0cdd5f6c367"
		calls := 0
		app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.URL.Query().Get("query") != "" {
				_ = json.NewEncoder(w).Encode(map[string]any{"users": []any{}, "total": 0})
				return
			}
			user := map[string]any{"id": "first", "name": "same", "email": uuid}
			if r.URL.Query().Get("page") == "2" {
				user = map[string]any{"id": uuid, "name": "same"}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"users": []any{user}, "total": 2})
		})
		resolver := principalResolver{app: app}
		id, err := resolver.resolveUserID(context.Background(), "ref", uuid)
		if err != nil || id != uuid {
			t.Fatalf("UUID resolution id=%q err=%v", id, err)
		}
		if _, err := resolver.resolveUserID(context.Background(), "name", "same"); err == nil {
			t.Fatal("cross-page ambiguity accepted")
		}
		if calls != 2 {
			t.Fatalf("inventory not cached: calls=%d", calls)
		}
	})
}

func TestReviewManifestIdentityResolution(t *testing.T) {
	for _, tc := range []struct {
		name, ref string
		objects   []map[string]any
		want      string
		bad       bool
	}{
		{"exact ID wins", "b", []map[string]any{{"id": "a", "name": "b"}, {"id": "b", "name": "other"}}, "b", false},
		{"ambiguous aliases", "same", []map[string]any{{"id": "a", "name": "same"}, {"id": "b", "name": "same"}}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(tc.objects) })
			got, err := manifestHandlers()["Knowledge"].Lookup(context.Background(), app, globalOptions{}, manifestDocument{Kind: "Knowledge", Metadata: manifestMetadata{Name: tc.ref}})
			if tc.bad {
				if err == nil {
					t.Fatalf("ambiguous target accepted: %+v", got)
				}
				return
			}
			if err != nil || got == nil || got.ID != tc.want {
				t.Fatalf("got=%+v err=%v", got, err)
			}
		})
	}
	t.Run("duplicate resolved manifests", func(t *testing.T) {
		app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode([]any{map[string]any{"id": "uuid", "name": "Alias"}})
		})
		docs := []manifestDocument{{Kind: "Knowledge", Metadata: manifestMetadata{Name: "uuid"}}, {Kind: "Knowledge", Metadata: manifestMetadata{Name: "Alias"}}}
		if _, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil); err == nil {
			t.Fatal("duplicate resolved target accepted")
		}
	})
}

func TestReviewManifestAliasPruningAndConflictGuard(t *testing.T) {
	remote := map[string]any{"id": "uuid", "command": "slash", "name": "Display", "content": "old"}
	mutations := 0
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mutations++
			if r.Method == http.MethodDelete {
				t.Error("declared prompt deleted")
			}
			_ = json.NewDecoder(r.Body).Decode(&remote)
			remote["id"] = "uuid"
		}
		if r.URL.Path == "/api/v1/prompts/" {
			_ = json.NewEncoder(w).Encode([]any{remote})
		} else {
			_ = json.NewEncoder(w).Encode(remote)
		}
	})
	docs := []manifestDocument{{Kind: "Prompt", Metadata: manifestMetadata{Name: "Display"}, Spec: map[string]any{"command": "slash", "content": "new"}}}
	plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), map[string]bool{"Prompt": true})
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range plan.Actions {
		if action.Action == "delete" {
			t.Fatalf("declared target pruned: %+v", plan.Actions)
		}
	}
	if err := executeManifestPlan(context.Background(), app, globalOptions{}, plan, "sync"); err != nil {
		t.Fatal(err)
	}
	if mutations != 1 {
		t.Fatalf("mutations=%d", mutations)
	}
	current := resourceState{Kind: "Prompt", ID: "uuid"}
	conflict := manifestPlan{Actions: []planAction{{Action: "unchanged", Kind: "Prompt", ID: "uuid", Current: &current}, {Action: "delete", Kind: "Prompt", ID: "uuid", Current: &current, Handler: manifestHandlers()["Prompt"]}}}
	if err := executeManifestPlan(context.Background(), app, globalOptions{}, conflict, "sync"); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("conflicting plan accepted: %v", err)
	}
}
