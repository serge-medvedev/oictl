package cli

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// The compact inventory closes over finite dispatch paths, not merely allowlist
// keys or JSON consumers. It survives archival of the planning change. Runtime
// behavior is exercised only through App.Run and existing injection seams.
// Baseline global/local missing-value emitters: parseGlobalFlags' setter and
// parseCommandFlags' scalar branch in app.go. Both need contextual errors.
type discoveryRow struct {
	path, kind, selectors, flags, longBaseline, shortBaseline string
}

type discoveryInventory struct {
	rows              []discoveryRow
	globals, booleans []string
}

func discoveryLoad(t *testing.T) discoveryInventory {
	t.Helper()
	body, err := os.ReadFile("testdata/discovery_inventory.tsv")
	if err != nil {
		t.Fatal(err)
	}
	var inv discoveryInventory
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "# globals: ") {
			inv.globals = strings.Fields(strings.TrimPrefix(line, "# globals: "))
		}
		if strings.HasPrefix(line, "# booleans: ") {
			inv.booleans = strings.Fields(strings.TrimPrefix(line, "# booleans: "))
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		v := strings.Split(line, "|")
		if len(v) != 6 {
			t.Fatalf("malformed inventory row: %q", line)
		}
		inv.rows = append(inv.rows, discoveryRow{v[0], v[1], v[2], v[3], v[4], v[5]})
	}
	if len(inv.rows) == 0 || len(inv.globals) == 0 || len(inv.booleans) == 0 {
		t.Fatal("empty inventory")
	}
	return inv
}

func discoveryHas(words, word string) bool {
	for _, w := range strings.Fields(words) {
		if w == word {
			return true
		}
	}
	return false
}

func discoveryJSONControls() map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(jsonHelpInventory, "\n") {
		v := strings.Split(strings.TrimSpace(line), "|")
		m[v[0]] = v[3]
	}
	return m
}

func TestDiscoveryInventoryConsistency(t *testing.T) {
	inv := discoveryLoad(t)
	seen := map[string]discoveryRow{}
	options := map[string]bool{}
	kinds, long, short := map[string]int{}, map[string]int{}, map[string]int{}
	helpCalls, valuedPairs := 0, 0
	for _, flag := range inv.globals {
		options[flag] = true
	}
	for _, r := range inv.rows {
		if _, ok := seen[r.path]; ok {
			t.Fatalf("duplicate path %q", r.path)
		}
		seen[r.path] = r
		if !discoveryHas("A G B", r.kind) {
			t.Fatalf("unknown kind %q", r.kind)
		}
		kinds[r.kind]++
		long[r.longBaseline]++
		short[r.shortBaseline]++
		helpCalls += 2
		if r.selectors != "" {
			helpCalls += 2
		}
		for _, flag := range strings.Fields(r.flags) {
			options[flag] = true
			if flag != "manifest" && !discoveryHas(strings.Join(inv.booleans, " "), flag) {
				valuedPairs++
			}
		}
		// Reconcile inherited allowlist ownership without assuming one key per leaf.
		parts := strings.Fields(r.path)
		actual := ""
		for n := len(parts); n > 0; n-- {
			if flags, ok := commandFlagAllowlist[strings.Join(parts[:n], " ")]; ok {
				actual = flags
				break
			}
		}
		wantFlags, actualFlags := strings.Fields(r.flags), strings.Fields(actual)
		sort.Strings(wantFlags)
		sort.Strings(actualFlags)
		if strings.Join(wantFlags, " ") != strings.Join(actualFlags, " ") {
			t.Errorf("%s option ownership: fixture=%v runtime=%v", r.path, wantFlags, actualFlags)
		}
	}
	for key := range commandFlagAllowlist {
		if _, ok := seen[key]; !ok {
			t.Errorf("allowlist key missing: %s", key)
		}
	}
	for key := range discoveryJSONControls() {
		if _, ok := seen[key]; !ok {
			t.Errorf("JSON action missing: %s", key)
		}
	}
	for _, r := range inv.rows {
		parent, _, nested := strings.Cut(r.path, " ")
		if i := strings.LastIndex(r.path, " "); i >= 0 {
			parent = r.path[:i]
		}
		if nested && seen[parent].kind != "G" && seen[parent].kind != "B" {
			t.Errorf("%s has no parent group %s", r.path, parent)
		}
	}
	for _, name := range inv.booleans {
		if !options[name] {
			t.Errorf("unowned boolean %s", name)
		}
	}
	for _, alias := range []string{"help", "--help", "-h", "version", "--version", "-v", "providers openai config set", "providers ollama config set"} {
		if _, ok := seen[alias]; !ok {
			t.Errorf("missing alias %s", alias)
		}
	}
	t.Logf("paths=%d kinds=%v ordinary-options=%d global=%d boolean=%d valued-local-pairs=%d help-invocations=%d long-baseline=%v short-baseline=%v", len(seen), kinds, len(options), len(inv.globals), len(inv.booleans), valuedPairs, helpCalls, long, short)
}

