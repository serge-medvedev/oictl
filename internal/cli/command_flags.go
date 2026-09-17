package cli

import (
	"fmt"
	"sort"
	"strings"
)

// Validate at dispatch, before target discovery or any mutation. A flag being
// understood by the shared parser does not mean the selected action supports it.
func validateCommandFlags(args []string) error {
	if len(args) == 0 || isHelp(args) || (len(args) > 1 && isHelp(args[1:])) {
		return nil
	}
	// Valve help is explicitly supported after the action and resource ID.
	if len(args) > 2 && (args[0] == "tools" || args[0] == "functions") && args[1] == "valves" && hasHelpFlag(args[2:]) {
		return nil
	}
	flags, pos, err := parseCommandFlags(args)
	if err != nil {
		return err
	}
	allowed := ""
	command := strings.Join(pos, " ")
	for n := len(pos); n > 0; n-- {
		if names, ok := commandFlagAllowlist[strings.Join(pos[:n], " ")]; ok {
			allowed = names
			command = strings.Join(pos[:n], " ")
			break
		}
	}
	set := map[string]bool{}
	for _, name := range strings.Fields(allowed) {
		set[name] = true
	}
	var supplied []string
	for name := range flags.values {
		supplied = append(supplied, name)
	}
	for name := range flags.bools {
		supplied = append(supplied, name)
	}
	if len(flags.headers) > 0 {
		supplied = append(supplied, "header")
	}
	sort.Strings(supplied)
	for _, name := range supplied {
		if !set[name] {
			return fmt.Errorf("--%s is not supported by %s", name, command)
		}
	}
	return nil
}

const bodyFlags = "data file data-file"
const confirmationFlags = "yes confirm"
const directoryFlags = "query q page order-by direction"
const analyticsFlags = "start-date end-date group-id user-id model-id chat-id skip limit days granularity order-by direction"

