package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestManifestsHelpAndValidation(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
		wantErr  string
	}{
		{name: "top help", args: []string{"--help"}, wantCode: 0, wantOut: "manifests"},
		{name: "family help", args: []string{"manifests", "--help"}, wantCode: 0, wantOut: "diff"},
		{name: "unknown", args: []string{"manifests", "bad"}, wantCode: 1, wantErr: "unknown manifests command"},
		{name: "missing input", args: []string{"manifests", "diff"}, wantCode: 1, wantErr: "at least one manifest"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, out, errOut := newTestApp(nil)
			code := app.Run(context.Background(), tt.args)
			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr=%q", code, tt.wantCode, errOut.String())
			}
			if tt.wantOut != "" && !strings.Contains(out.String(), tt.wantOut) {
				t.Fatalf("stdout missing %q:\n%s", tt.wantOut, out.String())
			}
			if tt.wantErr != "" && !strings.Contains(errOut.String(), tt.wantErr) {
				t.Fatalf("stderr missing %q:\n%s", tt.wantErr, errOut.String())
			}
		})
	}
}

func TestManifestsHelpIndentation(t *testing.T) {
	app, out, errOut := newTestApp(nil)

	code := app.Run(context.Background(), []string{"manifests", "--help"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
	assertHelpHasNoTabs(t, out.String())
	for _, want := range []string{
		"  oictl manifests sync --scope knowledge|models|prompts|tools|skills|functions|groups|channels|all",
		"  sync   Reconcile a scoped remote set to manifests and prune omitted resources after confirmation",
		"  --scope string     Sync scope: knowledge, models, prompts, tools, skills, functions, groups, channels, or all",
		"  --yes              Confirm destructive sync delete actions",
		"  --confirm          Alias for --yes",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("manifests help missing %q:\n%s", want, out.String())
		}
	}
}

func TestManifestLoadValidation(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, filepath.Join(dir, "b.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"b"},"spec":{"command":"b","content":"hi"}}`)
	writeManifest(t, filepath.Join(dir, "a.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"Knowledge","metadata":{"name":"a"},"spec":{"name":"a"},"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"},{"principal_type":"user","principal_id":"*","permission":"read"}]}`)

	docs, err := loadManifestDocuments(manifestOptions{directories: []string{dir}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load manifests: %v", err)
	}
	if len(docs) != 2 || docs[0].Metadata.Name != "a" || len(docs[0].AccessGrants) != 1 {
		t.Fatalf("docs = %+v", docs)
	}

	writeManifest(t, filepath.Join(dir, "bad.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"Widget","metadata":{"name":"x"},"spec":{}}`)
	if _, err := loadManifestDocuments(manifestOptions{files: []string{filepath.Join(dir, "bad.json")}}, manifestHandlers()); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported kind error = %v", err)
	}
}

func TestManifestYAMLFilesAndDirectoryDiscovery(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "a.json")
	yamlPath := filepath.Join(dir, "b.yaml")
	ymlPath := filepath.Join(dir, "c.yml")
	ignoredPath := filepath.Join(dir, "d.txt")
	writeManifest(t, jsonPath, `{"apiVersion":"oictl.openwebui/v1","kind":"Knowledge","metadata":{"name":"a"},"spec":{"name":"a"}}`)
	writeManifest(t, yamlPath, "apiVersion: oictl.openwebui/v1\nkind: Prompt\nmetadata:\n  name: b\nspec:\n  command: b\n  content: hi\naccess_grants: []\n")
	writeManifest(t, ymlPath, "apiVersion: oictl.openwebui/v1\nkind: Tool\nmetadata:\n  name: c\nspec:\n  name: c\n  content: |\n    class Tools:\n        pass\n")
	writeManifest(t, ignoredPath, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"ignored"},"spec":{"command":"ignored"}}`)

	explicit, err := loadManifestDocuments(manifestOptions{files: []string{yamlPath, ymlPath}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load explicit yaml manifests: %v", err)
	}
	if len(explicit) != 2 || explicit[0].Metadata.Name != "b" || !explicit[0].HasAccessGrants || explicit[1].Metadata.Name != "c" {
		t.Fatalf("explicit docs = %+v", explicit)
	}

	docs, err := loadManifestDocuments(manifestOptions{directories: []string{dir}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load manifest directory: %v", err)
	}
	if len(docs) != 3 || docs[0].Metadata.Name != "a" || docs[1].Metadata.Name != "b" || docs[2].Metadata.Name != "c" {
		t.Fatalf("directory docs = %+v", docs)
	}

	compatPath := filepath.Join(dir, "compat.manifest")
	writeManifest(t, compatPath, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"compat"},"spec":{"command":"compat","content":"hi"}}`)
	if _, err := loadManifestDocuments(manifestOptions{files: []string{compatPath}}, manifestHandlers()); err != nil {
		t.Fatalf("explicit non-yaml JSON manifest: %v", err)
	}
}

func TestManifestDirectoryDuplicateDetectionIncludesYAML(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.json")
	second := filepath.Join(dir, "b.yaml")
	writeManifest(t, first, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"dup"},"spec":{"command":"dup","content":"hi"}}`)
	writeManifest(t, second, "apiVersion: oictl.openwebui/v1\nkind: Prompt\nmetadata:\n  name: dup\nspec:\n  command: dup\n  content: hi\n")

	_, err := loadManifestDocuments(manifestOptions{directories: []string{dir}}, manifestHandlers())
	if err == nil || !strings.Contains(err.Error(), "duplicate manifest identity Prompt/dup") || !strings.Contains(err.Error(), first) || !strings.Contains(err.Error(), second) {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestManifestEnvironmentRendering(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.json")
	t.Setenv("OICTL_TEST_COMMAND", "hello")
	t.Setenv("OICTL_TEST_GROUP", "SRE")
	writeManifest(t, path, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"${OICTL_TEST_COMMAND}"},"spec":{"command":"${OICTL_TEST_COMMAND}","content":"page ${OICTL_TEST_GROUP}"},"access_grants":[{"principal_type":"group","principal_name":"${OICTL_TEST_GROUP}","permission":"read"}]}`)

	docs, err := loadManifestDocuments(manifestOptions{files: []string{path}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if docs[0].Metadata.Name != "hello" || docs[0].Spec["command"] != "hello" || docs[0].Spec["content"] != "page SRE" {
		t.Fatalf("rendered doc = %#v", docs[0])
	}
	if len(docs[0].AccessGrants) != 1 || docs[0].AccessGrants[0].PrincipalName != "SRE" {
		t.Fatalf("rendered grants = %#v", docs[0].AccessGrants)
	}
}

func TestManifestYAMLEnvironmentRendering(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.yaml")
	t.Setenv("OICTL_TEST_COMMAND", "hello")
	t.Setenv("OICTL_TEST_GROUP", "SRE")
	writeManifest(t, path, "apiVersion: oictl.openwebui/v1\nkind: Prompt\nmetadata:\n  name: ${OICTL_TEST_COMMAND}\nspec:\n  command: ${OICTL_TEST_COMMAND}\n  content: page ${OICTL_TEST_GROUP}\naccess_grants:\n  - principal_type: group\n    principal_name: ${OICTL_TEST_GROUP}\n    permission: read\n")

	docs, err := loadManifestDocuments(manifestOptions{files: []string{path}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if docs[0].Metadata.Name != "hello" || docs[0].Spec["command"] != "hello" || docs[0].Spec["content"] != "page SRE" {
		t.Fatalf("rendered doc = %#v", docs[0])
	}
	if len(docs[0].AccessGrants) != 1 || docs[0].AccessGrants[0].PrincipalName != "SRE" {
		t.Fatalf("rendered grants = %#v", docs[0].AccessGrants)
	}
}

func TestManifestMissingEnvironmentVariableFailsBeforeRequests(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.json")
	writeManifest(t, path, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"x"},"spec":{"command":"${OICTL_MISSING_VALUE}","content":"hi"},"access_grants":[]}`)

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})

	code := app.Run(context.Background(), []string{"manifests", "diff", "--file", path})
	if code == 0 || requests != 0 {
		t.Fatalf("code=%d requests=%d stderr=%q", code, requests, errOut.String())
	}
	if !strings.Contains(errOut.String(), path) || !strings.Contains(errOut.String(), "OICTL_MISSING_VALUE") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestManifestYAMLMissingEnvironmentVariableFailsBeforeRequests(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.yaml")
	writeManifest(t, path, "apiVersion: oictl.openwebui/v1\nkind: Prompt\nmetadata:\n  name: x\nspec:\n  command: ${OICTL_MISSING_VALUE}\n  content: hi\naccess_grants: []\n")

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})

	code := app.Run(context.Background(), []string{"manifests", "diff", "--file", path})
	if code == 0 || requests != 0 {
		t.Fatalf("code=%d requests=%d stderr=%q", code, requests, errOut.String())
	}
	if !strings.Contains(errOut.String(), path) || !strings.Contains(errOut.String(), "OICTL_MISSING_VALUE") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestManifestUnsupportedEnvironmentPlaceholderFailsBeforeRequests(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.json")
	writeManifest(t, path, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"x"},"spec":{"command":"x","content":"${OICTL_TEAM_NAME:-sre}"},"access_grants":[]}`)

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})

	code := app.Run(context.Background(), []string{"manifests", "diff", "--file", path})
	if code == 0 || requests != 0 {
		t.Fatalf("code=%d requests=%d stderr=%q", code, requests, errOut.String())
	}
	for _, want := range []string{path, "unsupported environment placeholder", "OICTL_TEAM_NAME:-sre"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr missing %q: %q", want, errOut.String())
		}
	}
	if strings.Contains(errOut.String(), "references unset environment variable") {
		t.Fatalf("stderr used unset-variable path for unsupported placeholder: %q", errOut.String())
	}
}

func TestManifestYAMLUnsupportedEnvironmentPlaceholderFailsBeforeRequests(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.yaml")
	writeManifest(t, path, "apiVersion: oictl.openwebui/v1\nkind: Prompt\nmetadata:\n  name: x\nspec:\n  command: x\n  content: ${OICTL_TEAM_NAME:-sre}\naccess_grants: []\n")

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})

	code := app.Run(context.Background(), []string{"manifests", "diff", "--file", path})
	if code == 0 || requests != 0 {
		t.Fatalf("code=%d requests=%d stderr=%q", code, requests, errOut.String())
	}
	for _, want := range []string{path, "unsupported environment placeholder", "OICTL_TEAM_NAME:-sre"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr missing %q: %q", want, errOut.String())
		}
	}
	if strings.Contains(errOut.String(), "references unset environment variable") {
		t.Fatalf("stderr used unset-variable path for unsupported placeholder: %q", errOut.String())
	}
}

func TestManifestParseErrorsIncludeSourcePath(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		file string
		body string
		want string
	}{
		{name: "yaml body with json extension", file: "prompt.json", body: "apiVersion: oictl.openwebui/v1\nkind: Prompt\nmetadata:\n  name: x\n", want: "parse manifest"},
		{name: "bad yaml body", file: "prompt.yaml", body: "apiVersion: [\n", want: "parse manifest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.file)
			writeManifest(t, path, tt.body)

			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				_, _ = io.WriteString(w, `[]`)
			}))
			defer server.Close()
			app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})

			code := app.Run(context.Background(), []string{"manifests", "diff", "--file", path})
			if code == 0 || requests != 0 {
				t.Fatalf("code=%d requests=%d stderr=%q", code, requests, errOut.String())
			}
			if !strings.Contains(errOut.String(), path) || !strings.Contains(errOut.String(), tt.want) {
				t.Fatalf("stderr = %q", errOut.String())
			}
		})
	}
}