type discoveryTransport func(*http.Request) (*http.Response, error)

func (f discoveryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type discoveryRequest struct{ method, path, query, body, authorization string }
type discoveryHarness struct {
	app                   *App
	out, errOut           bytes.Buffer
	root                  string
	configReads, envReads int
	requests              []discoveryRequest
}

func discoveryNew(t *testing.T) *discoveryHarness {
	t.Helper()
	h := &discoveryHarness{root: t.TempDir()}
	profiles := filepath.Join(h.root, "oictl", "profiles")
	if err := os.MkdirAll(profiles, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"default", "example-1", "-h", "help", "--version", "-v"} {
		if err := os.WriteFile(filepath.Join(profiles, name+".json"), []byte(`{"base_url":"https://example.invalid","token":"synthetic-token","scim_token":"synthetic-scim"}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	h.app = New(&h.out, &h.errOut, "discovery-test")
	h.app.getenv = func(key string) string {
		h.envReads++
		switch key {
		case "OPEN_WEBUI_URL":
			return "https://example.invalid"
		case "OPEN_WEBUI_API_KEY":
			return "synthetic-token"
		case "OPEN_WEBUI_SCIM_TOKEN":
			return "synthetic-scim"
		}
		return ""
	}
	h.app.userConfigDir = func() (string, error) { h.configReads++; return h.root, nil }
	// Never dial: even broken help reaches only this refusing in-process peer.
	h.app.httpClient = &http.Client{Transport: discoveryTransport(func(r *http.Request) (*http.Response, error) {
		var body []byte
		if r.Body != nil {
			body, _ = io.ReadAll(r.Body)
		}
		h.requests = append(h.requests, discoveryRequest{r.Method, r.URL.Path, r.URL.RawQuery, string(body), r.Header.Get("Authorization")})
		return &http.Response{StatusCode: 418, Status: "418 fixture refused", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"detail":"fixture refused"}`)), Request: r}, nil
	})}
	return h
}

func discoverySnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		info, err := d.Info()
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if d.Type()&os.ModeSymlink != 0 {
			dest, err := os.Readlink(path)
			if err != nil {
				return err
			}
			value += ":" + dest
		} else if !d.IsDir() {
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += ":" + string(body)
		}
		m[rel] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func (h *discoveryHarness) run(args ...string) int { return h.app.Run(context.Background(), args) }
func (h *discoveryHarness) inert(t *testing.T, before map[string]string) {
	t.Helper()
	if h.configReads != 0 || h.envReads != 0 || len(h.requests) != 0 {
		t.Errorf("informational/diagnostic side effects: config=%d env=%d requests=%+v", h.configReads, h.envReads, h.requests)
	}
	if after := discoverySnapshot(t, h.root); !reflect.DeepEqual(before, after) {
		t.Error("fixture files added, removed, changed or chmodded")
	}
}

