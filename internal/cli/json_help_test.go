package cli

import (
	"context"
	"strings"
	"testing"
)

func TestJSONHelpNestedTracer(t *testing.T) {
	for _, action := range []struct{ path, selectors, want string }{
		{"channels messages update", "channel-a message-a", "reply_to_id"},
		{"tools valves user update", "tool-a", "oictl tools valves user spec <tool-id>"},
	} {
		for _, flag := range []string{"--help", "-h"} {
			for _, selectors := range []string{"", action.selectors} {
				app, out, errOut := newTestApp(nil)
				args := append(strings.Fields(action.path+" "+selectors), flag)
				if code := app.Run(context.Background(), args); code != 0 || !strings.Contains(out.String(), action.want) || !strings.Contains(out.String(), "oictl "+action.path) {
					t.Errorf("%v: code=%d stdout=%s stderr=%s", args, code, out.String(), errOut.String())
				}
			}
		}
	}
}

func TestJSONHelpUserCreateTracer(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		app, out, errOut := newTestApp(nil)
		app.userConfigDir = func() (string, error) { t.Fatal("help resolved configuration"); return "", nil }
		if code := app.Run(context.Background(), []string{"users", "create", flag}); code != 0 {
			t.Fatalf("code=%d stderr=%s", code, errOut.String())
		}
		for _, want := range []string{"Usage:", "oictl users create", "Input:", "name, email, password", "pending", "/user.png", "--data", "--data-file", "Examples:"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("help missing %q: %s", want, out.String())
			}
		}
	}
}
