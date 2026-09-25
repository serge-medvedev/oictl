package cli

// Authored fragments shared only by help entries with the same input contract.
const nativeGrantsHelp = `
  access_grants elements are objects: principal_type: "user" or "group",
  principal_id: string ("*" for public user access), permission: "read" or "write".
  Native API bodies use IDs, not local principal_ref/name/email selectors.
  [] replaces grants with an empty list; server sharing permissions still apply.`

const localGrantsHelp = `  Required input: bare grant array or {"access_grants":[...]}.
  null (including access_grants:null) is decoded as an empty list by the CLI.
  Each grant requires principal_type:"user"|"group", principal_id:nonempty string,
  permission:"read"|"write". principal_ref, principal_email and principal_name
  are not accepted here (only declarative manifests resolve those selectors).
  Duplicate grants are normalized.
  [] explicitly clears grants; diff compares only, set replaces target grants
  while retaining the remaining terminal-server configuration.`

const modelFieldsHelp = `  Model object: required id: string, name: string, meta: object,
  params: object. Creation/import IDs must be nonempty and at most 256 characters.
  base_model_id: optional nullable string, server default null;
  is_active: optional boolean, server default true; access_grants: optional array
  (server omission default null; elements are grant objects or null).
  meta: optional nullable profile_image_url and description strings,
  capabilities: optional nullable free-form object; knowledge: optional nullable
  array of arbitrary values (file entries use type:"file" and id:string).
  All four meta defaults are null; additional metadata keys are accepted.
  meta.tags may be strings or objects with name:string; server normalizes strings.
  params has provider/model-defined inference keys (for example temperature:number),
  with no fixed required fields. These are server contracts, not CLI validation.` + nativeGrantsHelp

const modelIDHelp = `
  CLI inserts the positional model ID into the object; a conflicting id fails.
  No body means {"id":<model-id>}; supply other fields required by the endpoint.`

const toolServerFieldsHelp = `  Connection object: required url, path: strings; required but nullable
  auth_type, key: strings and config: object. Optional nullable type:string
  defaults to "openapi"; headers: object|string defaults null; info: object
  defaults null. config and info are extensible server-owned maps.
  config supports enable:boolean and access_grants:array of grants;
  info supports id, name, description, oauth_server_url, oauth_client_id,
  oauth_client_secret, oauth_scope (strings). These keys depend on connection type.
  headers object maps header names to values; type "mcp" selects MCP handling.` + nativeGrantsHelp

const terminalFieldsHelp = `  Connection object: required url:string. Optional nullable strings:
  id and name (default ""), path (default "/openapi.json"), key (default ""),
  auth_type (default "bearer"), server_type and policy_id (default null).
  enabled: optional nullable boolean, default true; config: optional nullable
  extensible object, default null. config.access_grants is an array of grants;
  other config keys belong to the terminal server. Defaults are server-owned.` + nativeGrantsHelp

const optionalForwardedHelp = `  Optional forwarded JSON body. The identified upstream route defines
  no request body and does not interpret JSON fields; no fields are required.
  The CLI still forwards a supplied body unchanged, including on GET/DELETE.
  Extensions may interpret it; their contract belongs to the selected endpoint.
  The empty object below illustrates forwarding, not an upstream requirement.`

const knowledgeFieldsHelp = `  JSON object: required name and description strings.
  access_grants: optional nullable array of grant objects, default null.
  Create uses server defaults. Update replaces name and description;
  omitted/null access_grants leaves existing grants unchanged in the server.
  CLI allows an absent body, but this upstream form requires an object.` + nativeGrantsHelp

const knowledgeFileHelp = `  JSON object: required file_id:string; optional nullable directory_id:string
  (default null). Identity is in JSON, not another positional file ID.
  Update reindexes that existing file; remove removes that association;
  directory_id is used by add/batch-add, not update/remove.
  CLI allows an absent body, but this upstream form requires an object.`