func discoveryCheckHelp(t *testing.T, text string, row discoveryRow, inv discoveryInventory) {
	t.Helper()
	if discoveryHas("version --version -v", row.path) {
		if text != "oictl discovery-test\n" {
			t.Errorf("version precedence: %q", text)
		}
		return
	}
	if !strings.Contains(text, "Usage:") {
		t.Errorf("missing usage: %q", text)
		return
	}
	path := row.path
	if path == "" || discoveryHas("help --help -h", path) {
		path = ""
	}
	if path != "" && !strings.Contains(text, "oictl "+path) {
		t.Errorf("missing local command %q: %s", path, text)
	}
	for _, option := range strings.Fields(row.flags) {
		if !strings.Contains(text, "--"+option) {
			t.Errorf("%q help omits supported option --%s", path, option)
		}
	}
	if row.kind == "G" || row.kind == "B" {
		for _, child := range inv.rows {
			prefix := path + " "
			if path == "" {
				prefix = ""
			}
			if !strings.HasPrefix(child.path, prefix) {
				continue
			}
			suffix := strings.TrimPrefix(child.path, prefix)
			if suffix == "" || strings.Contains(suffix, " ") || strings.HasPrefix(suffix, "-") {
				continue
			}
			if !strings.Contains(text, suffix) {
				t.Errorf("group %q missing child %q", path, suffix)
			}
		}
		if !strings.Contains(text, "--help") && !strings.Contains(text, "-h") {
			t.Errorf("group %q gives no child-help discovery", path)
		}
	} else if path != "" {
		// A family dump containing the leaf somewhere is not local leaf help.
		_, usage, _ := strings.Cut(text, "Usage:")
		first := ""
		for _, line := range strings.Split(usage, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "oictl ") {
				first = strings.TrimSpace(line)
				break
			}
		}
		if first != "oictl "+path && !strings.HasPrefix(first, "oictl "+path+" ") {
			t.Errorf("first usage is not selected leaf %q: %q", path, first)
		}
		if row.selectors != "" && !strings.Contains(first, "<") && !strings.Contains(first, "PATH") {
			t.Errorf("missing operand syntax in %q", first)
		}
	}
}

// Diagnostics assert categories, not an implementation-specific full sentence.
func discoveryDiagnostic(t *testing.T, args []string, option, category, scope string) {
	t.Helper()
	h := discoveryNew(t)
	before := discoverySnapshot(t, h.root)
	code := h.run(args...)
	h.inert(t, before)
	text := h.errOut.String()
	if code == 0 || h.out.Len() != 0 {
		t.Errorf("expected nonzero diagnostic, got code=%d stdout=%q", code, h.out.String())
	}
	if !strings.Contains(text, option) {
		t.Errorf("diagnostic does not identify %s: %q", option, text)
	}
	switch category {
	case "missing":
		if !strings.Contains(text, "requires a value") {
			t.Errorf("missing-value classification lost: %q", text)
		}
	case "unknown":
		if !strings.Contains(strings.ToLower(text), "unknown") || strings.Contains(text, "requires a value") {
			t.Errorf("unknown option misclassified: %q", text)
		}
	case "nested-version":
		if (!strings.Contains(strings.ToLower(text), "unknown") && !strings.Contains(text, "not supported") && !strings.Contains(text, "unsupported")) || strings.Contains(text, "requires a value") {
			t.Errorf("nested version must be rejected, not global discovery: %q", text)
		}
	case "unsupported":
		if (!strings.Contains(text, "not supported") && !strings.Contains(text, "unsupported")) || strings.Contains(text, "requires a value") {
			t.Errorf("unsupported option misclassified: %q", text)
		}
	}
	if !strings.Contains(strings.ToLower(text), "usage:") && !(strings.Contains(text, "oictl ") && strings.Contains(text, "--help")) {
		t.Errorf("no usable diagnostic context: %q", text)
	}
	if scope != "" && !strings.Contains(text, "oictl "+scope) {
		t.Errorf("missing selected context %q: %q", scope, text)
	}
	if strings.Contains(text, "discovery-secret-value") {
		t.Errorf("diagnostic leaked supplied value: %q", text)
	}
}

func TestDiscoveryMissingValues(t *testing.T) {
	inv := discoveryLoad(t)
	for _, flag := range inv.globals {
		t.Run("global/"+flag, func(t *testing.T) { discoveryDiagnostic(t, []string{"--" + flag}, "--"+flag, "missing", "") })
	}
	for _, row := range inv.rows {
		for _, flag := range strings.Fields(row.flags) {
			if flag == "manifest" || discoveryHas(strings.Join(inv.booleans, " "), flag) {
				continue
			}
			t.Run(row.path+"/"+flag, func(t *testing.T) {
				args := append(strings.Fields(row.path+" "+row.selectors), "--"+flag)
				discoveryDiagnostic(t, args, "--"+flag, "missing", row.path)
			})
		}
	}
	for _, path := range []string{"skills create", "skills update example-1"} {
		t.Run(path+"/manifest", func(t *testing.T) {
			discoveryDiagnostic(t, append(strings.Fields(path), "--manifest"), "--manifest", "missing", strings.Join(strings.Fields(path)[:2], " "))
		})
	}
}

