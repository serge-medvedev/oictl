package cli

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Closed inventory reconciled with consumers in app.go (runAPI/Auth/Tasks/Models/
// Config/Files/Knowledge), surfaces.go (native families), terminal_servers.go
// (decodeDesiredAccessGrants), manifests.go (loadManifestDocuments). F=fixed,
// D=endpoint/resource-defined, L=local normalization. B=bodyFlags, M=metadata,
// E=manifest envelope files/directories. Source locators live with authored help.
// No invariably unused B entries: knowledge optional bodies ARE forwarded.
// ensure consumes B only on creation. Explicit B overrides convenience values.
// knowledge sync's undocumented arbitrary operation suffix is not a public leaf.
// --users-file is newline text; skills --manifest is Markdown (boolean spelling
// is accepted but unused). tools export directory flags are output, not input.
const jsonHelpInventory = `api|POST /api/v1/auths/signin|DB|endpoint
 auth login||FB|email
 auth profile update||FB|date_of_birth
 auth password update||FB|new_password
 auth admin-config set||FB|CHANNEL_MODEL_RESPONSE_MODE
 auth ldap-server-config set||FB|attribute_for_groups
 auth ldap-config set||FB|enable_ldap
 auth oauth-config set||FB|OAUTH_GROUP_DEFAULT_SHARE
 tasks config set||FB|AUTOCOMPLETE_GENERATION_INPUT_MAX_LENGTH
 models create||LB|capabilities
 models import||LB|models
 models sync||LB|updated_at
 models update|model-a|LB|base_model_id
 models toggle|model-a|LB|id
 models access-update|model-a|LB|principal_type
 models delete|model-a|LB|id
 config import||FB|config
 config connections set||FB|ENABLE_BASE_MODELS_CACHE
 config tool-servers set||FB|TOOL_SERVER_CONNECTIONS
 config tool-servers verify||FB|auth_type
 config terminal-servers set||FB|TERMINAL_SERVER_CONNECTIONS
 config terminal-servers verify||FB|server_type
 config terminal-servers policy||FB|policy_data
 config terminal-servers lifecycle||FB|lifecycle_data
 config terminal-servers refresh||FB|only_idle
 config terminal-servers access-grants diff|shell-a|LB|principal_ref
 config terminal-servers access-grants set|shell-a|LB|principal_ref
 config code-execution set||FB|CODE_INTERPRETER_JUPYTER_TIMEOUT
 config models set||FB|DEFAULT_MODEL_METADATA
 config suggestions||FB|title
 config banners set||FB|dismissible
 config oauth-client register||FB|oauth_scope
 files upload|upload.txt|DM|metadata
 files update-content|file-a|FB|content
 knowledge create||FB|description
 knowledge reindex||DB|no request body
 knowledge metadata-reindex||DB|no request body
 knowledge get|knowledge-a|DB|no request body
 knowledge export|knowledge-a|DB|no request body
 knowledge delete|knowledge-a|DB|no request body
 knowledge reset|knowledge-a|DB|no request body
 knowledge update|knowledge-a|FB|description
 knowledge access-update|knowledge-a|FB|access_grants
 knowledge files update|knowledge-a|FB|file_id
 knowledge files remove|knowledge-a|FB|file_id
 knowledge files move|knowledge-a|FB|directory_id
 knowledge files batch-add|knowledge-a|FB|array
 knowledge dirs create|knowledge-a|FB|parent_id
 knowledge dirs update|knowledge-a dir-a|FB|__unset__
 knowledge dirs delete|knowledge-a dir-a|DB|no request body
 knowledge sync diff|knowledge-a|FB|checksum
 knowledge sync cleanup|knowledge-a|FB|dir_ids
 channels create||FB|is_private
 channels update|channel-a|FB|is_private
 channels members add|channel-a|FB|group_ids
 channels members remove|channel-a|FB|user_ids
 channels members active|channel-a|LB|is_active
 channels messages post|channel-a|FB|parent_id
 channels messages update|channel-a message-a|FB|reply_to_id
 channels reactions add|channel-a message-a|FB|name
 channels reactions remove|channel-a message-a|FB|name
 webhooks channels create|channel-a|LB|profile_image_url
 webhooks channels ensure|channel-a|LB|creation
 webhooks channels update|channel-a webhook-a|LB|replacement
 webhooks events create||FB|events
 webhooks events update|webhook-a|FB|events
 groups create||FB|permissions
 groups update|group-a|FB|permissions
 groups users add|group-a|FB|user_ids
 groups users remove|group-a|FB|user_ids
 users create||FB|pending
 users update|user-a|FB|password
 users settings update||DB|ui
 users ui-settings patch|user-a|DB|deployment extension
 users ui-settings bulk-patch|user-a user-b|DB|newline
 functions create||FB|manifest
 functions update|function-a|FB|manifest
 functions sync||LB|is_global
 functions valves update|function-a|DB|oictl functions valves spec <function-id>
 functions valves user update|function-a|DB|oictl functions valves user spec <function-id>
 skills create||FB|Markdown
 skills update|skill-a|FB|Markdown
 skills access-update|skill-a|FB|access_grants
 tools create||FB|has_user_valves
 tools update|tool-a|FB|has_user_valves
 tools access-update|tool-a|FB|access_grants
 tools load-url||FB|url
 tools valves update|tool-a|DB|oictl tools valves spec <tool-id>
 tools valves user update|tool-a|DB|oictl tools valves user spec <tool-id>
 chats import||FB|current_message_id
 chats compact|chat-a|FB|model
 chats tags set|chat-a|FB|name
 chats tags delete|chat-a|FB|name
 automations create||FB|rrule
 automations update|automation-a|FB|server_id
 scim users create||FB|emails
 scim users replace|user-a|FB|photos
 scim users patch|user-a|FB|Operations
 scim groups create||FB|members
 scim groups replace|group-a|FB|members
 scim groups patch|group-a|FB|Operations
 providers openai config update||FB|OPENAI_API_CONFIGS
 providers openai config set||FB|OPENAI_API_CONFIGS
 providers openai verify||FB|key
 providers openai request|chat/completions|DB|endpoint
 providers ollama config update||FB|OLLAMA_API_CONFIGS
 providers ollama config set||FB|OLLAMA_API_CONFIGS
 providers ollama verify||FB|url
 providers ollama request|api/generate|DB|endpoint
 manifests diff||LE|ToolValve
 manifests apply||LE|content_file
 manifests sync||LE|scope`

// Consumer checkpoint is independent of bodyFlags: a new consuming function or
// callsite prompts reconciliation even if someone forgets the flag allowlist.
// Counts include shared loader wrappers; inventory above expands dispatch leaves.
func TestJSONHelpConsumerClosure(t *testing.T) {
	expected := map[string]map[string]int{
		"app.go":              {"bodyWithID": 1, "runAPI": 1, "runAuth": 3, "getSet": 1, "runTasks": 1, "runModels": 2, "runConfig": 3, "configSubcommand": 1, "runFiles": 2, "runKnowledge": 2, "runKnowledgeFiles": 2, "runKnowledgeDirs": 1, "runKnowledgeSync": 1},
		"surfaces.go":         {"runChannels": 2, "runChannelMembers": 2, "runChannelMessages": 2, "runChannelReactions": 1, "channelWebhookMutationBody": 1, "channelWebhookUpdateBody": 1, "runWebhookEvents": 2, "runGroups": 2, "runUsers": 4, "runUsersUISettingsBulkPatch": 1, "userSettingsMutationBody": 1, "runGroupUsers": 1, "runFunctions": 3, "functionSyncBody": 1, "runSkills": 3, "runTools": 4, "runValves": 1, "skillPayloadBody": 1, "runChats": 2, "runChatTags": 1, "runAutomations": 2, "runSCIMResource": 1, "runProviders": 2, "runProviderRequest": 1, "requiredBody": 1},
		"terminal_servers.go": {"runTerminalServerAccessGrants": 2, "readDesiredAccessGrants": 1},
		"manifests.go":        {"runManifestWorkflow": 1},
	}
	loaders := map[string]bool{"requestBody": true, "requiredBody": true, "bodyWithID": true, "skillPayloadBody": true, "functionSyncBody": true, "userSettingsMutationBody": true, "readDesiredAccessGrants": true, "loadManifestDocuments": true, "multipartFileBody": true}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		path := entry.Name()
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		actual := map[string]int{}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				ident, ok := call.Fun.(*ast.Ident)
				if ok && loaders[ident.Name] {
					actual[fn.Name.Name]++
				}
				return true
			})
		}
		if len(actual) == 0 {
			continue
		}
		if !reflect.DeepEqual(actual, expected[path]) {
			t.Errorf("JSON consumers changed in %s; reconcile inventory: got %v expected %v", path, actual, expected[path])
		}
	}
}