func TestManifestContentFileResolution(t *testing.T) {
	dir := t.TempDir()
	body := filepath.Join(dir, "body.py")
	writeManifest(t, body, "class Tools:\n    pass\n")
	relative := filepath.Join(dir, "tool-relative.json")
	writeManifest(t, relative, `{"apiVersion":"oictl.openwebui/v1","kind":"Tool","metadata":{"name":"relative"},"spec":{"name":"relative","content_file":"body.py"},"access_grants":[]}`)

	docs, err := loadManifestDocuments(manifestOptions{files: []string{relative}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load relative content_file: %v", err)
	}
	if docs[0].Spec["content"] != "class Tools:\n    pass\n" {
		t.Fatalf("content = %#v", docs[0].Spec["content"])
	}
	if _, ok := docs[0].Spec["content_file"]; ok {
		t.Fatalf("content_file leaked into spec: %#v", docs[0].Spec)
	}

	t.Setenv("OICTL_CONTENT_DIR", dir)
	envPath := filepath.Join(dir, "tool-env.json")
	writeManifest(t, envPath, `{"apiVersion":"oictl.openwebui/v1","kind":"Tool","metadata":{"name":"env"},"spec":{"name":"env","content_file":"${OICTL_CONTENT_DIR}/body.py"},"access_grants":[]}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tools/list" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	envDocs, err := loadManifestDocuments(manifestOptions{files: []string{envPath}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load env content_file: %v", err)
	}
	plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, envDocs, manifestHandlers(), nil)
	if err != nil {
		t.Fatalf("build plan: %v; stderr=%s", err, errOut.String())
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Desired.Spec["content"] != "class Tools:\n    pass\n" {
		t.Fatalf("plan = %#v", plan.Actions)
	}
	if _, ok := plan.Actions[0].Desired.Spec["content_file"]; ok {
		t.Fatalf("content_file leaked into planned desired spec: %#v", plan.Actions[0].Desired.Spec)
	}
}

func TestManifestContentFileValidationFailsBeforeRequests(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "ambiguous", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Tool","metadata":{"name":"x"},"spec":{"name":"x","content":"inline","content_file":"body.py"},"access_grants":[]}`, want: "both spec.content and spec.content_file"},
		{name: "unreadable", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Tool","metadata":{"name":"x"},"spec":{"name":"x","content_file":"missing.py"},"access_grants":[]}`, want: "missing.py"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".json")
			writeManifest(t, path, tt.body)
			app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
			code := app.Run(context.Background(), []string{"manifests", "diff", "--file", path})
			if code == 0 || !strings.Contains(errOut.String(), path) || !strings.Contains(errOut.String(), tt.want) {
				t.Fatalf("code=%d stderr=%q", code, errOut.String())
			}
		})
	}
}

func TestAdditionalManifestKindsValidationAndScopes(t *testing.T) {
	dir := t.TempDir()
	valid := map[string]string{
		"channel":        `{"apiVersion":"oictl.openwebui/v1","kind":"Channel","metadata":{"name":"ops"},"spec":{"name":"Ops"},"access_grants":[]}`,
		"function":       `{"apiVersion":"oictl.openwebui/v1","kind":"Function","metadata":{"name":"fn_a"},"spec":{"id":"fn_a","name":"Fn","content":"print(1)"}}`,
		"function-valve": `{"apiVersion":"oictl.openwebui/v1","kind":"FunctionValve","metadata":{"name":"fn_a"},"spec":{"level":2}}`,
		"group":          `{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"group-a"},"spec":{"id":"group-a","name":"Group A"}}`,
		"skill":          `{"apiVersion":"oictl.openwebui/v1","kind":"Skill","metadata":{"name":"skill-a"},"spec":{"id":"skill-a","name":"Skill","content":"Do work"},"access_grants":[]}`,
		"terminal":       `{"apiVersion":"oictl.openwebui/v1","kind":"TerminalServerConnection","metadata":{"name":"shell-a"},"spec":{"url":"https://terminal"},"access_grants":[]}`,
		"tool-valve":     `{"apiVersion":"oictl.openwebui/v1","kind":"ToolValve","metadata":{"name":"weather_tool"},"spec":{"enabled":true}}`,
	}
	for name, body := range valid {
		path := filepath.Join(dir, name+".json")
		writeManifest(t, path, body)
		if _, err := loadManifestDocuments(manifestOptions{files: []string{path}}, manifestHandlers()); err != nil {
			t.Fatalf("load %s manifest: %v", name, err)
		}
	}

	for _, kind := range []string{"Function", "FunctionValve", "Group", "ToolValve"} {
		path := filepath.Join(dir, strings.ToLower(kind)+"-grants.json")
		writeManifest(t, path, `{"apiVersion":"oictl.openwebui/v1","kind":"`+kind+`","metadata":{"name":"x"},"spec":{"id":"x"},"access_grants":[]}`)
		if _, err := loadManifestDocuments(manifestOptions{files: []string{path}}, manifestHandlers()); err == nil || !strings.Contains(err.Error(), "does not support access_grants") {
			t.Fatalf("%s access_grants error = %v", kind, err)
		}
	}

	for _, kind := range []string{"ToolUserValve", "FunctionUserValve"} {
		path := filepath.Join(dir, strings.ToLower(kind)+".json")
		writeManifest(t, path, `{"apiVersion":"oictl.openwebui/v1","kind":"`+kind+`","metadata":{"name":"x"},"spec":{"enabled":true}}`)
		if _, err := loadManifestDocuments(manifestOptions{files: []string{path}}, manifestHandlers()); err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("%s unsupported error = %v", kind, err)
		}
	}

	first := filepath.Join(dir, "channel-dup-1.json")
	second := filepath.Join(dir, "channel-dup-2.json")
	writeManifest(t, first, `{"apiVersion":"oictl.openwebui/v1","kind":"Channel","metadata":{"name":"dup"},"spec":{"name":"Dup"}}`)
	writeManifest(t, second, `{"apiVersion":"oictl.openwebui/v1","kind":"Channel","metadata":{"name":"dup"},"spec":{"name":"Dup"}}`)
	if _, err := loadManifestDocuments(manifestOptions{files: []string{first, second}}, manifestHandlers()); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate channel error = %v", err)
	}

	for scope, want := range map[string]string{"channel": "Channel", "channels": "Channel", "function": "Function", "functions": "Function", "group": "Group", "groups": "Group", "skill": "Skill", "skills": "Skill", "model": "Model", "models": "Model"} {
		kinds, err := syncScopeKinds("sync", scope)
		if err != nil || !kinds[want] || len(kinds) != 1 {
			t.Fatalf("scope %q = %#v, err=%v", scope, kinds, err)
		}
	}
	all, err := syncScopeKinds("sync", "all")
	if err != nil {
		t.Fatalf("all scope: %v", err)
	}
	for _, want := range []string{"Knowledge", "Prompt", "Tool", "Model", "Skill", "Function", "Group", "Channel"} {
		if !all[want] {
			t.Fatalf("all scope missing %s in %#v", want, all)
		}
	}
	if all["TerminalServerConnection"] {
		t.Fatalf("all scope must not prune terminal server connections: %#v", all)
	}
	if all["ToolValve"] || all["FunctionValve"] {
		t.Fatalf("all scope must not prune valve manifests: %#v", all)
	}
	for _, scope := range []string{"tool-valves", "function-valves"} {
		if _, err := syncScopeKinds("sync", scope); err == nil || !strings.Contains(err.Error(), "unsupported sync scope") {
			t.Fatalf("scope %q error = %v", scope, err)
		}
	}
}

