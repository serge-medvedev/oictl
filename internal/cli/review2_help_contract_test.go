package cli

import (
	"context"
	"strings"
	"testing"
)

// Contracts: Open WebUI f8ae8a6c3 routers/scim.py:773-827.
// Forwarding remains unrestricted; these assertions concern published help.
func TestReview2HelpSCIMUserPatch(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		app, out, errOut := newTestApp(nil)
		if code := app.Run(context.Background(), []string{"scim", "users", "patch", flag}); code != 0 {
			t.Fatalf("code=%d stderr=%s", code, errOut.String())
		}
		for _, want := range []string{"add/replace", "pathless", "nonempty attribute object", "remove externalId", "400", "invalidPath", "invalidValue", "noTarget", "mutability", "active:boolean", "externalId:string|null"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("missing %q", want)
			}
		}
		for _, obsolete := range []string{"replace only", "not implemented", "ignored upstream"} {
			if strings.Contains(out.String(), obsolete) {
				t.Errorf("obsolete guidance %q", obsolete)
			}
		}
	}
}

// Contract: routers/chats.py:926-938 at Open WebUI f8ae8a6c3.
func TestReview2HelpChatSearch(t *testing.T) {
	app, out, errOut := newTestApp(nil)
	if code := app.Run(context.Background(), []string{"chats", "search", "--help"}); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
	for _, want := range []string{"--query TEXT", "--q", "required", "--page", "--limit", "--skip", "not supported"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(out.String(), "optional query") {
		t.Error("obsolete optional-query guidance")
	}
}

func TestReview2HelpExportAndReportContracts(t *testing.T) {
	for _, tc := range []struct {
		path  string
		wants []string
	}{
		{"skills export", []string{"package", "supporting files", "SKILL.md-only", "not a package backup", "ACL"}},
		{"skills get", []string{"version_id", "root content", "not a package"}},
		{"analytics messages", []string{"chat > model > user", "not AND", "chat ignores", "user ignores"}},
		{"analytics summary", []string{"dates and group", "user/model/chat", "not supported"}},
		{"chats list", []string{"--page", "--limit", "--skip", "not supported"}},
		{"automations list", []string{"--page", "--limit", "--skip", "not supported"}},
		{"config terminal-servers access-grants get", []string{"--out", "stdout"}},
		{"config terminal-servers access-grants diff", []string{"--out", "stdout"}},
		{"config terminal-servers access-grants set", []string{"--out", "stdout"}},
		{"files upload", []string{"MIME", "media", "binary fallback"}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			app, out, errOut := newTestApp(nil)
			if code := app.Run(context.Background(), append(strings.Fields(tc.path), "--help")); code != 0 {
				t.Fatalf("code=%d stderr=%s", code, errOut.String())
			}
			for _, want := range tc.wants {
				if !strings.Contains(out.String(), want) {
					t.Errorf("missing %q", want)
				}
			}
		})
	}
}

func TestReview2HelpSkillManifestSubset(t *testing.T) {
	app, out, errOut := newTestApp(nil)
	if code := app.Run(context.Background(), []string{"manifests", "apply", "--help"}); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
	_, skill, ok := strings.Cut(out.String(), "    Skill:")
	if !ok {
		t.Fatal("missing Skill section")
	}
	skill, _, _ = strings.Cut(skill, "    TerminalServerConnection:")
	for _, want := range []string{"content:optional", "empty SKILL.md", "native skills create/update", "packages", "not reconciled"} {
		if !strings.Contains(skill, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(skill, "content:string required") {
		t.Error("obsolete required content")
	}
}

// Contracts: models/skills.py:105-117,165-206,384-448;
// utils/skill_files.py:59-156 at Open WebUI f8ae8a6c3.
func TestReview2HelpSkillPackageInputs(t *testing.T) {
	for _, action := range []string{"create", "update"} {
		t.Run(action, func(t *testing.T) {
			app, out, errOut := newTestApp(nil)
			if code := app.Run(context.Background(), []string{"skills", action, "--help"}); code != 0 {
				t.Fatalf("code=%d stderr=%s", code, errOut.String())
			}
			wants := []string{"id and name", "content:optional", "files", "data.files", "SKILL.md", "base64", "commit_message", "meta.i18n", "Markdown"}
			if action == "create" {
				wants = append(wants, "empty SKILL.md", "not both", "operations", "not applied on creation")
			} else {
				wants = append(wants, "expected_version_id", "409", "400", "put", "move", "delete", "omitted", "preserve", "supporting files", "not more than one", "content:null")
			}
			for _, want := range wants {
				if !strings.Contains(out.String(), want) {
					t.Errorf("missing %q", want)
				}
			}
			for _, old := range []string{"id, name, content are required", "form defaults\n  for omitted"} {
				if strings.Contains(out.String(), old) {
					t.Errorf("obsolete %q", old)
				}
			}
		})
	}
}