func TestJSONHelpInventory(t *testing.T) {
	seen := map[string]bool{}
	invocations := 0
	for _, line := range strings.Split(jsonHelpInventory, "\n") {
		fields := strings.Split(strings.TrimSpace(line), "|")
		key, selectors, kind, want := fields[0], fields[1], fields[2], fields[3]
		if seen[key] {
			t.Fatalf("duplicate inventory key %s", key)
		}
		seen[key] = true
		t.Run(key, func(t *testing.T) {
			for _, flag := range []string{"--help", "-h"} {
				variants := []string{""}
				if selectors != "" {
					variants = append(variants, selectors)
				}
				for _, selector := range variants {
					invocations++
					app, out, errOut := newTestApp(nil)
					app.getenv = func(string) string { t.Fatal("help read environment/target"); return "" }
					app.userConfigDir = func() (string, error) { t.Fatal("help read config"); return "", nil }
					args := append(strings.Fields(key+" "+selector), flag)
					code := app.Run(context.Background(), args)
					if code != 0 {
						t.Fatalf("%v: code=%d stderr=%s", args, code, errOut.String())
					}
					for _, text := range []string{"Usage:", "oictl " + key, "Input:", "Examples:", want} {
						if !strings.Contains(out.String(), text) {
							t.Errorf("%v missing %q: %s", args, text, out.String())
						}
					}
					if kind[1] == 'B' && !strings.Contains(out.String(), "--data-file") {
						t.Errorf("missing body sources")
					}
					_, section, ok := strings.Cut(out.String(), "Examples:\n")
					if !ok {
						t.Fatal("examples absent")
					}
					if strings.Contains(out.String(), "\nSource:") {
						t.Error("rendered help contains a source-provenance footer")
					}
					// Contextual help appends action/global options after the examples.
					// Validate every example, but do not parse the following section as JSON.
					examples, _, _ := strings.Cut(section, "\nOptions:\n")
					count := 0
					for _, example := range strings.Split(examples, "\n") {
						example = strings.TrimSpace(example)
						if example == "" {
							continue
						}
						if !json.Valid([]byte(example)) {
							t.Errorf("invalid published example: %s", example)
						}
						count++
					}
					if count == 0 {
						t.Error("no examples")
					}
				}
			}
		})
	}
	for key, flags := range commandFlagAllowlist {
		if !strings.Contains(" "+flags+" ", " data ") {
			continue
		}
		if key == "knowledge sync" {
			if !seen[key+" diff"] || !seen[key+" cleanup"] {
				t.Error("sync leaves absent")
			}
			continue
		}
		if !seen[key] {
			t.Errorf("bodyFlags entry not accounted for: %s", key)
		}
	}
	for key := range jsonInputReferences {
		if !seen[key] {
			t.Errorf("help entry missing inventory: %s", key)
		}
	}
	t.Logf("JSON input actions=%d; help invocations=%d; manifest kinds=11", len(seen), invocations)
}
