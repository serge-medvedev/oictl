package cli

// Local envelope contract: manifests.go loadManifestDocuments, manifestHandlers,
// manifestPayload, validateManifestDocument/Plan, connectionFromManifest,
// modelPayload and principalResolver. Native forms at the upstream revision
// identified in json_help.go supply the fields carried inside spec.
const manifestInputHelp = `  One JSON object per file (YAML .yaml/.yml also accepted), not an array,
  JSONL, or multi-document stream. Required envelope fields:
    apiVersion: string, exactly "oictl.openwebui/v1"
    kind: string, one of the kind names below (case-sensitive)
    metadata: object with required nonempty name:string (resource identity)
    spec: non-null object, kind-specific desired fields described below
    access_grants: optional top-level array of grant objects (or null)
  Sources: --file PATH[,PATH...] and --directory DIR[,DIR...] (alias --dir).
  Files/directories may be combined; directories recursively discover .json,
  .yaml and .yml paths, sorted before reading. Repeated source flags retain the
  last value; comma-separated paths select multiple files/directories.
  --file - is a literal filename, NOT stdin. No --data or --data-file here.
  spec.content_file: optional nonempty path string, mutually exclusive with content;
  reads UTF-8 text relative to the manifest (or an absolute path), not JSON.
  Simple ${VAR} substitution applies inside parsed string values; missing variables
  fail. Substituted strings are not reparsed as JSON. Duplicate kind/name identities
  fail. IDs are matched before names; ambiguous names fail. Not every kind can
  create a resource from an empty spec or preserve every omitted field.

  Access grants:
    Top-level grants are supported by Channel, Knowledge, Model, Prompt, Skill,
    TerminalServerConnection and Tool only. Omission leaves grants unmanaged;
    [] or null explicitly clears them. Each grant requires principal_type:
    "user"|"group", permission:"read"|"write", and exactly one nonempty string
    selector: principal_id, principal_ref, principal_email (user only), principal_name.
    principal_ref may be user:selector or group:selector, matching principal_type.
    Name/email references are resolved during execution, including Group dependencies.
    Native nested access_grants fields use principal_id, not these local selectors.

  Kind-specific spec fields (server defaults unless marked CLI):
    Channel:
      name:string (CLI falls back to metadata.name for creation), description:
      nullable string, is_private:nullable boolean, meta/data:nullable free-form
      objects (default null). type:nullable string=null on creation; group/dm
      creation accepts user_ids/group_ids:nullable arrays of strings=null; dm needs
      user_ids. Server lowercases creation name. Existing channel updates merge
      desired fields over fetched state; omitted fields are preserved. type and
      membership are not changed by the update form. id is server-assigned.
    Function:
      id/name:strings (CLI creation defaults to metadata.name), content:string
      required on creation, or content_file. Existing source/name/meta are retained
      when omitted. meta:object (CLI creation default {}), containing optional
      nullable description:string=null, manifest:free-form object={}; extra keys
      accepted. Python source must define a valid Pipe/Filter/Action class.
      Creation id must be a Python identifier; server lowercases it.
      Server derives type and source manifest. is_active/is_global:optional
      booleans; explicitly supplied values are reconciled with toggles, omission
      preserves existing state (new functions default false).
    FunctionValve:
      metadata.name is the function ID; spec is its direct resource-defined
      valve-values object, not a {"valves":...} wrapper. Inspect separately with
      oictl functions valves spec <function-id>. Parent function must exist or be
      created in the set. Null/default handling belongs to its Valves class.
      No access_grants and no sync pruning. Empty spec example is illustrative.
    Group:
      name:string (CLI creation fallback metadata.name), description:string
      required by upstream create/update; permissions:nullable nested capability
      object and data:nullable free-form object default null. Permission values
      include workspace.models:boolean, sharing.public_models:boolean and
      features.web_search:boolean; additional keys belong to server configuration.
      id may select existing identity but is never sent as a writable field.
      Update replaces name/description; omitted/null permissions/data are preserved
      by the server. user_ids in inventory is not a writable GroupForm field;
      membership uses groups users actions, not this manifest.
    Knowledge:
      name:string (CLI fallback metadata.name), description:string required by
      the upstream create/update form; access_grants:nullable native array=null.
      Update is a full name/description form, not a generic merge patch.
      Top-level grants manage sharing separately. id is server-assigned.
    Model:
      metadata.name is the model ID; spec.id:string defaults to it in the CLI
      and must match when supplied. Server creation IDs have a 256-character limit.
      name:nonempty string required even on update.
      meta/params:objects; CLI fills omitted/null with {}. base_model_id:nullable
      string defaults null; is_active:optional boolean, server create default true,
      CLI preserves current value on update when omitted. meta has optional nullable
      profile_image_url/description:string, capabilities:free-form object,
      knowledge:array of arbitrary values, all default null; extra keys accepted.
      meta.tags supports strings or {"name":string} objects; file knowledge entries
      use type:"file", id:string. params is free-form model/provider inference
      configuration, such as temperature:number. These are replacement objects;
      CLI does not deep-merge omitted metadata/params. Server preserves omitted
      base_model_id and meta.profile_image_url. Top-level grants are reconciled
      separately; omitted grants are retained.
    Prompt:
      command/name:strings (CLI creation defaults metadata.name), content:string
      required on creation or content_file; existing command/name/content and
      other fetched fields retained when omitted. Optional nullable data/meta:
      free-form objects=null, tags:array of string|null elements (omission null),
      access_grants:native grant array (omission null), version_id:string=null,
      commit_message:string=null, is_production:boolean=true. command is the
      slash-command identity; server owns prompt version creation/selection.
    Skill:
      id:string required on creation (CLI does not synthesize it); name:string
      defaults to metadata.name only if neither id nor name supplies identity.
      Therefore supply both id and name for new skills. Server lowercases creation
      IDs and replaces spaces with hyphens. content:string required
      on creation or content_file. description:nullable string=null;
      meta:object defaults {"tags":[]}, tags:nullable array of strings default [];
      is_active:boolean=true on creation. Existing fetched fields, including
      content and active state, are preserved if omitted. Top-level grants are
      separate. This envelope is NOT skills create --manifest Markdown.
    TerminalServerConnection:
      One connection, not {"TERMINAL_SERVER_CONNECTIONS":[...]}.
      url:string required on creation; id/name:nullable strings (CLI fills blank
      values from metadata.name); enabled:nullable boolean=true;
      path:nullable string="/openapi.json", key:nullable string="",
      auth_type:nullable string="bearer", server_type/policy_id:nullable strings=null;
      config:nullable extensible object=null, with native access_grants:array.
      Existing connection fields are shallow-merged; omitted fields/grants retained.
      Explicit top-level grants replace config.access_grants. Other connections
      and unknown configuration fields are retained. No delete or sync pruning.
    Tool:
      id/name:strings (CLI creation defaults metadata.name); content:string
      required on creation or content_file. Existing source/name/meta retained
      when omitted. meta:object (CLI creation default {}): optional nullable
      description:string=null, manifest:free-form object={}, has_user_valves:
      boolean=false. Creation id must be a Python identifier, lowercased by server.
      Python source defines a Tools class; server derives tool
      specs and source manifest. access_grants:native array (omission null);
      top-level grants reconcile sharing separately.
    ToolValve:
      metadata.name is tool ID; spec is the direct resource-defined valve object.
      Inspect separately: oictl tools valves spec <tool-id>. Parent tool must
      exist or be created in the set. No access_grants or sync pruning.
      Empty spec example is illustrative, not a universal valid valve payload.

  diff computes a read-only plan; apply changes described resources; --dry-run
  previews apply/sync. sync additionally prunes absent resources in --scope:
  knowledge, models, prompts, tools, skills, functions, groups, channels, or all
  (singular aliases except knowledge also accepted). Destructive sync requires
  --yes or --confirm unless --dry-run. Valves and terminal connections may be
  applied but are never pruned. Required server fields still apply when a planned
  update sends a full native form; only the preservation cases above are promised.
  Examples below are separate files, one per supported kind.`