func TestGlobalValveManifestPlanning(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/tools/id/weather_tool/valves":
			_, _ = io.WriteString(w, `null`)
		case "/api/v1/tools/id/same_tool/valves":
			_, _ = io.WriteString(w, `{"enabled":true}`)
		case "/api/v1/functions/id/fn_a/valves":
			_, _ = io.WriteString(w, `{"level":1,"enabled":false}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	docs := []manifestDocument{
		manifestDoc("ToolValve", "weather_tool", map[string]any{"enabled": true}, nil, false),
		manifestDoc("FunctionValve", "fn_a", map[string]any{"level": float64(2), "enabled": false}, nil, false),
		manifestDoc("ToolValve", "same_tool", map[string]any{"enabled": true}, nil, false),
	}
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
	if err != nil {
		t.Fatalf("build valve plan: %v; stderr=%s", err, errOut.String())
	}
	actions := map[string]planAction{}
	for _, action := range plan.Actions {
		actions[action.Action+":"+action.Kind+":"+action.Name] = action
	}
	if _, ok := actions["create:ToolValve:weather_tool"]; !ok {
		t.Fatalf("missing create action in %+v", plan.Actions)
	}
	update, ok := actions["update:FunctionValve:fn_a"]
	if !ok || len(update.Changes) != 1 || update.Changes[0].Field != "level" {
		t.Fatalf("function update action = %+v", update)
	}
	if _, ok := actions["unchanged:ToolValve:same_tool"]; !ok {
		t.Fatalf("missing unchanged action in %+v", plan.Actions)
	}
	for _, request := range requests {
		if strings.Contains(request, "/valves/user") {
			t.Fatalf("user-scoped valve endpoint was called: %v", requests)
		}
	}
}

func TestModelManifestLoadValidation(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "model.json")
	writeManifest(t, valid, `{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"llama-ops"},"spec":{"name":"Llama Ops"}}`)

	docs, err := loadManifestDocuments(manifestOptions{files: []string{valid}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load model manifest: %v", err)
	}
	if got := docs[0].Spec["id"]; got != "llama-ops" {
		t.Fatalf("spec.id = %v, want metadata name", got)
	}
	if _, ok := docs[0].Spec["meta"].(map[string]any); !ok {
		t.Fatalf("spec.meta = %#v, want object", docs[0].Spec["meta"])
	}
	if _, ok := docs[0].Spec["params"].(map[string]any); !ok {
		t.Fatalf("spec.params = %#v, want object", docs[0].Spec["params"])
	}

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "identity mismatch", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"llama-ops"},"spec":{"id":"other","name":"Llama Ops"}}`, want: "identity is inconsistent"},
		{name: "missing name", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"llama-ops"},"spec":{}}`, want: "spec.name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".json")
			writeManifest(t, path, tt.body)
			_, err := loadManifestDocuments(manifestOptions{files: []string{path}}, manifestHandlers())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestManifestRejectsInvalidEnvelopeDuplicateIdentityAndInvalidGrants(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "missing apiVersion", body: `{"kind":"Prompt","metadata":{"name":"x"},"spec":{}}`, want: "apiVersion"},
		{name: "missing metadata name", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{},"spec":{}}`, want: "metadata.name"},
		{name: "missing spec", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"x"}}`, want: "spec"},
		{name: "bad principal", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"x"},"spec":{},"access_grants":[{"principal_type":"role","principal_id":"admin","permission":"read"}]}`, want: "user and group"},
		{name: "bad permission", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"x"},"spec":{},"access_grants":[{"principal_type":"user","principal_id":"u","permission":"admin"}]}`, want: "read and write"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".json")
			writeManifest(t, path, tt.body)
			_, err := loadManifestDocuments(manifestOptions{files: []string{path}}, manifestHandlers())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}

	first := filepath.Join(dir, "dup1.json")
	second := filepath.Join(dir, "dup2.json")
	writeManifest(t, first, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"dup"},"spec":{}}`)
	writeManifest(t, second, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"dup"},"spec":{}}`)
	_, err := loadManifestDocuments(manifestOptions{files: []string{first, second}}, manifestHandlers())
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestManifestAccessGrantSelectorsValidateAndResolve(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "selectors.json")
	writeManifest(t, valid, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"x"},"spec":{"command":"x"},"access_grants":[{"principal_type":"user","principal_email":"user@example.com","permission":"read"},{"principal_type":"user","principal_name":"User One","permission":"read"},{"principal_type":"group","principal_name":"SRE","permission":"write"},{"principal_type":"group","principal_ref":"group:ops","permission":"read"}]}`)
	docs, err := loadManifestDocuments(manifestOptions{files: []string{valid}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load selectors: %v", err)
	}

	userRequests := 0
	groupRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/users/":
			userRequests++
			if r.URL.Query().Get("query") != "" {
				t.Fatalf("unexpected filtered user inventory %s", r.URL)
			}
			_, _ = io.WriteString(w, `{"users":[{"id":"u-email","email":"user@example.com","name":"User Email"},{"id":"u-name","email":"one@example.com","name":"User One"}],"total":2}`)
		case "/api/v1/groups/":
			groupRequests++
			_, _ = io.WriteString(w, `[{"id":"g-sre","name":"SRE"},{"id":"ops","name":"Ops"}]`)
		default:
			t.Fatalf("unexpected request %s", r.URL.String())
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	resolver := principalResolver{app: app, opts: globalOptions{}, userCache: map[string][]principalUser{}}
	if err := resolver.resolveDocuments(context.Background(), docs); err != nil {
		t.Fatalf("resolve selectors: %v; stderr=%s", err, errOut.String())
	}
	ids := map[string]bool{}
	for _, grant := range docs[0].AccessGrants {
		ids[grant.PrincipalID] = true
		if grant.PrincipalEmail != "" || grant.PrincipalName != "" || grant.PrincipalRef != "" {
			t.Fatalf("authoring fields were not cleared: %+v", grant)
		}
	}
	for _, want := range []string{"u-email", "u-name", "g-sre", "ops"} {
		if !ids[want] {
			t.Fatalf("missing resolved id %s in %+v", want, docs[0].AccessGrants)
		}
	}
	if userRequests != 1 || groupRequests != 1 {
		t.Fatalf("resolver requests users=%d groups=%d", userRequests, groupRequests)
	}

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "multiple selectors", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"x"},"spec":{},"access_grants":[{"principal_type":"user","principal_id":"u1","principal_email":"u@example.com","permission":"read"}]}`, want: "only one principal selector"},
		{name: "group email", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"x"},"spec":{},"access_grants":[{"principal_type":"group","principal_email":"team@example.com","permission":"read"}]}`, want: "principal_email"},
		{name: "ref mismatch", body: `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"x"},"spec":{},"access_grants":[{"principal_type":"user","principal_ref":"group:SRE","permission":"read"}]}`, want: "does not match"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".json")
			writeManifest(t, path, tt.body)
			_, err := loadManifestDocuments(manifestOptions{files: []string{path}}, manifestHandlers())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestManifestPrincipalResolutionFailuresAndRawIDSkipLookups(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	writeManifest(t, missing, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"missing"},"spec":{},"access_grants":[{"principal_type":"user","principal_email":"missing@example.com","permission":"read"}]}`)
	ambiguous := filepath.Join(dir, "ambiguous.json")
	writeManifest(t, ambiguous, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"ambiguous"},"spec":{},"access_grants":[{"principal_type":"group","principal_name":"SRE","permission":"read"}]}`)
	raw := filepath.Join(dir, "raw.json")
	writeManifest(t, raw, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"raw"},"spec":{},"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"},{"principal_type":"group","principal_id":"ops","permission":"write"}]}`)

	lookupRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/users/":
			lookupRequests++
			_, _ = io.WriteString(w, `[]`)
		case "/api/v1/groups/":
			lookupRequests++
			_, _ = io.WriteString(w, `[{"id":"g1","name":"SRE"},{"id":"g2","name":"SRE"}]`)
		default:
			t.Fatalf("unexpected request %s", r.URL.String())
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	missingDocs, err := loadManifestDocuments(manifestOptions{files: []string{missing}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	resolver := principalResolver{app: app, opts: globalOptions{}, userCache: map[string][]principalUser{}}
	if err := resolver.resolveDocuments(context.Background(), missingDocs); err == nil || !strings.Contains(err.Error(), "could not be resolved") {
		t.Fatalf("missing error = %v; stderr=%s", err, errOut.String())
	}

	ambiguousDocs, err := loadManifestDocuments(manifestOptions{files: []string{ambiguous}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load ambiguous: %v", err)
	}
	resolver = principalResolver{app: app, opts: globalOptions{}, userCache: map[string][]principalUser{}}
	if err := resolver.resolveDocuments(context.Background(), ambiguousDocs); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous error = %v", err)
	}

	rawDocs, err := loadManifestDocuments(manifestOptions{files: []string{raw}}, manifestHandlers())
	if err != nil {
		t.Fatalf("load raw: %v", err)
	}
	before := lookupRequests
	resolver = principalResolver{app: app, opts: globalOptions{}, userCache: map[string][]principalUser{}}
	if err := resolver.resolveDocuments(context.Background(), rawDocs); err != nil {
		t.Fatalf("raw resolve: %v", err)
	}
	if lookupRequests != before {
		t.Fatalf("raw IDs triggered resolver lookups: before=%d after=%d", before, lookupRequests)
	}
}

func TestPlannerCreateUpdateGrantReplacementUnchangedDeleteAndJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/prompts/":
			_, _ = io.WriteString(w, `[{"id":"p1","command":"same","content":"ok","access_grants":[{"id":"server","principal_type":"user","principal_id":"*","permission":"read"}]},{"id":"p2","command":"change","content":"old","access_grants":[]},{"id":"p3","command":"grant","content":"ok","access_grants":[{"principal_type":"group","principal_id":"g1","permission":"read"}]},{"id":"p4","command":"remote-only","content":"old"}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	docs := []manifestDocument{
		manifestDoc("Prompt", "new", map[string]any{"command": "new", "content": "ok"}, nil, false),
		manifestDoc("Prompt", "same", map[string]any{"command": "same", "content": "ok"}, []accessGrant{{PrincipalType: "user", PrincipalID: "*", Permission: "read"}}, true),
		manifestDoc("Prompt", "change", map[string]any{"command": "change", "content": "new"}, nil, false),
		manifestDoc("Prompt", "grant", map[string]any{"command": "grant", "content": "ok"}, []accessGrant{{PrincipalType: "group", PrincipalID: "g2", Permission: "write"}}, true),
	}
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), map[string]bool{"Prompt": true})
	if err != nil {
		t.Fatalf("build plan: %v; stderr=%s", err, errOut.String())
	}
	actions := map[string]bool{}
	for _, action := range plan.Actions {
		actions[action.Action+":"+action.Name] = true
	}
	for _, want := range []string{"create:new", "unchanged:same", "update:change", "replace-grants:grant", "delete:remote-only"} {
		if !actions[want] {
			t.Fatalf("missing action %s in %+v", want, plan.Actions)
		}
	}
	var out strings.Builder
	if err := writePlan(&out, plan, "json"); err != nil {
		t.Fatalf("write json: %v", err)
	}
	var decoded manifestPlan
	if err := json.Unmarshal([]byte(out.String()), &decoded); err != nil || len(decoded.Actions) == 0 {
		t.Fatalf("invalid json plan %q err=%v", out.String(), err)
	}
}

func TestModelManifestPlanningDiffsManagedFieldsAndPrunes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models/list" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = io.WriteString(w, `{"items":[{"id":"llama-ops","base_model_id":"llama3","name":"Llama Ops","meta":{"description":"old"},"params":{"temperature":0.2},"is_active":true,"user_id":"u1","created_at":1,"updated_at":2,"write_access":true,"access_grants":[]},{"id":"same","name":"Same","meta":{},"params":{},"is_active":true,"user_id":"u2","created_at":3,"updated_at":4,"write_access":false}],"total":4}`)
		case "2":
			_, _ = io.WriteString(w, `{"items":[{"id":"grant-model","name":"Grant Model","meta":{},"params":{},"is_active":true,"access_grants":[{"principal_type":"group","principal_id":"ops","permission":"read"}]},{"id":"remote-only","name":"Remote Only","meta":{},"params":{},"is_active":true}],"total":4}`)
		default:
			t.Fatalf("page = %q", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	docs := []manifestDocument{
		manifestDoc("Model", "llama-ops", map[string]any{"id": "llama-ops", "base_model_id": "llama3", "name": "Llama Ops", "meta": map[string]any{"description": "new"}, "params": map[string]any{"temperature": 0.7}, "is_active": false}, nil, false),
		manifestDoc("Model", "same", map[string]any{"id": "same", "name": "Same", "meta": map[string]any{}, "params": map[string]any{}, "is_active": true}, nil, false),
		manifestDoc("Model", "grant-model", map[string]any{"id": "grant-model", "name": "Grant Model", "meta": map[string]any{}, "params": map[string]any{}, "is_active": true}, []accessGrant{{PrincipalType: "user", PrincipalID: "*", Permission: "read"}}, true),
	}
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), map[string]bool{"Model": true})
	if err != nil {
		t.Fatalf("build model plan: %v; stderr=%s", err, errOut.String())
	}

	actions := map[string]planAction{}
	for _, action := range plan.Actions {
		actions[action.Action+":"+action.Name] = action
	}
	update, ok := actions["update:llama-ops"]
	if !ok {
		t.Fatalf("missing update action in %+v", plan.Actions)
	}
	fields := map[string]bool{}
	for _, change := range update.Changes {
		fields[change.Field] = true
	}
	for _, want := range []string{"is_active", "meta", "params"} {
		if !fields[want] {
			t.Fatalf("missing field %s in %+v", want, update.Changes)
		}
	}
	for _, want := range []string{"unchanged:same", "replace-grants:grant-model", "delete:remote-only"} {
		if _, ok := actions[want]; !ok {
			t.Fatalf("missing action %s in %+v", want, plan.Actions)
		}
	}
}

func TestManifestPlanningPreservesOmittedActiveState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models/list" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		_, _ = io.WriteString(w, `{"items":[{"id":"llama-ops","name":"Llama Ops","meta":{},"params":{},"is_active":true}],"total":1}`)
	}))
	defer server.Close()

	docs := []manifestDocument{
		manifestDoc("Model", "llama-ops", map[string]any{"id": "llama-ops", "name": "Llama Ops", "meta": map[string]any{}, "params": map[string]any{}}, nil, false),
	}
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	plan, err := buildManifestPlan(context.Background(), app, globalOptions{}, docs, manifestHandlers(), nil)
	if err != nil {
		t.Fatalf("build plan: %v; stderr=%s", err, errOut.String())
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Action != "unchanged" {
		t.Fatalf("plan actions = %#v", plan.Actions)
	}
}