const adminConfigHelp = `  JSON object. Required booleans: SHOW_ADMIN_DETAILS, ENABLE_SIGNUP,
  ENABLE_API_KEYS, ENABLE_API_KEYS_ENDPOINT_RESTRICTIONS, ENABLE_COMMUNITY_SHARING,
  ENABLE_MESSAGE_RATING, ENABLE_FOLDERS, ENABLE_AUTOMATIONS, ENABLE_CHANNELS,
  ENABLE_CALENDAR, ENABLE_MEMORIES, ENABLE_MEMORY_SYSTEM_CONTEXT, ENABLE_NOTES,
  ENABLE_USER_WEBHOOKS, ENABLE_USER_STATUS.
  Required strings: WEBUI_URL, API_KEYS_ALLOWED_ENDPOINTS, DEFAULT_USER_ROLE,
  DEFAULT_GROUP_ID, JWT_EXPIRES_IN.
  Optional nullable strings, default null: ADMIN_EMAIL, PENDING_USER_OVERLAY_TITLE,
  PENDING_USER_OVERLAY_CONTENT, RESPONSE_WATERMARK.
  Optional nullable integer|string values, default null: FOLDER_MAX_FILE_COUNT,
  AUTOMATION_MAX_COUNT, AUTOMATION_MIN_INTERVAL.
  DEFAULT_INTERFACE_SETTINGS: optional nullable free-form UI object, default null.
  CHANNEL_MODEL_RESPONSE_MODE: optional string, default "thread".
  Accepted DEFAULT_USER_ROLE: "pending"|"user"|"admin";
  CHANNEL_MODEL_RESPONSE_MODE: "thread"|"channel".
  JWT_EXPIRES_IN accepts "-1", "0", or a decimal number with an optional minus sign
  followed by ms, s, m, h, d, or w (for example "4w", "1.5h", "-1s"). Server pattern:
    ^(-1|0|(-?\d+(\.\d+)?)(ms|s|m|h|d|w))$
  For an unsupported role, mode, or expiry, the server retains the existing setting
  for that field while applying the other fields. CLI does not validate these values.
  This is a full configuration form, not a merge patch: omitted optional fields
  take server defaults, subject to the preservation rules above.
  CLI permits no body, but the server requires this form.`

const adminConfigExample = `{"SHOW_ADMIN_DETAILS":false,"WEBUI_URL":"https://webui.example.com","ENABLE_SIGNUP":false,"ENABLE_API_KEYS":true,"ENABLE_API_KEYS_ENDPOINT_RESTRICTIONS":false,"API_KEYS_ALLOWED_ENDPOINTS":"","DEFAULT_USER_ROLE":"pending","DEFAULT_GROUP_ID":"","JWT_EXPIRES_IN":"4w","ENABLE_COMMUNITY_SHARING":false,"ENABLE_MESSAGE_RATING":true,"ENABLE_FOLDERS":true,"ENABLE_AUTOMATIONS":false,"ENABLE_CHANNELS":true,"ENABLE_CALENDAR":true,"ENABLE_MEMORIES":true,"ENABLE_MEMORY_SYSTEM_CONTEXT":false,"ENABLE_NOTES":true,"ENABLE_USER_WEBHOOKS":false,"ENABLE_USER_STATUS":true}`