func TestDiscoveryOptionClassification(t *testing.T) {
	inv := discoveryLoad(t)
	local := map[string]bool{}
	for _, row := range inv.rows {
		for _, flag := range strings.Fields(row.flags) {
			local[flag] = true
		}
	}
	names := make([]string, 0, len(local))
	for name := range local {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, flag := range names {
		// users get admits no local flags: every one of the 65 known local
		// names has an unsupported control, including explicit-false booleans.
		values := [][]string{{"--" + flag}, {"--" + flag + "=discovery-secret-value"}}
		if discoveryHas(strings.Join(inv.booleans, " "), flag) || flag == "manifest" {
			values = [][]string{{"--" + flag}, {"--" + flag + "=false"}}
		}
		for i, value := range values {
			t.Run(flag+"/"+string(rune('0'+i)), func(t *testing.T) {
				discoveryDiagnostic(t, append([]string{"users", "get", "example-1"}, value...), "--"+flag, "unsupported", "users get")
			})
		}
	}
	for _, path := range []string{"api", "users", "users list", "config terminal-servers"} {
		for _, tail := range [][]string{{"--unknown-option"}, {"--unknown-option", "discovery-secret-value"}, {"--unknown-option=discovery-secret-value"}} {
			t.Run(path+"/"+strings.Join(tail, " "), func(t *testing.T) {
				discoveryDiagnostic(t, append(strings.Fields(path), tail...), "--unknown-option", "unknown", path)
			})
		}
	}
	for _, tail := range [][]string{{"--version"}} {
		t.Run(strings.Join(tail, " "), func(t *testing.T) {
			discoveryDiagnostic(t, append([]string{"users", "list"}, tail...), strings.SplitN(tail[0], "=", 2)[0], "nested-version", "users list")
		})
	}
	for _, tail := range [][]string{{"--data"}, {"--data", "discovery-secret-value"}, {"--data=discovery-secret-value"}} {
		t.Run("unsupported-data/"+strings.Join(tail, " "), func(t *testing.T) {
			discoveryDiagnostic(t, append([]string{"users", "list"}, tail...), "--data", "unsupported", "users list")
		})
	}
	for _, flag := range []string{"--dry-run", "--dry-run=false", "--allow-sensitive-ui-keys=false", "--allow-ui-settings-extension=false"} {
		t.Run("safety/"+flag, func(t *testing.T) {
			discoveryDiagnostic(t, []string{"users", "delete", "example-1", "--yes", flag}, strings.SplitN(flag, "=", 2)[0], "unsupported", "users delete")
		})
	}
}

func TestDiscoveryVersion(t *testing.T) {
	for _, spelling := range []string{"version", "--version", "-v"} {
		for _, suffix := range []string{"", "--help", "-h", "help"} {
			for _, globals := range []bool{false, true} {
				name := spelling + "/plain"
				if suffix != "" {
					name = spelling + "/" + suffix
				}
				if globals {
					name += "/globals"
				} else {
					name += "/no-globals"
				}
				t.Run(name, func(t *testing.T) {
					h := discoveryNew(t)
					before := discoverySnapshot(t, h.root)
					args := []string{spelling}
					if suffix != "" {
						args = append(args, suffix)
					}
					if globals {
						args = append([]string{"--profile", "missing", "--base-url", "invalid-but-unused", "--token", "synthetic-token", "--timeout", "invalid-but-unused", "--output", "invalid-but-unused", "--out", filepath.Join(h.root, "must-not-exist")}, args...)
					}
					code := h.run(args...)
					h.inert(t, before)
					if code != 0 || h.out.String() != "oictl discovery-test\n" || h.errOut.Len() != 0 {
						t.Errorf("argv=%q code=%d stdout=%q stderr=%q", args, code, h.out.String(), h.errOut.String())
					}
				})
			}
		}
	}
}

func TestDiscoveryUnknownCommands(t *testing.T) {
	for _, path := range []string{"not-a-command", "users not-an-action", "config terminal-servers not-an-action", "channels messages not-an-action", "tools valves not-an-action", "providers openai tags", "providers openai version", "providers openai ps"} {
		for _, flag := range []string{"--help", "-h"} {
			t.Run(path+"/"+flag, func(t *testing.T) {
				h := discoveryNew(t)
				before := discoverySnapshot(t, h.root)
				args := append(strings.Fields(path), flag)
				code := h.run(args...)
				h.inert(t, before)
				text := h.errOut.String()
				last := strings.Fields(path)
				lastToken := last[len(last)-1]
				if code == 0 || h.out.Len() != 0 || !strings.Contains(text, lastToken) {
					t.Errorf("unsupported path reported as help: code=%d stdout=%q stderr=%q", code, h.out.String(), text)
				}
				if !strings.Contains(strings.ToLower(text), "usage:") && !strings.Contains(text, "--help") {
					t.Errorf("no nearest help context: %q", text)
				}
			})
		}
	}
}

func TestDiscoveryPositionalHelpControls(t *testing.T) {
	inv := discoveryLoad(t)
	paths := []string{"", "tools valves", "functions valves"}
	for _, row := range inv.rows {
		if row.kind == "G" && row.path != "" && !strings.Contains(row.path, " ") {
			paths = append(paths, row.path)
		}
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			h := discoveryNew(t)
			before := discoverySnapshot(t, h.root)
			code := h.run(append(strings.Fields(path), "help")...)
			h.inert(t, before)
			if code != 0 || h.errOut.Len() != 0 || !strings.Contains(h.out.String(), "Usage:") {
				t.Errorf("existing positional help lost: code=%d stdout=%q stderr=%q", code, h.out.String(), h.errOut.String())
			}
		})
	}
	for _, args := range [][]string{{"help", "models", "list"}, {"models", "help", "list"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			h := discoveryNew(t)
			before := discoverySnapshot(t, h.root)
			code := h.run(args...)
			h.inert(t, before)
			want := "Commands:"
			if args[0] == "models" {
				want = "oictl models <list|"
			}
			if code != 0 || !strings.Contains(h.out.String(), want) {
				t.Errorf("positional help incorrectly rerouted: code=%d stdout=%q stderr=%q", code, h.out.String(), h.errOut.String())
			}
		})
	}
}

