package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Authored help only. Execution remains in the existing command handlers; option
// ownership remains in commandFlagAllowlist. JSON actions reuse their references.
type actionHelp struct{ arguments, purpose string }

var actionHelpReferences = map[string]actionHelp{
	"analytics daily":                           {"", "Inspect daily analytics."},
	"analytics messages":                        {"", "Inspect message analytics."},
	"analytics models":                          {"", "Inspect model analytics, or select a model report below."},
	"analytics models chats":                    {"<model-id>", "List chats for a model."},
	"analytics models overview":                 {"<model-id>", "Inspect a model analytics overview."},
	"analytics summary":                         {"", "Inspect the analytics summary."},
	"analytics tokens":                          {"", "Inspect token usage analytics."},
	"analytics users":                           {"", "Inspect user analytics."},
	"auth me":                                   {"", "Get the authenticated user."},
	"auth logout":                               {"", "Sign out and remove the selected local profile token after remote success."},
	"auth timezone set":                         {"<timezone>", "Set the authenticated user's timezone."},
	"auth api-key get":                          {"", "Get the current API key."},
	"auth api-key create":                       {"", "Create an API key."},
	"auth api-key delete":                       {"", "Delete the current API key."},
	"auth admin-config get":                     {"", "Get authentication administration settings."},
	"auth ldap-config get":                      {"", "Get LDAP settings."},
	"auth ldap-server-config get":               {"", "Get LDAP server settings."},
	"auth oauth-config get":                     {"", "Get OAuth settings."},
	"automations list":                          {"", "List scheduled automations."},
	"automations get":                           {"<automation-id>", "Get one automation."},
	"automations delete":                        {"<automation-id> (--yes | --confirm)", "Delete one automation."},
	"automations toggle":                        {"<automation-id>", "Toggle an automation's enabled state."},
	"automations run":                           {"<automation-id>", "Run an automation."},
	"automations runs list":                     {"<automation-id>", "List runs of an automation."},
	"channels list":                             {"", "List accessible channels."},
	"channels get":                              {"<channel-id>", "Get one channel."},
	"channels delete":                           {"<channel-id> (--yes | --confirm)", "Delete a channel."},
	"channels members list":                     {"<channel-id>", "List channel members."},
	"channels messages list":                    {"<channel-id>", "List channel messages."},
	"channels messages get":                     {"<channel-id> <message-id>", "Get one channel message."},
	"channels messages data":                    {"<channel-id> <message-id>", "Get a message's data object."},
	"channels messages thread":                  {"<channel-id> <message-id>", "List replies in a message thread."},
	"channels messages delete":                  {"<channel-id> <message-id> (--yes | --confirm)", "Delete a channel message."},
	"channels pins list":                        {"<channel-id>", "List pinned channel messages."},
	"channels pins set":                         {"<channel-id> <message-id>", "Pin a channel message."},
	"channels pins unset":                       {"<channel-id> <message-id>", "Unpin a channel message."},
	"chats list":                                {"", "List chats. Choose at most one scope: --user-id, --folder-id, or --archived."},
	"chats search":                              {"", "Search chats with optional query and pagination filters."},
	"chats get":                                 {"<chat-id>", "Get one chat."},
	"chats export":                              {"", "Export own chats as NDJSON; --chat-id selects one chat. --all-users selects database-wide JSON instead; it cannot be combined with --chat-id."},
	"chats share":                               {"<chat-id>", "Share a chat."},
	"chats archive":                             {"<chat-id>", "Archive or unarchive a chat."},
	"chats pin":                                 {"<chat-id>", "Pin or unpin a chat."},
	"chats delete":                              {"<chat-id> (--yes | --confirm)", "Delete a chat."},
	"chats tags get":                            {"<chat-id>", "Get a chat's tags."},
	"config export":                             {"", "Export configuration."},
	"config namespace":                          {"<namespace>", "Get one configuration namespace."},
	"config connections get":                    {"", "Get connection settings."},
	"config tool-servers get":                   {"", "Get tool server settings."},
	"config terminal-servers get":               {"", "Get terminal server settings."},
	"config terminal-servers access-grants get": {"<connection-id-or-name>", "Get a terminal server connection's access grants."},
	"config code-execution get":                 {"", "Get code execution settings."},
	"config models get":                         {"", "Get model settings."},
	"config models defaults":                    {"", "Get default model settings."},
	"config banners get":                        {"", "Get banner settings."},
	"files list":                                {"", "List files."},
	"files search":                              {"", "Search files by --filename or --query; the default filename pattern is *."},
	"files count":                               {"", "Count files."},
	"files get":                                 {"<file-id>", "Get file metadata."},
	"files status":                              {"<file-id>", "Get file processing status."},
	"files content":                             {"<file-id>", "Get file content."},
	"files html-content":                        {"<file-id>", "Get file content as HTML."},
	"files data-content":                        {"<file-id>", "Get a file's data content."},
	"files named-content":                       {"<file-id> <file-name>", "Get file content with the selected filename."},
	"files rename":                              {"<file-id> --name NAME", "Rename a file."},
	"files delete":                              {"<file-id>", "Delete one file."},
	"files delete-all":                          {"", "Delete all files."},
	"functions list":                            {"", "List functions; --type filter selects filters."},
	"functions get":                             {"<function-id>", "Get a function's source and metadata."},
	"functions delete":                          {"<function-id> (--yes | --confirm)", "Delete a function."},
	"functions export":                          {"", "Export functions; --include-valves includes valve values."},
	"functions load-url":                        {"<url>", "Load function source from a URL."},
	"functions toggle":                          {"<function-id>", "Toggle a function's active state."},
	"functions toggle-global":                   {"<function-id>", "Toggle a function's global state."},
	"functions valves get":                      {"<function-id>", "Get global function valve values."},
	"functions valves spec":                     {"<function-id>", "Inspect the global function valve schema."},
	"functions valves user get":                 {"<function-id>", "Get function valve values for the authenticated user."},
	"functions valves user spec":                {"<function-id>", "Inspect the user-scoped function valve schema."},
	"groups list":                               {"", "List native groups; --share filters shared groups."},
	"groups get":                                {"<group-id>", "Get one native group."},
	"groups info":                               {"<group-id>", "Get native group information."},
	"groups export":                             {"<group-id>", "Export a native group."},
	"groups preview":                            {"<group-id>", "Preview access controlled by a native group."},
	"groups delete":                             {"<group-id> (--yes | --confirm)", "Delete a native group."},
	"groups users list":                         {"<group-id>", "List users in a native group."},
	"knowledge list":                            {"", "List knowledge bases."},
	"knowledge search":                          {"", "Search knowledge bases."},
	"knowledge search-files":                    {"", "Search knowledge files."},
	"knowledge files list":                      {"<knowledge-id>", "List files in a knowledge base."},
	"knowledge files pending":                   {"<knowledge-id>", "List pending files in a knowledge base."},
	"knowledge files add":                       {"<knowledge-id> <file-id>", "Add an existing file to a knowledge base."},
	"profiles get":                              {"<name>", "Get a local profile with credentials redacted."},
	"profiles set":                              {"<name> --base-url URL [--token TOKEN]", "Save a local connection profile."},
	"profiles delete":                           {"<name>", "Delete a local connection profile."},
	"models list":                               {"", "List models with optional filters and ordering."},
	"models base":                               {"", "List base models."},
	"models base-tags":                          {"", "List base model tags."},
	"models tags":                               {"", "List model tags."},
	"models get":                                {"<model-id>", "Get one model."},
	"models delete-all":                         {"", "Delete all models."},
	"models export":                             {"", "Export the model inventory."},
	"providers openai config get":               {"", "Get OpenAI-compatible provider settings."},
	"providers openai models":                   {"", "List OpenAI-compatible provider models."},
	"providers ollama config get":               {"", "Get Ollama-compatible provider settings."},
	"providers ollama models":                   {"", "List Ollama-compatible provider models."},
	"providers ollama tags":                     {"", "List Ollama model tags."},
	"providers ollama version":                  {"", "Get the Ollama provider version."},
	"providers ollama ps":                       {"", "List running Ollama models."},
	"scim service-provider-config":              {"", "Get SCIM service provider configuration."},
	"scim resource-types":                       {"", "List SCIM resource types."},
	"scim schemas":                              {"", "List SCIM schemas."},
	"scim users list":                           {"", "List SCIM users."},
	"scim users get":                            {"<id>", "Get one SCIM user."},
	"scim users delete":                         {"<id> (--yes | --confirm)", "Delete one SCIM user."},
	"scim groups list":                          {"", "List SCIM groups."},
	"scim groups get":                           {"<id>", "Get one SCIM group."},
	"scim groups delete":                        {"<id> (--yes | --confirm)", "Delete one SCIM group."},
	"skills list":                               {"", "List skills with optional filters."},
	"skills get":                                {"<skill-id>", "Get one skill."},
	"skills delete":                             {"<skill-id> (--yes | --confirm)", "Delete one skill."},
	"skills toggle":                             {"<skill-id>", "Toggle a skill's active state."},
	"skills export":                             {"[<skill-id>]", "Export all skills or one skill as JSON; --format manifest requires a skill ID."},
	"tasks config get":                          {"", "Get task configuration."},
	"tasks skills attach":                       {"<skill-id-or-name>...", "Attach skills to the configured default task model, or the external task model with --external."},
	"tools list":                                {"", "List tools."},
	"tools get":                                 {"<tool-id>", "Get one tool."},
	"tools delete":                              {"<tool-id>", "Delete one tool."},
	"tools export":                              {"[<tool-id>]", "Export raw tool records, or Tool manifests with --manifest; --directory/--dir writes separate manifest files."},
	"tools valves get":                          {"<tool-id>", "Get global tool valve values."},
	"tools valves spec":                         {"<tool-id>", "Inspect the global tool valve schema."},
	"tools valves user get":                     {"<tool-id>", "Get tool valve values for the authenticated user."},
	"tools valves user spec":                    {"<tool-id>", "Inspect the user-scoped tool valve schema."},
	"users list":                                {"", "List users with optional query, pagination, and ordering."},
	"users search":                              {"", "Search users with optional query, pagination, and ordering."},
	"users get":                                 {"<user-id>", "Get one user."},
	"users delete":                              {"<user-id> (--yes | --confirm)", "Delete one user."},
	"users settings get":                        {"", "Get the authenticated user's settings."},
	"webhooks channels list":                    {"<channel-id>", "List incoming webhooks for a channel."},
	"webhooks channels get":                     {"<channel-id> <webhook-id>", "Get one incoming channel webhook."},
	"webhooks channels delete":                  {"<channel-id> <webhook-id> (--yes | --confirm)", "Delete an incoming channel webhook."},
	"webhooks channels url":                     {"<channel-id> <webhook-id> --show-url", "Show an incoming webhook URL; --show-url is required."},
	"webhooks events list":                      {"", "List event webhooks."},
	"webhooks events catalog":                   {"", "List supported event types."},
	"webhooks events get":                       {"<webhook-id>", "Get one event webhook."},
	"webhooks events delete":                    {"<webhook-id> (--yes | --confirm)", "Delete one event webhook."},
	"webhooks events enable":                    {"<webhook-id>", "Enable an event webhook."},
	"webhooks events disable":                   {"<webhook-id>", "Disable an event webhook."},
}