func TestManifestsDiffApplyDryRunSyncConfirmationAndHTTP(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "prompt.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"hello"},"spec":{"command":"hello","content":"hi"},"access_grants":[]}`)

	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/prompts/":
			_, _ = io.WriteString(w, `[{"id":"old","command":"old","content":"old"}]`)
		case "/api/v1/prompts/create":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"id":"new","command":"hello","content":"hi","access_grants":[]}`)
		case "/api/v1/prompts/id/new":
			_, _ = io.WriteString(w, `{"id":"new","command":"hello","content":"hi","access_grants":[]}`)
		case "/api/v1/prompts/id/new/access/update":
			_, _ = io.WriteString(w, `{"id":"new","command":"hello","content":"hi","access_grants":[]}`)
		case "/api/v1/prompts/id/old/delete":
			_, _ = io.WriteString(w, `true`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "diff", "--file", manifest}); code != 0 {
		t.Fatalf("diff exit code = %d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "CREATE Prompt/hello") || containsRequest(requests, "POST ") {
		t.Fatalf("diff output=%q requests=%v", out.String(), requests)
	}

	out.Reset()
	errOut.Reset()
	requests = nil
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--dry-run", "--file", manifest}); code != 0 {
		t.Fatalf("dry-run exit code = %d; stderr=%q", code, errOut.String())
	}
	if containsRequest(requests, "POST ") || containsRequest(requests, "DELETE ") {
		t.Fatalf("dry-run mutated: %v", requests)
	}

	out.Reset()
	errOut.Reset()
	requests = nil
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", manifest}); code != 0 {
		t.Fatalf("apply exit code = %d; stderr=%q", code, errOut.String())
	}
	if !containsRequest(requests, "POST /api/v1/prompts/create") || containsRequest(requests, "DELETE ") {
		t.Fatalf("apply requests=%v", requests)
	}

	out.Reset()
	errOut.Reset()
	requests = nil
	code := app.Run(context.Background(), []string{"manifests", "sync", "--scope", "prompts", "--file", manifest})
	errText := errOut.String()
	if code == 0 {
		t.Fatalf("sync confirmation code=%d stderr=%q", code, errText)
	}
	if !strings.Contains(errText, "destructive sync actions") {
		t.Fatalf("sync confirmation stderr=%q", errText)
	}

	out.Reset()
	errOut.Reset()
	requests = nil
	if code := app.Run(context.Background(), []string{"manifests", "sync", "--scope", "prompts", "--yes", "--file", manifest}); code != 0 {
		t.Fatalf("sync exit code = %d; stderr=%q", code, errOut.String())
	}
	if !containsRequest(requests, "DELETE /api/v1/prompts/id/old/delete") {
		t.Fatalf("sync requests=%v", requests)
	}
}

func TestGlobalValveManifestApplyDryRunHTTPAndErrors(t *testing.T) {
	dir := t.TempDir()
	toolManifest := filepath.Join(dir, "tool-valve.json")
	functionManifest := filepath.Join(dir, "function-valve.json")
	writeManifest(t, toolManifest, `{"apiVersion":"oictl.openwebui/v1","kind":"ToolValve","metadata":{"name":"weather_tool"},"spec":{"enabled":true,"api_key":"secret"}}`)
	writeManifest(t, functionManifest, `{"apiVersion":"oictl.openwebui/v1","kind":"FunctionValve","metadata":{"name":"fn_a"},"spec":{"level":2}}`)

	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer sk" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/v1/tools/list":
			_, _ = io.WriteString(w, `[{"id":"weather_tool","name":"weather_tool"}]`)
		case "/api/v1/functions/list":
			_, _ = io.WriteString(w, `[{"id":"fn_a","name":"fn_a"}]`)
		case "/api/v1/tools/id/weather_tool/valves":
			_, _ = io.WriteString(w, `null`)
		case "/api/v1/functions/id/fn_a/valves":
			_, _ = io.WriteString(w, `{"level":1}`)
		case "/api/v1/tools/id/weather_tool/valves/update":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode tool valve body: %v", err)
			}
			if body["enabled"] != true || body["api_key"] != "secret" {
				t.Fatalf("tool valve update body = %#v", body)
			}
			_, _ = io.WriteString(w, `{"enabled":true,"api_key":"secret","normalized":true}`)
		case "/api/v1/functions/id/fn_a/valves/update":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode function valve body: %v", err)
			}
			if body["level"] != float64(2) {
				t.Fatalf("function valve update body = %#v", body)
			}
			_, _ = io.WriteString(w, `{"level":2}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--dry-run", "--directory", dir}); code != 0 {
		t.Fatalf("dry-run code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "CREATE ToolValve/weather_tool") || !strings.Contains(out.String(), "UPDATE FunctionValve/fn_a") || containsRequest(requests, "POST ") {
		t.Fatalf("dry-run output=%q requests=%v", out.String(), requests)
	}

	out.Reset()
	errOut.Reset()
	requests = nil
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code != 0 {
		t.Fatalf("apply code=%d stderr=%q", code, errOut.String())
	}
	for _, want := range []string{"POST /api/v1/tools/id/weather_tool/valves/update", "POST /api/v1/functions/id/fn_a/valves/update"} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}
	for _, request := range requests {
		if strings.Contains(request, "/valves/user") {
			t.Fatalf("user-scoped valve endpoint was called: %v", requests)
		}
	}

	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tools/list":
			_, _ = io.WriteString(w, `[{"id":"weather_tool","name":"weather_tool"}]`)
		case "/api/v1/tools/id/weather_tool/valves":
			_, _ = io.WriteString(w, `null`)
		case "/api/v1/tools/id/weather_tool/valves/update":
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = io.WriteString(w, `{"detail":"invalid valve"}`)
		default:
			t.Fatalf("unexpected error request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer errorServer.Close()
	errorDir := t.TempDir()
	errorManifest := filepath.Join(errorDir, "tool-valve.json")
	writeManifest(t, errorManifest, `{"apiVersion":"oictl.openwebui/v1","kind":"ToolValve","metadata":{"name":"weather_tool"},"spec":{"enabled":true}}`)
	errorApp, _, errorErr := newTestApp(map[string]string{"OPEN_WEBUI_URL": errorServer.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := errorApp.Run(context.Background(), []string{"manifests", "apply", "--file", errorManifest}); code == 0 || !strings.Contains(errorErr.String(), "422 Unprocessable Entity") || !strings.Contains(errorErr.String(), "invalid valve") {
		t.Fatalf("error code=%d stderr=%q", code, errorErr.String())
	}
}

func TestManifestWorkflowsResolvePrincipalSelectors(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "prompt.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"hello"},"spec":{"command":"hello","content":"hi"},"access_grants":[{"principal_type":"user","principal_email":"user@example.com","permission":"read"}]}`)

	run := func(args []string, remoteGrants string) ([]string, string, string, int) {
		t.Helper()
		requests := []string{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Method+" "+r.URL.String())
			switch r.URL.Path {
			case "/api/v1/users/":
				if r.URL.Query().Get("query") != "" {
					t.Fatalf("user query = %q", r.URL.Query().Get("query"))
				}
				_, _ = io.WriteString(w, `[{"id":"u1","email":"user@example.com","name":"User One"}]`)
			case "/api/v1/prompts/":
				_, _ = io.WriteString(w, `[{"id":"p1","command":"hello","content":"hi","access_grants":`+remoteGrants+`},{"id":"old","command":"old","content":"old","access_grants":[]}]`)
			case "/api/v1/prompts/id/p1/access/update":
				var body map[string][]accessGrant
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode access body: %v", err)
				}
				grants := body["access_grants"]
				if len(grants) != 1 || grants[0].PrincipalID != "u1" || grants[0].PrincipalEmail != "" || grants[0].PrincipalName != "" || grants[0].PrincipalRef != "" {
					t.Fatalf("access grants body = %+v", grants)
				}
				_, _ = io.WriteString(w, `{"id":"p1","command":"hello","content":"hi","access_grants":[{"principal_type":"user","principal_id":"u1","permission":"read"}]}`)
			case "/api/v1/prompts/id/p1":
				_, _ = io.WriteString(w, `{"id":"p1","command":"hello","content":"hi","access_grants":[{"principal_type":"user","principal_id":"u1","permission":"read"}]}`)
			case "/api/v1/prompts/id/old/delete":
				_, _ = io.WriteString(w, `true`)
			default:
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
			}
		}))
		defer server.Close()

		app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
		fullArgs := append([]string{}, args...)
		for i, arg := range fullArgs {
			if arg == "SERVER_MANIFEST" {
				fullArgs[i] = manifest
			}
		}
		code := app.Run(context.Background(), fullArgs)
		return requests, out.String(), errOut.String(), code
	}

	requests, out, errText, code := run([]string{"manifests", "diff", "--file", "SERVER_MANIFEST", "--output", "json"}, `[{"principal_type":"user","principal_id":"u1","permission":"read"}]`)
	if code != 0 || !containsRequest(requests, "GET /api/v1/users/") || !strings.Contains(out, `"action": "unchanged"`) {
		t.Fatalf("diff code=%d out=%q err=%q requests=%v", code, out, errText, requests)
	}

	requests, _, errText, code = run([]string{"manifests", "apply", "--dry-run", "--file", "SERVER_MANIFEST"}, `[]`)
	if code != 0 || !containsRequest(requests, "GET /api/v1/users/") || containsRequest(requests, "POST ") || containsRequest(requests, "DELETE ") {
		t.Fatalf("dry-run code=%d err=%q requests=%v", code, errText, requests)
	}

	requests, _, errText, code = run([]string{"manifests", "apply", "--file", "SERVER_MANIFEST"}, `[]`)
	if code != 0 || !containsRequest(requests, "POST /api/v1/prompts/id/p1/access/update") {
		t.Fatalf("apply code=%d err=%q requests=%v", code, errText, requests)
	}

	requests, _, errText, code = run([]string{"manifests", "sync", "--scope", "prompts", "--file", "SERVER_MANIFEST"}, `[]`)
	if code == 0 || !containsRequest(requests, "GET /api/v1/users/") || containsRequest(requests, "DELETE ") || !strings.Contains(errText, "destructive sync actions") {
		t.Fatalf("sync confirmation code=%d err=%q requests=%v", code, errText, requests)
	}
}