func TestDiscoveryLiteralValuesControls(t *testing.T) {
	for _, value := range []string{"--help", "-h", "--version", "-v", "help", "-example"} {
		for _, equals := range []bool{false, true} {
			t.Run(value+map[bool]string{false: "/separate", true: "/equals"}[equals], func(t *testing.T) {
				h := discoveryNew(t)
				args := []string{"users", "list", "--query", value}
				if equals {
					args = []string{"users", "list", "--query=" + value}
				}
				code := h.run(args...)
				if code == 0 || len(h.requests) != 1 || h.out.Len() != 0 {
					t.Fatalf("literal query turned into information: code=%d requests=%+v stdout=%q stderr=%q", code, h.requests, h.out.String(), h.errOut.String())
				}
				r := h.requests[0]
				q, err := url.ParseQuery(r.query)
				if err != nil {
					t.Fatal(err)
				}
				if q.Get("query") != value {
					t.Errorf("query value changed: %q -> %q", value, r.query)
				}
			})
		}
		for _, mode := range []string{"body", "token", "profile", "operand"} {
			// Long options remain options; only ordinary dash-leading operands
			// (-v/-example) and the positional word help are data here.
			if mode == "operand" && (strings.HasPrefix(value, "--") || value == "-h") {
				continue
			}
			t.Run(value+"/"+mode, func(t *testing.T) {
				h := discoveryNew(t)
				var args []string
				switch mode {
				case "body":
					args = []string{"api", "POST", "/example", "--data", value}
				case "token":
					args = []string{"--token", value, "api", "/example"}
				case "profile":
					args = []string{"--profile", value, "api", "/example"}
				case "operand":
					args = []string{"users", "get", value}
				}
				code := h.run(args...)
				if code == 0 || len(h.requests) != 1 || h.out.Len() != 0 {
					t.Fatalf("value became informational: argv=%q code=%d requests=%+v stdout=%q stderr=%q", args, code, h.requests, h.out.String(), h.errOut.String())
				}
				if mode == "body" && h.requests[0].body != value {
					t.Errorf("body changed: %+v", h.requests[0])
				}
				if mode == "token" && h.requests[0].authorization != "Bearer "+value {
					t.Errorf("token value changed")
				}
				if mode == "operand" && !strings.Contains(h.requests[0].path, value) {
					t.Errorf("operand changed: %+v", h.requests[0])
				}
			})
		}
		for _, flag := range []string{"--help", "-h"} {
			t.Run(value+"/later/"+flag, func(t *testing.T) {
				h := discoveryNew(t)
				before := discoverySnapshot(t, h.root)
				code := h.run("users", "list", "--query", value, flag)
				h.inert(t, before)
				if code != 0 || !strings.Contains(h.out.String(), "oictl users list") {
					t.Errorf("separate help not recognized: code=%d stdout=%q stderr=%q", code, h.out.String(), h.errOut.String())
				}
			})
		}
	}
}

