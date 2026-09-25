package cli

import (
	"fmt"
	"strings"
)

// Help-only authored references, not dispatch or validation metadata. Upstream
// locators refer to backend/open_webui at 0aa65cd1c9d42d6458976598b0551f3e37968ace.
type jsonInputHelp struct {
	usage, input, examples string
}

const jsonBodySources = `  Sources: --data JSON | --file PATH | --data-file PATH (choose one).
  Both body-file spellings read stdin when PATH is -. --data is literal text,
  not @file expansion. Repeated scalar options retain the last value.
`

var jsonInputReferences = map[string]jsonInputHelp{
	"manifests diff":  {"manifests diff (--file PATH[,PATH...] | --directory DIR[,DIR...] | --dir DIR[,DIR...])", manifestInputHelp, manifestExamples},                                               // Source: internal/cli/manifests.go: loadManifestDocuments, manifestHandlers, manifestPayload, validateManifestPlan; models/{channels,functions,groups,knowledge,models,prompts,skills,tools}.py; routers/configs.py: TerminalServerConnection
	"manifests apply": {"manifests apply (--file PATH[,PATH...] | --directory DIR[,DIR...] | --dir DIR[,DIR...]) [--dry-run]", manifestInputHelp, manifestExamples},                                  // Source: internal/cli/manifests.go: loadManifestDocuments, manifestHandlers, manifestPayload, validateManifestPlan; models/{channels,functions,groups,knowledge,models,prompts,skills,tools}.py; routers/configs.py: TerminalServerConnection
	"manifests sync":  {"manifests sync (--file PATH[,PATH...] | --directory DIR[,DIR...] | --dir DIR[,DIR...]) --scope SCOPE [--dry-run] [--yes | --confirm]", manifestInputHelp, manifestExamples}, // Source: internal/cli/manifests.go: loadManifestDocuments, manifestHandlers, manifestPayload, syncScopeKinds; models/{channels,functions,groups,knowledge,models,prompts,skills,tools}.py; routers/configs.py: TerminalServerConnection
	"api": {"api [METHOD] <endpoint> [--method METHOD] [--header NAME:VALUE] [--api-key-header NAME]", `  Optional request body. The selected method/path endpoint owns its JSON
  contract: root type, fields, requiredness and defaults are endpoint-specific.
  CLI forwards bytes unchanged, without a universal object schema or validation.
  Method defaults GET; a positional METHOD overrides --method.
  --header may repeat; --api-key-header selects a credential header.
  Global connection flags: --base-url/--url (OPEN_WEBUI_URL or OPEN_WEBUI_BASE_URL),
  --token (OPEN_WEBUI_API_KEY), --profile; --output, --timeout and --out retain
  normal output/timeout behavior.
  Illustrative POST /api/v1/auths/signin body (email/password strings required):`, `{"email":"user@example.com","password":"example-password"}`}, // Source: app.go: runAPI, requestBody; routers/auths.py: signin (example only)
	"functions create": {"functions create", functionFieldsHelp, functionExample},               // Source: models/functions.py: FunctionForm, FunctionMeta; routers/functions.py: create_new_function
	"functions update": {"functions update <function-id>", functionFieldsHelp, functionExample}, // Source: models/functions.py: FunctionForm; routers/functions.py: update_function_by_id
	"functions sync": {"functions sync (--yes | --confirm)", `  Required array or {"functions":[...]} object. CLI wraps a bare array;
  confirmation is always required. Omitted remote functions are removed; [] clears.
  Entries are complete function records, not only creation forms:
  required id, name, type, content:string, meta:object, created_at, updated_at:integer.
  user_id:optional nullable string=null; is_active, is_global:optional booleans=false;
  valves:optional nullable resource-defined object=null. meta has optional nullable
  description:string=null and manifest:free-form object={} plus extension keys.
  Source must define a valid function class. Server owns type/source validation
  and replaces inventory state rather than preserving omitted fields.`, `[{"id":"example_function","name":"Example Function","type":"pipe","content":"class Pipe:\n    def pipe(self, body: dict):\n        return 'Example'\n","meta":{},"created_at":0,"updated_at":0}]
{"functions":[{"id":"example_function","name":"Example Function","type":"pipe","content":"class Pipe:\n    def pipe(self, body: dict):\n        return 'Example'\n","meta":{},"created_at":0,"updated_at":0}]}`}, // Source: models/functions.py: FunctionWithValvesModel; routers/functions.py: SyncFunctionsForm, sync_functions; surfaces.go: functionSyncBody
	"functions valves update": {"functions valves update <function-id>", `  Required valve-values JSON object. Keys/types/requiredness/defaults are
  resource-defined by the function's valve schema. Inspect separately:
  oictl functions valves spec <function-id>
  Server removes null values and stores explicitly supplied validated values.
  No universal valve fields. Empty-object example is illustrative for a resource
  with no required valves; it is not a guarantee for every function.`, `{}`}, // Source: routers/functions.py: update_function_valves_by_id; surfaces.go: runValves
	"functions valves user update": {"functions valves user update <function-id>", `  Required valve-values JSON object for the authenticated user's function
  user valves. Keys/types/defaults are resource-defined; no positional user ID.
  Inspect separately: oictl functions valves user spec <function-id>
  Server removes null values and stores explicitly supplied validated values.
  Example is illustrative for a resource with no required user valves.`, `{}`}, // Source: routers/functions.py: update_function_user_valves_by_id; surfaces.go: runValves
	"skills create": {"skills create [--manifest PATH]", skillFieldsHelp, skillExample},            // Source: models/skills.py: SkillForm, SkillMeta; routers/skills.py: create_new_skill; surfaces.go: skillPayloadBody, parseSkillManifest
	"skills update": {"skills update <skill-id> [--manifest PATH]", skillFieldsHelp, skillExample}, // Source: models/skills.py: SkillForm; routers/skills.py: update_skill_by_id; surfaces.go: skillPayloadBody
	"skills access-update": {"skills access-update <skill-id>", `  Required JSON object with required access_grants:array of grant objects.
  This replaces grants only, not the full skill form.` + nativeGrantsHelp, `{"access_grants":[]}`}, // Source: routers/skills.py: SkillAccessGrantsForm, update_skill_access_by_id
	"tools create": {"tools create", toolFieldsHelp, toolExample},           // Source: models/tools.py: ToolForm, ToolMeta; routers/tools.py: create_new_tools
	"tools update": {"tools update <tool-id>", toolFieldsHelp, toolExample}, // Source: models/tools.py: ToolForm; routers/tools.py: update_tools_by_id
	"tools access-update": {"tools access-update <tool-id>", `  Required JSON object with required access_grants:array of grant objects.
  Replaces grants only, not tool source/metadata.` + nativeGrantsHelp, `{"access_grants":[]}`}, // Source: routers/tools.py: ToolAccessGrantsForm, update_tool_access_by_id
	"tools load-url": {"tools load-url", `  Required JSON object: url:string required, server validates an HTTP(S) URL.
  Server fetches tool source from it. No positional URL or field defaults.`, `{"url":"https://tools.example.com/example.py"}`}, // Source: routers/tools.py: LoadUrlForm, load_tool_from_url; surfaces.go: runTools
	"tools valves update": {"tools valves update <tool-id>", `  Required valve-values JSON object. Keys/types/requiredness/defaults belong
  to the selected tool's resource-defined valve schema. Inspect separately:
  oictl tools valves spec <tool-id>
  Server removes null values and stores explicitly supplied validated values.
  Example is illustrative for a tool with no required valves, not a fixed schema.`, `{}`}, // Source: routers/tools.py: update_tools_valves_by_id; surfaces.go: runValves
	"providers openai config update": {"providers openai config update", openAIConfigHelp, openAIConfigExample}, // Source: routers/openai.py: OpenAIConfigForm, update_config, get_headers_and_cookies
	"providers openai config set": {"providers openai config set", openAIConfigHelp + `
  Existing alias of config update; identical body and replacement semantics.`, openAIConfigExample}, // Source: routers/openai.py: OpenAIConfigForm, update_config; surfaces.go: runProviders
	"providers ollama config update": {"providers ollama config update", ollamaConfigHelp, ollamaConfigExample}, // Source: routers/ollama.py: OllamaConfigForm, update_config
	"providers ollama config set": {"providers ollama config set", ollamaConfigHelp + `
  Existing alias of config update; identical input and replacement semantics.`, ollamaConfigExample}, // Source: routers/ollama.py: OllamaConfigForm, update_config; surfaces.go: runProviders
	"providers openai verify": {"providers openai verify", `  JSON object with required url:string and key:string; optional nullable
  config:object, default null (treated as {}). CLI permits no body; upstream
  requires this verification form, NOT the OPENAI_API_CONFIGS envelope.
` + openAIOptionsHelp, `{"url":"https://provider.example.com/v1","key":"example-key","config":{}}`}, // Source: routers/openai.py: ConnectionVerificationForm, verify_connection
	"providers ollama verify": {"providers ollama verify", `  JSON object with required url:string; key:optional nullable string=null.
  CLI permits no body; upstream requires this connection verification object.
  Supply the connection directly; no OLLAMA_API_CONFIGS envelope.`, `{"url":"https://ollama.example.com","key":null}`}, // Source: routers/ollama.py: ConnectionVerificationForm, verify_connection
	"providers openai request": {"providers openai request <path> [--method METHOD] [--header NAME:VALUE] [--stream]", `  Optional JSON body forwarded to the selected OpenAI-compatible endpoint. Endpoint and
  HTTP method own root shape/fields/defaults; no universal fixed schema.
  --method defaults GET; unlike api, no positional METHOD is accepted.
  --stream enables the existing streaming response path; headers may repeat.
  Illustrative --method POST chat/completions body: required model:string,
  messages:array of message objects with role:string and content:string for this
  simple text example; additional message shapes/options belong to the provider.`, `{"model":"example-model","messages":[{"role":"user","content":"Hello"}],"stream":false}`}, // Source: surfaces.go: runProviders; routers/openai.py: generate_chat_completion (example)
	"providers ollama request": {"providers ollama request <path> [--method METHOD] [--header NAME:VALUE] [--stream]", `  Optional JSON body forwarded to the selected Ollama-compatible endpoint. Endpoint and
  HTTP method own input types/requiredness/defaults, not a universal CLI schema.
  --method defaults GET; no positional METHOD. --stream uses streaming output.
  Illustrative --method POST api/generate body: model:string required;
  prompt:string and stream:boolean are endpoint options (server stream default true).`, `{"model":"example-model","prompt":"Hello","stream":false}`}, // Source: surfaces.go: runProviders; routers/ollama.py: GenerateCompletionForm (example)
	"channels create": {"channels create", channelFieldsHelp + `
  Creation also accepts type:optional nullable string, default null.
  type "group" or "dm" uses user_ids/group_ids to populate membership; dm needs
  user_ids. Standard channels (null type) require admin. Name is lowercased.`, `{"name":"example-channel","description":"Examples","is_private":false}`}, // Source: models/channels.py: CreateChannelForm, insert_new_channel; routers/channels.py: create_new_channel
	"channels update": {"channels update <channel-id>", channelFieldsHelp + `
  Full replacement: omitted name becomes ""; description/is_private/data/meta
  become null. Omitted/null access_grants preserves grants; [] clears them.
  type is creation-only and not changed by this form.`, `{"name":"example-channel","description":"Updated examples","is_private":false}`}, // Source: models/channels.py: ChannelForm, update_channel_by_id; routers/channels.py: update_channel_by_id
	"channels members add": {"channels members add <channel-id>", `  Required JSON object. user_ids and group_ids:optional arrays of strings,
  each server default []. Adds the listed users and members of listed groups;
  omission adds nobody for that selector. No nullable fields in this form.`, `{"user_ids":["user-a"],"group_ids":[]}`}, // Source: routers/channels.py: UpdateMembersForm, add_members_by_id
	"channels members remove": {"channels members remove <channel-id>", `  Required JSON object. user_ids:optional array of strings, server default [].
  Removes listed users; omission does not select all members. No group_ids field.`, `{"user_ids":["user-a"]}`}, // Source: routers/channels.py: RemoveMembersForm, remove_members_by_id
	"channels members active": {"channels members active <channel-id>", `  Required JSON object with is_active:boolean (required, no default).
  CLI accepts active as an alias: copies it only if is_active is absent, then
  removes active. Canonical is_active takes precedence. This changes the current
  user's membership activity, not an arbitrary user identified in the body.`, `{"is_active":true}
{"active":true}`}, // Source: routers/channels.py: UpdateActiveMemberForm; surfaces.go: normalizeChannelMemberActiveBody
	"channels messages post": {"channels messages post <channel-id>", `  Required JSON object with content:string (required).
  Optional nullable strings, default null: temp_id, reply_to_id, parent_id.
  Optional nullable free-form objects, default null: data and meta.
  parent_id selects a thread; reply_to_id selects a quoted message.
  data.files, when supplied, is an array of file objects with id:string.
  Server assigns message identity/timestamps; those are not required input.`, `{"content":"Example message","data":{"files":[]},"parent_id":null}`}, // Source: models/messages.py: MessageForm; routers/channels.py: post_new_message
	"channels reactions add": {"channels reactions add <channel-id> <message-id> [--name NAME]", `  JSON object with required name:string, no default. Body or --name is required.
  With no JSON, --name builds {"name":"..."}; explicit JSON takes precedence.`, `{"name":"thumbsup"}`}, // Source: routers/channels.py: ReactionForm; surfaces.go: runChannelReactions
	"channels reactions remove": {"channels reactions remove <channel-id> <message-id> [--name NAME]", `  JSON object with required name:string, no default. Body or --name is required.
  With no JSON, --name builds {"name":"..."}; explicit JSON takes precedence.`, `{"name":"thumbsup"}`}, // Source: routers/channels.py: ReactionForm; surfaces.go: runChannelReactions
	"webhooks channels create": {"webhooks channels create <channel-id> [--name NAME] [--profile-image-url URL]", channelWebhookHelp + `
  JSON or --name is required for creation.`, `{"name":"Example webhook","profile_image_url":null}`}, // Source: models/channels.py: ChannelWebhookForm; surfaces.go: runWebhookChannels, channelWebhookMutationBody
	"webhooks channels ensure": {"webhooks channels ensure <channel-id> --name NAME [--profile-image-url URL]", channelWebhookHelp + `
  --name is always required to find an existing webhook. JSON is consumed only
  for creation when no name matches; existing matches skip even missing/invalid
  JSON files. No JSON means convenience fields build the creation form.`, `{"name":"Example webhook"}`}, // Source: surfaces.go: ensureChannelWebhook; models/channels.py: ChannelWebhookForm
	"webhooks channels update": {"webhooks channels update <channel-id> <webhook-id> [--name NAME] [--profile-image-url URL]", channelWebhookHelp + `
  Explicit JSON is a replacement form: name remains required by the server,
  omitted image resets to null. Without JSON, convenience updates fetch current
  fields and preserve omitted values. Supply JSON or a convenience field.`, `{"name":"Updated webhook","profile_image_url":null}`}, // Source: surfaces.go: runWebhookChannels, channelWebhookUpdateBody; models/channels.py: ChannelWebhookForm
	"webhooks events create": {"webhooks events create", `  Required JSON object: url:string required; name:optional nullable string,
  default null; enabled:optional boolean, default true.
` + eventWebhookFieldsHelp, `{"name":"Example events","url":"https://hooks.example.com/events","enabled":true,"events":["*"],"targets":[{"type":"user","id":"user-a"}]}`}, // Source: main.py: EventWebhookForm, create_event_webhook; events.py: normalize_event_webhook, normalize_event_targets
	"webhooks events update": {"webhooks events update <webhook-id>", `  Required JSON object. All fields optional nullable, default null:
  name:string, url:string, enabled:boolean, events:array, targets:array.
  Omitted fields retain existing values; explicit null is applied:
  enabled:null becomes false, url:null becomes "", name:null resets the name.
` + eventWebhookFieldsHelp, `{"enabled":false,"events":["*"],"targets":null}`}, // Source: main.py: EventWebhookUpdateForm, update_event_webhook; events.py: normalize_event_webhook
	"groups create":       {"groups create", groupFieldsHelp, `{"name":"Example Team","description":"Example group","permissions":{"workspace":{"models":true}}}`},            // Source: models/groups.py: GroupForm; routers/groups.py: create_new_group
	"groups update":       {"groups update <group-id>", groupFieldsHelp, `{"name":"Example Team","description":"Updated group","permissions":{"workspace":{"models":true}}}`}, // Source: models/groups.py: GroupUpdateForm, update_group_by_id; routers/groups.py: update_group_by_id
	"groups users add":    {"groups users add <group-id> [<user-id>...]", groupMembersHelp, `{"user_ids":["user-a","user-b"]}`},                                               // Source: models/groups.py: UserIdsForm; surfaces.go: runGroupUsers
	"groups users remove": {"groups users remove <group-id> [<user-id>...]", groupMembersHelp, `{"user_ids":["user-a"]}`},                                                     // Source: models/groups.py: UserIdsForm; surfaces.go: runGroupUsers
	"users update": {"users update <user-id>", `  Required JSON object. Optional nullable strings, all default null:
  role, name, email, profile_image_url, password. Omitted/null values preserve
  existing values; empty password also preserves it. Nonempty password changes
  it under server password policy. Email is lowercased; profile URLs validated.
  Role is an unconstrained form string, not a CLI enum.`, `{"name":"Updated Example User","role":"user"}`}, // Source: models/users.py: UserUpdateForm; routers/users.py: update_user_by_id
	"users settings update": {"users settings update [--allow-sensitive-ui-keys]", `  Required JSON settings object. ui:optional nullable free-form object,
  server default {}. ui keys/types belong to the UI version; for example
  theme:string. Server replaces the ui value, not a recursive merge, while
  preserving unrelated stored top-level settings. Omission resets ui to {};
  ui:null stores null. Settings also allow extra top-level keys.
  ui.toolServers requires --allow-sensitive-ui-keys in the CLI.
  Example is illustrative of current-user UI preferences, not a cross-user patch.`, `{"ui":{"theme":"dark"}}`}, // Source: models/users.py: UserSettings, update_user_settings_by_id; routers/users.py: update_user_settings_by_session_user; surfaces.go: userSettingsMutationBody
	"users ui-settings patch": {"users ui-settings patch <user-id> --allow-ui-settings-extension [--allow-sensitive-ui-keys]", uiPatchHelp, `{"theme":"dark"}`}, // Source: surfaces.go: runUsers, userSettingsMutationBody (deployment extension)
	"users ui-settings bulk-patch": {"users ui-settings bulk-patch [<user-id>...] --allow-ui-settings-extension [--user-id ID] [--users-file PATH] [--all | --query TEXT] [--dry-run] [--yes | --confirm]", uiPatchHelp + `
  Targets: positional IDs, repeated --user-id, and --users-file may be combined.
  --users-file contains newline-delimited IDs, NOT JSON; - is a literal filename.
  Alternatively --all or --query discovers targets, mutually exclusive with
  explicit sources; non-dry-run discovery requires --yes or --confirm.
  The same direct UI object is sent to each selected user.`, `{"theme":"dark"}`}, // Source: surfaces.go: runUsersUISettingsBulkPatch, collectBulkUserIDs, bulkUISettingsTargetSourceFor
	"chats import": {"chats import", `  Required JSON object with chats:array (required). Each element requires
  chat:free-form object; optional nullable variables:object=null, folder_id:string=null,
  meta:object={}, pinned:boolean=false, current_message_id:string=null,
  created_at:integer=null, updated_at:integer=null (server timestamps when absent).
  chat is the UI chat document: title:string, models:array of model ID strings,
  history:object with messages:object keyed by message ID and currentId:string|null;
  message objects conventionally carry id, role, content, parentId and childrenIds.
  These UI keys are extensible, not additional form-required fields.
  Import creates records; CLI does not normalize bare arrays or NDJSON exports.`, `{"chats":[{"chat":{"title":"Example chat","models":[],"history":{"messages":{},"currentId":null}},"pinned":false}]}`}, // Source: models/chats.py: ChatForm, ChatImportForm, ChatsImportForm, import_chats; routers/chats.py: import_chats
	"chats compact": {"chats compact <chat-id> [--model MODEL]", `  Optional JSON object: model:optional nullable string, server default null.
  Omitted model uses the server-selected compaction model. Without JSON,
  --model builds {"model":"..."}; explicit JSON wins over --model.`, `{"model":"example-model"}`}, // Source: routers/chats.py: CompactChatForm, compact_chat_by_id; surfaces.go: runChats
	"chats tags set": {"chats tags set <chat-id> [--tag TAG]", `  JSON object with required name:string, no default. Adds one tag (not a list
  replacement). Supply JSON or exactly one --tag to build the name object.
  Explicit JSON takes precedence over --tag.`, `{"name":"example"}`}, // Source: routers/chats.py: TagForm, add_tag_by_id_and_tag_name; surfaces.go: runChatTags
	"chats tags delete": {"chats tags delete <chat-id> [--tag TAG]", `  JSON object with required name:string, no default. Deletes the named tag.
  Supply JSON or exactly one --tag to build the object; explicit JSON wins.`, `{"name":"example"}`}, // Source: routers/chats.py: TagForm, delete_tag_by_id_and_tag_name; surfaces.go: runChatTags
	"automations create": {"automations create", automationHelp, automationExample},                 // Source: models/automations.py: AutomationForm, AutomationData, AutomationTerminalConfig; routers/automations.py: create_new_automation
	"automations update": {"automations update <automation-id>", automationHelp, automationExample}, // Source: models/automations.py: AutomationForm, update_by_id; routers/automations.py: update_automation_by_id
	"scim users create": {"scim users create [--scim-token TOKEN]", `  Required JSON resource object: userName:string, displayName:string,
  emails:array (required). Optional nullable externalId:string, name:object,
  password:string, photos:array, default null; active:optional boolean=true.
  Server uses first email, falling back to userName; active chooses user/pending.
` + scimUserNestedHelp, `{"userName":"user@example.com","displayName":"Example User","emails":[{"value":"user@example.com","primary":true}],"active":true}`}, // Source: routers/scim.py: SCIMUserCreateRequest, SCIMName, SCIMEmail, SCIMPhoto, create_user
	"scim users replace": {"scim users replace <id> [--scim-token TOKEN]", `  Required JSON object; all resource fields are optional nullable, default null:
  id:string, externalId:string, userName:string, name:object, displayName:string,
  emails:array, active:boolean, photos:array. No password field on replacement.
  Despite PUT, this upstream implementation retains omitted/null values; only
  nonempty names/email/photo lists update, while active:false deactivates a user.
` + scimUserNestedHelp, `{"displayName":"Updated Example User","emails":[{"value":"user@example.com"}],"photos":null}`}, // Source: routers/scim.py: SCIMUserUpdateRequest, update_user
	"scim users patch": {"scim users patch <id> [--scim-token TOKEN]", scimPatchHelp + `
  Upstream implements replace only, for active (boolean), userName, displayName,
  emails[primary eq true].value, name.formatted, externalId (string values).
  add/remove and pathless operations are not implemented by this route.`, `{"Operations":[{"op":"replace","path":"active","value":false}]}`}, // Source: routers/scim.py: SCIMPatchRequest, SCIMPatchOperation, patch_user
	"scim groups create": {"scim groups create [--scim-token TOKEN]", `  Required JSON object: displayName:string required; members:optional nullable
  array, server default []. Creates a group with listed membership.
` + scimGroupNestedHelp, `{"displayName":"Example Team","members":[{"value":"user-a"}]}`}, // Source: routers/scim.py: SCIMGroupCreateRequest, SCIMGroupMember, create_group
	"scim groups replace": {"scim groups replace <id> [--scim-token TOKEN]", `  Required JSON object: displayName:optional nullable string=null;
  members:optional nullable array=null. Omitted/null members preserves membership;
  [] clears it; supplied list replaces it. Omitted/empty name preserves the name.
` + scimGroupNestedHelp, `{"displayName":"Example Team","members":[]}`}, // Source: routers/scim.py: SCIMGroupUpdateRequest, update_group
	"scim groups patch": {"scim groups patch <id> [--scim-token TOKEN]", scimPatchHelp + `
  Supported operations: replace displayName with string; add/replace members
  with array of {"value":"user-id"}; remove with path members[value eq "user-id"].
  Omitted fields not targeted by these operations retain existing values.`, `{"Operations":[{"op":"add","path":"members","value":[{"value":"user-a"}]}]}`}, // Source: routers/scim.py: SCIMPatchRequest, SCIMPatchOperation, patch_group
	"auth login": {"auth login [--save]", `  JSON object: required email:string and password:string. CLI permits no body,
  but the server requires both fields. --save saves the returned token;
  selecting --profile also saves it. No fields/defaults are inserted by the CLI.`, `{"email":"user@example.com","password":"example-password"}`}, // Source: models/auths.py: SigninForm; routers/auths.py: signin; app.go: runAuth
	"auth profile update": {"auth profile update", `  JSON object: required name:string, profile_image_url:string.
  Optional nullable bio:string, gender:string, date_of_birth:string (YYYY-MM-DD),
  all default null. Full profile form: omission clears those optional values.
  Server validates profile image URLs. CLI permits no body; upstream requires it.`, `{"name":"Example User","profile_image_url":"/user.png","bio":null,"gender":null,"date_of_birth":null}`}, // Source: models/users.py: UpdateProfileForm; routers/auths.py: update_profile
	"auth password update": {"auth password update", `  JSON object: required password:string (current), new_password:string.
  Server checks current password and password policy. No defaults.
  CLI permits no body; upstream requires both fields.`, `{"password":"old-example-password","new_password":"new-example-password"}`}, // Source: models/auths.py: UpdatePasswordForm; routers/auths.py: update_password
	"auth admin-config set": {"auth admin-config set", adminConfigHelp, adminConfigExample}, // Source: routers/auths.py: AdminConfig, update_admin_config
	"auth ldap-server-config set": {"auth ldap-server-config set", `  JSON object. Required strings: label, host, app_dn, app_dn_password, search_base.
  Optional nullable port:integer, certificate_path:string default null;
  ciphers:nullable string defaults "ALL".
  Optional strings: attribute_for_mail="mail", attribute_for_username="uid",
  search_filters="", attribute_for_groups="memberOf".
  Optional booleans: use_tls=true, validate_cert=true,
  enable_group_management=false, enable_group_creation=false.
  Values above are server defaults on this full replacement form.
  CLI permits no body; upstream requires the object.`, `{"label":"Directory","host":"ldap.example.com","app_dn":"cn=reader,dc=example,dc=com","app_dn_password":"example-password","search_base":"dc=example,dc=com"}`}, // Source: routers/auths.py: LdapServerConfig, update_ldap_server
	"auth ldap-config set": {"auth ldap-config set", `  JSON object: enable_ldap is optional nullable boolean, server default null.
  The endpoint writes the supplied/default value; this is not a merge patch.
  CLI permits no body; upstream requires an object.`, `{"enable_ldap":false}`}, // Source: routers/auths.py: LdapConfigForm, update_ldap_config
	"auth oauth-config set": {"auth oauth-config set", oauthConfigHelp, `{"ENABLE_OAUTH":true,"OAUTH_PROVIDER_NAME":"Example"}`}, // Source: routers/auths.py: OAuthConfigForm, update_oauth_config
	"tasks config set":      {"tasks config set", taskConfigHelp, taskConfigExample},                                             // Source: routers/tasks.py: TaskConfigForm, update_task_config; app.go: runTasks
	"models create": {"models create", modelFieldsHelp + `
  CLI permits no body, but creation needs the model form. A supplied body must
  be a non-null object; omitted params is inserted as {} by the CLI.`, `{"id":"example-model","name":"Example Model","meta":{}}`}, // Source: models/models.py: ModelForm, ModelMeta, ModelParams; routers/models.py: create_new_model; app.go: normalizeModelParamsBody
	"models update": {"models update <model-id>", modelFieldsHelp + modelIDHelp + `
  Full model update: params is required (unlike create, CLI does not add it).
  Server preserves omitted base_model_id and meta.profile_image_url; other meta
  values use form defaults. is_active defaults true. Omitted/null grants preserve
  grants. This is not a general merge patch.`, `{"name":"Example Model","meta":{"description":"Updated"},"params":{}}`}, // Source: routers/models.py: update_model_by_id; models/models.py: update_model_by_id; app.go: bodyWithID
	"models import": {"models import", `  Required array of model objects or object with required models:array.
  CLI wraps a bare array and inserts params:{} in every entry when omitted.
  Import upserts by id, without deleting absent models. Existing entries merge
  metadata and retain unspecified form values, but omitted params becomes {}.
  New entries require id/name; server supplies missing meta:{}.
` + modelFieldsHelp, `[{"id":"example-model","name":"Example Model","meta":{}}]
{"models":[{"id":"example-model","name":"Example Model","meta":{},"params":{}}]}`}, // Source: routers/models.py: ModelsImportForm, import_models; app.go: normalizeModelInventory
	"models sync": {"models sync [--yes | --confirm]", `  Required array or {"models":[...]} envelope; replaces the complete inventory.
  CLI wraps arrays, inserts omitted params:{}, and requires confirmation for [].
  Supply complete model records, NOT create forms.
  Each record requires id, user_id, name:string; params, meta:object;
  created_at, updated_at:integer; is_active:boolean (no default).
  Optional base_model_id:nullable string defaults null. access_grants:array
  record defaults to []; elements are stored grant objects, not null: required
  id, resource_type, resource_id, principal_type, principal_id, permission:string,
  created_at:integer. principal_type is user/group; permission is read/write.
  meta: optional nullable profile_image_url/description:string, capabilities:
  free-form object, knowledge:array of arbitrary values (all default null),
  plus extra keys. meta.tags accepts strings or objects with name:string;
  file knowledge entries use type:"file", id:string. params contains model/provider
  inference options, not a fixed field set. Server rewrites owner/updated_at;
  omission of grants clears them during inventory replacement. Export records
  match this form; the CLI does not invent required record timestamps.`, `[{"id":"example-model","user_id":"user-a","name":"Example Model","meta":{},"params":{},"is_active":true,"created_at":0,"updated_at":0}]
{"models":[{"id":"example-model","user_id":"user-a","name":"Example Model","meta":{},"params":{},"is_active":true,"created_at":0,"updated_at":0}]}`}, // Source: routers/models.py: SyncModelsForm; models/models.py: ModelModel, sync_models; models/access_grants.py: AccessGrantModel; app.go: normalizeModelInventory
	"models toggle": {"models toggle <model-id>", `  Optional JSON object; the upstream form requires only id:string.
  The server toggles existing active state; this is not an is_active setter.` + modelIDHelp, `{}`}, // Source: routers/models.py: ModelIdForm, toggle_model_by_id; app.go: bodyWithID
	"models delete": {"models delete <model-id>", `  Optional JSON object; upstream requires only id:string and deletes that model.` + modelIDHelp, `{}`}, // Source: routers/models.py: ModelIdForm, delete_model_by_id; app.go: bodyWithID, deleteModel
	"models access-update": {"models access-update <model-id>", `  JSON object: required access_grants:array; optional nullable name:string,
  default null, used if the server must create a base-model entry.` + modelIDHelp + nativeGrantsHelp, `{"access_grants":[{"principal_type":"user","principal_id":"*","permission":"read"}]}`}, // Source: routers/models.py: ModelAccessGrantsForm, update_model_access_by_id; app.go: bodyWithID
	"config import": {"config import", `  JSON object with required config:object, a flat configuration key/value map.
  Keys such as ui.default_locale belong to server configuration namespaces;
  values may be any JSON type appropriate to that key. Server upserts supplied
  keys and retains unmentioned keys; this is not a whole-configuration deletion.
  CLI permits no body; upstream requires this envelope.
  Example is illustrative of one UI configuration key.`, `{"config":{"ui.default_locale":"en"}}`}, // Source: routers/configs.py: ImportConfigForm, import_config; models/config.py: Config.upsert
	"config connections set": {"config connections set", `  JSON object: required ENABLE_DIRECT_CONNECTIONS:boolean and
  ENABLE_BASE_MODELS_CACHE:boolean. No defaults; both settings are replaced.
  CLI permits no body; server requires the object.`, `{"ENABLE_DIRECT_CONNECTIONS":false,"ENABLE_BASE_MODELS_CACHE":true}`}, // Source: routers/configs.py: ConnectionsConfigForm, set_connections_config
	"config tool-servers set": {"config tool-servers set", `  JSON object with required TOOL_SERVER_CONNECTIONS:array of connection objects.
  Replaces the connection list; [] clears it. CLI permits no body; server requires it.
` + toolServerFieldsHelp, `{"TOOL_SERVER_CONNECTIONS":[{"url":"https://tools.example.com","path":"/openapi.json","auth_type":"bearer","key":"example-key","config":{}}]}`}, // Source: routers/configs.py: ToolServersConfigForm, ToolServerConnection, set_tool_servers_config
	"config tool-servers verify": {"config tool-servers verify", `  One connection object, NOT a TOOL_SERVER_CONNECTIONS envelope.
  CLI permits no body; server requires the form.
` + toolServerFieldsHelp, `{"url":"https://tools.example.com","path":"/openapi.json","auth_type":"bearer","key":"example-key","config":{}}`}, // Source: routers/configs.py: ToolServerConnection, verify_tool_servers_config
	"config terminal-servers set": {"config terminal-servers set", `  JSON object with required TERMINAL_SERVER_CONNECTIONS:array of connections.
  Replaces the list; [] clears it. CLI permits no body; server requires the form.
` + terminalFieldsHelp, `{"TERMINAL_SERVER_CONNECTIONS":[{"id":"shell-a","name":"Shell","url":"https://terminal.example.com","config":{"access_grants":[]}}]}`}, // Source: routers/configs.py: TerminalServersConfigForm, TerminalServerConnection, set_terminal_servers_config
	"config terminal-servers verify": {"config terminal-servers verify", `  One connection object, not the full configuration envelope.
  CLI permits no body; server requires the form.
` + terminalFieldsHelp, `{"url":"https://terminal.example.com"}`}, // Source: routers/configs.py: TerminalServerConnection, verify_terminal_server_connection
	"config terminal-servers policy": {"config terminal-servers policy", `  JSON object: required url:string, policy_id:string.
  Optional nullable key:string="", auth_type:string="bearer", policy_data:object=null.
  Omitted policy_data or null reads the policy (GET); an object writes it (PUT).
  policy_data is a terminal-orchestrator-defined object. Defaults are server-owned;
  no universal policy fields. CLI permits no body; upstream requires the form.
  Example illustrates an empty policy document.`, `{"url":"https://terminal.example.com","policy_id":"example","policy_data":{}}`}, // Source: routers/configs.py: TerminalServerPolicyForm, put_terminal_server_policy
	"config terminal-servers lifecycle": {"config terminal-servers lifecycle", `  JSON object: required url:string, policy_id:string.
  Optional nullable key:string="", auth_type:string="bearer", lifecycle_data:object=null.
  Omitted lifecycle_data or null reads lifecycle (GET); an object writes it (PUT).
  lifecycle_data is terminal-orchestrator-defined. Defaults are server-owned.
  CLI permits no body; upstream requires the form. Example is illustrative.`, `{"url":"https://terminal.example.com","policy_id":"example","lifecycle_data":{}}`}, // Source: routers/configs.py: TerminalServerLifecycleForm, put_terminal_server_lifecycle
	"config terminal-servers refresh": {"config terminal-servers refresh", `  JSON object: required url:string. Optional nullable strings: key="",
  auth_type="bearer", user_id=null, policy_id=null. Optional booleans:
  only_idle=true, reset=false. Defaults are server-owned; selectors may be omitted.
  CLI permits no body; upstream requires the form.`, `{"url":"https://terminal.example.com","only_idle":true,"reset":false}`}, // Source: routers/configs.py: TerminalServerRefreshForm, refresh_terminal_server_terminals
	"config terminal-servers access-grants diff": {"config terminal-servers access-grants diff <connection-id-or-name>", localGrantsHelp, `[{"principal_type":"group","principal_id":"group-a","permission":"read"}]
{"access_grants":[]}
null`}, // Source: terminal_servers.go: decodeDesiredAccessGrants, runTerminalServerAccessGrants; manifests.go: validateAndNormalizeResolvedAccessGrants
	"config terminal-servers access-grants set": {"config terminal-servers access-grants set <connection-id-or-name>", localGrantsHelp, `[{"principal_type":"user","principal_id":"user-a","permission":"read"}]
{"access_grants":[]}
null`}, // Source: terminal_servers.go: decodeDesiredAccessGrants, runTerminalServerAccessGrants; manifests.go: validateAndNormalizeResolvedAccessGrants
	"config code-execution set": {"config code-execution set", codeConfigHelp, codeConfigExample}, // Source: routers/configs.py: CodeInterpreterConfigForm, set_code_execution_config
	"config models set": {"config models set", `  JSON object. Required nullable strings: DEFAULT_MODELS, DEFAULT_PINNED_MODELS.
  Required MODEL_ORDER_LIST:array of string|null elements.
  Optional nullable free-form objects: DEFAULT_MODEL_METADATA, DEFAULT_MODEL_PARAMS
  (both default null). Inference parameters and metadata keys are model-dependent.
  Full settings form; omitted optional fields use server defaults, not a merge.
  CLI permits no body; upstream requires the object.`, `{"DEFAULT_MODELS":null,"DEFAULT_PINNED_MODELS":null,"MODEL_ORDER_LIST":[],"DEFAULT_MODEL_METADATA":{},"DEFAULT_MODEL_PARAMS":{}}`}, // Source: routers/configs.py: ModelsConfigForm, set_models_config
	"config suggestions": {"config suggestions", `  JSON object with required suggestions:array. Each element has required
  title:array of strings and content:string; no defaults. [] clears suggestions.
  CLI permits no body; upstream requires the object. There is no set subcommand.`, `{"suggestions":[{"title":["Example","Question"],"content":"Explain this topic."}]}`}, // Source: routers/configs.py: PromptSuggestion, SetDefaultSuggestionsForm, set_default_suggestions
	"config banners set": {"config banners set", `  JSON object with required banners:array; [] clears banners. Each banner:
  required id:string, type:string, content:string, dismissible:boolean,
  timestamp:integer; optional nullable title:string defaults null.
  Full replacement list; CLI permits no body, upstream requires the form.`, `{"banners":[{"id":"notice","type":"info","content":"Example notice","dismissible":true,"timestamp":0}]}`}, // Source: routers/configs.py: SetBannersForm, set_banners; config.py: BannerModel
	"config oauth-client register": {"config oauth-client register [--type TYPE]", `  JSON object: required url:string and client_id:string.
  Optional nullable strings, default null: client_name, client_secret,
  oauth_server_url, oauth_scope. --type is a query parameter, not a JSON field.
  CLI permits no body; upstream requires the form.`, `{"url":"https://tools.example.com","client_id":"example-client","oauth_scope":"openid"}`}, // Source: routers/configs.py: OAuthClientRegistrationForm, register_oauth_client
	"files upload": {"files upload <path> [--metadata JSON] [--process BOOL] [--process-in-background BOOL]", `  <path> is raw uploaded file content, not a JSON body file.
  --metadata is optional inline JSON sent unchanged as a multipart text field;
  the CLI does not validate it or read it from a file/stdin. The server expects
  an extensible object: optional file_hash:string (otherwise hashes file bytes),
  channel_id:string (associates upload with an accessible channel),
  knowledge_id:string (auto-links processed file to writable knowledge),
  directory_id:nullable string (directory for that knowledge association), and
  processing metadata such as language:string for transcription.
  Omission uses {} on the server. Stored under file.meta.data, not root file fields.
  Server process and process_in_background query defaults are true.
  Example illustrates metadata, not file bytes.`, `{"file_hash":"example-checksum","language":"en"}`}, // Source: app.go: multipartFileBody; routers/files.py: upload_file_handler, process_uploaded_file
	"files update-content": {"files update-content <file-id>", `  JSON object with required content:string (new text, including empty string).
  Replaces stored extracted content. No field defaults. CLI permits no body;
  the server requires the object.`, `{"content":"Updated file text"}`}, // Source: routers/files.py: ContentForm, update_file_data_content_by_id
	"knowledge create":           {"knowledge create", knowledgeFieldsHelp, `{"name":"Example Knowledge","description":"Reference documents"}`},                        // Source: models/knowledge.py: KnowledgeForm; routers/knowledge.py: create_new_knowledge
	"knowledge update":           {"knowledge update <knowledge-id>", knowledgeFieldsHelp, `{"name":"Example Knowledge","description":"Updated reference documents"}`}, // Source: models/knowledge.py: KnowledgeForm, update_knowledge_by_id; routers/knowledge.py: update_knowledge_by_id
	"knowledge reindex":          {"knowledge reindex", optionalForwardedHelp, `{}`},                                                                                   // Source: routers/knowledge.py: reindex_knowledge_files; app.go: runKnowledge
	"knowledge metadata-reindex": {"knowledge metadata-reindex", optionalForwardedHelp, `{}`},                                                                          // Source: routers/knowledge.py: reindex_knowledge_base_metadata_embeddings; app.go: runKnowledge
	"knowledge get":              {"knowledge get <knowledge-id>", optionalForwardedHelp, `{}`},                                                                        // Source: routers/knowledge.py: get_knowledge_by_id; app.go: runKnowledge
	"knowledge export":           {"knowledge export <knowledge-id>", optionalForwardedHelp, `{}`},                                                                     // Source: routers/knowledge.py: export_knowledge_by_id; app.go: runKnowledge
	"knowledge delete":           {"knowledge delete <knowledge-id>", optionalForwardedHelp, `{}`},                                                                     // Source: routers/knowledge.py: delete_knowledge_by_id; app.go: runKnowledge
	"knowledge reset":            {"knowledge reset <knowledge-id>", optionalForwardedHelp, `{}`},                                                                      // Source: routers/knowledge.py: reset_knowledge_by_id; app.go: runKnowledge
	"knowledge access-update": {"knowledge access-update <knowledge-id>", `  JSON object with required access_grants:array of grant objects.
  CLI permits no body; upstream requires this replacement form.` + nativeGrantsHelp, `{"access_grants":[]}`}, // Source: routers/knowledge.py: KnowledgeAccessGrantsForm, update_knowledge_access_by_id
	"knowledge files update": {"knowledge files update <knowledge-id>", knowledgeFileHelp, `{"file_id":"file-a"}`}, // Source: routers/knowledge.py: KnowledgeFileIdForm, update_file_from_knowledge_by_id
	"knowledge files remove": {"knowledge files remove <knowledge-id>", knowledgeFileHelp, `{"file_id":"file-a"}`}, // Source: routers/knowledge.py: KnowledgeFileIdForm, remove_file_from_knowledge_by_id
	"knowledge files move": {"knowledge files move <knowledge-id>", `  JSON object with required file_id:string and optional nullable
  directory_id:string (server default null moves file to root).
  CLI permits no body; upstream requires the object.`, `{"file_id":"file-a","directory_id":null}`}, // Source: routers/knowledge.py: KnowledgeFileMoveForm, move_file_in_knowledge
	"knowledge files batch-add": {"knowledge files batch-add <knowledge-id>", `  JSON array (not {"file_ids":[...]}). Each element requires file_id:string;
  directory_id is optional nullable string, default null (root directory).
  Existing associations are skipped. CLI permits no body; server requires array.`, `[{"file_id":"file-a","directory_id":null}]`}, // Source: routers/knowledge.py: KnowledgeFileIdForm, add_files_to_knowledge_batch
	"knowledge dirs create": {"knowledge dirs create <knowledge-id>", `  JSON object: required name:string; optional nullable parent_id:string,
  server default null (root). CLI permits no body; server requires the form.`, `{"name":"Examples","parent_id":null}`}, // Source: routers/knowledge.py: KnowledgeDirectoryCreateForm, create_knowledge_directory
	"knowledge dirs update": {"knowledge dirs update <knowledge-id> <dir-id>", `  JSON object. name:optional nullable string defaults null (no rename);
  parent_id:optional nullable string defaults "__unset__" (no move).
  Explicit parent_id:null moves to root; a directory ID moves beneath it.
  CLI permits no body; upstream requires an object.`, `{"name":"Updated Examples","parent_id":null}`}, // Source: routers/knowledge.py: KnowledgeDirectoryUpdateForm, update_knowledge_directory; models/knowledge.py: update_directory
	"knowledge dirs delete": {"knowledge dirs delete <knowledge-id> <dir-id>", optionalForwardedHelp + `
  Upstream move_files is a query parameter with default true, not a JSON field.`, `{}`}, // Source: routers/knowledge.py: delete_knowledge_directory; app.go: runKnowledgeDirs
	"knowledge sync diff": {"knowledge sync diff <knowledge-id>", `  JSON object with required manifest:array. Each file entry requires
  filename:string, path:string, checksum:string, size:integer. No defaults.
  This is a file inventory, not the declarative oictl manifest envelope.
  CLI permits no body; upstream requires this object. Diff does not upload files.`, `{"manifest":[{"filename":"example.txt","path":"docs/example.txt","checksum":"example-checksum","size":12}]}`}, // Source: routers/knowledge.py: FileManifestEntry, SyncDiffForm, sync_knowledge_diff; app.go: runKnowledgeSync
	"knowledge sync cleanup": {"knowledge sync cleanup <knowledge-id>", `  JSON object: required file_ids:array of strings; optional dir_ids:array
  of strings, server default []. Removes listed stale files/directories.
  CLI permits no body; upstream requires this object.`, `{"file_ids":["file-a"],"dir_ids":[]}`}, // Source: routers/knowledge.py: SyncCleanupForm, sync_knowledge_cleanup; app.go: runKnowledgeSync
	"channels messages update": {
		"channels messages update <channel-id> <message-id>",
		`  Required JSON object with content: string (required).
  Optional nullable strings, server default null: temp_id, reply_to_id, parent_id.
  Optional nullable free-form objects, default null: data, meta.
  data.files, when supplied, is an array of file objects with id: string.
  Update replaces content; data/meta shallow-merge with stored objects.
  Omitted/null data/meta preserve existing keys, rather than clearing them.
  temp_id and reply/thread selectors are used on posting, not changed by update.`,
		`{"content":"Updated message","data":{"files":[]}}`,
	}, // Source: models/messages.py: MessageForm, update_message_by_id; routers/channels.py: update_message_by_id
	"tools valves user update": {
		"tools valves user update <tool-id>",
		`  Required JSON valve-values object. Keys and value types are resource-defined
  by the selected tool's user-valve schema, not a universal CLI schema.
  Inspect separately: oictl tools valves user spec <tool-id>
  The server removes null values and stores explicitly supplied validated values;
  defaults and required fields belong to that resource. No positional user ID.
  Example is illustrative for a resource with no required user valves.`,
		`{}`,
	}, // Source: routers/tools.py: update_tools_user_valves_by_id; surfaces.go: runValves
	"users create": {
		"users create",
		`  JSON object. Required strings: name, email, password.
  Optional nullable strings: role (server default "pending"),
  profile_image_url (server default "/user.png"). Role is a string, not a CLI enum.
  The server validates email, password policy and profile image URLs.
  A body source is required; the CLI does not insert these server defaults.`,
		`{"name":"Example User","email":"user@example.com","password":"example-password"}`,
	}, // Source: models/auths.py: SignupForm, AddUserForm; routers/auths.py: add_user
}

