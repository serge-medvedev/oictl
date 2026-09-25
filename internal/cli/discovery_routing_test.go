package cli

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The independent acceptance lane owns the complete maintained command inventory.
// This focused closure keeps newly authored references and their groups reachable
// without depending on an active OpenSpec change after archival.
func TestDiscoveryRoutingHelpReferences(t *testing.T) {
	paths := map[string]bool{}
	for key := range actionHelpReferences {
		paths[key] = true
	}
	for key := range jsonInputReferences {
		paths[key] = true
	}
	for key := range paths {
		words := strings.Fields(key)
		for n := 1; n < len(words); n++ {
			paths[strings.Join(words[:n], " ")] = true
		}
	}
	for path := range paths {
		for _, flag := range []string{"--help", "-h"} {
			t.Run(path+flag, func(t *testing.T) {
				app, out, errOut := newTestApp(nil)
				app.getenv = func(string) string { t.Fatal("help read environment"); return "" }
				app.userConfigDir = func() (string, error) { t.Fatal("help read config"); return "", nil }
				if code := app.Run(context.Background(), append(strings.Fields(path), flag)); code != 0 || !strings.Contains(out.String(), "oictl "+path) || errOut.Len() != 0 {
					t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
				}
			})
		}
	}
}

func TestDiscoveryRoutingLocalContent(t *testing.T) {
	for _, tc := range []struct {
		command string
		want    []string
	}{
		{"users get", []string{"Get", "oictl users get <user-id>", "--profile"}},
		{"users create", []string{"Create", "oictl users create", "--data"}},
		{"channels messages update", []string{"Update", "oictl channels messages update", "--data"}},
		{"providers openai request", []string{"Send", "oictl providers openai request", "--method"}},
		{"files rename", []string{"Rename", "<file-id>", "--name"}},
		{"channels messages", []string{"Commands:", "list", "post", "thread", "--help"}},
		{"config terminal-servers", []string{"Commands:", "access-grants", "verify", "refresh"}},
		{"profiles delete", []string{"Delete", "<name>"}},
		{"tools export --manifest", []string{"Export", "--manifest", "--directory"}},
		{"tools export tool-a --manifest", []string{"Export", "--manifest", "--directory"}},
	} {
		for _, flag := range []string{"--help", "-h"} {
			t.Run(tc.command+flag, func(t *testing.T) {
				app, out, errOut := newTestApp(nil)
				if code := app.Run(context.Background(), append(strings.Fields(tc.command), flag)); code != 0 {
					t.Fatalf("%d %s", code, errOut)
				}
				for _, want := range tc.want {
					if !strings.Contains(out.String(), want) {
						t.Errorf("missing %q in %s", want, out)
					}
				}
			})
		}
	}
}

func TestDiscoveryRoutingHelpInert(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "output")
	if err := os.WriteFile(target, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"profiles delete example", "auth logout", "tools export --manifest --directory missing", "users delete example", "files upload missing --metadata invalid", "manifests sync --directory missing"} {
		for _, flag := range []string{"--help", "-h"} {
			app, out, errOut := newTestApp(nil)
			app.userConfigDir = func() (string, error) { t.Fatal("read config"); return "", nil }
			app.getenv = func(string) string { t.Fatal("read environment"); return "" }
			args := append(strings.Fields(command), "--out", target, flag)
			if code := app.Run(context.Background(), args); code != 0 || out.Len() == 0 {
				t.Fatalf("%v: %d %s", args, code, errOut)
			}
		}
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "unchanged" {
		t.Fatalf("output changed: %q %v", data, err)
	}
}

func TestDiscoveryRoutingDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		args       []string
		want, hint string
	}{
		{[]string{"--profile"}, "--profile requires a value", "oictl --help"},
		{[]string{"users", "create", "--data"}, "--data requires a value", "oictl users create --help"},
		{[]string{"users", "list", "--unknown-option"}, "unknown option --unknown-option", "oictl users list --help"},
		{[]string{"users", "list", "--unknown-option=secret-marker"}, "unknown option --unknown-option", "oictl users list --help"},
		{[]string{"users", "list", "--unknown-option", "secret-marker"}, "unknown option --unknown-option", "oictl users list --help"},
		{[]string{"users", "list", "--data"}, "--data is not supported by users list", "oictl users list --help"},
		{[]string{"users", "list", "--data", "secret-marker"}, "--data is not supported by users list", "oictl users list --help"},
		{[]string{"users", "delete", "u", "--dry-run=false"}, "--dry-run is not supported by users delete", "oictl users delete --help"},
		{[]string{"users", "delete", "u", "--yes=secret-marker"}, "--yes requires a boolean", "oictl users delete --help"},
		{[]string{"api", "/example", "--header", "secret-marker"}, "malformed header", "oictl api --help"},
		{[]string{"skills", "create", "--manifest"}, "--manifest requires a value", "oictl skills create --help"},
		{[]string{"users", "list", "--version"}, "unknown option --version", "oictl users list --help"},
	} {
		t.Run(strings.Join(tc.args, "/"), func(t *testing.T) {
			app, out, errOut := newTestApp(nil)
			app.getenv = func(string) string { t.Fatal("diagnostic resolved environment"); return "" }
			app.userConfigDir = func() (string, error) { t.Fatal("diagnostic resolved config"); return "", nil }
			if code := app.Run(context.Background(), tc.args); code == 0 || out.Len() != 0 {
				t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
			}
			for _, want := range []string{tc.want, tc.hint} {
				if !strings.Contains(errOut.String(), want) {
					t.Errorf("missing %q in %s", want, errOut)
				}
			}
			if strings.Contains(errOut.String(), "secret-marker") {
				t.Errorf("value disclosed: %s", errOut)
			}
		})
	}
}