var commandFlagAllowlist = map[string]string{
	"api":                                        bodyFlags + " method header api-key-header",
	"auth login":                                 bodyFlags + " save",
	"auth profile update":                        bodyFlags,
	"auth password update":                       bodyFlags,
	"auth admin-config set":                      bodyFlags,
	"auth ldap-server-config set":                bodyFlags,
	"auth ldap-config set":                       bodyFlags,
	"auth oauth-config set":                      bodyFlags,
	"tasks config set":                           bodyFlags,
	"tasks skills attach":                        "external",
	"models list":                                "query view-option tag order-by direction page",
	"models base":                                "tag",
	"models create":                              bodyFlags,
	"models import":                              bodyFlags,
	"models sync":                                bodyFlags + " " + confirmationFlags,
	"models update":                              bodyFlags,
	"models toggle":                              bodyFlags,
	"models access-update":                       bodyFlags,
	"models delete":                              bodyFlags,
	"config import":                              bodyFlags,
	"config connections set":                     bodyFlags,
	"config tool-servers set":                    bodyFlags,
	"config tool-servers verify":                 bodyFlags,
	"config terminal-servers set":                bodyFlags,
	"config terminal-servers verify":             bodyFlags,
	"config terminal-servers policy":             bodyFlags,
	"config terminal-servers lifecycle":          bodyFlags,
	"config terminal-servers refresh":            bodyFlags,
	"config terminal-servers access-grants diff": bodyFlags,
	"config terminal-servers access-grants set":  bodyFlags,
	"config code-execution set":                  bodyFlags,
	"config models set":                          bodyFlags,
	"config suggestions":                         bodyFlags,
	"config banners set":                         bodyFlags,
	"config oauth-client register":               bodyFlags + " type",
	"files upload":                               "metadata process process-in-background",
	"files list":                                 "content page",
	"files search":                               "filename query skip limit content",
	"files update-content":                       bodyFlags,
	"files rename":                               "name",
	"knowledge list":                             "page",
	"knowledge search":                           "query view-option source page",
	"knowledge search-files":                     "query include-content page",
	"knowledge create":                           bodyFlags,
	"knowledge reindex":                          bodyFlags,
	"knowledge metadata-reindex":                 bodyFlags,
	"knowledge get":                              bodyFlags,
	"knowledge export":                           bodyFlags,
	"knowledge delete":                           bodyFlags,
	"knowledge reset":                            bodyFlags,
	"knowledge update":                           bodyFlags,
	"knowledge access-update":                    bodyFlags,
	"knowledge files list":                       "page query q include-content view-option order-by direction directory-id limit",
	"knowledge files update":                     bodyFlags,
	"knowledge files remove":                     bodyFlags,
	"knowledge files move":                       bodyFlags,
	"knowledge files batch-add":                  bodyFlags,
	"knowledge dirs create":                      bodyFlags,
	"knowledge dirs update":                      bodyFlags,
	"knowledge dirs delete":                      bodyFlags,
	"knowledge sync":                             bodyFlags,
	"manifests diff":                             "file directory dir",
	"manifests apply":                            "file directory dir dry-run",
	"manifests sync":                             "file directory dir scope dry-run " + confirmationFlags,
	"channels create":                            bodyFlags,
	"channels update":                            bodyFlags,
	"channels delete":                            confirmationFlags,
	"channels members list":                      directoryFlags,
	"channels members add":                       bodyFlags,
	"channels members remove":                    bodyFlags,
	"channels members active":                    bodyFlags,
	"channels messages list":                     "skip limit",
	"channels messages post":                     bodyFlags,
	"channels messages thread":                   "skip limit",
	"channels messages update":                   bodyFlags,
	"channels messages delete":                   confirmationFlags,
	"channels pins list":                         "page",
	"channels reactions add":                     bodyFlags + " name",
	"channels reactions remove":                  bodyFlags + " name",
	"webhooks channels create":                   bodyFlags + " name profile-image-url",
	"webhooks channels ensure":                   bodyFlags + " name profile-image-url show-url verify-url expected-url expected-url-env",
	"webhooks channels update":                   bodyFlags + " name profile-image-url",
	"webhooks channels delete":                   confirmationFlags,
	"webhooks channels url":                      "show-url",
	"webhooks events create":                     bodyFlags,
	"webhooks events update":                     bodyFlags,
	"webhooks events delete":                     confirmationFlags,
	"groups list":                                "share",
	"groups create":                              bodyFlags,
	"groups update":                              bodyFlags,
	"groups delete":                              confirmationFlags,
	"groups users add":                           bodyFlags,
	"groups users remove":                        bodyFlags,
	"users list":                                 directoryFlags,
	"users search":                               directoryFlags,
	"users create":                               bodyFlags,
	"users update":                               bodyFlags,
	"users delete":                               confirmationFlags,
	"users settings update":                      bodyFlags + " allow-sensitive-ui-keys",
	"users ui-settings patch":                    bodyFlags + " allow-sensitive-ui-keys allow-ui-settings-extension",
	"users ui-settings bulk-patch":               bodyFlags + " allow-sensitive-ui-keys allow-ui-settings-extension user-id users-file all query dry-run " + confirmationFlags,
	"functions list":                             "type",
	"functions create":                           bodyFlags,
	"functions update":                           bodyFlags,
	"functions delete":                           confirmationFlags,
	"functions export":                           "include-valves",
	"functions sync":                             bodyFlags + " " + confirmationFlags,
	"functions valves update":                    bodyFlags,
	"functions valves user update":               bodyFlags,
	"skills list":                                "query view-option page",
	"skills create":                              bodyFlags + " manifest",
	"skills update":                              bodyFlags + " manifest",
	"skills access-update":                       bodyFlags,
	"skills delete":                              confirmationFlags,
	"skills export":                              "format",
	"tools create":                               bodyFlags,
	"tools update":                               bodyFlags,
	"tools access-update":                        bodyFlags,
	"tools load-url":                             bodyFlags,
	"tools export":                               "manifest directory dir",
	"tools valves update":                        bodyFlags,
	"tools valves user update":                   bodyFlags,
	"chats list":                                 "user-id archived folder-id page limit skip",
	"chats search":                               "query q page limit",
	"chats export":                               "chat-id all-users",
	"chats import":                               bodyFlags,
	"chats compact":                              bodyFlags + " model",
	"chats delete":                               confirmationFlags,
	"chats tags set":                             bodyFlags + " tag",
	"chats tags delete":                          bodyFlags + " tag",
	"analytics models":                           analyticsFlags,
	"analytics users":                            analyticsFlags,
	"analytics messages":                         analyticsFlags,
	"analytics summary":                          analyticsFlags,
	"analytics daily":                            analyticsFlags,
	"analytics tokens":                           analyticsFlags,
	"automations list":                           "page status limit skip",
	"automations create":                         bodyFlags,
	"automations update":                         bodyFlags,
	"automations delete":                         confirmationFlags,
	"automations runs list":                      "limit skip",
	"scim":                                       "scim-token",
	"scim users list":                            "scim-token start-index startIndex count filter",
	"scim users create":                          bodyFlags + " scim-token",
	"scim users replace":                         bodyFlags + " scim-token",
	"scim users patch":                           bodyFlags + " scim-token",
	"scim users delete":                          "scim-token " + confirmationFlags,
	"scim groups list":                           "scim-token start-index startIndex count filter",
	"scim groups create":                         bodyFlags + " scim-token",
	"scim groups replace":                        bodyFlags + " scim-token",
	"scim groups patch":                          bodyFlags + " scim-token",
	"scim groups delete":                         "scim-token " + confirmationFlags,
	"providers openai config update":             bodyFlags,
	"providers openai config set":                bodyFlags,
	"providers openai verify":                    bodyFlags,
	"providers openai request":                   bodyFlags + " method header stream",
	"providers ollama config update":             bodyFlags,
	"providers ollama config set":                bodyFlags,
	"providers ollama verify":                    bodyFlags,
	"providers ollama request":                   bodyFlags + " method header stream",
}
