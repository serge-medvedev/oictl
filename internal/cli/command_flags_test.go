package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// R06 and R24: unsupported flags must fail before even a discovery request.
func TestUnsupportedCommandFlagsMakeNoRequests(t *testing.T) {
	cases := []struct{ command, flag string }{
		{"users delete u --yes", "--dry-run"},
		{"users delete u --yes", "--dry-run=false"},
		{"users delete u --yes", "--page 2"},
		{"users delete u --yes", "--typo value"},
		{"models delete m", "--dry-run"},
		{"models delete-all", "--dry-run"},
		{"models sync --data {}", "--dry-run"},
		{"functions sync --yes --data []", "--dry-run"},
		{"functions delete f --yes", "--dry-run"},
		{"tools delete t", "--dry-run"},
		{"skills delete s --yes", "--dry-run"},
		{"files delete f", "--dry-run"},
		{"files delete-all", "--dry-run"},
		{"knowledge delete k", "--dry-run"},
		{"channels delete c --yes", "--dry-run"},
		{"channels messages delete c m --yes", "--dry-run"},
		{"chats delete c --yes", "--dry-run"},
		{"groups delete g --yes", "--dry-run"},
		{"groups users list g", "--page 2"},
		{"groups users add g u", "--user-id other"},
		{"automations delete a --yes", "--dry-run"},
		{"scim users delete u --yes --scim-token fixture", "--dry-run"},
		{"webhooks channels delete c w --yes", "--dry-run"},
		{"webhooks events delete w --yes", "--dry-run"},
		{"config import --data {}", "--dry-run"},
		{"tasks skills attach s", "--dry-run"},
		{"auth logout", "--dry-run"},
		{"api DELETE /resource", "--dry-run"},
		{"providers openai request /models --method DELETE", "--dry-run"},
		{"channels list", "--page 2"},
		{"channels list", "--query nonexistent"},
		{"channels list", "--q nonexistent"},
		{"channels list", "--order-by name"},
		{"channels list", "--direction desc"},
	}
	for _, tc := range cases {
		t.Run(tc.command+"/"+tc.flag, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				_, _ = io.WriteString(w, `true`)
			}))
			defer server.Close()
			var out, errOut bytes.Buffer
			app := New(&out, &errOut, "test")
			app.getenv = func(string) string { return "" }
			app.userConfigDir = func() (string, error) { return t.TempDir(), nil }
			args := append([]string{"--base-url", server.URL, "--token", "fixture"}, strings.Fields(tc.command+" "+tc.flag)...)
			code := app.Run(context.Background(), args)
			flagName := strings.Split(strings.Fields(tc.flag)[0], "=")[0]
			if code == 0 || requests != 0 || !strings.Contains(errOut.String(), flagName) {
				t.Fatalf("code=%d requests=%d stderr=%q; expected local rejection of %s", code, requests, errOut.String(), flagName)
			}
		})
	}
}

func TestUnimplementedFlagAliasesAreRejected(t *testing.T) {
	for _, command := range []string{
		"manifests diff --file missing.json --scope models",
		"users ui-settings bulk-patch u --data {} --q alice --allow-ui-settings-extension",
		"knowledge files pending k --page 2",
	} {
		if err := validateCommandFlags(strings.Fields(command)); err == nil {
			t.Errorf("accepted unimplemented flag in %q", command)
		}
	}
}

func TestModelEmptySyncConfirmationAlias(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	app := New(&out, &errOut, "test")
	app.getenv = func(string) string { return "" }
	app.userConfigDir = func() (string, error) { return t.TempDir(), nil }
	code := app.Run(context.Background(), []string{"--base-url", server.URL, "--token", "fixture", "models", "sync", "--data", `[]`, "--confirm"})
	if code != 0 || requests != 1 {
		t.Fatalf("code=%d requests=%d stderr=%q", code, requests, errOut.String())
	}
}

func TestSupportedCommandFlags(t *testing.T) {
	commands := []string{
		"api POST /resource --header X-Test:yes --data {}",
		"auth login --file credentials.json --save",
		"models list --query ops --view-option owned --tag ops --page 2 --order-by name --direction asc",
		"models sync --data [] --yes",
		"files list --page 2 --content false",
		"knowledge files list k --page 2 --query ops",
		"manifests apply --file manifest.json --dry-run",
		"manifests sync --dir manifests --scope all --dry-run --confirm",
		"channels members list c --page 2 --query ops",
		"channels messages thread c m --skip 10 --limit 20",
		"channels reactions add c m --name smile",
		"webhooks channels ensure c --name hook --verify-url --expected-url-env HOOK_URL",
		"webhooks channels url c w --show-url",
		"users ui-settings bulk-patch --all --data {} --dry-run --allow-ui-settings-extension",
		"functions export --include-valves",
		"functions valves user update f --data {}",
		"skills create --manifest skill.md",
		"skills export s --format manifest",
		"tools export --manifest --directory manifests",
		"chats export --all-users",
		"chats tags delete c --tag ops",
		"scim groups list --scim-token fixture --startIndex 2 --count 30 --filter ops",
		"providers openai request /chat/completions --method POST --data {} --stream",
	}
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			if err := validateCommandFlags(strings.Fields(command)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
