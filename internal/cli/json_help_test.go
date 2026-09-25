package cli

import (
	"context"
	"strings"
	"testing"
)

// These rejected phrases are implementation locators/explanations, not a ban on
// Python input source, URLs, API paths, or useful resource-schema commands.
func TestJSONHelpOperatorFocusedContent(t *testing.T) {
	for _, tc := range []struct {
		path         string
		want, reject []string
	}{
		{"functions sync", []string{"function records", "created_at", "valves", "class Pipe:"}, []string{"FunctionWithValvesModel"}},
		{"models sync", []string{"model records", "record requires", "user_id", "record defaults", "created_at"}, []string{"SyncModelsForm", "ModelModel"}},
		{"auth login", []string{"email:string", "password:string", "--save"}, []string{"SigninForm"}},
		{"files update-content", []string{"content:string", "empty string", "server requires"}, []string{"ContentForm"}},
		{"users settings update", []string{"extra top-level keys", "ui:null", "Omission resets"}, []string{"UserSettings"}},
		{"webhooks events update", []string{"Omitted fields retain existing values", "explicit null is applied"}, []string{"exclude_unset"}},
		{"providers ollama verify", []string{"url:string", "key:optional nullable", "no OLLAMA_API_CONFIGS envelope"}, []string{"backend /api/version"}},
		{"providers openai request", []string{"--method POST chat/completions", "messages:array"}, []string{"selected /openai endpoint"}},
		{"providers ollama request", []string{"--method POST api/generate", "model:string required"}, []string{"selected /ollama endpoint"}},
		{"tools valves update", []string{"oictl tools valves spec <tool-id>", "resource-defined", "removes null"}, []string{"Valves class"}},
		{"tools valves user update", []string{"oictl tools valves user spec <tool-id>", "No positional user ID"}, []string{"UserValves class"}},
		{"functions valves update", []string{"oictl functions valves spec <function-id>", "resource-defined"}, []string{"Valves class"}},
		{"functions valves user update", []string{"oictl functions valves user spec <function-id>", "authenticated user"}, []string{"UserValves"}},
		{"manifests apply", []string{"writable group field", "oictl functions valves spec <function-id>", "Pipe/Filter/Action", "Tools class"}, []string{"GroupForm", "Valves class"}},
		{"knowledge get", []string{"no request body", "forwards a supplied body unchanged", "Extensions may interpret it"}, []string{"identified upstream route"}},
		{"api", []string{"POST /api/v1/auths/signin", "--method METHOD", "--header NAME:VALUE"}, nil},
		{"tools load-url", []string{"https://tools.example.com/example.py", "url:string required"}, nil},
	} {
		t.Run(tc.path, func(t *testing.T) {
			for _, flag := range []string{"--help", "-h"} {
				app, out, errOut := newTestApp(nil)
				if code := app.Run(context.Background(), append(strings.Fields(tc.path), flag)); code != 0 {
					t.Fatalf("code=%d stderr=%s", code, errOut.String())
				}
				for _, want := range tc.want {
					if !strings.Contains(out.String(), want) {
						t.Errorf("missing operator contract %q", want)
					}
				}
				for _, reject := range tc.reject {
					if strings.Contains(out.String(), reject) {
						t.Errorf("rendered implementation detail %q", reject)
					}
				}
			}
		})
	}
}

func TestJSONHelpFamilyReferenceWording(t *testing.T) {
	for _, tc := range []struct{ family, want, reject string }{
		{"users", "Cross-user ui-settings commands require --allow-ui-settings-extension", "the route is a deployment extension absent from the original API"},
		{"users", "--dry-run previews discovered targets", "use --dry-run to audit discovered targets first"},
		{"functions", "Function source is arbitrary Python loaded by Open WebUI", "Only create, update, load, or sync code from trusted sources"},
		{"functions", "fetch Python source from a URL", "from a trusted URL"},
		{"webhooks", "Table output redacts", "Use default table output"},
	} {
		app, out, errOut := newTestApp(nil)
		if code := app.Run(context.Background(), []string{tc.family, "--help"}); code != 0 {
			t.Fatalf("%s: code=%d stderr=%s", tc.family, code, errOut.String())
		}
		if !strings.Contains(out.String(), tc.want) || strings.Contains(out.String(), tc.reject) {
			t.Errorf("%s: want %q without %q", tc.family, tc.want, tc.reject)
		}
	}
}

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