func TestDiscoveryGlobalValueBoundaries(t *testing.T) {
	inv := discoveryLoad(t)
	for _, option := range inv.globals {
		for _, value := range []string{"--help", "-h", "--version", "-v", "help"} {
			t.Run(option+"/"+value, func(t *testing.T) {
				h := discoveryNew(t)
				// Owned cwd also contains any literal --out filename.
				t.Chdir(h.root)
				code := h.run("--"+option, value, "users", "list")
				if strings.Contains(h.out.String(), "Usage:") || strings.HasPrefix(h.out.String(), "oictl discovery-test") || code == 0 {
					t.Errorf("global value became information: option=%s value=%q code=%d stdout=%q stderr=%q", option, value, code, h.out.String(), h.errOut.String())
				}
				// Flags validated only at execution may fail before HTTP; neither
				// discovery success nor a value-arity error is acceptable here.
				if strings.Contains(h.errOut.String(), "requires a value") {
					t.Errorf("complete global value lost: %q", h.errOut.String())
				}
			})
		}
	}
}

func TestDiscoveryManifestBoundaries(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		t.Run("tool-export/"+flag, func(t *testing.T) {
			h := discoveryNew(t)
			before := discoverySnapshot(t, h.root)
			code := h.run("tools", "export", "--manifest", flag)
			h.inert(t, before)
			if code != 0 || h.errOut.Len() != 0 || !strings.Contains(h.out.String(), "oictl tools export") {
				t.Errorf("boolean manifest consumed help: code=%d stdout=%q stderr=%q", code, h.out.String(), h.errOut.String())
			}
		})
	}
	for _, path := range []string{"skills create", "skills update example-1"} {
		for _, value := range []string{"-h", "--help"} {
			t.Run(path+"/literal/"+value, func(t *testing.T) {
				h := discoveryNew(t)
				// The owned cwd guarantees these literal paths do not exist.
				t.Chdir(h.root)
				code := h.run(append(strings.Fields(path), "--manifest", value)...)
				if code == 0 || h.out.Len() != 0 || len(h.requests) != 0 || !strings.Contains(h.errOut.String(), value) || !strings.Contains(h.errOut.String(), "no such file") {
					t.Errorf("manifest path was not consumed literally: code=%d stdout=%q stderr=%q", code, h.out.String(), h.errOut.String())
				}
			})
			for _, flag := range []string{"--help", "-h"} {
				t.Run(path+"/later/"+value+"/"+flag, func(t *testing.T) {
					h := discoveryNew(t)
					before := discoverySnapshot(t, h.root)
					code := h.run(append(strings.Fields(path), "--manifest", value, flag)...)
					h.inert(t, before)
					if code != 0 || h.errOut.Len() != 0 || !strings.Contains(h.out.String(), "Usage:") {
						t.Errorf("later help missing: code=%d stdout=%q stderr=%q", code, h.out.String(), h.errOut.String())
					}
				})
			}
		}
	}
}

