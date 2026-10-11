package cli

import (
	"context"
	"strings"
	"testing"
)

func TestReview2CoreR03ZeroOperandActionsRejectSelectors(t *testing.T) {
	for _, group := range []string{"models", "files"} {
		actions := []string{"list", "search", "count", "delete-all"}
		if group == "models" {
			actions = []string{"list", "base", "base-tags", "tags", "create", "import", "sync", "delete-all", "export"}
		}
		for _, action := range actions {
			t.Run(group+"/"+action, func(t *testing.T) {
				app, calls := coreReviewApp(t, `true`)
				var diagnostics strings.Builder
				app.err = &diagnostics
				args := []string{group, action, "unexpected-selector"}
				if group == "models" && (action == "create" || action == "import" || action == "sync") {
					args = append(args, "--file", "/nonexistent-review2-input")
				}
				if code := app.Run(context.Background(), args); code == 0 || *calls != 0 || !strings.Contains(diagnostics.String(), "usage:") {
					t.Fatalf("code=%d requests=%d stderr=%q", code, *calls, diagnostics.String())
				}
			})
		}
	}
}

func TestReview2CoreR03ZeroOperandActionsRemainSupported(t *testing.T) {
	for _, args := range [][]string{
		{"models", "delete-all"}, {"files", "delete-all"}, {"models", "list"}, {"files", "list"},
		{"models", "create", "--data", `{"id":"example","params":{}}`},
		{"models", "import", "--data", `[]`}, {"models", "sync", "--yes", "--data", `[]`},
	} {
		app, calls := coreReviewApp(t, `true`)
		if code := app.Run(context.Background(), args); code != 0 || *calls != 1 {
			t.Errorf("args=%q code=%d requests=%d", args, code, *calls)
		}
	}
}