const oauthConfigHelp = `  JSON object. Every field is optional and nullable (default null).
  Boolean fields: ENABLE_OAUTH, ENABLE_OAUTH_SIGNUP, OAUTH_MERGE_ACCOUNTS_BY_EMAIL,
  OAUTH_AUTO_REDIRECT, ENABLE_OAUTH_ROLE_MANAGEMENT, ENABLE_OAUTH_GROUP_MANAGEMENT,
  ENABLE_OAUTH_GROUP_CREATION, OAUTH_UPDATE_EMAIL_ON_LOGIN,
  OAUTH_UPDATE_NAME_ON_LOGIN, OAUTH_UPDATE_PICTURE_ON_LOGIN,
  OAUTH_REFRESH_TOKEN_INCLUDE_SCOPE.
  String fields: OAUTH_ALLOWED_DOMAINS, OAUTH_BLOCKED_GROUPS, OAUTH_ROLES_CLAIM,
  OAUTH_ADMIN_ROLES, OAUTH_ALLOWED_ROLES, OAUTH_GROUP_CLAIM, OAUTH_PROVIDER_NAME,
  OPENID_PROVIDER_URL, OAUTH_CLIENT_ID, OAUTH_CLIENT_SECRET, OPENID_REDIRECT_URI,
  OAUTH_SCOPES, OAUTH_CODE_CHALLENGE_METHOD, OAUTH_TOKEN_ENDPOINT_AUTH_METHOD,
  OPENID_END_SESSION_ENDPOINT, OAUTH_EMAIL_CLAIM, OAUTH_USERNAME_CLAIM,
  OAUTH_PICTURE_CLAIM, OAUTH_SUB_CLAIM, OAUTH_AUDIENCE.
  OAUTH_GROUP_DEFAULT_SHARE: boolean|string|null.
  OAUTH_TIMEOUT, OAUTH_CLIENT_TIMEOUT: integer|string|null.
  Omitted/null fields are not updated; server configuration owns existing values.
  CLI permits an absent body; the upstream route requires an object.`

const taskConfigHelp = `  Required JSON object (full configuration replacement).
  Required nullable strings: TASK_MODEL, TASK_MODEL_EXTERNAL, VOICE_MODE_PROMPT_TEMPLATE.
  Required booleans: ENABLE_TITLE_GENERATION, ENABLE_AUTOCOMPLETE_GENERATION,
  ENABLE_FOLLOW_UP_GENERATION, ENABLE_TAGS_GENERATION, ENABLE_SEARCH_QUERY_GENERATION,
  ENABLE_RETRIEVAL_QUERY_GENERATION, ENABLE_VOICE_MODE_PROMPT.
  Required integer: AUTOCOMPLETE_GENERATION_INPUT_MAX_LENGTH.
  Required strings: TITLE_GENERATION_PROMPT_TEMPLATE, IMAGE_PROMPT_GENERATION_PROMPT_TEMPLATE,
  AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE, TAGS_GENERATION_PROMPT_TEMPLATE,
  FOLLOW_UP_GENERATION_PROMPT_TEMPLATE, QUERY_GENERATION_PROMPT_TEMPLATE,
  TOOLS_FUNCTION_CALLING_PROMPT_TEMPLATE.
  TASK_MODEL_PARAMS: optional nullable free-form inference object, default null.
  Required fields have no server form defaults; nullability does not mean optional.
  TASK_MODEL/TASK_MODEL_EXTERNAL select task models, not positional resource IDs.`

const taskConfigExample = `{"TASK_MODEL":null,"TASK_MODEL_EXTERNAL":null,"ENABLE_TITLE_GENERATION":true,"TITLE_GENERATION_PROMPT_TEMPLATE":"","IMAGE_PROMPT_GENERATION_PROMPT_TEMPLATE":"","ENABLE_AUTOCOMPLETE_GENERATION":false,"AUTOCOMPLETE_GENERATION_INPUT_MAX_LENGTH":2048,"AUTOCOMPLETE_GENERATION_PROMPT_TEMPLATE":"","TAGS_GENERATION_PROMPT_TEMPLATE":"","FOLLOW_UP_GENERATION_PROMPT_TEMPLATE":"","ENABLE_FOLLOW_UP_GENERATION":false,"ENABLE_TAGS_GENERATION":false,"ENABLE_SEARCH_QUERY_GENERATION":false,"ENABLE_RETRIEVAL_QUERY_GENERATION":false,"QUERY_GENERATION_PROMPT_TEMPLATE":"","TOOLS_FUNCTION_CALLING_PROMPT_TEMPLATE":"","ENABLE_VOICE_MODE_PROMPT":false,"VOICE_MODE_PROMPT_TEMPLATE":null}`