func TestDiscoveryBooleanControls(t *testing.T) {
	for _, value := range []string{"", "=true", "=false"} {
		t.Run("valid/"+value, func(t *testing.T) {
			h := discoveryNew(t)
			code := h.run("users", "delete", "example-1", "--yes"+value)
			if code == 0 {
				t.Error("refusing fixture/confirmation unexpectedly succeeded")
			}
			want := 1
			if value == "=false" {
				want = 0
			}
			if len(h.requests) != want {
				t.Errorf("boolean admission changed: requests=%+v stderr=%q", h.requests, h.errOut.String())
			}
			if want == 1 && (h.requests[0].method != "DELETE" || !strings.Contains(h.requests[0].path, "example-1")) {
				t.Errorf("wrong delete: %+v", h.requests)
			}
		})
	}
	for _, flag := range []string{"--yes=maybe", "--confirm=maybe"} {
		t.Run("invalid/"+flag, func(t *testing.T) {
			h := discoveryNew(t)
			before := discoverySnapshot(t, h.root)
			code := h.run("users", "delete", "example-1", flag)
			h.inert(t, before)
			if code == 0 || !strings.Contains(h.errOut.String(), "boolean") {
				t.Errorf("invalid boolean admitted: code=%d stderr=%q", code, h.errOut.String())
			}
		})
	}
}

func TestDiscoveryDynamicBoundaryControls(t *testing.T) {
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "get"} {
		for _, endpoint := range []string{"/example/nested?key=value", "https://other.example.invalid/example"} {
			t.Run(method+"/"+endpoint, func(t *testing.T) {
				h := discoveryNew(t)
				code := h.run("api", method, endpoint)
				if code == 0 || len(h.requests) != 1 || h.requests[0].method != strings.ToUpper(method) {
					t.Errorf("dynamic API grammar changed: code=%d requests=%+v stderr=%q", code, h.requests, h.errOut.String())
				}
			})
		}
		for _, flag := range []string{"--help", "-h"} {
			for _, endpoint := range []string{"", "/example"} {
				t.Run("help/"+method+"/"+flag+"/"+endpoint, func(t *testing.T) {
					h := discoveryNew(t)
					before := discoverySnapshot(t, h.root)
					args := []string{"api", method}
					if endpoint != "" {
						args = append(args, endpoint)
					}
					args = append(args, flag)
					code := h.run(args...)
					h.inert(t, before)
					if code != 0 || !strings.Contains(h.out.String(), "oictl api") {
						t.Errorf("method help missing: code=%d stdout=%q stderr=%q", code, h.out.String(), h.errOut.String())
					}
				})
			}
		}
	}
	for _, tc := range []struct {
		args         []string
		method, path string
	}{
		{[]string{"api", "/example", "--method", "CUSTOM"}, "CUSTOM", "/example"},
		{[]string{"providers", "openai", "request", "arbitrary/nested", "--method", "POST"}, "POST", "/openai/arbitrary/nested"},
		{[]string{"providers", "ollama", "request", "arbitrary/nested", "--method", "GET"}, "GET", "/ollama/arbitrary/nested"},
		{[]string{"knowledge", "sync", "example-operation", "example-1"}, "POST", "/api/v1/knowledge/example-1/sync/example-operation"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			h := discoveryNew(t)
			code := h.run(tc.args...)
			if code == 0 || len(h.requests) != 1 || h.requests[0].method != tc.method || h.requests[0].path != tc.path {
				t.Errorf("dynamic boundary constrained: code=%d requests=%+v stderr=%q", code, h.requests, h.errOut.String())
			}
		})
	}
}

func TestDiscoveryHelpInertInputsAndOutputs(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		for _, command := range []string{
			"profiles delete example-1", "profiles set example-1", "auth logout", "users delete example-1", "channels messages delete example-1 example-2",
			"functions sync --file MISSING", "models sync --data-file MISSING", "manifests sync --directory MISSING", "files upload MISSING",
			"users ui-settings bulk-patch --users-file MISSING", "skills create --manifest MISSING", "tools export --manifest --directory MISSING",
			"config terminal-servers access-grants set example-1 --file MISSING", "users create --file -",
		} {
			t.Run(command+"/"+flag, func(t *testing.T) {
				h := discoveryNew(t)
				stdinPath := filepath.Join(h.root, "stdin")
				if err := os.WriteFile(stdinPath, []byte(`{"sentinel":"unread"}`), 0600); err != nil {
					t.Fatal(err)
				}
				stdin, err := os.Open(stdinPath)
				if err != nil {
					t.Fatal(err)
				}
				defer stdin.Close()
				old := os.Stdin
				os.Stdin = stdin
				defer func() { os.Stdin = old }()
				output := filepath.Join(h.root, "output")
				if err := os.WriteFile(output, []byte("preserve output"), 0640); err != nil {
					t.Fatal(err)
				}
				before := discoverySnapshot(t, h.root)
				args := strings.Fields(command)
				for i, v := range args {
					if v == "MISSING" {
						args[i] = filepath.Join(h.root, "absent")
					}
				}
				args = append(args, "--profile", "missing-profile", "--base-url", "https://example.invalid", "--out", output, flag)
				code := h.run(args...)
				h.inert(t, before)
				offset, err := stdin.Seek(0, io.SeekCurrent)
				if err != nil || offset != 0 {
					t.Errorf("stdin consumed: offset=%d err=%v", offset, err)
				}
				if code != 0 || h.errOut.Len() != 0 || !strings.Contains(h.out.String(), "Usage:") {
					t.Errorf("help reached operation prerequisites: argv=%q code=%d stdout=%q stderr=%q", args, code, h.out.String(), h.errOut.String())
				}
			})
		}
	}
}