func helpChildren(path string) []string {
	children := map[string]bool{}
	prefix := path + " "
	if path == "" {
		prefix = ""
	}
	add := func(key string) {
		if strings.HasPrefix(key, prefix) {
			child := strings.Split(strings.TrimPrefix(key, prefix), " ")[0]
			children[child] = true
		}
	}
	for key := range actionHelpReferences {
		add(key)
	}
	for key := range jsonInputReferences {
		add(key)
	}
	result := make([]string, 0, len(children))
	for child := range children {
		result = append(result, child)
	}
	sort.Strings(result)
	return result
}

// Match fixed command words only until a leaf; everything after that is an
// operand. knowledge sync deliberately owns an unbounded action operand.
func contextualHelpPath(pos []string) (string, error) {
	path := ""
	for i, word := range pos {
		if word == "help" && (i == 0 || i == 1 || (i == 2 && (path == "tools valves" || path == "functions valves"))) {
			return path, nil
		}
		children := helpChildren(path)
		if len(children) == 0 {
			return path, nil
		}
		if path == "knowledge sync" && word != "diff" && word != "cleanup" {
			return path, nil
		}
		found := false
		for _, child := range children {
			if word == child {
				found = true
				break
			}
		}
		if !found {
			return path, fmt.Errorf("unknown command %q", word)
		}
		path = strings.TrimSpace(path + " " + word)
	}
	return path, nil
}

