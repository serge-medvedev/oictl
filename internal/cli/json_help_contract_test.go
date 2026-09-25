package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONHelpSourceContracts(t *testing.T) {
	// Expectations transcribed from the named source revision in json_help.go:
	// models/messages.py update_message_by_id merges data/meta; native terminal
	// grants use validateAndNormalizeResolvedAccessGrants, unlike manifest grants.
	for _, tc := range []struct {
		key    string
		want   []string
		reject string
	}{
		{"channels messages update", []string{"shallow-merge", "omitted/null"}, "omitted data/meta become null"},
		{"config terminal-servers access-grants diff", []string{"principal_id", "principal_ref", "not accepted"}, "principal_name\":\"Example Team"},
		{"users create", []string{"required", "pending", "/user.png", "nullable"}, "shell history"},
		// routers/auths.py update_admin_config preserves unsupported role/mode/expiry values.
		{"auth admin-config set", []string{`DEFAULT_USER_ROLE: "pending"|"user"|"admin"`, `CHANNEL_MODEL_RESPONSE_MODE: "thread"|"channel"`, `^(-1|0|(-?\d+(\.\d+)?)(ms|s|m|h|d|w))$`, "retains the existing setting", "CLI does not validate"}, ""},
		{"tasks config set", []string{"Required nullable strings", "TASK_MODEL_PARAMS", "AUTOCOMPLETE_GENERATION_INPUT_MAX_LENGTH"}, ""},
		{"config tool-servers verify", []string{"required but nullable", "auth_type", "config"}, ""},
		{"models sync", []string{"ModelModel", "user_id", "created_at", "updated_at"}, ""},
		{"functions sync", []string{"FunctionWithValvesModel", "type", "is_global", "valves"}, ""},
		{"knowledge files batch-add", []string{"JSON array", "file_id", "directory_id"}, ""},
		{"automations create", []string{"rrule:string", "server_id:string", "cwd:string"}, ""},
		{"scim users create", []string{"displayName:string", "emails:array", "familyName", "primary:boolean=true"}, ""},
		{"scim groups patch", []string{"Operations:array", "members[value eq", "value"}, ""},
	} {
		t.Run(tc.key, func(t *testing.T) {
			ref := jsonInputReferences[tc.key]
			text := ref.input + ref.examples
			for _, s := range tc.want {
				if !strings.Contains(strings.ToLower(text), strings.ToLower(s)) {
					t.Errorf("missing source contract %q", s)
				}
			}
			if tc.reject != "" && strings.Contains(text, tc.reject) {
				t.Errorf("inaccurate/unsolicited help: %s", tc.reject)
			}
		})
	}
	// All published grant alternatives must pass the actual local decoder, including null.
	for _, key := range []string{"config terminal-servers access-grants diff", "config terminal-servers access-grants set"} {
		for _, ex := range strings.Split(jsonInputReferences[key].examples, "\n") {
			if _, err := decodeDesiredAccessGrants([]byte(ex)); err != nil {
				t.Errorf("%s example: %v", key, err)
			}
		}
	}
	handlers := manifestHandlers()
	kinds := map[string]bool{}
	for _, ex := range strings.Split(manifestExamples, "\n") {
		var doc manifestDocument
		if err := json.Unmarshal([]byte(ex), &doc); err != nil {
			t.Fatal(err)
		}
		if err := normalizeManifestDocument(&doc); err != nil {
			t.Fatal(err)
		}
		if err := validateManifestDocument(&doc, handlers); err != nil {
			t.Errorf("%s: %v", doc.Kind, err)
		}
		if err := validateManifestPlan(manifestPlan{Actions: []planAction{{Action: "create", Kind: doc.Kind, Name: doc.Metadata.Name, Desired: &doc}}}); err != nil {
			t.Errorf("%s: %v", doc.Kind, err)
		}
		kinds[doc.Kind] = true
	}
	for kind := range handlers {
		if !kinds[kind] || !strings.Contains(manifestInputHelp, kind+":") {
			t.Errorf("manifest kind missing contract/example: %s", kind)
		}
	}
	if len(kinds) != len(handlers) {
		t.Errorf("kind count examples=%d handlers=%d", len(kinds), len(handlers))
	}
	t.Logf("manifest contracts/examples=%d", len(kinds))
}

func TestJSONHelpInert(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "output.json")
	input := filepath.Join(dir, "input.json")
	if err := os.WriteFile(output, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("stdin sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	stdin, err := os.Open(input)
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	old := os.Stdin
	os.Stdin = stdin
	defer func() { os.Stdin = old }()
	for _, args := range [][]string{
		{"users", "create", "--file", "-", "--data", "conflict", "--help"},
		{"functions", "sync", "--file", filepath.Join(dir, "missing"), "--help"},
		{"skills", "create", "--manifest", filepath.Join(dir, "missing"), "-h"},
		{"manifests", "sync", "--directory", filepath.Join(dir, "missing"), "-h"},
		{"files", "upload", filepath.Join(dir, "missing"), "--metadata", "invalid", "--help"},
		{"users", "ui-settings", "bulk-patch", "--users-file", filepath.Join(dir, "missing"), "--help"},
		{"channels", "messages", "update", "channel-a", "message-a", "--file", dir, "-h"},
		{"webhooks", "channels", "ensure", "channel-a", "--name", "Example", "--file", dir, "--help"},
	} {
		app, out, errOut := newTestApp(nil)
		app.getenv = func(string) string { t.Fatal("help read target environment"); return "" }
		app.userConfigDir = func() (string, error) { t.Fatal("help resolved profile"); return "", nil }
		app.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("help made HTTP request"); return nil, nil })}
		args = append([]string{"--profile", "absent", "--out", output}, args...)
		if code := app.Run(context.Background(), args); code != 0 || !strings.Contains(out.String(), "Input:") {
			t.Errorf("%v code=%d stderr=%s", args, code, errOut.String())
		}
	}
	if pos, err := stdin.Seek(0, io.SeekCurrent); err != nil || pos != 0 {
		t.Errorf("help consumed stdin: offset=%d err=%v", pos, err)
	}
	if b, err := os.ReadFile(output); err != nil || string(b) != "keep" {
		t.Errorf("help wrote output: %q err=%v", b, err)
	}
}

func TestJSONHelpOptionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		args []string
		help bool
	}{
		{[]string{"users", "create", "--data", "--help"}, false},
		{[]string{"users", "create", "--file", "-h"}, false},
		{[]string{"users", "create", "--data", `{"name":"--help","email":"-h","password":"help"}`}, false},
		{[]string{"users", "create", "--data=--help"}, false},
		{[]string{"users", "create", "--profile", "--help"}, false},
		{[]string{"users", "create", "--token", "-h"}, false},
		{[]string{"users", "create", "--data", "--help", "-h"}, true},
		{[]string{"skills", "create", "--manifest", "--help"}, true},
		{[]string{"skills", "create", "--manifest", "-h"}, false},
		{[]string{"models", "sync", "--yes", "--help"}, true},
		{[]string{"users", "ui-settings", "bulk-patch", "--all", "--help"}, true},
		{[]string{"users", "create", "help"}, false},
	} {
		app, out, _ := newTestApp(nil)
		got := app.printJSONInputHelp(tc.args)
		if got != tc.help {
			t.Errorf("%v help=%v want %v output=%s", tc.args, got, tc.help, out.String())
		}
	}
	// Non-help calls still validate and execute rather than returning a help page.
	for _, value := range []string{"--help", "-h", "help", `{"name":"--help"}`} {
		app, out, errOut := newTestApp(nil)
		requests := 0
		app.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests++
			b, _ := io.ReadAll(r.Body)
			if string(b) != value {
				t.Errorf("body changed: %q", b)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: http.Header{}}, nil
		})}
		code := app.Run(context.Background(), []string{"--base-url", "https://webui.example.com", "--token", "example-token", "users", "create", "--data", value})
		if code != 0 || requests != 1 || strings.Contains(out.String(), "Input:") {
			t.Errorf("value=%s code=%d requests=%d err=%s", value, code, requests, errOut.String())
		}
	}
}

func TestJSONHelpValveValuesRemainExecution(t *testing.T) {
	for _, value := range []string{"--help", "-h"} {
		app, out, errOut := newTestApp(nil)
		requests := 0
		app.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests++
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: http.Header{}}, nil
		})}
		code := app.Run(context.Background(), []string{"--base-url", "https://webui.example.com", "--token", "example-token", "tools", "valves", "update", "tool-a", "--data", value})
		if code != 0 || requests != 1 || strings.Contains(out.String(), "Usage:") {
			t.Errorf("value=%s code=%d requests=%d stdout=%s err=%s", value, code, requests, out.String(), errOut.String())
		}
	}
}

func TestJSONHelpMetadataAndEventSourceDetails(t *testing.T) {
	for key, wants := range map[string][]string{
		"models sync":                       {"record requires", "record defaults"},
		"functions create":                  {"Python identifier", "lowercases"},
		"tools create":                      {"Python identifier", "lowercases"},
		"skills create":                     {"replaces spaces with hyphens"},
		"models create":                     {"256"},
		"config terminal-servers policy":    {"null reads", "PUT"},
		"config terminal-servers lifecycle": {"null reads", "PUT"},
		"config import":                     {"flat configuration key/value map", "ui.default_locale", "retains unmentioned keys"},
		"automations update":                {"is_active:null preserves"},
		"files upload":                      {"knowledge_id", "directory_id"},
		"webhooks events create":            {"[] matches events with no associated users"},
	} {
		for _, s := range wants {
			if !strings.Contains(jsonInputReferences[key].input, s) {
				t.Errorf("%s missing %s", key, s)
			}
		}
	}
}

func TestJSONHelpFamilyDiscovery(t *testing.T) {
	families := map[string]bool{}
	for _, line := range strings.Split(jsonHelpInventory, "\n") {
		families[strings.Fields(strings.Split(line, "|")[0])[0]] = true
	}
	for family := range families {
		if family == "api" {
			continue
		}
		for _, spelling := range []string{"--help", "-h", "help", ""} {
			app, out, errOut := newTestApp(nil)
			args := []string{family}
			if spelling != "" {
				args = append(args, spelling)
			}
			if code := app.Run(context.Background(), args); code != 0 || !strings.Contains(out.String(), "--help or -h") {
				t.Errorf("%v: code=%d stdout=%s err=%s", args, code, out.String(), errOut.String())
			}
		}
	}
}