const codeConfigHelp = `  JSON object, full configuration form. All fields below are required.
  Booleans: ENABLE_CODE_EXECUTION, ENABLE_CODE_INTERPRETER.
  Strings: CODE_EXECUTION_ENGINE, CODE_INTERPRETER_ENGINE.
  Nullable strings: CODE_EXECUTION_JUPYTER_URL, CODE_EXECUTION_JUPYTER_AUTH,
  CODE_EXECUTION_JUPYTER_AUTH_TOKEN, CODE_EXECUTION_JUPYTER_AUTH_PASSWORD,
  CODE_INTERPRETER_PROMPT_TEMPLATE, CODE_INTERPRETER_JUPYTER_URL,
  CODE_INTERPRETER_JUPYTER_AUTH, CODE_INTERPRETER_JUPYTER_AUTH_TOKEN,
  CODE_INTERPRETER_JUPYTER_AUTH_PASSWORD.
  Nullable integers: CODE_EXECUTION_JUPYTER_TIMEOUT, CODE_INTERPRETER_JUPYTER_TIMEOUT.
  No form defaults; nullable fields still must be present. CLI permits no body,
  but the upstream endpoint requires this complete object.`

const channelFieldsHelp = `  Required JSON object. name:optional string, server default "".
  Optional nullable fields, default null: description:string, is_private:boolean,
  data:free-form object, meta:free-form object, access_grants:array of grants,
  group_ids:array of strings, user_ids:array of strings.
  Membership arrays are for group/dm creation; update does not change membership.` + nativeGrantsHelp

const channelWebhookHelp = `  JSON object: required name:string (CLI creation requires nonempty);
  optional nullable profile_image_url:string, server default null.
  --name NAME and --profile-image-url URL are convenience inputs.
  Explicit JSON takes precedence, not a merge with convenience fields.
  The server validates profile image URLs.`

const eventWebhookFieldsHelp = `  events:optional nullable array of strings; null/[] normalizes to ["*"].
  Filters are "*", catalog event names, or supported prefix patterns ending ".*".
  targets:optional nullable array of objects with required type:"user"|"group"
  and nonempty id:string. null means unfiltered;
  [] matches events with no associated users.
  Duplicate targets are deduplicated. Name null/empty normalizes to "Webhook".`

const groupFieldsHelp = `  Required JSON object: name:string, description:string (both required).
  permissions and data:optional nullable free-form objects, server default null.
  permissions is a nested capability map (for example workspace.models:boolean,
  sharing.public_models:boolean, features.web_search:boolean); configuration owns
  additional permission keys. On update name/description are replaced, while
  omitted/null permissions and data are excluded and retain existing values.`

const groupMembersHelp = `  JSON object: user_ids:optional nullable array of strings, default null.
  Supply an array of IDs to add/remove membership. Omission/null is not an
  all-users selector. Alternatively provide positional <user-id>... with no body;
  CLI builds {"user_ids":[...]}. Explicit JSON supersedes positional user IDs.`

const uiPatchHelp = `  Required direct JSON UI object, NOT {"ui":{...}}. Keys/value types belong
  to the deployment UI settings extension (for example theme:string).
  This cross-user route is a deployment extension, not standard upstream;
  --allow-ui-settings-extension is required even for previews.
  A top-level toolServers key requires --allow-sensitive-ui-keys.
  Merge/null/default semantics belong to that extension; CLI sends JSON unchanged.
  Example is illustrative, not a universal UI schema.`