func TestManifestApplyCreatesSameSetValveOwnersFirstAndConverges(t *testing.T) {
	tests := []struct {
		ownerKind  string
		valveKind  string
		listPath   string
		createPath string
		getPath    string
		valvePath  string
	}{
		{ownerKind: "Function", valveKind: "FunctionValve", listPath: "/api/v1/functions/list", createPath: "/api/v1/functions/create", getPath: "/api/v1/functions/id/dep", valvePath: "/api/v1/functions/id/dep/valves"},
		{ownerKind: "Tool", valveKind: "ToolValve", listPath: "/api/v1/tools/list", createPath: "/api/v1/tools/create", getPath: "/api/v1/tools/id/dep", valvePath: "/api/v1/tools/id/dep/valves"},
	}
	for _, tt := range tests {
		t.Run(tt.ownerKind, func(t *testing.T) {
			dir := t.TempDir()
			writeManifest(t, filepath.Join(dir, "z-owner.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"`+tt.ownerKind+`","metadata":{"name":"dep"},"spec":{"id":"dep","name":"dep","content":"x"}}`)
			writeManifest(t, filepath.Join(dir, "a-valve.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"`+tt.valveKind+`","metadata":{"name":"dep"},"spec":{"enabled":true}}`)

			ownerExists := true
			var valves map[string]any
			var writes []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case tt.listPath:
					if ownerExists {
						_, _ = io.WriteString(w, `[{"id":"dep","name":"dep","content":"x"}]`)
					} else {
						_, _ = io.WriteString(w, `[]`)
					}
				case tt.createPath:
					writes = append(writes, "owner")
					ownerExists = true
					_, _ = io.WriteString(w, `{"id":"dep","name":"dep","content":"x"}`)
				case tt.getPath:
					_, _ = io.WriteString(w, `{"id":"dep","name":"dep","content":"x"}`)
				case tt.valvePath, tt.valvePath + "/update":
					if !ownerExists {
						w.WriteHeader(http.StatusNotFound)
						_, _ = io.WriteString(w, `{"detail":"owner missing"}`)
						return
					}
					if r.Method == http.MethodPost {
						writes = append(writes, "valve")
						if err := json.NewDecoder(r.Body).Decode(&valves); err != nil {
							t.Fatalf("decode valves: %v", err)
						}
					}
					if valves == nil {
						_, _ = io.WriteString(w, `null`)
					} else {
						_ = json.NewEncoder(w).Encode(valves)
					}
				default:
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
			var diffOutput string
			for _, args := range [][]string{
				{"manifests", "diff", "--directory", dir},
				{"manifests", "apply", "--dry-run", "--directory", dir},
			} {
				out.Reset()
				errOut.Reset()
				writes = nil
				if code := app.Run(context.Background(), args); code != 0 {
					t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
				}
				if len(writes) != 0 {
					t.Fatalf("%v writes = %v", args, writes)
				}
				if diffOutput == "" {
					diffOutput = out.String()
				} else if out.String() != diffOutput {
					t.Fatalf("dry-run output %q differs from diff %q", out.String(), diffOutput)
				}
			}

			ownerExists = false
			out.Reset()
			errOut.Reset()
			writes = nil
			if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code != 0 {
				t.Fatalf("first apply code=%d stderr=%q", code, errOut.String())
			}
			if got, want := strings.Join(writes, ","), "owner,valve"; got != want {
				t.Fatalf("write order = %q, want %q", got, want)
			}
			ownerResult := "CREATE " + tt.ownerKind + "/dep"
			valveResult := "CREATE " + tt.valveKind + "/dep"
			if ownerIndex, valveIndex := strings.Index(out.String(), ownerResult), strings.Index(out.String(), valveResult); ownerIndex < 0 || valveIndex < 0 || ownerIndex > valveIndex {
				t.Fatalf("result order = %q", out.String())
			}

			writes = nil
			out.Reset()
			errOut.Reset()
			if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code != 0 {
				t.Fatalf("second apply code=%d stderr=%q", code, errOut.String())
			}
			if len(writes) != 0 {
				t.Fatalf("second apply writes = %v", writes)
			}
		})
	}
}

func TestManifestApplyValveOwnerResolutionUsesIDNotDisplayName(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, filepath.Join(dir, "owner.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"Function","metadata":{"name":"owner"},"spec":{"id":"fn-id","name":"Display Name","content":"x"}}`)
	writeManifest(t, filepath.Join(dir, "valve.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"FunctionValve","metadata":{"name":"Display Name"},"spec":{"enabled":true}}`)

	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodDelete {
			writes++
		}
		switch r.URL.Path {
		case "/api/v1/functions/list":
			_, _ = io.WriteString(w, `[{"id":"fn-id","name":"Display Name","content":"x"}]`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code == 0 {
		t.Fatalf("code=0 stderr=%q", errOut.String())
	}
	if writes != 0 {
		t.Fatalf("invalid display-name owner caused %d writes", writes)
	}
	if !strings.Contains(errOut.String(), "could not be resolved") {
		t.Fatalf("stderr=%q", errOut.String())
	}
}

func TestManifestApplyUsesGeneratedSameSetGroupIDAndConverges(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, filepath.Join(dir, "z-group.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"sre"},"spec":{"name":"SRE"}}`)
	writeManifest(t, filepath.Join(dir, "a-skill.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"Skill","metadata":{"name":"skill-a"},"spec":{"id":"skill-a","name":"Skill A","content":"x"},"access_grants":[{"principal_type":"group","principal_ref":"group:sre","permission":"write"}]}`)

	groupExists := false
	var grants []accessGrant
	var writes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/groups/":
			if groupExists {
				_, _ = io.WriteString(w, `[{"id":"generated-group","name":"SRE"}]`)
			} else {
				_, _ = io.WriteString(w, `[]`)
			}
		case "/api/v1/groups/create":
			writes = append(writes, "group")
			groupExists = true
			_, _ = io.WriteString(w, `{"id":"generated-group","name":"SRE"}`)
		case "/api/v1/groups/id/generated-group":
			_, _ = io.WriteString(w, `{"id":"generated-group","name":"SRE"}`)
		case "/api/v1/skills/list":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "skill-a", "name": "Skill A", "content": "x", "access_grants": grants}})
		case "/api/v1/skills/id/skill-a/access/update":
			writes = append(writes, "grant")
			var body map[string][]accessGrant
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode grants: %v", err)
			}
			grants = body["access_grants"]
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "skill-a", "name": "Skill A", "content": "x", "access_grants": grants})
		case "/api/v1/skills/id/skill-a":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "skill-a", "name": "Skill A", "content": "x", "access_grants": grants})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code != 0 {
		t.Fatalf("first apply code=%d stderr=%q", code, errOut.String())
	}
	if got, want := strings.Join(writes, ","), "group,grant"; got != want {
		t.Fatalf("write order = %q, want %q", got, want)
	}
	if len(grants) != 1 || grants[0].PrincipalID != "generated-group" {
		t.Fatalf("resolved grants = %+v", grants)
	}
	if groupIndex, grantIndex := strings.Index(out.String(), "CREATE Group/sre"), strings.Index(out.String(), "REPLACE-GRANTS Skill/skill-a"); groupIndex < 0 || grantIndex < 0 || groupIndex > grantIndex {
		t.Fatalf("result order = %q", out.String())
	}

	writes = nil
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code != 0 {
		t.Fatalf("second apply code=%d stderr=%q", code, errOut.String())
	}
	if len(writes) != 0 {
		t.Fatalf("second apply writes = %v", writes)
	}
}

func TestGeneratedGroupManifestConvergesWithoutDependent(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "group.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"sre"},"spec":{"id":"sre","name":"SRE"}}`)

	groupExists := false
	creates := 0
	updates := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/groups/":
			if groupExists {
				_, _ = io.WriteString(w, `[{"id":"generated-group","name":"SRE"}]`)
			} else {
				_, _ = io.WriteString(w, `[]`)
			}
		case "/api/v1/groups/create":
			creates++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			if _, ok := body["id"]; ok {
				t.Fatalf("group create body contains declarative id: %#v", body)
			}
			groupExists = true
			_, _ = io.WriteString(w, `{"id":"generated-group","name":"SRE"}`)
		case "/api/v1/groups/id/generated-group":
			if r.Method == http.MethodPost {
				updates++
			}
			_, _ = io.WriteString(w, `{"id":"generated-group","name":"SRE"}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	for attempt := 1; attempt <= 2; attempt++ {
		errOut.Reset()
		if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", manifest}); code != 0 {
			t.Fatalf("apply %d code=%d stderr=%q", attempt, code, errOut.String())
		}
	}
	if creates != 1 {
		t.Fatalf("group create count = %d, want 1", creates)
	}
	if updates != 0 {
		t.Fatalf("group update count = %d, want 0", updates)
	}
}

func TestGroupManifestExactIDWinsNameCollision(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "group.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"sre"},"spec":{"id":"sre","name":"SRE"}}`)
	groups := []string{
		`[{"id":"generated-group","name":"SRE"},{"id":"sre","name":"Other"}]`,
		`[{"id":"sre","name":"Other"},{"id":"generated-group","name":"SRE"}]`,
	}
	for index, response := range groups {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/groups/" {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				_, _ = io.WriteString(w, response)
			}))
			defer server.Close()

			app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
			if code := app.Run(context.Background(), []string{"manifests", "diff", "--file", manifest}); code != 0 || !strings.Contains(out.String(), "id=sre") {
				t.Fatalf("exact ID did not win: stdout=%q stderr=%q", out.String(), errOut.String())
			}
		})
	}
}

func TestGeneratedGroupSyncDoesNotPruneMatchedServerID(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "group.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"sre"},"spec":{"id":"sre","name":"SRE"}}`)

	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodDelete {
			writes++
		}
		if r.URL.Path != "/api/v1/groups/" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, `[{"id":"generated-group","name":"SRE"}]`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "sync", "--scope", "groups", "--yes", "--dry-run", "--file", manifest}); code != 0 {
		t.Fatalf("sync code=%d stderr=%q", code, errOut.String())
	}
	if writes != 0 {
		t.Fatalf("dry-run writes = %d", writes)
	}
	if strings.Contains(out.String(), "DELETE Group/") || !strings.Contains(out.String(), "UNCHANGED Group/sre id=generated-group") {
		t.Fatalf("sync plan = %q", out.String())
	}
}

func TestManifestApplyComposesSameSetGroupToolAndValveDependencies(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, filepath.Join(dir, "a-valve.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"ToolValve","metadata":{"name":"tool"},"spec":{"enabled":true}}`)
	writeManifest(t, filepath.Join(dir, "b-tool.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"Tool","metadata":{"name":"tool"},"spec":{"name":"tool","content":"x"},"access_grants":[{"principal_type":"group","principal_ref":"group:sre","permission":"write"}]}`)
	writeManifest(t, filepath.Join(dir, "c-group.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"sre"},"spec":{"name":"SRE"}}`)

	groupExists := false
	toolExists := false
	var grants []accessGrant
	var valves map[string]any
	var writes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/groups/":
			if groupExists {
				_, _ = io.WriteString(w, `[{"id":"generated-group","name":"SRE"}]`)
			} else {
				_, _ = io.WriteString(w, `[]`)
			}
		case "/api/v1/groups/create":
			writes = append(writes, "group")
			groupExists = true
			_, _ = io.WriteString(w, `{"id":"generated-group","name":"SRE"}`)
		case "/api/v1/groups/id/generated-group":
			_, _ = io.WriteString(w, `{"id":"generated-group","name":"SRE"}`)
		case "/api/v1/tools/list":
			if toolExists {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "tool", "name": "tool", "content": "x", "access_grants": grants}})
			} else {
				_, _ = io.WriteString(w, `[]`)
			}
		case "/api/v1/tools/create":
			writes = append(writes, "tool")
			toolExists = true
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "tool", "name": "tool", "content": "x", "access_grants": grants})
		case "/api/v1/tools/id/tool":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "tool", "name": "tool", "content": "x", "access_grants": grants})
		case "/api/v1/tools/id/tool/access/update":
			writes = append(writes, "grant")
			var body map[string][]accessGrant
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode grants: %v", err)
			}
			grants = body["access_grants"]
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "tool", "name": "tool", "content": "x", "access_grants": grants})
		case "/api/v1/tools/id/tool/valves":
			if !toolExists {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"detail":"owner missing"}`)
			} else if valves == nil {
				_, _ = io.WriteString(w, `null`)
			} else {
				_ = json.NewEncoder(w).Encode(valves)
			}
		case "/api/v1/tools/id/tool/valves/update":
			writes = append(writes, "valve")
			if err := json.NewDecoder(r.Body).Decode(&valves); err != nil {
				t.Fatalf("decode valves: %v", err)
			}
			_ = json.NewEncoder(w).Encode(valves)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code != 0 {
		t.Fatalf("first apply code=%d stderr=%q", code, errOut.String())
	}
	if got, want := strings.Join(writes, ","), "group,tool,grant,valve"; got != want {
		t.Fatalf("write order = %q, want %q", got, want)
	}
	if len(grants) != 1 || grants[0].PrincipalID != "generated-group" {
		t.Fatalf("resolved grants = %+v", grants)
	}
	previous := -1
	for _, result := range []string{"CREATE Group/sre", "CREATE Tool/tool", "CREATE ToolValve/tool"} {
		index := strings.Index(out.String(), result)
		if index <= previous {
			t.Fatalf("result order = %q", out.String())
		}
		previous = index
	}

	writes = nil
	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code != 0 {
		t.Fatalf("second apply code=%d stderr=%q", code, errOut.String())
	}
	if len(writes) != 0 {
		t.Fatalf("second apply writes = %v", writes)
	}
}

func TestManifestApplyRejectsMissingOrDuplicateDependenciesBeforeMutation(t *testing.T) {
	tests := []struct {
		name      string
		manifests []string
		want      string
	}{
		{name: "missing valve owner", manifests: []string{
			`{"apiVersion":"oictl.openwebui/v1","kind":"FunctionValve","metadata":{"name":"dep"},"spec":{"enabled":true}}`,
		}},
		{name: "duplicate valve owners", want: "ambiguous", manifests: []string{
			`{"apiVersion":"oictl.openwebui/v1","kind":"Function","metadata":{"name":"a"},"spec":{"id":"shared","name":"a","content":"x"}}`,
			`{"apiVersion":"oictl.openwebui/v1","kind":"Function","metadata":{"name":"b"},"spec":{"id":"shared","name":"b","content":"x"}}`,
			`{"apiVersion":"oictl.openwebui/v1","kind":"FunctionValve","metadata":{"name":"shared"},"spec":{"enabled":true}}`,
		}},
		{name: "missing group", manifests: []string{
			`{"apiVersion":"oictl.openwebui/v1","kind":"Skill","metadata":{"name":"skill"},"spec":{"id":"skill","name":"skill","content":"x"},"access_grants":[{"principal_type":"group","principal_ref":"group:missing","permission":"read"}]}`,
		}},
		{name: "duplicate groups", want: "ambiguous", manifests: []string{
			`{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"a"},"spec":{"name":"SRE"}}`,
			`{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"b"},"spec":{"name":"SRE"}}`,
			`{"apiVersion":"oictl.openwebui/v1","kind":"Skill","metadata":{"name":"skill"},"spec":{"id":"skill","name":"skill","content":"x"},"access_grants":[{"principal_type":"group","principal_name":"SRE","permission":"read"}]}`,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			filenames := []string{"a.json", "b.json", "c.json"}
			for index, body := range tt.manifests {
				writeManifest(t, filepath.Join(dir, filenames[index]), body)
			}
			writes := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost || r.Method == http.MethodDelete {
					writes++
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = io.WriteString(w, `{"detail":"mutation reached"}`)
					return
				}
				switch r.URL.Path {
				case "/api/v1/groups/", "/api/v1/functions/list", "/api/v1/skills/list":
					_, _ = io.WriteString(w, `[]`)
				case "/api/v1/functions/id/dep/valves", "/api/v1/functions/id/shared/valves":
					_, _ = io.WriteString(w, `null`)
				default:
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
			if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code == 0 {
				t.Fatalf("code=0 stderr=%q", errOut.String())
			}
			if writes != 0 {
				t.Fatalf("dependency failure caused %d writes; stderr=%q", writes, errOut.String())
			}
			if tt.want != "" && !strings.Contains(strings.ToLower(errOut.String()), tt.want) {
				t.Fatalf("stderr=%q, want %q", errOut.String(), tt.want)
			}
		})
	}
}

func TestManifestHTTPErrorAndServerFilteredGrant(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "prompt.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"hello"},"spec":{"command":"hello","content":"hi"},"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)

	t.Run("non-2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"detail":"forbidden"}`)
		}))
		defer server.Close()
		app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
		if code := app.Run(context.Background(), []string{"manifests", "diff", "--file", manifest}); code == 0 || !strings.Contains(errOut.String(), "403 Forbidden") {
			t.Fatalf("code=%d stderr=%q", code, errOut.String())
		}
	})

	t.Run("server filtered grants", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/prompts/":
				_, _ = io.WriteString(w, `[]`)
			case "/api/v1/prompts/create":
				_, _ = io.WriteString(w, `{"id":"new","command":"hello","content":"hi","access_grants":[]}`)
			case "/api/v1/prompts/id/new":
				_, _ = io.WriteString(w, `{"id":"new","command":"hello","content":"hi","access_grants":[]}`)
			case "/api/v1/prompts/id/new/access/update":
				_, _ = io.WriteString(w, `{"id":"new","command":"hello","content":"hi","access_grants":[]}`)
			default:
				t.Fatalf("unexpected request %s", r.URL.Path)
			}
		}))
		defer server.Close()
		app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
		if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", manifest}); code == 0 || !strings.Contains(errOut.String(), "server filtered access grants") {
			t.Fatalf("code=%d stderr=%q", code, errOut.String())
		}
	})
}