func (a *App) printContextualHelp(ctx context.Context, args []string) (bool, int) {
	pos, help := jsonHelpPositionals(args)
	if !help {
		return false, 0
	}
	// Version is handled after global syntax has been parsed, including appended help.
	if len(pos) > 0 && (pos[0] == "version" || pos[0] == "--version" || pos[0] == "-v") {
		return false, 0
	}
	path, err := contextualHelpPath(pos)
	if err != nil {
		fmt.Fprintln(a.err, err)
		a.printHelpHint(a.err, path)
		return true, 1
	}
	if path == "" {
		a.printHelp(a.out)
		fmt.Fprintln(a.out, "\nCommand help: oictl <command> --help (or -h).")
		return true, 0
	}
	if !strings.Contains(path, " ") && path != "api" {
		// Reuse each existing family's authored overview through its inert bare form.
		code := a.Run(ctx, []string{path})
		fmt.Fprintf(a.out, "\nChild help: oictl %s <command> --help (or -h).\n", path)
		return true, code
	}
	if path == "tools valves" || path == "functions valves" {
		resource := strings.Fields(path)[0]
		id := "tool-id"
		if resource == "functions" {
			id = "function-id"
		}
		a.printValvesHelp(resource, id)
		fmt.Fprintf(a.out, "\nChild help: oictl %s <command> --help (or -h).\n", path)
		return true, 0
	}
	if _, ok := jsonInputReferences[path]; ok {
		words := strings.Fields(path)
		verb := words[len(words)-1]
		subject := strings.Join(words[:len(words)-1], " ")
		if path == "api" {
			fmt.Fprint(a.out, "Send an authenticated API request.\n\n")
		} else if verb == "request" {
			fmt.Fprintf(a.out, "Send a %s request.\n\n", subject)
		} else {
			fmt.Fprintf(a.out, "%s %s.\n\n", strings.ToUpper(verb[:1])+verb[1:], subject)
		}
		a.printJSONInputHelp(args)
		a.printActionOptions(a.out, path)
		return true, 0
	}
	ref, leaf := actionHelpReferences[path]
	children := helpChildren(path)
	if leaf {
		fmt.Fprintf(a.out, "%s\n\nUsage:\n  oictl %s %s\n", ref.purpose, path, ref.arguments)
	} else {
		fmt.Fprintf(a.out, "Manage %s.\n\nUsage:\n  oictl %s <command>\n", path, path)
	}
	if path == "knowledge sync" {
		fmt.Fprintln(a.out, "  oictl knowledge sync <action> <knowledge-id>\n\nThe action is forwarded as supplied; documented actions follow.")
	}
	if len(children) > 0 {
		fmt.Fprintln(a.out, "\nCommands:")
		for _, child := range children {
			fmt.Fprintf(a.out, "  %s\n", child)
		}
		fmt.Fprintf(a.out, "\nChild help: oictl %s <command> --help (or -h).\n", path)
	}
	a.printActionOptions(a.out, path)
	return true, 0
}

func (a *App) printHelpHint(w io.Writer, path string) {
	if path == "" {
		fmt.Fprintln(w, "Run 'oictl --help' for usage.")
	} else {
		fmt.Fprintf(w, "Run 'oictl %s --help' for usage.\n", path)
	}
}

func (a *App) printActionOptions(w io.Writer, path string) {
	names := allowedCommandFlags(strings.Fields(path))
	fmt.Fprintln(w, "\nOptions:")
	for _, name := range strings.Fields(names) {
		if isBooleanCommandFlag(name) || (path == "tools export" && name == "manifest") {
			fmt.Fprintf(w, "  --%s[=true|false]\n", name)
		} else {
			fmt.Fprintf(w, "  --%s VALUE\n", name)
		}
	}
	fmt.Fprintln(w, "  --help, -h        Show this action's help\n\nGlobal options:\n  --profile NAME  --base-url URL (or --url URL)  --token TOKEN\n  --output FORMAT  --timeout DURATION  --out PATH")
}
