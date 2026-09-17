package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// listAllResources reads the entire inventory before callers resolve identities
// or prune. Array endpoints are unpaged; envelope endpoints must report a total.
// A repeated/empty page or changing total fails closed, never exposing a partial
// inventory to a destructive caller.
func (a *App) listAllResources(ctx context.Context, opts globalOptions, path string) ([]map[string]any, error) {
	result := []map[string]any{}
	seen := map[string]bool{}
	expected := -1
	for page := 1; ; page++ {
		requestPath := path
		if page > 1 || path == "/api/v1/models/list" || path == "/api/v1/knowledge/" || path == "/api/v1/skills/list" || path == "/api/v1/users/" {
			requestPath = appendQuery(path, map[string]string{"page": fmt.Sprint(page)})
		}
		body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: requestPath, authRequired: true})
		if err != nil {
			return nil, err
		}
		trimmed := bytes.TrimSpace(body)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
			return nil, fmt.Errorf("invalid inventory from %s", path)
		}
		objects, err := decodeResourceList(body)
		if err != nil {
			return nil, fmt.Errorf("decode inventory %s: %w", path, err)
		}
		total := -1
		if trimmed[0] == '{' {
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(body, &envelope); err != nil {
				return nil, err
			}
			raw, ok := envelope["total"]
			if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				return nil, fmt.Errorf("inventory %s is missing total", path)
			}
			if err := json.Unmarshal(raw, &total); err != nil || total < 0 {
				return nil, fmt.Errorf("inventory %s has invalid total", path)
			}
			if expected >= 0 && total != expected {
				return nil, fmt.Errorf("inventory %s total changed during pagination", path)
			}
			expected = total
		} else if page > 1 {
			return nil, fmt.Errorf("inventory %s lost pagination envelope", path)
		}
		before := len(result)
		for _, object := range objects {
			id, ok := object["id"].(string)
			if !ok || id == "" {
				return nil, fmt.Errorf("inventory %s resource missing id", path)
			}
			if !seen[id] {
				seen[id] = true
				result = append(result, object)
			}
		}
		if total < 0 {
			return result, nil
		}
		if len(result) > total {
			return nil, fmt.Errorf("inventory %s exceeds declared total", path)
		}
		if len(result) == total {
			return result, nil
		}
		if len(result) == before {
			return nil, fmt.Errorf("incomplete inventory %s: pagination made no progress (%d of %d)", path, len(result), total)
		}
	}
}
