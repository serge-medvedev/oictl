package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// f8ae8a6 routers/models.py:227-264,305-322; models/models.py:242-253,350-357:
// /list excludes null-base records, /all is the administrative complete inventory.
func TestReview2ManifestBaseModelInventory(t *testing.T) {
	var remote map[string]any
	creates, deletes := 0, 0
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/models/list":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "total": 0})
			return
		case "/api/v1/models/all":
			items := []any{}
			if remote != nil {
				items = append(items, remote, remote)
			}
			_ = json.NewEncoder(w).Encode(items)
			return
		case "/api/v1/models/create":
			creates++
			if remote != nil {
				http.Error(w, "duplicate ID", 401)
				return
			}
			_ = json.NewDecoder(r.Body).Decode(&remote)
		case "/api/v1/models/model/delete":
			deletes++
			remote = nil
			_ = json.NewEncoder(w).Encode(true)
			return
		}
		_ = json.NewEncoder(w).Encode(remote)
	})
	doc := manifestDocument{Kind: "Model", Metadata: manifestMetadata{Name: "base-record"}, Spec: map[string]any{"id": "base-record", "base_model_id": nil, "name": "Base", "meta": map[string]any{}, "params": map[string]any{}}}
	review2ManifestApply(t, app, []manifestDocument{doc}, nil)
	review2ManifestApply(t, app, []manifestDocument{doc}, map[string]bool{"Model": true})
	if creates != 1 || deletes != 0 {
		t.Fatalf("duplicate create/prune: creates=%d deletes=%d", creates, deletes)
	}
	plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, nil, manifestHandlers(), map[string]bool{"Model": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Action != "delete" {
		t.Fatalf("base omitted from prune: %+v", plan.Actions)
	}
	if err := executeManifestPlan(context.Background(), app, globalOptions{}, plan, "sync"); err != nil {
		t.Fatal(err)
	}
	if remote != nil || deletes != 1 {
		t.Fatalf("base deletion not verified: %v deletes=%d", remote, deletes)
	}
}

func TestReview2ManifestBaseInventoryUnauthorizedFailsClosed(t *testing.T) {
	mutations := 0
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mutations++
		}
		if r.URL.Path == "/api/v1/models/all" {
			http.Error(w, "admin required", 403)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "total": 0})
	})
	_, err := buildManifestPlan(context.Background(), app, globalOptions{}, nil, manifestHandlers(), map[string]bool{"Model": true})
	if err == nil || mutations != 0 {
		t.Fatalf("incomplete unauthorized inventory accepted: err=%v mutations=%d", err, mutations)
	}
}