// Scan only syntax: never read inputs or resolve configuration. Like the existing
// parser, scalar flags consume the next token even when it looks like help.
func jsonHelpPositionals(args []string) ([]string, bool) {
	var pos []string
	help := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--help" || arg == "-h" {
			help = true
			continue
		}
		if arg == "--version" && len(pos) == 0 {
			pos = append(pos, arg)
			continue
		}
		name, _, hasValue := splitFlag(arg)
		if name == "" {
			pos = append(pos, arg)
			continue
		}
		if hasValue {
			continue
		}
		switch name {
		case "save", "yes", "dry-run", "confirm", "stream", "include-valves", "allow-sensitive-ui-keys", "show-url", "verify-url", "external", "all", "all-users", "allow-ui-settings-extension":
			continue
		case "manifest":
			if len(pos) >= 2 && pos[0] == "skills" && (pos[1] == "create" || pos[1] == "update") {
				i++
				continue
			}
			if (len(pos) >= 2 && pos[0] == "tools" && pos[1] == "export") || i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				continue
			}
		}
		i++
	}
	return pos, help
}

func (a *App) printJSONInputHelp(args []string) bool {
	pos, help := jsonHelpPositionals(args)
	if !help {
		return false
	}
	for n := len(pos); n > 0; n-- {
		ref, ok := jsonInputReferences[strings.Join(pos[:n], " ")]
		if !ok {
			continue
		}
		fmt.Fprintf(a.out, "Usage:\n  oictl %s\n\nInput:\n%s\n", ref.usage, ref.input)
		if pos[0] != "manifests" && !(pos[0] == "files" && n > 1 && pos[1] == "upload") {
			fmt.Fprint(a.out, jsonBodySources)
		}
		fmt.Fprintf(a.out, "\nExamples:\n")
		for _, example := range strings.Split(ref.examples, "\n") {
			fmt.Fprintf(a.out, "  %s\n", example)
		}
		return true
	}
	return false
}