func TestModelManifestUpdatePreservesCurrentAccessGrants(t *testing.T) {
	desiredGrants := []accessGrant{{PrincipalType: "user", PrincipalID: "*", Permission: "read"}}
	tests := []struct {
		name          string
		currentGrants []accessGrant
	}{
		{
			name:          "non-empty",
			currentGrants: []accessGrant{{PrincipalType: "group", PrincipalID: "ops", Permission: "write"}},
		},
		{name: "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantGrants := normalizeAccessGrants(tt.currentGrants)
			model := map[string]any{
				"id":            "llama-ops",
				"name":          "Llama Ops",
				"meta":          map[string]any{},
				"params":        map[string]any{},
				"is_active":     false,
				"access_grants": wantGrants,
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/models/model/update":
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						http.Error(w, "invalid JSON", http.StatusBadRequest)
						return
					}
					value, ok := body["access_grants"]
					if !ok || value == nil {
						http.Error(w, "access_grants must be a list", http.StatusUnprocessableEntity)
						return
					}
					rawGrants, ok := value.([]any)
					if !ok {
						http.Error(w, "access_grants must be a list", http.StatusUnprocessableEntity)
						return
					}
					if got := normalizeAccessGrants(accessGrantsFromValue(rawGrants)); !grantsEqual(got, wantGrants) {
						http.Error(w, "access_grants changed", http.StatusBadRequest)
						return
					}
					_ = json.NewEncoder(w).Encode(model)
				case "/api/v1/models/model":
					_ = json.NewEncoder(w).Encode(model)
				default:
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
				}
			}))
			defer server.Close()

			app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
			current := resourceState{
				Kind:         "Model",
				Name:         "llama-ops",
				ID:           "llama-ops",
				Spec:         map[string]any{"id": "llama-ops", "name": "Llama Ops", "meta": map[string]any{}, "params": map[string]any{}, "is_active": true},
				AccessGrants: tt.currentGrants,
			}
			desired := manifestDoc("Model", "llama-ops", map[string]any{"id": "llama-ops", "name": "Llama Ops", "meta": map[string]any{}, "params": map[string]any{}, "is_active": false}, desiredGrants, true)
			if _, err := (modelHandler{}).Update(context.Background(), app, globalOptions{}, current, desired); err != nil {
				t.Fatalf("update model: %v; stderr=%s", err, errOut.String())
			}
		})
	}
}

func TestModelManifestApplyOpenWebUI011UpdateAndRerun(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "model.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"llama-ops"},"spec":{"name":"Llama Ops","meta":{"description":"New managed description"},"params":{},"is_active":true},"access_grants":[{"principal_type":"group","principal_id":"example-users","permission":"read"}]}`)

	currentDescription := "Old managed description"
	currentGrants := []accessGrant{{PrincipalType: "group", PrincipalID: "legacy-ops", Permission: "write"}}
	plannedGrants := append([]accessGrant(nil), currentGrants...)
	desiredGrants := []accessGrant{{PrincipalType: "group", PrincipalID: "example-users", Permission: "read"}}
	listRequests := map[string]int{}
	fetchRequests := 0
	updateRequests := 0
	accessUpdateRequests := 0
	mutationEndpoints := []string{}

	modelObject := func() map[string]any {
		return map[string]any{
			"id":   "llama-ops",
			"name": "Llama Ops",
			"meta": map[string]any{
				"description":       currentDescription,
				"capabilities":      nil,
				"knowledge":         nil,
				"profile_image_url": nil,
			},
			"params":        map[string]any{},
			"is_active":     true,
			"access_grants": currentGrants,
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/models/list":
			if r.Method != http.MethodGet {
				t.Fatalf("list method = %s, want GET", r.Method)
			}
			page := r.URL.Query().Get("page")
			listRequests[page]++
			switch page {
			case "1":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"items": []map[string]any{{"id": "other-model", "name": "Other", "meta": map[string]any{}, "params": map[string]any{}, "is_active": true}},
					"total": 2,
				})
			case "2":
				plannedGrants = append(plannedGrants[:0], currentGrants...)
				_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{modelObject()}, "total": 2})
			default:
				t.Fatalf("list page = %q", page)
			}
		case "/api/v1/models/model":
			if r.Method != http.MethodGet || r.URL.Query().Get("id") != "llama-ops" {
				t.Fatalf("model fetch = %s %s", r.Method, r.URL.String())
			}
			fetchRequests++
			_ = json.NewEncoder(w).Encode(modelObject())
		case "/api/v1/models/model/update":
			if r.Method != http.MethodPost {
				t.Fatalf("update method = %s, want POST", r.Method)
			}
			updateRequests++
			mutationEndpoints = append(mutationEndpoints, "update")
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			rawGrants, ok := body["access_grants"]
			if !ok || rawGrants == nil {
				http.Error(w, "access_grants must be a list", http.StatusUnprocessableEntity)
				return
			}
			grantValues, ok := rawGrants.([]any)
			if !ok {
				http.Error(w, "access_grants must be a list", http.StatusUnprocessableEntity)
				return
			}
			updatedGrants := normalizeAccessGrants(accessGrantsFromValue(grantValues))
			if !grantsEqual(updatedGrants, plannedGrants) {
				http.Error(w, fmt.Sprintf("spec update access_grants = %#v, want planned %#v", updatedGrants, plannedGrants), http.StatusConflict)
				return
			}
			meta, ok := body["meta"].(map[string]any)
			if !ok || stringValue(meta["description"]) != "New managed description" {
				http.Error(w, "missing updated description", http.StatusBadRequest)
				return
			}
			currentDescription = stringValue(meta["description"])
			currentGrants = append(currentGrants[:0], updatedGrants...)
			_ = json.NewEncoder(w).Encode(modelObject())
		case "/api/v1/models/model/access/update":
			if r.Method != http.MethodPost {
				t.Fatalf("access update method = %s, want POST", r.Method)
			}
			accessUpdateRequests++
			mutationEndpoints = append(mutationEndpoints, "replace-grants")
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			if body["id"] != "llama-ops" {
				http.Error(w, "wrong model id", http.StatusBadRequest)
				return
			}
			grantValues, ok := body["access_grants"].([]any)
			if !ok {
				http.Error(w, "access_grants must be a list", http.StatusUnprocessableEntity)
				return
			}
			updatedGrants := normalizeAccessGrants(accessGrantsFromValue(grantValues))
			if !grantsEqual(updatedGrants, desiredGrants) {
				http.Error(w, "wrong declared access_grants", http.StatusBadRequest)
				return
			}
			currentGrants = updatedGrants
			_ = json.NewEncoder(w).Encode(modelObject())
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", manifest, "--output", "json"}); code != 0 {
		t.Fatalf("first apply code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"action": "update"`) || !strings.Contains(out.String(), `"action": "replace-grants"`) {
		t.Fatalf("first apply output = %q", out.String())
	}
	if updateRequests != 1 || accessUpdateRequests != 1 || fetchRequests != 2 {
		t.Fatalf("first apply request counts: update=%d access-update=%d fetch=%d", updateRequests, accessUpdateRequests, fetchRequests)
	}
	if got := strings.Join(mutationEndpoints, " -> "); got != "update -> replace-grants" {
		t.Fatalf("mutation endpoint order = %q, want %q", got, "update -> replace-grants")
	}
	if !grantsEqual(currentGrants, desiredGrants) {
		t.Fatalf("final grants = %#v, want %#v", currentGrants, desiredGrants)
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", manifest, "--output", "json"}); code != 0 {
		t.Fatalf("rerun apply code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"action": "unchanged"`) {
		t.Fatalf("rerun apply output = %q", out.String())
	}
	if updateRequests != 1 || accessUpdateRequests != 1 || fetchRequests != 2 {
		t.Fatalf("rerun mutated model: update=%d access-update=%d fetch=%d", updateRequests, accessUpdateRequests, fetchRequests)
	}
	if len(mutationEndpoints) != 2 {
		t.Fatalf("rerun mutation endpoints = %v, want no mutations after first apply", mutationEndpoints)
	}
	if listRequests["1"] != 2 || listRequests["2"] != 2 {
		t.Fatalf("paginated list request counts = %#v, want two requests per page", listRequests)
	}
}

func TestModelManifestApplyCreateUpdateDeleteAndGrants(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "model.json")
	writeManifest(t, model, `{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"llama-ops"},"spec":{"name":"Llama Ops","base_model_id":"llama3","is_active":false},"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)

	deleted := false
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/models/list":
			if deleted {
				_, _ = io.WriteString(w, `{"items":[{"id":"llama-ops","name":"Llama Ops","meta":{},"params":{},"is_active":false}],"total":1}`)
				return
			}
			_, _ = io.WriteString(w, `{"items":[{"id":"old-model","name":"Old","meta":{},"params":{},"is_active":true}],"total":1}`)
		case "/api/v1/models/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			if body["id"] != "llama-ops" || body["name"] != "Llama Ops" || body["is_active"] != false || body["access_grants"] != nil {
				t.Fatalf("create body = %#v", body)
			}
			_, _ = io.WriteString(w, `{"id":"llama-ops","name":"Llama Ops","base_model_id":"llama3","meta":{},"params":{},"is_active":false,"access_grants":[]}`)
		case "/api/v1/models/model":
			if r.URL.Query().Get("id") != "llama-ops" {
				t.Fatalf("get query = %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"id":"llama-ops","name":"Llama Ops","base_model_id":"llama3","meta":{},"params":{},"is_active":false,"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)
		case "/api/v1/models/model/access/update":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode access body: %v", err)
			}
			if body["id"] != "llama-ops" {
				t.Fatalf("access body = %#v", body)
			}
			_, _ = io.WriteString(w, `{"id":"llama-ops","name":"Llama Ops","base_model_id":"llama3","meta":{},"params":{},"is_active":false,"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)
		case "/api/v1/models/model/delete":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode delete body: %v", err)
			}
			if body["id"] != "old-model" {
				t.Fatalf("delete body = %#v", body)
			}
			deleted = true
			_, _ = io.WriteString(w, `true`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "sync", "--scope", "models", "--yes", "--file", model}); code != 0 {
		t.Fatalf("sync code=%d stderr=%q", code, errOut.String())
	}
	for _, want := range []string{"POST /api/v1/models/create", "POST /api/v1/models/model/access/update", "POST /api/v1/models/model/delete"} {
		if !containsRequest(requests, want) {
			t.Fatalf("missing request %s in %v", want, requests)
		}
	}
}

