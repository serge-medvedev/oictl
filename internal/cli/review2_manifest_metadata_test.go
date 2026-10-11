package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// f8ae8a6 models/models.py:143-155; models/tools.py:39-44;
// routers/functions.py:230-231,402-404. Only schema defaults or unmanaged
// source-derived keys are ignored; extras and explicitly declared values drift.
func TestReview2ManifestMetadataDefaultsConverge(t *testing.T) {
	for _, kind := range []string{"Model", "Tool", "Function"} {
		t.Run(kind, func(t *testing.T) {
			meta := map[string]any{"description": nil}
			switch kind {
			case "Model":
				for _, k := range []string{"profile_image_url", "background_image_url", "i18n", "capabilities", "knowledge", "voice", "voice_avatar"} {
					meta[k] = nil
				}
			case "Tool":
				meta["i18n"] = nil
				meta["has_user_valves"] = false
				meta["manifest"] = map[string]any{}
			case "Function":
				meta["toggle"] = false
				meta["manifest"] = map[string]any{}
			}
			remote := map[string]any{"id": "sample", "name": "Sample", "content": "source", "meta": meta, "params": map[string]any{}, "base_model_id": "upstream", "is_active": true, "is_global": false}
			mutations := 0
			app := reviewManifestApp(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					mutations++
				}
				if strings.HasSuffix(r.URL.Path, "/all") || strings.HasSuffix(r.URL.Path, "/list") {
					_ = json.NewEncoder(w).Encode([]any{remote})
				} else {
					_ = json.NewEncoder(w).Encode(remote)
				}
			})
			spec := map[string]any{"name": "Sample", "content": "source", "meta": map[string]any{}}
			if kind == "Model" {
				spec = map[string]any{"id": "sample", "name": "Sample", "base_model_id": "upstream", "meta": map[string]any{}, "params": map[string]any{}}
			}
			doc := manifestDocument{Kind: kind, Metadata: manifestMetadata{Name: "sample"}, Spec: spec}
			for i := 0; i < 2; i++ {
				review2ManifestApply(t, app, []manifestDocument{doc}, nil)
			}
			if mutations != 0 {
				t.Fatalf("defaults cause updates=%d meta=%v", mutations, meta)
			}
		})
	}
}

func TestReview2ManifestMetadataManagedValuesRemainVisible(t *testing.T) {
	for _, kind := range []string{"Model", "Tool", "Function"} {
		t.Run(kind, func(t *testing.T) {
			key := "i18n"
			value := any(map[string]any{"en": "Declared"})
			if kind == "Function" {
				key = "toggle"
				value = true
			}
			desired := map[string]any{"meta": map[string]any{key: value}}
			current := map[string]any{"meta": map[string]any{key: nil}}
			var changes []fieldChange
			if kind == "Model" {
				changes = diffSpec(normalizeModelSpecForCompare(desired), normalizeModelSpecForCompare(current))
			} else {
				d, c := functionSpecsForCompare(desired, current)
				changes = diffSpec(d, c)
			}
			if len(changes) == 0 {
				t.Fatal("explicit metadata value stripped")
			}
			current = map[string]any{"meta": map[string]any{"extra": "retain"}}
			desired = map[string]any{"meta": map[string]any{}}
			if kind == "Model" {
				changes = diffSpec(normalizeModelSpecForCompare(desired), normalizeModelSpecForCompare(current))
			} else {
				d, c := functionSpecsForCompare(desired, current)
				changes = diffSpec(d, c)
			}
			if len(changes) == 0 {
				t.Fatal("real extra metadata stripped")
			}
			if kind == "Tool" {
				desired = map[string]any{"meta": map[string]any{}}
				current = map[string]any{"meta": map[string]any{"toggle": true}}
				d, c := functionSpecsForCompare(desired, current)
				if len(diffSpec(d, c)) == 0 {
					t.Fatal("Tool's real toggle metadata stripped as Function-derived state")
				}
			}
		})
	}
}