func TestDiscoveryRoutingUnknownHelp(t *testing.T) {
	for _, command := range []string{"unknown-command", "users unknown-command", "channels messages unknown-command", "providers openai tags", "analytics models unknown-command", "tools valves unknown-command"} {
		for _, flag := range []string{"--help", "-h"} {
			app, out, errOut := newTestApp(nil)
			args := append(strings.Fields(command), flag)
			if code := app.Run(context.Background(), args); code == 0 || out.Len() != 0 || !strings.Contains(errOut.String(), "unknown command") || !strings.Contains(errOut.String(), "--help") {
				t.Errorf("%v: %d %q %q", args, code, out, errOut)
			}
		}
	}
}

func TestDiscoveryRoutingBoundaryControls(t *testing.T) {
	for _, value := range []string{"--help", "-h", "--version", "-v", "help", "-identifier"} {
		t.Run(value, func(t *testing.T) {
			app, out, errOut := newTestApp(nil)
			requests := 0
			app.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.Query().Get("query") != value {
					t.Errorf("query changed: %s", r.URL)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("[]")), Header: http.Header{}}, nil
			})}
			args := []string{"--base-url", "https://webui.example.com", "--token", "fixture", "users", "list", "--query", value}
			if code := app.Run(context.Background(), args); code != 0 || requests != 1 || strings.Contains(out.String(), "Usage:") {
				t.Fatalf("code=%d requests=%d out=%q err=%q", code, requests, out, errOut)
			}
			app, out, errOut = newTestApp(nil)
			app.getenv = func(string) string { t.Fatal("separate help reached execution"); return "" }
			if code := app.Run(context.Background(), append(args, "-h")); code != 0 || !strings.Contains(out.String(), "oictl users list") {
				t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
			}
		})
	}
	for _, tc := range []struct {
		args []string
		path string
	}{
		{[]string{"knowledge", "sync", "custom-action", "knowledge-a"}, "/api/v1/knowledge/knowledge-a/sync/custom-action"},
		{[]string{"users", "get", "-identifier"}, "/api/v1/users/-identifier"},
	} {
		app, out, errOut := newTestApp(nil)
		requests := 0
		app.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests++
			if r.URL.Path != tc.path {
				t.Errorf("path=%s want=%s", r.URL.Path, tc.path)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}")), Header: http.Header{}}, nil
		})}
		args := append([]string{"--base-url", "https://webui.example.com", "--token", "fixture"}, tc.args...)
		if code := app.Run(context.Background(), args); code != 0 || requests != 1 {
			t.Errorf("%v code=%d requests=%d out=%q err=%q", tc.args, code, requests, out, errOut)
		}
	}
}

func TestDiscoveryRoutingGroupDiscovery(t *testing.T) {
	for _, path := range []string{"", "profiles", "analytics", "tools valves", "functions valves"} {
		app, out, errOut := newTestApp(nil)
		if code := app.Run(context.Background(), append(strings.Fields(path), "--help")); code != 0 || !strings.Contains(out.String(), "--help") {
			t.Errorf("%q code=%d out=%q err=%q", path, code, out, errOut)
		}
	}
}

func TestDiscoveryRoutingSkillManifestValues(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, action := range [][]string{{"skills", "create"}, {"skills", "update", "skill-a"}} {
		for _, value := range []string{"--help", "-h"} {
			args := append(append([]string{}, action...), "--manifest", value)
			app, out, errOut := newTestApp(nil)
			if code := app.Run(context.Background(), args); code == 0 || out.Len() != 0 || !strings.Contains(errOut.String(), value) || !strings.Contains(errOut.String(), "no such file") {
				t.Fatalf("%v code=%d out=%q err=%q", args, code, out, errOut)
			}
			for _, flag := range []string{"--help", "-h"} {
				app, out, errOut = newTestApp(nil)
				if code := app.Run(context.Background(), append(args, flag)); code != 0 || !strings.Contains(out.String(), "Usage:") {
					t.Fatalf("%v code=%d out=%q err=%q", args, code, out, errOut)
				}
			}
		}
	}
}

func TestDiscoveryRoutingVersion(t *testing.T) {
	for _, spelling := range []string{"version", "--version", "-v"} {
		for _, suffix := range [][]string{nil, {"--help"}, {"-h"}, {"help"}} {
			t.Run(spelling+strings.Join(suffix, "/"), func(t *testing.T) {
				app, out, errOut := newTestApp(nil)
				app.getenv = func(string) string { t.Fatal("version read environment"); return "" }
				app.userConfigDir = func() (string, error) { t.Fatal("version resolved profile"); return "", nil }
				args := append([]string{"--profile", "missing", "--out", "missing/output", spelling}, suffix...)
				if code := app.Run(context.Background(), args); code != 0 || out.String() != "oictl test\n" || errOut.Len() != 0 {
					t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errOut)
				}
			})
		}
	}
}
