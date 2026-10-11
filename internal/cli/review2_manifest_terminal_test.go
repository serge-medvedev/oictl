package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// f8ae8a6 routers/configs.py:306-357 replaces the complete connections config;
// names are aliases, while connection ID identifies rename readback.
func TestReview2ManifestTerminalRenameStableReadback(t *testing.T) {
	remote := map[string]any{"id": "term", "name": "Old", "url": "https://terminal.example.com", "config": map[string]any{"keep": true, "access_grants": []accessGrant{{PrincipalType: "group", PrincipalID: "ops", Permission: "read"}}}}
	stored := map[string]any{"ENABLE_TERMINAL_SERVER": true, "TERMINAL_SERVER_CONNECTIONS": []any{remote}}
	mutations := 0
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mutations++
			_ = json.NewDecoder(r.Body).Decode(&stored)
		}
		_ = json.NewEncoder(w).Encode(stored)
	})
	handler := terminalServerConnectionHandler{}
	doc := manifestDocument{Kind: "TerminalServerConnection", Metadata: manifestMetadata{Name: "Old"}, Spec: map[string]any{"name": "New"}}
	current, err := handler.Lookup(context.Background(), app, globalOptions{}, doc)
	if err != nil || current == nil {
		t.Fatalf("lookup=%v err=%v", current, err)
	}
	updated, err := handler.Update(context.Background(), app, globalOptions{}, *current, doc)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != "term" || updated.Spec["name"] != "New" || !grantsEqual(updated.AccessGrants, current.AccessGrants) {
		t.Fatalf("rename lost identity/ACL: %+v", updated)
	}
	doc.Metadata.Name = "term"
	review2ManifestApply(t, app, []manifestDocument{doc}, nil)
	if mutations != 1 {
		t.Fatalf("stable ID repeat mutated: %d", mutations)
	}
}

func TestReview2ManifestTerminalIDChangeRejectedBeforeMutation(t *testing.T) {
	mutations := 0
	remote := map[string]any{"id": "term", "name": "Old"}
	app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mutations++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ENABLE_TERMINAL_SERVER": true, "TERMINAL_SERVER_CONNECTIONS": []any{remote}})
	})
	docs := []manifestDocument{{Kind: "TerminalServerConnection", Metadata: manifestMetadata{Name: "term"}, Spec: map[string]any{"id": "different"}}}
	_, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
	if err == nil || mutations != 0 {
		t.Fatalf("mutable ID accepted: err=%v mutations=%d", err, mutations)
	}
}