const manifestExamples = `{"apiVersion":"oictl.openwebui/v1","kind":"Channel","metadata":{"name":"example-channel"},"spec":{"name":"example-channel","description":"Examples"}}
{"apiVersion":"oictl.openwebui/v1","kind":"Function","metadata":{"name":"example_function"},"spec":{"content":"class Pipe:\n    def pipe(self, body: dict):\n        return 'Example'\n"}}
{"apiVersion":"oictl.openwebui/v1","kind":"FunctionValve","metadata":{"name":"example_function"},"spec":{}}
{"apiVersion":"oictl.openwebui/v1","kind":"Group","metadata":{"name":"Example Team"},"spec":{"name":"Example Team","description":"Examples"}}
{"apiVersion":"oictl.openwebui/v1","kind":"Knowledge","metadata":{"name":"Example Knowledge"},"spec":{"name":"Example Knowledge","description":"Reference documents"}}
{"apiVersion":"oictl.openwebui/v1","kind":"Model","metadata":{"name":"example-model"},"spec":{"name":"Example Model","meta":{},"params":{}}}
{"apiVersion":"oictl.openwebui/v1","kind":"Prompt","metadata":{"name":"example-prompt"},"spec":{"content":"Explain this topic."}}
{"apiVersion":"oictl.openwebui/v1","kind":"Skill","metadata":{"name":"example-skill"},"spec":{"id":"example-skill","name":"Example Skill","content":"# Example\nDescribe the result."}}
{"apiVersion":"oictl.openwebui/v1","kind":"TerminalServerConnection","metadata":{"name":"shell-a"},"spec":{"url":"https://terminal.example.com","config":{}}}
{"apiVersion":"oictl.openwebui/v1","kind":"Tool","metadata":{"name":"example_tool"},"spec":{"content":"class Tools:\n    def example(self) -> str:\n        return 'Example'\n"},"access_grants":[{"principal_type":"group","principal_name":"Example Team","permission":"read"}]}
{"apiVersion":"oictl.openwebui/v1","kind":"ToolValve","metadata":{"name":"example_tool"},"spec":{}}`