const automationHelp = `  Required JSON object: name:string and data:object (required).
  data requires prompt:string, model_id:string, rrule:string (recurrence rule).
  data.terminal:optional nullable object, default null; when present requires
  server_id:string, with optional nullable cwd:string (default null).
  folder_id:optional nullable string, meta:optional nullable free-form object
  (both default null); is_active:optional nullable boolean, default true.
  Server validates the recurrence and scheduling limits. Full-form update:
  omitted optional fields take server defaults; data is replaced, not patched.
  On update only, is_active:null preserves current active state; omission sets true.
  Creation's persisted active field is non-null, so use a boolean or omit it.`

const automationExample = `{"name":"Example schedule","data":{"prompt":"Summarize the day","model_id":"example-model","rrule":"FREQ=DAILY;BYHOUR=9;BYMINUTE=0;BYSECOND=0","terminal":{"server_id":"shell-a","cwd":"/tmp"}},"is_active":false}`

const scimUserNestedHelp = `  name:nullable object whose optional nullable string fields default null:
  formatted, familyName, givenName, middleName, honorificPrefix, honorificSuffix.
  emails elements: required value:string; optional nullable type:string="work",
  optional primary:boolean=true, optional nullable display:string=null.
  photos elements: required value:string; optional nullable type:string="photo",
  optional primary:boolean=true, optional nullable display:string=null.
  schemas:optional string array defaults ["urn:ietf:params:scim:schemas:core:2.0:User"].
  These are the inspected upstream fields; arbitrary extension attributes are
  not interpreted by its form. Other deployments may define extensions.`

const scimGroupNestedHelp = `  members elements: required value:string (user ID); optional nullable
  $ref:string=null, type:string="User", display:string=null.
  schemas:optional string array defaults ["urn:ietf:params:scim:schemas:core:2.0:Group"].
  Additional deployment extensions are not interpreted by this upstream form.`

const scimPatchHelp = `  Required JSON object with Operations:array (required, case-sensitive).
  schemas:optional string array defaults ["urn:ietf:params:scim:api:messages:2.0:PatchOp"].
  Each operation requires op:string; path:optional nullable string, default null;
  value:optional arbitrary JSON, default null. Operation names are case-insensitive.
  This is a SCIM operation envelope, not JSON Patch or a resource document.
  The CLI forwards it unchanged; unsupported paths/operations are ignored upstream.`

const functionFieldsHelp = `  Required JSON object: id, name, content are required strings; meta is a
  required object with optional nullable description:string=null and
  manifest:free-form object={} (extra metadata keys accepted).
  content is Python function source defining the appropriate Pipe/Filter/Action
  class; server loads it and replaces meta.manifest with parsed source frontmatter.
  Update still requires body id (not injected by CLI), but route ID selects target.
  Creation requires a Python identifier as id; server lowercases it.
  Name/content/meta are replaced; active/global state uses separate toggle commands.`

const toolFieldsHelp = `  Required JSON object: id, name, content are required strings; meta is a
  required object: optional nullable description:string=null, manifest:object={},
  optional has_user_valves:boolean=false. manifest is a free-form source metadata map.
  Creation requires a Python identifier as id; server lowercases it.
  content is Python source with a Tools class; server loads it and derives tool
  specs, meta.manifest and has_user_valves. access_grants:optional array of
  grant objects or null elements, omission default null.
  Update requires body id (CLI does not inject it); route ID selects target.
  Source/name/meta use full-form update, not a merge; null grants preserve grants.` + nativeGrantsHelp

const skillFieldsHelp = `  Required JSON object: id, name, content are required strings.
  description:optional nullable string=null; meta:optional object defaults {"tags":[]};
  meta.tags:optional nullable array of strings, default []; is_active:optional
  boolean=true; access_grants:optional nullable array of grant objects=null.
  On creation the server lowercases id and replaces spaces with hyphens.
  content is Markdown text. Update still requires body id (not injected by CLI);
  name/content/meta/description/is_active replace existing values with form defaults
  for omitted optionals; omitted/null grants preserve grants.
  Alternative: --manifest PATH reads Markdown with limited frontmatter, not JSON.
  Supported frontmatter: id/name/description strings, is_active boolean, tags list;
  body becomes content. Complex access_grants requires JSON. Choose one input source.
  --manifest - is a literal filename; boolean --manifest supplies no payload.` + nativeGrantsHelp