func TestDiscoveryObserverControls(t *testing.T) {
	t.Run("HTTP", func(t *testing.T) {
		h := discoveryNew(t)
		code := h.run("users", "get", "example-1")
		if code == 0 || len(h.requests) != 1 || h.configReads == 0 || h.envReads == 0 {
			t.Fatalf("observers insensitive: code=%d config=%d env=%d requests=%+v", code, h.configReads, h.envReads, h.requests)
		}
	})
	t.Run("profile-mutation", func(t *testing.T) {
		h := discoveryNew(t)
		before := discoverySnapshot(t, h.root)
		if code := h.run("profiles", "delete", "example-1"); code != 0 {
			t.Fatal(h.errOut.String())
		}
		if reflect.DeepEqual(before, discoverySnapshot(t, h.root)) || h.configReads == 0 {
			t.Fatal("profile mutation observer insensitive")
		}
	})
	t.Run("stdin-and-output", func(t *testing.T) {
		h := discoveryNew(t)
		path := filepath.Join(h.root, "stdin")
		if err := os.WriteFile(path, []byte("literal input"), 0600); err != nil {
			t.Fatal(err)
		}
		input, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer input.Close()
		old := os.Stdin
		os.Stdin = input
		defer func() { os.Stdin = old }()
		h.run("api", "POST", "/example", "--file", "-")
		offset, err := input.Seek(0, io.SeekCurrent)
		if err != nil || offset == 0 || len(h.requests) != 1 || h.requests[0].body != "literal input" {
			t.Fatalf("stdin observer insensitive: offset=%d requests=%+v err=%v", offset, h.requests, err)
		}
		before := discoverySnapshot(t, h.root)
		if code := h.run("profiles", "get", "example-1", "--out", filepath.Join(h.root, "new-output")); code != 0 {
			t.Fatal(h.errOut.String())
		}
		if reflect.DeepEqual(before, discoverySnapshot(t, h.root)) {
			t.Fatal("output observer insensitive")
		}
	})
}

func TestDiscoveryHelpMatrix(t *testing.T) {
	inv := discoveryLoad(t)
	jsonControls := discoveryJSONControls()
	for _, row := range inv.rows {
		for _, flag := range []string{"--help", "-h"} {
			selectors := []string{""}
			if row.selectors != "" {
				selectors = append(selectors, row.selectors)
			}
			for _, selector := range selectors {
				name := row.path + "/" + flag + "/omitted"
				if selector != "" {
					name = row.path + "/" + flag + "/supplied"
				}
				t.Run(name, func(t *testing.T) {
					h := discoveryNew(t)
					before := discoverySnapshot(t, h.root)
					args := append(strings.Fields(row.path+" "+selector), flag)
					code := h.run(args...)
					h.inert(t, before)
					if code != 0 || h.errOut.Len() != 0 {
						t.Fatalf("argv=%q code=%d stdout=%q stderr=%q (baseline %s/%s)", args, code, h.out.String(), h.errOut.String(), row.longBaseline, row.shortBaseline)
					}
					discoveryCheckHelp(t, h.out.String(), row, inv)
					if want, ok := jsonControls[row.path]; ok {
						for _, fragment := range []string{"Input:", "Examples:", want} {
							if !strings.Contains(h.out.String(), fragment) {
								t.Errorf("JSON help lost %q", fragment)
							}
						}
					}
				})
			}
		}
	}
}