func TestModelManifestApplyUpdatesActiveStateWithoutToggle(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "model.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"llama-ops"},"spec":{"name":"Llama Ops","params":{},"meta":{},"is_active":false}}`)

	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/models/list":
			_, _ = io.WriteString(w, `{"items":[{"id":"llama-ops","name":"Llama Ops","meta":{},"params":{},"is_active":true}],"total":1}`)
		case "/api/v1/models/model/update":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			if body["id"] != "llama-ops" || body["is_active"] != false {
				t.Fatalf("update body = %#v", body)
			}
			_, _ = io.WriteString(w, `{"id":"llama-ops","name":"Llama Ops","meta":{},"params":{},"is_active":false}`)
		case "/api/v1/models/model":
			_, _ = io.WriteString(w, `{"id":"llama-ops","name":"Llama Ops","meta":{},"params":{},"is_active":false}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", manifest}); code != 0 {
		t.Fatalf("apply code=%d stderr=%q", code, errOut.String())
	}
	if containsRequest(requests, "POST /api/v1/models/model/toggle") {
		t.Fatalf("used toggle endpoint: %v", requests)
	}
}

func TestModelManifestDiffNormalizesOptionalNullMeta(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "model.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"llama-ops"},"spec":{"name":"Llama Ops","meta":{},"params":{},"is_active":true}}`)

	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/models/list":
			_, _ = io.WriteString(w, `{"items":[{"id":"llama-ops","name":"Llama Ops","meta":{"description":null,"capabilities":null,"knowledge":null,"profile_image_url":null},"params":{},"is_active":true}],"total":1}`)
		case "/api/v1/models/model":
			if r.URL.Query().Get("id") != "llama-ops" {
				t.Fatalf("model id = %q", r.URL.Query().Get("id"))
			}
			_, _ = io.WriteString(w, `{"id":"llama-ops","name":"Llama Ops","meta":{"description":null,"capabilities":null,"knowledge":null,"profile_image_url":null},"params":{},"is_active":true}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", manifest, "--output", "json"}); code != 0 {
		t.Fatalf("apply code=%d stderr=%q", code, errOut.String())
	}
	if containsRequest(requests, "POST ") || !strings.Contains(out.String(), `"action": "unchanged"`) {
		t.Fatalf("apply output=%q requests=%v", out.String(), requests)
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"manifests", "diff", "--file", manifest, "--output", "json"}); code != 0 {
		t.Fatalf("diff code=%d stderr=%q", code, errOut.String())
	}
	if containsRequest(requests, "POST ") || !strings.Contains(out.String(), `"action": "unchanged"`) || strings.Contains(out.String(), `"field": "meta"`) {
		t.Fatalf("diff output=%q requests=%v", out.String(), requests)
	}

	fetched, err := (modelHandler{}).Fetch(context.Background(), app, globalOptions{}, "llama-ops")
	if err != nil {
		t.Fatalf("fetch model: %v", err)
	}
	desiredSpec := map[string]any{"id": "llama-ops", "name": "Llama Ops", "meta": map[string]any{}, "params": map[string]any{}, "is_active": true}
	if changes := diffSpec(normalizeModelSpecForCompare(desiredSpec), normalizeModelSpecForCompare(fetched.Spec)); len(changes) != 0 {
		t.Fatalf("fetched model changes = %#v", changes)
	}
}

func TestModelManifestDiffDetectsChangedDescriptionWithNullMetaDefaults(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "model.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"llama-ops"},"spec":{"name":"Llama Ops","meta":{"description":"New description"},"params":{},"is_active":true}}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models/list" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		_, _ = io.WriteString(w, `{"items":[{"id":"llama-ops","name":"Llama Ops","meta":{"description":"Old description","capabilities":null,"knowledge":null,"profile_image_url":null},"params":{},"is_active":true}],"total":1}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "diff", "--file", manifest, "--output", "json"}); code != 0 {
		t.Fatalf("diff code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"action": "update"`) || !strings.Contains(out.String(), `"field": "meta"`) || !strings.Contains(out.String(), "New description") {
		t.Fatalf("diff output=%q", out.String())
	}
}

func TestModelManifestDiffPreservesNonNullOptionalMetaDiff(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "model.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"llama-ops"},"spec":{"name":"Llama Ops","meta":{"description":"Company graph"},"params":{},"is_active":true}}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models/list" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		_, _ = io.WriteString(w, `{"items":[{"id":"llama-ops","name":"Llama Ops","meta":{"description":"Company graph","capabilities":{"vision":true},"knowledge":[],"profile_image_url":null},"params":{},"is_active":true}],"total":1}`)
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "diff", "--file", manifest, "--output", "json"}); code != 0 {
		t.Fatalf("diff code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"action": "update"`) || !strings.Contains(out.String(), `"field": "meta"`) || !strings.Contains(out.String(), "vision") || !strings.Contains(out.String(), "knowledge") {
		t.Fatalf("diff output=%q", out.String())
	}
}

func TestModelManifestServerFilteredGrant(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "model.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"llama-ops"},"spec":{"name":"Llama Ops"},"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/models/list":
			_, _ = io.WriteString(w, `{"items":[],"total":0}`)
		case "/api/v1/models/create", "/api/v1/models/model", "/api/v1/models/model/access/update":
			_, _ = io.WriteString(w, `{"id":"llama-ops","name":"Llama Ops","meta":{},"params":{},"is_active":true,"access_grants":[]}`)
		default:
			t.Fatalf("unexpected request %s", r.URL.Path)
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", manifest}); code == 0 || !strings.Contains(errOut.String(), "server filtered access grants") {
		t.Fatalf("code=%d stderr=%q", code, errOut.String())
	}
}

func TestFunctionManifestPlanFetchesExistingFunctionBeforeCompare(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "function.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Function","metadata":{"name":"fn_a"},"spec":{"id":"fn_a","name":"Fn","content":"print(1)"}}`)

	detailReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/functions/list":
			_, _ = io.WriteString(w, `[{"id":"fn_a","name":"Fn"}]`)
		case "/api/v1/functions/id/fn_a":
			detailReads++
			_, _ = io.WriteString(w, `{"id":"fn_a","name":"Fn","content":"print(1)"}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "diff", "--file", manifest}); code != 0 {
		t.Fatalf("diff code=%d stderr=%q", code, errOut.String())
	}
	if detailReads != 1 {
		t.Fatalf("Function detail reads = %d, want 1", detailReads)
	}
	if !strings.Contains(out.String(), "UNCHANGED Function/fn_a") {
		t.Fatalf("diff output=%q", out.String())
	}
}

func TestFunctionManifestPlanIgnoresUnmanagedServerManifestMetadata(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "function.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Function","metadata":{"name":"fn_a"},"spec":{"id":"fn_a","name":"Fn","content":"print(1)","meta":{"description":"Managed"}}}`)

	function := `{"id":"fn_a","name":"Fn","content":"print(1)","meta":{"description":"Managed","manifest":{"title":"Parsed frontmatter"}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/functions/list":
			_, _ = io.WriteString(w, "["+function+"]")
		case "/api/v1/functions/id/fn_a":
			_, _ = io.WriteString(w, function)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "diff", "--file", manifest}); code != 0 {
		t.Fatalf("diff code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "UNCHANGED Function/fn_a") {
		t.Fatalf("diff output=%q", out.String())
	}
}

func TestFunctionManifestSpecsForCompareIgnoresServerManifestMetadataWhenDesiredMetaIsOmitted(t *testing.T) {
	desired := map[string]any{"id": "fn_a", "name": "Fn", "content": "print(1)"}
	current := map[string]any{
		"id":      "fn_a",
		"name":    "Fn",
		"content": "print(1)",
		"meta": map[string]any{
			"description": "Server metadata",
			"manifest":    map[string]any{"title": "Parsed frontmatter"},
		},
	}

	normalizedDesired, normalizedCurrent := functionSpecsForCompare(desired, current)
	if !reflect.DeepEqual(normalizedDesired, desired) {
		t.Fatalf("desired = %#v, want unchanged %#v", normalizedDesired, desired)
	}
	currentMeta, ok := normalizedCurrent["meta"].(map[string]any)
	if !ok {
		t.Fatalf("current meta = %#v, want map", normalizedCurrent["meta"])
	}
	if _, exists := currentMeta["manifest"]; exists {
		t.Fatalf("current meta = %#v, want unmanaged manifest removed", currentMeta)
	}
	if got := currentMeta["description"]; got != "Server metadata" {
		t.Fatalf("current meta description = %#v, want preserved", got)
	}
}

func TestFunctionManifestPlanComparesManagedManifestMetadata(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "function.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Function","metadata":{"name":"fn_a"},"spec":{"id":"fn_a","name":"Fn","content":"print(1)","meta":{"description":"Managed","manifest":{"title":"Desired frontmatter"}}}}`)

	function := `{"id":"fn_a","name":"Fn","content":"print(1)","meta":{"description":"Managed","manifest":{"title":"Current frontmatter"}}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/functions/list":
			_, _ = io.WriteString(w, "["+function+"]")
		case "/api/v1/functions/id/fn_a":
			_, _ = io.WriteString(w, function)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "diff", "--file", manifest}); code != 0 {
		t.Fatalf("diff code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "UPDATE Function/fn_a") || !strings.Contains(out.String(), "fields=meta") {
		t.Fatalf("diff output=%q", out.String())
	}
}

func TestFunctionManifestApplyReconcilesNewInactiveState(t *testing.T) {
	testFunctionManifestActivationReconciliation(t, false, true, false)
}

func TestFunctionManifestApplyReconcilesActiveToInactive(t *testing.T) {
	testFunctionManifestActivationReconciliation(t, true, true, false)
}

func TestFunctionManifestApplyReconcilesInactiveToActive(t *testing.T) {
	testFunctionManifestActivationReconciliation(t, true, false, true)
}

func testFunctionManifestActivationReconciliation(t *testing.T, exists bool, initialActive bool, desiredActive bool) {
	t.Helper()
	wasExisting := exists
	manifest := filepath.Join(t.TempDir(), "function.json")
	writeManifest(t, manifest, fmt.Sprintf(`{"apiVersion":"oictl.openwebui/v1","kind":"Function","metadata":{"name":"fn_a"},"spec":{"id":"fn_a","name":"Fn","content":"print(1)","is_active":%t}}`, desiredActive))

	active := initialActive
	createCount := 0
	updateCount := 0
	toggleCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeFunction := func() {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "fn_a", "name": "Fn", "content": "print(1)", "is_active": active})
		}
		switch r.URL.Path {
		case "/api/v1/functions/list":
			if !exists {
				_, _ = io.WriteString(w, `[]`)
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "fn_a", "name": "Fn", "content": "print(1)", "is_active": active}})
		case "/api/v1/functions/create":
			createCount++
			exists = true
			active = true // Open WebUI creates Functions active even when FunctionForm says false.
			assertFunctionManifestActivePayload(t, r, desiredActive)
			writeFunction()
		case "/api/v1/functions/id/fn_a/update":
			updateCount++
			assertFunctionManifestActivePayload(t, r, desiredActive)
			writeFunction() // FunctionForm is accepted, but Open WebUI leaves activation unchanged.
		case "/api/v1/functions/id/fn_a/toggle":
			if r.Method != http.MethodPost {
				t.Fatalf("toggle method = %s, want POST", r.Method)
			}
			toggleCount++
			active = !active
			writeFunction()
		case "/api/v1/functions/id/fn_a":
			writeFunction()
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", manifest}); code != 0 {
		t.Fatalf("apply code=%d stderr=%q", code, errOut.String())
	}
	if active != desiredActive {
		t.Fatalf("active = %t after apply, want %t", active, desiredActive)
	}
	if toggleCount != 1 {
		t.Fatalf("toggle count = %d, want 1", toggleCount)
	}
	wantCreateCount, wantUpdateCount := 1, 0
	if wasExisting {
		wantCreateCount, wantUpdateCount = 0, 1
	}
	if createCount != wantCreateCount || updateCount != wantUpdateCount {
		t.Fatalf("create count = %d, update count = %d; want %d and %d", createCount, updateCount, wantCreateCount, wantUpdateCount)
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"manifests", "diff", "--file", manifest}); code != 0 {
		t.Fatalf("convergence diff code=%d stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "UNCHANGED Function/fn_a") {
		t.Fatalf("convergence diff = %q", out.String())
	}
	if toggleCount != 1 {
		t.Fatalf("convergence diff toggled Function: count=%d", toggleCount)
	}
}

func assertFunctionManifestActivePayload(t *testing.T, r *http.Request, desired bool) {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode FunctionForm: %v", err)
	}
	if body["is_active"] != desired {
		t.Fatalf("FunctionForm is_active = %#v, want %t", body["is_active"], desired)
	}
}