const openAIConfigHelp = `  Required JSON object: OPENAI_API_BASE_URLS:array of strings,
  OPENAI_API_KEYS:array of strings, OPENAI_API_CONFIGS:object (all required).
  ENABLE_OPENAI_API:optional nullable boolean, default null (written as such).
  Full configuration replacement. Server pads/truncates keys to match URL count
  and discards config entries whose keys are not URL indices ("0", "1", ...).
  Each indexed config is a provider-owned extensible options object.
` + openAIOptionsHelp

const openAIOptionsHelp = `  Known optional options: enable:boolean (reader default true),
  prefix_id:string (default ""), model_ids:array of strings (default []),
  tags:array (default []), headers:object mapping header names to values,
  auth_type:string (default "bearer"), azure:boolean, provider:string,
  api_version:string, api_type:string, connection_type:string.
  Provider/auth mode determines applicable options; there is no fixed universal
  required option set. Azure legacy verification defaults api_version to
  "2023-03-15-preview" when empty. CLI neither validates nor inserts these values.`

const ollamaConfigHelp = `  Required JSON object: OLLAMA_BASE_URLS:array of strings and
  OLLAMA_API_CONFIGS:object. ENABLE_OLLAMA_API:optional nullable boolean, default null.
  Full replacement, not merge. Config keys are URL indices ("0", "1", ...);
  server discards entries outside URL count. Each value is an extensible
  connection-options object: optional enable:boolean (reader default true),
  key:string, headers:object of header values, prefix_id:string (default ""),
  model_ids:array of strings (default []), tags:array (default []),
  connection_type:string. These options belong to the provider integration;
  CLI forwards them without inserting defaults.`

const functionExample = `{"id":"example_function","name":"Example Function","content":"class Pipe:\n    def pipe(self, body: dict):\n        return 'Example'\n","meta":{}}`
const toolExample = `{"id":"example_tool","name":"Example Tool","content":"class Tools:\n    def example(self) -> str:\n        return 'Example'\n","meta":{}}`
const skillExample = `{"id":"example-skill","name":"Example Skill","content":"# Example\nDescribe the result.","meta":{"tags":["example"]}}`
const openAIConfigExample = `{"ENABLE_OPENAI_API":true,"OPENAI_API_BASE_URLS":["https://provider.example.com/v1"],"OPENAI_API_KEYS":["example-key"],"OPENAI_API_CONFIGS":{"0":{"enable":true,"model_ids":["example-model"]}}}`
const ollamaConfigExample = `{"ENABLE_OLLAMA_API":true,"OLLAMA_BASE_URLS":["https://ollama.example.com"],"OLLAMA_API_CONFIGS":{"0":{"enable":true}}}`

const codeConfigExample = `{"ENABLE_CODE_EXECUTION":false,"CODE_EXECUTION_ENGINE":"pyodide","CODE_EXECUTION_JUPYTER_URL":null,"CODE_EXECUTION_JUPYTER_AUTH":null,"CODE_EXECUTION_JUPYTER_AUTH_TOKEN":null,"CODE_EXECUTION_JUPYTER_AUTH_PASSWORD":null,"CODE_EXECUTION_JUPYTER_TIMEOUT":null,"ENABLE_CODE_INTERPRETER":false,"CODE_INTERPRETER_ENGINE":"pyodide","CODE_INTERPRETER_PROMPT_TEMPLATE":null,"CODE_INTERPRETER_JUPYTER_URL":null,"CODE_INTERPRETER_JUPYTER_AUTH":null,"CODE_INTERPRETER_JUPYTER_AUTH_TOKEN":null,"CODE_INTERPRETER_JUPYTER_AUTH_PASSWORD":null,"CODE_INTERPRETER_JUPYTER_TIMEOUT":null}`