func TestAdditionalManifestKindsApplyUseExpectedEndpoints(t *testing.T) {
	dir := t.TempDir()
	channel := filepath.Join(dir, "channel.json")
	function := filepath.Join(dir, "function.json")
	group := filepath.Join(dir, "group.json")
	skill := filepath.Join(dir, "skill.json")
	writeManifest(t, channel, `{"apiVersion":"oictl.openwebui/v1","kind":"Channel","metadata":{"name":"channel-a"},"spec":{"name":"Channel A"},"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)
	writeManifest(t, function, `{"apiVersion":"oictl.openwebui/v1","kind":"Function","metadata":{"name":"fn_a"},"spec":{"id":"fn_a","name":"Fn","content":"print(1)"}}`)
	writeManifest(t, group, `{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"group-a"},"spec":{"id":"group-a","name":"Group A"}}`)
	writeManifest(t, skill, `{"apiVersion":"oictl.openwebui/v1","kind":"Skill","metadata":{"name":"skill-a"},"spec":{"id":"skill-a","name":"Skill A","content":"Do work"},"access_grants":[{"principal_type":"group","principal_id":"ops","permission":"write"}]}`)

	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.Method+" "+r.URL.Path] = true
		switch r.URL.Path {
		case "/api/v1/channels/list", "/api/v1/functions/list", "/api/v1/groups/", "/api/v1/skills/list":
			_, _ = io.WriteString(w, `[]`)
		case "/api/v1/channels/create":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode channel create: %v", err)
			}
			if body["name"] != "Channel A" || body["access_grants"] == nil {
				t.Fatalf("channel create body = %#v", body)
			}
			_, _ = io.WriteString(w, `{"id":"channel-a","name":"Channel A","access_grants":[]}`)
		case "/api/v1/channels/channel-a", "/api/v1/channels/channel-a/update":
			_, _ = io.WriteString(w, `{"id":"channel-a","name":"Channel A","access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)
		case "/api/v1/functions/create":
			_, _ = io.WriteString(w, `{"id":"fn_a","name":"Fn","content":"print(1)"}`)
		case "/api/v1/functions/id/fn_a":
			_, _ = io.WriteString(w, `{"id":"fn_a","name":"Fn","content":"print(1)"}`)
		case "/api/v1/groups/create":
			_, _ = io.WriteString(w, `{"id":"group-a","name":"Group A"}`)
		case "/api/v1/groups/id/group-a":
			_, _ = io.WriteString(w, `{"id":"group-a","name":"Group A"}`)
		case "/api/v1/skills/create":
			_, _ = io.WriteString(w, `{"id":"skill-a","name":"Skill A","content":"Do work","access_grants":[]}`)
		case "/api/v1/skills/id/skill-a":
			_, _ = io.WriteString(w, `{"id":"skill-a","name":"Skill A","content":"Do work","access_grants":[{"principal_type":"group","principal_id":"ops","permission":"write"}]}`)
		case "/api/v1/skills/id/skill-a/access/update":
			_, _ = io.WriteString(w, `{"id":"skill-a","name":"Skill A","content":"Do work","access_grants":[{"principal_type":"group","principal_id":"ops","permission":"write"}]}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code != 0 {
		t.Fatalf("apply code=%d stderr=%q", code, errOut.String())
	}
	for _, want := range []string{"POST /api/v1/channels/create", "POST /api/v1/channels/channel-a/update", "POST /api/v1/functions/create", "GET /api/v1/functions/id/fn_a", "POST /api/v1/groups/create", "GET /api/v1/groups/id/group-a", "POST /api/v1/skills/create", "POST /api/v1/skills/id/skill-a/access/update"} {
		if !seen[want] {
			t.Fatalf("missing endpoint %s in %#v", want, seen)
		}
	}
}

func TestChannelGrantReplacementUsesChannelUpdatePayload(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "channel.json")
	writeManifest(t, manifest, `{"apiVersion":"oictl.openwebui/v1","kind":"Channel","metadata":{"name":"channel-a"},"spec":{"name":"Channel New"},"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)

	channelName := "Channel Old"
	channelGrants := any([]map[string]any{{"principal_type": "group", "principal_id": "ops", "permission": "read"}})
	updates := []string{}
	writeChannel := func(w http.ResponseWriter) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "channel-a", "name": channelName, "access_grants": channelGrants})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/channels/list":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "channel-a", "name": channelName, "access_grants": channelGrants, "created_at": 1}})
		case "/api/v1/channels/channel-a/update":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode channel update: %v", err)
			}
			if body["created_at"] != nil {
				t.Fatalf("channel update included server metadata: %#v", body)
			}
			if grants, ok := body["access_grants"]; ok {
				updates = append(updates, "replace-grants")
				channelGrants = grants
			} else {
				updates = append(updates, "update")
			}
			if name, ok := body["name"].(string); ok {
				channelName = name
			}
			writeChannel(w)
		case "/api/v1/channels/channel-a":
			writeChannel(w)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", manifest}); code != 0 {
		t.Fatalf("apply code=%d stderr=%q", code, errOut.String())
	}
	if len(updates) != 2 || updates[0] != "replace-grants" || updates[1] != "replace-grants" {
		t.Fatalf("channel update order = %v", updates)
	}
	if channelName != "Channel New" {
		t.Fatalf("channel name = %q", channelName)
	}
	wantGrants := []accessGrant{{PrincipalType: "user", PrincipalID: "*", Permission: "read"}}
	if got := normalizeAccessGrants(accessGrantsFromValue(channelGrants)); !grantsEqual(wantGrants, got) {
		t.Fatalf("channel grants = %#v", got)
	}
}

func TestTerminalServerConnectionManifestPlanApplyAndGrantPreservation(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, filepath.Join(dir, "new.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"TerminalServerConnection","metadata":{"name":"new-shell"},"spec":{"url":"https://new","path":"/ws"},"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)
	writeManifest(t, filepath.Join(dir, "ops.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"TerminalServerConnection","metadata":{"name":"ops-shell"},"spec":{"url":"https://new-ops","config":{"keep":true}},"access_grants":[{"principal_type":"group","principal_id":"ops","permission":"write"}]}`)
	writeManifest(t, filepath.Join(dir, "preserve.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"TerminalServerConnection","metadata":{"name":"preserve"},"spec":{"url":"https://preserve-new"}}`)
	writeManifest(t, filepath.Join(dir, "same.json"), `{"apiVersion":"oictl.openwebui/v1","kind":"TerminalServerConnection","metadata":{"name":"same"},"spec":{"url":"https://same"},"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`)

	connections := []map[string]any{
		{
			"id":   "ops-shell",
			"name": "Ops Shell",
			"url":  "https://old-ops",
			"config": map[string]any{
				"keep":          true,
				"access_grants": []any{map[string]any{"principal_type": "group", "principal_id": "old", "permission": "read"}},
			},
		},
		{
			"id":   "preserve",
			"name": "Preserve",
			"url":  "https://preserve-old",
			"config": map[string]any{
				"access_grants": []any{map[string]any{"principal_type": "group", "principal_id": "keep", "permission": "read"}},
			},
		},
		{
			"id":   "same",
			"name": "Same",
			"url":  "https://same",
			"config": map[string]any{
				"access_grants": []any{map[string]any{"principal_type": "user", "principal_id": "*", "permission": "read"}},
			},
		},
		{"id": "remote-only", "name": "Remote Only", "url": "https://remote"},
	}
	postCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/configs/terminal_servers" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{terminalServerConnectionsKey: connections, "OTHER": map[string]any{"keep": true}})
		case http.MethodPost:
			postCount++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode terminal config post: %v", err)
			}
			if _, ok := body["OTHER"].(map[string]any); !ok {
				t.Fatalf("unknown config missing from post: %#v", body)
			}
			encoded, _ := json.Marshal(body[terminalServerConnectionsKey])
			if err := json.Unmarshal(encoded, &connections); err != nil {
				t.Fatalf("decode posted connections: %v", err)
			}
			_ = json.NewEncoder(w).Encode(body)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "diff", "--directory", dir}); code != 0 {
		t.Fatalf("diff code=%d stderr=%q", code, errOut.String())
	}
	for _, want := range []string{"CREATE TerminalServerConnection/new-shell", "UPDATE TerminalServerConnection/ops-shell", "REPLACE-GRANTS TerminalServerConnection/ops-shell", "UPDATE TerminalServerConnection/preserve", "UNCHANGED TerminalServerConnection/same"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("diff missing %q:\n%s", want, out.String())
		}
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--directory", dir}); code != 0 {
		t.Fatalf("apply code=%d stderr=%q", code, errOut.String())
	}
	if postCount == 0 {
		t.Fatal("apply did not post terminal server config")
	}
	byID := map[string]map[string]any{}
	for _, conn := range connections {
		byID[stringValue(conn["id"])] = conn
	}
	if byID["new-shell"] == nil || byID["new-shell"]["name"] != "new-shell" || len(terminalServerAccessGrants(byID["new-shell"])) != 1 {
		t.Fatalf("new connection = %#v", byID["new-shell"])
	}
	if byID["ops-shell"]["url"] != "https://new-ops" || terminalServerAccessGrants(byID["ops-shell"])[0].PrincipalID != "ops" {
		t.Fatalf("updated ops connection = %#v", byID["ops-shell"])
	}
	if grants := terminalServerAccessGrants(byID["preserve"]); len(grants) != 1 || grants[0].PrincipalID != "keep" {
		t.Fatalf("preserved grants = %#v", grants)
	}
	if byID["remote-only"] == nil {
		t.Fatalf("remote-only terminal server was pruned: %#v", byID)
	}
}

func TestResourceHandlersUseExpectedEndpoints(t *testing.T) {
	dir := t.TempDir()
	knowledge := filepath.Join(dir, "knowledge.json")
	tool := filepath.Join(dir, "tool.json")
	writeManifest(t, knowledge, `{"apiVersion":"oictl.openwebui/v1","kind":"Knowledge","metadata":{"name":"kb"},"spec":{"name":"kb","description":"new"}}`)
	writeManifest(t, tool, `{"apiVersion":"oictl.openwebui/v1","kind":"Tool","metadata":{"name":"hammer"},"spec":{"name":"hammer","content":"new"},"access_grants":[]}`)

	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.Method+" "+r.URL.Path] = true
		switch r.URL.Path {
		case "/api/v1/knowledge/":
			_, _ = io.WriteString(w, `[{"id":"kb-id","name":"kb","description":"old"}]`)
		case "/api/v1/knowledge/kb-id/update":
			_, _ = io.WriteString(w, `{"id":"kb-id","name":"kb","description":"new"}`)
		case "/api/v1/knowledge/kb-id":
			_, _ = io.WriteString(w, `{"id":"kb-id","name":"kb","description":"new"}`)
		case "/api/v1/tools/list":
			_, _ = io.WriteString(w, `[]`)
		case "/api/v1/tools/create":
			_, _ = io.WriteString(w, `{"id":"tool-id","name":"hammer","content":"new","access_grants":[]}`)
		case "/api/v1/tools/id/tool-id":
			_, _ = io.WriteString(w, `{"id":"tool-id","name":"hammer","content":"new","access_grants":[]}`)
		case "/api/v1/tools/id/tool-id/access/update":
			_, _ = io.WriteString(w, `{"id":"tool-id","name":"hammer","content":"new","access_grants":[]}`)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "sk"})
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", knowledge}); code != 0 {
		t.Fatalf("knowledge apply code=%d stderr=%q", code, errOut.String())
	}
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--file", tool}); code != 0 {
		t.Fatalf("tool apply code=%d stderr=%q", code, errOut.String())
	}
	for _, want := range []string{"POST /api/v1/knowledge/kb-id/update", "GET /api/v1/knowledge/kb-id", "POST /api/v1/tools/create", "GET /api/v1/tools/id/tool-id", "POST /api/v1/tools/id/tool-id/access/update"} {
		if !seen[want] {
			t.Fatalf("missing endpoint %s in %#v", want, seen)
		}
	}
}

func writeManifest(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func manifestDoc(kind string, name string, spec map[string]any, grants []accessGrant, hasGrants bool) manifestDocument {
	return manifestDocument{APIVersion: manifestAPIVersion, Kind: kind, Metadata: manifestMetadata{Name: name}, Spec: spec, AccessGrants: grants, HasAccessGrants: hasGrants}
}

func containsRequest(requests []string, prefix string) bool {
	for _, request := range requests {
		if strings.HasPrefix(request, prefix) {
			return true
		}
	}
	return false
}
