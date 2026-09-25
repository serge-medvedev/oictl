package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type App struct {
	out           io.Writer
	err           io.Writer
	version       string
	getenv        func(string) string
	userConfigDir func() (string, error)
	httpClient    *http.Client
}

func New(out io.Writer, err io.Writer, version string) *App {
	return &App{
		out:           out,
		err:           err,
		version:       version,
		getenv:        os.Getenv,
		userConfigDir: os.UserConfigDir,
		httpClient:    http.DefaultClient,
	}
}

func (a *App) Run(ctx context.Context, args []string) int {
	select {
	case <-ctx.Done():
		fmt.Fprintln(a.err, ctx.Err())
		return 1
	default:
	}

	if len(args) == 0 {
		a.printHelp(a.out)
		return 0
	}

	if a.printJSONInputHelp(args) {
		return 0
	}

	global, args, err := parseGlobalFlags(args)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if len(args) == 0 {
		a.printHelp(a.out)
		return 0
	}

	if len(args) == 1 || (len(args) == 2 && isHelp(args[1:])) {
		for key := range jsonInputReferences {
			if strings.HasPrefix(key, args[0]+" ") {
				defer fmt.Fprintln(a.out, "\nJSON input fields and examples: append --help or -h to an action (selectors optional).")
				break
			}
		}
	}
	if err := validateCommandFlags(args); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	switch args[0] {
	case "api":
		return a.runAPI(ctx, global, args[1:])
	case "auth":
		return a.runAuth(ctx, global, args[1:])
	case "analytics":
		return a.runAnalytics(ctx, global, args[1:])
	case "automations":
		return a.runAutomations(ctx, global, args[1:])
	case "chats":
		return a.runChats(ctx, global, args[1:])
	case "channels":
		return a.runChannels(ctx, global, args[1:])
	case "config":
		return a.runConfig(ctx, global, args[1:])
	case "files":
		return a.runFiles(ctx, global, args[1:])
	case "functions":
		return a.runFunctions(ctx, global, args[1:])
	case "groups":
		return a.runGroups(ctx, global, args[1:])
	case "knowledge":
		return a.runKnowledge(ctx, global, args[1:])
	case "manifests":
		return a.runManifests(ctx, global, args[1:])
	case "models":
		return a.runModels(ctx, global, args[1:])
	case "profiles":
		return a.runProfiles(global, args[1:])
	case "providers":
		return a.runProviders(ctx, global, args[1:])
	case "scim":
		return a.runSCIM(ctx, global, args[1:])
	case "skills":
		return a.runSkills(ctx, global, args[1:])
	case "tasks":
		return a.runTasks(ctx, global, args[1:])
	case "tools":
		return a.runTools(ctx, global, args[1:])
	case "users":
		return a.runUsers(ctx, global, args[1:])
	case "webhooks":
		return a.runWebhooks(ctx, global, args[1:])
	case "help", "-h", "--help":
		a.printHelp(a.out)
		return 0
	case "version", "-v", "--version":
		fmt.Fprintf(a.out, "oictl %s\n", a.version)
		return 0
	default:
		fmt.Fprintf(a.err, "unknown command %q\n\n", args[0])
		a.printHelp(a.err)
		return 1
	}
}

func (a *App) printHelp(w io.Writer) {
	fmt.Fprint(w, `oictl controls Open WebUI from the command line.

Usage:
  oictl [global flags] <command> [args]

Global Flags:
  --profile string   Named profile to use
  --base-url string  Open WebUI base URL (or --url)
  --token string     Open WebUI API key or JWT token
  --output string    Output format: json or table (default json)
  --timeout string   Request timeout, such as 30s
  --out path         Write raw response output to a file

Commands:
  api          Make an authenticated Open WebUI API request
  analytics    Inspect administrative analytics reports
  auth         Manage session, profile, API key, and auth configuration
  automations  Manage scheduled automation workflows
  chats        Manage chats and chat workflows
  channels     Manage Open WebUI channels, members, messages, pins, and reactions
  config       Manage Open WebUI configuration
  files        Manage Open WebUI files
  functions    Manage Open WebUI functions and filters
  groups       Manage native Open WebUI groups
  knowledge    Manage Open WebUI knowledge bases
  manifests    Diff, apply, and sync declarative resource manifests
  models       Manage Open WebUI models
  profiles     Manage local oictl profiles
  providers    Send guarded provider passthrough requests via Open WebUI
  scim         Manage SCIM 2.0 provisioning resources
  skills       Manage Open WebUI skills
  tasks        Manage Open WebUI task configuration
  tools        Manage Open WebUI tools
  users        Manage Open WebUI user settings
  webhooks     Manage channel incoming webhooks and global event webhooks
  help         Show this help message
  version      Print version information
`)
}

type globalOptions struct {
	profile string
	baseURL string
	token   string
	output  string
	timeout string
	outPath string
}

type profileConfig struct {
	BaseURL   string `json:"base_url"`
	Token     string `json:"token,omitempty"`
	SCIMToken string `json:"scim_token,omitempty"`
}

type requestSpec struct {
	method       string
	path         string
	body         io.Reader
	contentType  string
	headers      []headerValue
	authRequired bool
	apiKeyHeader string
	outPath      string
	skipAuth     bool
	bearerToken  string
	redactValues []string
	quietErrors  bool
}

type headerValue struct {
	name  string
	value string
}

type commandFlags struct {
	values      map[string]string
	multiValues map[string][]string
	bools       map[string]bool
	headers     []headerValue
}

type targetSettings struct {
	baseURL string
	token   string
}

type httpStatusError struct {
	status string
	body   string
}

func (e httpStatusError) Error() string {
	return "request failed with " + e.status
}

func parseGlobalFlags(args []string) (globalOptions, []string, error) {
	var opts globalOptions
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, value, hasValue := splitFlag(arg)
		if name == "" {
			remaining = append(remaining, arg)
			continue
		}

		set := func(target *string) error {
			if !hasValue {
				if i+1 >= len(args) {
					return fmt.Errorf("%s requires a value", arg)
				}
				i++
				value = args[i]
			}
			*target = value
			return nil
		}

		switch name {
		case "profile":
			if err := set(&opts.profile); err != nil {
				return globalOptions{}, nil, err
			}
		case "base-url", "url":
			if err := set(&opts.baseURL); err != nil {
				return globalOptions{}, nil, err
			}
		case "token":
			if err := set(&opts.token); err != nil {
				return globalOptions{}, nil, err
			}
		case "output":
			if err := set(&opts.output); err != nil {
				return globalOptions{}, nil, err
			}
		case "timeout":
			if err := set(&opts.timeout); err != nil {
				return globalOptions{}, nil, err
			}
		case "out":
			if err := set(&opts.outPath); err != nil {
				return globalOptions{}, nil, err
			}
		default:
			remaining = append(remaining, arg)
		}
	}
	return opts, remaining, nil
}

func splitFlag(arg string) (name string, value string, hasValue bool) {
	if !strings.HasPrefix(arg, "--") || arg == "--" {
		return "", "", false
	}
	trimmed := strings.TrimPrefix(arg, "--")
	if before, after, ok := strings.Cut(trimmed, "="); ok {
		return before, after, true
	}
	return trimmed, "", false
}

func parseCommandFlags(args []string) (commandFlags, []string, error) {
	flags := commandFlags{values: map[string]string{}, multiValues: map[string][]string{}, bools: map[string]bool{}}
	remaining := make([]string, 0, len(args))
	boolFlags := map[string]bool{"save": true, "yes": true, "dry-run": true, "confirm": true, "stream": true, "include-valves": true, "allow-sensitive-ui-keys": true, "show-url": true, "verify-url": true, "external": true, "all": true, "all-users": true, "allow-ui-settings-extension": true}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, value, hasValue := splitFlag(arg)
		if name == "" {
			remaining = append(remaining, arg)
			continue
		}
		if boolFlags[name] {
			if hasValue {
				parsed, err := strconv.ParseBool(value)
				if err != nil {
					return commandFlags{}, nil, fmt.Errorf("--%s requires a boolean: %w", name, err)
				}
				flags.bools[name] = parsed
			} else {
				flags.bools[name] = true
			}
			continue
		}
		if name == "manifest" {
			if hasValue && (value == "true" || value == "false") {
				flags.bools[name] = value == "true"
				continue
			}
			if !hasValue && (i+1 >= len(args) || strings.HasPrefix(args[i+1], "--")) {
				flags.bools[name] = true
				continue
			}
		}
		if !hasValue {
			if i+1 >= len(args) {
				return commandFlags{}, nil, fmt.Errorf("--%s requires a value", name)
			}
			i++
			value = args[i]
		}
		if name == "header" {
			header, err := parseHeader(value)
			if err != nil {
				return commandFlags{}, nil, err
			}
			flags.headers = append(flags.headers, header)
			continue
		}
		flags.values[name] = value
		flags.multiValues[name] = append(flags.multiValues[name], value)
	}
	return flags, remaining, nil
}

func parseHeader(input string) (headerValue, error) {
	name, value, ok := strings.Cut(input, ":")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return headerValue{}, fmt.Errorf("malformed header %q; expected name:value", input)
	}
	return headerValue{name: name, value: strings.TrimSpace(value)}, nil
}

func (a *App) resolveTarget(opts globalOptions, authRequired bool) (targetSettings, error) {
	profile, err := a.loadProfile(opts.profile)
	if err != nil {
		return targetSettings{}, err
	}

	baseURL := firstNonEmpty(opts.baseURL, a.getenv("OPEN_WEBUI_URL"), a.getenv("OPEN_WEBUI_BASE_URL"), profile.BaseURL)
	if baseURL == "" {
		return targetSettings{}, errors.New("Open WebUI URL is required; set --base-url, OPEN_WEBUI_URL, OPEN_WEBUI_BASE_URL, or a profile base_url")
	}
	if _, err := parseBaseURL(baseURL); err != nil {
		return targetSettings{}, err
	}

	token := firstNonEmpty(opts.token, a.getenv("OPEN_WEBUI_API_KEY"), profile.Token)
	if authRequired && token == "" {
		return targetSettings{}, errors.New("OPEN_WEBUI_API_KEY is required; set --token, OPEN_WEBUI_API_KEY, or a profile token")
	}

	return targetSettings{baseURL: baseURL, token: token}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseBaseURL(baseURL string) (*url.URL, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, errors.New("Open WebUI URL must be an absolute URL")
	}
	return base, nil
}

func joinURL(baseURL string, endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", errors.New("endpoint path or URL is required")
	}
	rel, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if rel.IsAbs() {
		return endpoint, nil
	}
	base, err := parseBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	baseEscapedPath := strings.TrimRight(base.EscapedPath(), "/")
	relEscapedPath := strings.TrimLeft(rel.EscapedPath(), "/")
	base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(rel.Path, "/")
	if relEscapedPath != strings.TrimLeft(rel.Path, "/") {
		base.RawPath = baseEscapedPath + "/" + relEscapedPath
	}
	base.RawQuery = rel.RawQuery
	return base.String(), nil
}

func resolveTarget(flagURL string, envURL string, endpoint string) (string, error) {
	baseURL := firstNonEmpty(flagURL, envURL)
	return joinURL(baseURL, endpoint)
}

func (a *App) profilePath(name string) (string, error) {
	name = firstNonEmpty(name, a.getenv("OICTL_PROFILE"), "default")
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return "", errors.New("profile name must not contain path separators")
	}
	configDir, err := a.userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "oictl", "profiles", name+".json"), nil
}

func (a *App) loadProfile(name string) (profileConfig, error) {
	name = firstNonEmpty(name, a.getenv("OICTL_PROFILE"), "default")
	path, err := a.profilePath(name)
	if err != nil {
		return profileConfig{}, err
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return profileConfig{}, nil
	}
	if err != nil {
		return profileConfig{}, err
	}
	var profile profileConfig
	if err := json.Unmarshal(body, &profile); err != nil {
		return profileConfig{}, fmt.Errorf("read profile %q: %w", name, err)
	}
	return profile, nil
}

func (a *App) saveProfile(name string, profile profileConfig) error {
	path, err := a.profilePath(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return writeSecureFile(path, body)
}

func (a *App) deleteProfileToken(name string) error {
	profile, err := a.loadProfile(name)
	if err != nil {
		return err
	}
	profile.Token = ""
	return a.saveProfile(name, profile)
}

func (a *App) doRequest(ctx context.Context, opts globalOptions, spec requestSpec) ([]byte, error) {
	target, err := a.resolveTarget(opts, spec.authRequired)
	if err != nil {
		return nil, err
	}
	targetURL, err := joinURL(target.baseURL, spec.path)
	if err != nil {
		return nil, err
	}
	if spec.method == "" {
		spec.method = http.MethodGet
	}

	duration := 30 * time.Second
	if opts.timeout != "" {
		var err error
		duration, err = time.ParseDuration(opts.timeout)
		if err != nil {
			return nil, fmt.Errorf("parse timeout: %w", err)
		}
		if duration <= 0 {
			return nil, errors.New("timeout must be positive")
		}
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, spec.method, targetURL, spec.body)
	if err != nil {
		return nil, err
	}
	if spec.contentType != "" {
		req.Header.Set("Content-Type", spec.contentType)
	} else if spec.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, header := range spec.headers {
		req.Header.Set(header.name, header.value)
	}
	if spec.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+spec.bearerToken)
	} else if !spec.skipAuth && target.token != "" {
		if spec.apiKeyHeader != "" {
			req.Header.Set(spec.apiKeyHeader, target.token)
		} else {
			req.Header.Set("Authorization", "Bearer "+target.token)
		}
	}

	client := a.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		secrets := append([]string{target.token, spec.bearerToken}, spec.redactValues...)
		redactedBody := redactSecrets(string(body), secrets...)
		if !spec.quietErrors {
			fmt.Fprintln(a.err, resp.Status)
			_, _ = a.err.Write([]byte(redactedBody))
		}
		return nil, httpStatusError{status: resp.Status, body: redactedBody}
	}
	return body, nil
}

func (a *App) execute(ctx context.Context, opts globalOptions, spec requestSpec) int {
	body, err := a.doRequest(ctx, opts, spec)
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, err)
		}
		return 1
	}
	return a.writeResponse(opts, spec, body)
}

func (a *App) writeResponse(opts globalOptions, spec requestSpec, body []byte) int {
	outPath := firstNonEmpty(spec.outPath, opts.outPath)
	if outPath != "" {
		if err := writeSecureFile(outPath, body); err != nil {
			fmt.Fprintf(a.err, "write output: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeAll(a.out, body); err != nil {
		fmt.Fprintf(a.err, "write output: %v\n", err)
		return 1
	}
	return 0
}

func requestBody(flags commandFlags) (io.Reader, error) {
	data := flags.values["data"]
	file := firstNonEmpty(flags.values["file"], flags.values["data-file"])
	sources := 0
	for _, key := range []string{"data", "file", "data-file"} {
		if _, present := flags.values[key]; present {
			sources++
		}
	}
	if sources > 1 {
		return nil, errors.New("use only one of --data, --file, or --data-file")
	}
	if data != "" {
		return strings.NewReader(data), nil
	}
	if file == "" {
		return nil, nil
	}
	if file == "-" {
		return os.Stdin, nil
	}
	body, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read request body file: %w", err)
	}
	return bytes.NewReader(body), nil
}

func appendQueryValues(path string, pairs map[string][]string) string {
	values := url.Values{}
	for key, entries := range pairs {
		for _, value := range entries {
			if strings.TrimSpace(value) != "" {
				values.Add(key, value)
			}
		}
	}
	if len(values) == 0 {
		return path
	}
	return path + "?" + values.Encode()
}

func jsonBody(value any) io.Reader {
	body, _ := json.Marshal(value)
	return bytes.NewReader(body)
}

func bodyWithID(flags commandFlags, id string) (io.Reader, error) {
	body, err := requestBody(flags)
	if err != nil || body == nil {
		return jsonBody(map[string]string{"id": id}), err
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, fmt.Errorf("model payload must be a JSON object: %w", err)
	}
	if object == nil {
		return nil, errors.New("model payload must be a non-null JSON object")
	}
	if bodyID, ok := object["id"]; ok && bodyID != id {
		return nil, errors.New("payload id conflicts with positional model id")
	}
	object["id"] = id
	return jsonBody(object), nil
}

func appendQuery(path string, pairs map[string]string) string {
	values := url.Values{}
	for key, value := range pairs {
		if strings.TrimSpace(value) != "" {
			values.Set(key, value)
		}
	}
	if len(values) == 0 {
		return path
	}
	return path + "?" + values.Encode()
}

func requireArgs(args []string, count int, usage string) error {
	if len(args) != count {
		return fmt.Errorf("usage: %s", usage)
	}
	return nil
}

func isHelp(args []string) bool {
	return len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help"
}

func (a *App) runAPI(ctx context.Context, opts globalOptions, args []string) int {
	if len(args) > 0 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help") {
		a.printAPIHelp(a.out)
		return 0
	}
	flags, positionals, err := parseCommandFlags(args)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	method := strings.ToUpper(firstNonEmpty(flags.values["method"], "GET"))
	endpoint := ""
	if len(positionals) == 1 {
		endpoint = positionals[0]
	} else if len(positionals) == 2 && isHTTPMethod(positionals[0]) {
		method = strings.ToUpper(positionals[0])
		endpoint = positionals[1]
	} else {
		fmt.Fprintln(a.err, "exactly one endpoint path or URL is required")
		return 1
	}
	body, err := requestBody(flags)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	apiKeyHeader := strings.TrimSpace(flags.values["api-key-header"])
	if value, ok := flags.values["api-key-header"]; ok && strings.TrimSpace(value) == "" {
		fmt.Fprintln(a.err, "custom API key header name cannot be empty")
		return 1
	}
	return a.execute(ctx, opts, requestSpec{
		method:       method,
		path:         endpoint,
		body:         body,
		headers:      flags.headers,
		authRequired: true,
		apiKeyHeader: apiKeyHeader,
	})
}

func isHTTPMethod(value string) bool {
	switch strings.ToUpper(value) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func (a *App) printAPIHelp(w io.Writer) {
	fmt.Fprint(w, `Make an authenticated Open WebUI API request.

Usage:
  oictl api [flags] [METHOD] <endpoint>

Flags:
  --base-url string        Open WebUI base URL (or --url, OPEN_WEBUI_URL, OPEN_WEBUI_BASE_URL)
  --url string             Alias for --base-url
  --token string           API key or JWT token (default OPEN_WEBUI_API_KEY)
  --method string          HTTP method to use when METHOD is omitted (default GET)
  --header name:value      Request header; may be repeated
  --data string            Inline request body
  --file path              File to read as request body
  --data-file path         Alias for --file
  --api-key-header string  Header name for the credential instead of Authorization bearer auth
`)
}

func (a *App) runProfiles(opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Manage local oictl profiles.

Usage:
  oictl profiles get <name>
  oictl profiles set <name> --base-url URL [--token TOKEN]
  oictl profiles delete <name>
`)
		return 0
	}
	action := args[0]
	flags, positionals, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	switch action {
	case "get":
		if err := requireArgs(positionals, 1, "oictl profiles get <name>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		profile, err := a.loadProfile(positionals[0])
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if profile.Token != "" {
			profile.Token = "<redacted>"
		}
		if profile.SCIMToken != "" {
			profile.SCIMToken = "<redacted>"
		}
		var body bytes.Buffer
		encoder := json.NewEncoder(&body)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(profile)
		return a.writeResponse(opts, requestSpec{}, body.Bytes())
	case "set":
		if err := requireArgs(positionals, 1, "oictl profiles set <name> --base-url URL [--token TOKEN]"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		baseURL := firstNonEmpty(opts.baseURL, flags.values["base-url"], flags.values["url"])
		if baseURL == "" {
			fmt.Fprintln(a.err, "profile base URL is required")
			return 1
		}
		if _, err := parseBaseURL(baseURL); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := a.saveProfile(positionals[0], profileConfig{BaseURL: baseURL, Token: firstNonEmpty(opts.token, flags.values["token"])}); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		fmt.Fprintln(a.out, "profile saved")
		return 0
	case "delete":
		if err := requireArgs(positionals, 1, "oictl profiles delete <name>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		path, err := a.profilePath(positionals[0])
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(a.err, err)
			return 1
		}
		fmt.Fprintln(a.out, "profile deleted")
		return 0
	default:
		fmt.Fprintf(a.err, "unknown profiles command %q\n", action)
		return 1
	}
}

func (a *App) runAuth(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Usage: oictl auth <me|login|logout|profile|timezone|password|api-key|admin-config|ldap-server-config|ldap-config|oauth-config>
`)
		return 0
	}
	action := args[0]
	flags, positionals, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	simple := func(method, path string, auth bool, body io.Reader) int {
		return a.execute(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: auth})
	}
	switch action {
	case "me":
		return simple(http.MethodGet, "/api/v1/auths/", true, nil)
	case "login":
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		resp, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/auths/signin", body: body})
		if err != nil {
			var statusErr httpStatusError
			if !errors.As(err, &statusErr) {
				fmt.Fprintln(a.err, err)
			}
			return 1
		}
		if flags.bools["save"] || opts.profile != "" {
			var session struct {
				Token string `json:"token"`
			}
			if err := json.Unmarshal(resp, &session); err != nil || session.Token == "" {
				fmt.Fprintln(a.err, "authentication succeeded but session response has no valid token to save")
				return 1
			}
			profile, err := a.loadProfile(opts.profile)
			if err == nil {
				profile.BaseURL = firstNonEmpty(opts.baseURL, a.getenv("OPEN_WEBUI_URL"), a.getenv("OPEN_WEBUI_BASE_URL"), profile.BaseURL)
				profile.Token = session.Token
				err = a.saveProfile(opts.profile, profile)
			}
			if err != nil {
				fmt.Fprintf(a.err, "authentication succeeded but saving profile failed: %v\n", err)
				return 1
			}
		}
		return a.writeResponse(opts, requestSpec{}, resp)
	case "logout":
		response, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/auths/signout", authRequired: true})
		if err != nil {
			var statusErr httpStatusError
			if !errors.As(err, &statusErr) {
				fmt.Fprintln(a.err, err)
			}
			return 1
		}
		if err := a.deleteProfileToken(opts.profile); err != nil {
			fmt.Fprintf(a.err, "remote logout succeeded but local token removal failed: %v\n", err)
			return 1
		}
		return a.writeResponse(opts, requestSpec{}, response)
	case "profile":
		if len(positionals) == 1 && positionals[0] == "update" {
			body, err := requestBody(flags)
			if err != nil {
				fmt.Fprintln(a.err, err)
				return 1
			}
			return simple(http.MethodPost, "/api/v1/auths/update/profile", true, body)
		}
	case "timezone":
		if len(positionals) == 2 && positionals[0] == "set" {
			return simple(http.MethodPost, "/api/v1/auths/update/timezone", true, jsonBody(map[string]string{"timezone": positionals[1]}))
		}
	case "password":
		if len(positionals) == 1 && positionals[0] == "update" {
			body, err := requestBody(flags)
			if err != nil {
				fmt.Fprintln(a.err, err)
				return 1
			}
			return simple(http.MethodPost, "/api/v1/auths/update/password", true, body)
		}
	case "api-key":
		if err := requireArgs(positionals, 1, "oictl auth api-key <get|create|delete>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		switch positionals[0] {
		case "get":
			return simple(http.MethodGet, "/api/v1/auths/api_key", true, nil)
		case "create":
			return simple(http.MethodPost, "/api/v1/auths/api_key", true, nil)
		case "delete":
			return simple(http.MethodDelete, "/api/v1/auths/api_key", true, nil)
		}
	case "admin-config", "ldap-server-config", "ldap-config", "oauth-config":
		paths := map[string]string{
			"admin-config":       "/api/v1/auths/admin/config",
			"ldap-server-config": "/api/v1/auths/admin/config/ldap/server",
			"ldap-config":        "/api/v1/auths/admin/config/ldap",
			"oauth-config":       "/api/v1/auths/admin/config/oauth",
		}
		return a.getSet(ctx, opts, paths[action], positionals, flags)
	}
	fmt.Fprintln(a.err, "invalid auth command")
	return 1
}

func (a *App) getSet(ctx context.Context, opts globalOptions, path string, args []string, flags commandFlags) int {
	if err := requireArgs(args, 1, "<get|set>"); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if args[0] == "get" {
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: path, authRequired: true})
	}
	if args[0] == "set" {
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: path, body: body, authRequired: true})
	}
	fmt.Fprintln(a.err, "expected get or set")
	return 1
}

func (a *App) runTasks(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Manage Open WebUI task configuration.

Usage:
  oictl tasks config get
  oictl tasks config set --file tasks-config.json
  oictl tasks skills attach [--external] <skill-id-or-name>...

Task model settings are fields in the task config payload; there is no separate task-model resource.
TASK_MODEL maps to task.model.default; TASK_MODEL_EXTERNAL maps to task.model.external.
`)
		return 0
	}
	if args[0] == "skills" {
		return a.runTaskSkills(ctx, opts, args[1:])
	}
	if args[0] != "config" {
		fmt.Fprintf(a.err, "unknown tasks command %q\n", args[0])
		return 1
	}
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if len(pos) != 1 {
		fmt.Fprintln(a.err, "usage: oictl tasks config <get|set> [--file tasks-config.json]")
		return 1
	}
	switch pos[0] {
	case "get":
		return a.executeRaw(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/tasks/config", authRequired: true, outPath: flags.values["out"]})
	case "set":
		body, err := requiredBody(flags, "a task config payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.executeRaw(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/tasks/config/update", body: body, authRequired: true, outPath: flags.values["out"]})
	default:
		fmt.Fprintf(a.err, "unknown tasks config operation %q\n", pos[0])
		return 1
	}
}

func (a *App) runTaskSkills(ctx context.Context, opts globalOptions, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.err, "usage: oictl tasks skills attach [--external] <skill-id-or-name>...")
		return 1
	}
	if args[0] != "attach" {
		fmt.Fprintf(a.err, "unknown tasks skills command %q\n", args[0])
		return 1
	}
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if len(pos) == 0 {
		fmt.Fprintln(a.err, "usage: oictl tasks skills attach [--external] <skill-id-or-name>...")
		return 1
	}
	if err := a.attachTaskSkills(ctx, opts, flags, pos); err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, err)
		}
		return 1
	}
	return 0
}

func (a *App) attachTaskSkills(ctx context.Context, opts globalOptions, flags commandFlags, skillRefs []string) error {
	configBody, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/tasks/config", authRequired: true})
	if err != nil {
		return err
	}
	config := map[string]any{}
	if err := json.Unmarshal(configBody, &config); err != nil {
		return fmt.Errorf("decode task config: %w", err)
	}
	configKey := "TASK_MODEL"
	if flags.bools["external"] {
		configKey = "TASK_MODEL_EXTERNAL"
	}
	modelID := strings.TrimSpace(stringValue(config[configKey]))
	if modelID == "" {
		return fmt.Errorf("task config %s is empty", configKey)
	}

	model, err := a.fetchTaskModel(ctx, opts, modelID)
	if err != nil {
		return err
	}
	skills, err := a.listAllResources(ctx, opts, "/api/v1/skills/list")
	if err != nil {
		return fmt.Errorf("list skills: %w", err)
	}
	resolved, err := resolveTaskSkillIDs(skillRefs, skills)
	if err != nil {
		return err
	}
	if err := mergeTaskModelSkillIDs(model, resolved); err != nil {
		return err
	}

	updated, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/models/model/update", body: jsonBody(model), authRequired: true})
	if err != nil {
		return err
	}
	outPath := firstNonEmpty(flags.values["out"], opts.outPath)
	if outPath != "" {
		if err := writeSecureFile(outPath, updated); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	}
	return writeAll(a.out, updated)
}

func (a *App) fetchTaskModel(ctx context.Context, opts globalOptions, modelID string) (map[string]any, error) {
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: appendQuery("/api/v1/models/model", map[string]string{"id": modelID}), authRequired: true})
	if err != nil {
		var statusErr httpStatusError
		if errors.As(err, &statusErr) {
			return nil, fmt.Errorf("task model %q was not found", modelID)
		}
		return nil, fmt.Errorf("fetch task model %q: %w", modelID, err)
	}
	var model map[string]any
	if err := json.Unmarshal(body, &model); err != nil {
		return nil, fmt.Errorf("decode task model %q: %w", modelID, err)
	}
	if len(model) == 0 {
		return nil, fmt.Errorf("task model %q was not found", modelID)
	}
	if strings.TrimSpace(stringValue(model["id"])) == "" {
		model["id"] = modelID
	}
	if strings.TrimSpace(stringValue(model["name"])) == "" {
		return nil, fmt.Errorf("task model %q response missing name", modelID)
	}
	if model["meta"] == nil {
		model["meta"] = map[string]any{}
	}
	if _, ok := model["meta"].(map[string]any); !ok {
		return nil, fmt.Errorf("task model %q meta must be an object", modelID)
	}
	if model["params"] == nil {
		model["params"] = map[string]any{}
	}
	return model, nil
}

func resolveTaskSkillIDs(refs []string, skills []map[string]any) ([]string, error) {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return nil, errors.New("skill reference must not be empty")
		}
		resolvedID := ""
		for _, skill := range skills {
			if stringValue(skill["id"]) == ref {
				resolvedID = ref
				break
			}
		}
		if resolvedID != "" {
			ids = append(ids, resolvedID)
			continue
		}
		matches := []string{}
		for _, skill := range skills {
			if stringValue(skill["name"]) == ref {
				if id := stringValue(skill["id"]); id != "" {
					matches = append(matches, id)
				}
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("skill %q was not found", ref)
		}
		if len(matches) > 1 {
			return nil, fmt.Errorf("skill name %q is ambiguous", ref)
		}
		ids = append(ids, matches[0])
	}
	return ids, nil
}

func mergeTaskModelSkillIDs(model map[string]any, newIDs []string) error {
	meta := model["meta"].(map[string]any)
	existing, err := taskModelSkillIDs(meta["skillIds"])
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	merged := make([]string, 0, len(existing)+len(newIDs))
	for _, id := range existing {
		if !seen[id] {
			seen[id] = true
			merged = append(merged, id)
		}
	}
	for _, id := range newIDs {
		if !seen[id] {
			seen[id] = true
			merged = append(merged, id)
		}
	}
	meta["skillIds"] = merged
	return nil
}

func taskModelSkillIDs(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil, errors.New("task model meta.skillIds must be an array of strings")
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		id, ok := item.(string)
		if !ok {
			return nil, errors.New("task model meta.skillIds must be an array of strings")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (a *App) runModels(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Usage: oictl models <list|base|base-tags|tags|get|create|update|toggle|access-update|delete|delete-all|import|export|sync>
`)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	q := map[string]string{"query": flags.values["query"], "view_option": flags.values["view-option"], "tag": flags.values["tag"], "order_by": flags.values["order-by"], "direction": flags.values["direction"], "page": flags.values["page"]}
	switch action {
	case "list":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: appendQuery("/api/v1/models/list", q), authRequired: true})
	case "base":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: appendQuery("/api/v1/models/base", map[string]string{"tag": flags.values["tag"]}), authRequired: true})
	case "base-tags":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/models/base/tags", authRequired: true})
	case "tags":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/models/tags", authRequired: true})
	case "get":
		if err := requireArgs(pos, 1, "oictl models get <model-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: appendQuery("/api/v1/models/model", map[string]string{"id": pos[0]}), authRequired: true})
	case "create", "import", "sync":
		paths := map[string]string{"create": "/api/v1/models/create", "import": "/api/v1/models/import", "sync": "/api/v1/models/sync"}
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if action == "sync" || action == "import" {
			body, err = normalizeModelInventory(body, action == "import" || flags.bools["yes"] || flags.bools["confirm"])
		} else {
			body, err = normalizeModelParamsBody(body)
		}
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: paths[action], body: body, authRequired: true})
	case "update", "toggle", "access-update", "delete":
		if err := requireArgs(pos, 1, "oictl models "+action+" <model-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		paths := map[string]string{"update": "/api/v1/models/model/update", "toggle": "/api/v1/models/model/toggle", "access-update": "/api/v1/models/model/access/update", "delete": "/api/v1/models/model/delete"}
		body, err := bodyWithID(flags, pos[0])
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if action == "delete" {
			return a.deleteModel(ctx, opts, pos[0], body)
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: paths[action], body: body, authRequired: true})
	case "delete-all":
		return a.execute(ctx, opts, requestSpec{method: http.MethodDelete, path: "/api/v1/models/delete/all", authRequired: true})
	case "export":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/models/export", authRequired: true, outPath: flags.values["out"]})
	}
	fmt.Fprintf(a.err, "unknown models command %q\n", action)
	return 1
}

func normalizeModelInventory(body io.Reader, allowEmpty bool) (io.Reader, error) {
	if body == nil {
		return nil, errors.New("models inventory is required")
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode models inventory: %w", err)
	}
	envelope, ok := payload.(map[string]any)
	if items, isArray := payload.([]any); isArray {
		envelope = map[string]any{"models": items}
		ok = true
	}
	if !ok {
		return nil, errors.New("models inventory must be an object containing a models array or an array")
	}
	items, ok := envelope["models"].([]any)
	if !ok {
		return nil, errors.New("models inventory requires an explicit models array")
	}
	if len(items) == 0 && !allowEmpty {
		return nil, errors.New("empty models sync requires --yes")
	}
	for _, item := range items {
		model, ok := item.(map[string]any)
		if !ok {
			return nil, errors.New("each models inventory entry must be an object")
		}
		ensureModelParams(model)
	}
	return jsonBody(envelope), nil
}

func normalizeModelParamsBody(body io.Reader) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var model map[string]any
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("model payload must be a JSON object: %w", err)
	}
	if model == nil {
		return nil, errors.New("model payload must be a non-null JSON object")
	}
	ensureModelParams(model)
	return jsonBody(model), nil
}

func ensureModelParams(model map[string]any) {
	if _, ok := model["params"]; !ok {
		model["params"] = map[string]any{}
	}
}

func (a *App) deleteModel(ctx context.Context, opts globalOptions, id string, body io.Reader) int {
	response, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/models/model/delete", body: body, authRequired: true, quietErrors: true})
	if err == nil {
		var deleted *bool
		if err := json.Unmarshal(response, &deleted); err != nil || deleted == nil {
			fmt.Fprintln(a.err, "model deletion response must be a boolean")
			return 1
		}
		if code := a.writeResponse(opts, requestSpec{}, response); code != 0 {
			return code
		}
		if !*deleted {
			fmt.Fprintf(a.err, "model %q deletion failed\n", id)
			return 1
		}
		return 0
	}
	var statusErr httpStatusError
	if errors.As(err, &statusErr) {
		if isModelDeleteNotFound(statusErr) {
			fmt.Fprintf(a.err, "model %q not found\n", id)
			return 1
		}
		fmt.Fprintln(a.err, statusErr.status)
		_, _ = a.err.Write([]byte(statusErr.body))
		return 1
	}
	fmt.Fprintln(a.err, err)
	return 1
}

func isModelDeleteNotFound(err httpStatusError) bool {
	if !strings.HasPrefix(err.status, "401 ") {
		return false
	}
	text := strings.ToLower(err.body)
	return strings.Contains(text, "not found") || strings.Contains(text, "could not find what you're looking for")
}

func (a *App) runConfig(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Usage: oictl config <import|export|namespace|connections|tool-servers|terminal-servers|code-execution|models|suggestions|banners|oauth-client>

Terminal server access grants:
  oictl config terminal-servers access-grants get <connection-id-or-name>
  oictl config terminal-servers access-grants diff <connection-id-or-name> --file grants.json
  oictl config terminal-servers access-grants set <connection-id-or-name> --file grants.json
`)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	switch action {
	case "import":
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/configs/import", body: body, authRequired: true})
	case "export":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/configs/export", authRequired: true})
	case "namespace":
		if err := requireArgs(pos, 1, "oictl config namespace <namespace>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/configs/namespace/" + url.PathEscape(pos[0]), authRequired: true})
	case "connections":
		return a.getSet(ctx, opts, "/api/v1/configs/connections", pos, flags)
	case "tool-servers":
		return a.configSubcommand(ctx, opts, "/api/v1/configs/tool_servers", pos, flags, map[string]string{"verify": "/api/v1/configs/tool_servers/verify"})
	case "terminal-servers":
		if len(pos) > 0 && pos[0] == "access-grants" {
			return a.runTerminalServerAccessGrants(ctx, opts, pos[1:], flags)
		}
		return a.configSubcommand(ctx, opts, terminalServersConfigPath, pos, flags, map[string]string{"verify": "/api/v1/configs/terminal_servers/verify", "policy": "/api/v1/configs/terminal_servers/policy", "lifecycle": "/api/v1/configs/terminal_servers/lifecycle", "refresh": "/api/v1/configs/terminal_servers/refresh"})
	case "code-execution":
		return a.getSet(ctx, opts, "/api/v1/configs/code_execution", pos, flags)
	case "models":
		if len(pos) == 1 && pos[0] == "defaults" {
			return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/configs/models/defaults", authRequired: true})
		}
		return a.getSet(ctx, opts, "/api/v1/configs/models", pos, flags)
	case "suggestions":
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/configs/suggestions", body: body, authRequired: true})
	case "banners":
		return a.getSet(ctx, opts, "/api/v1/configs/banners", pos, flags)
	case "oauth-client":
		if len(pos) == 1 && pos[0] == "register" {
			body, err := requestBody(flags)
			if err != nil {
				fmt.Fprintln(a.err, err)
				return 1
			}
			return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: appendQuery("/api/v1/configs/oauth/clients/register", map[string]string{"type": flags.values["type"]}), body: body, authRequired: true})
		}
	}
	fmt.Fprintf(a.err, "unknown config command %q\n", action)
	return 1
}

func (a *App) configSubcommand(ctx context.Context, opts globalOptions, basePath string, args []string, flags commandFlags, posts map[string]string) int {
	if err := requireArgs(args, 1, "<get|set|operation>"); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if args[0] == "get" {
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: basePath, authRequired: true})
	}
	body, err := requestBody(flags)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if args[0] == "set" {
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: basePath, body: body, authRequired: true})
	}
	if path, ok := posts[args[0]]; ok {
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: path, body: body, authRequired: true})
	}
	fmt.Fprintln(a.err, "unknown config operation")
	return 1
}

func (a *App) runFiles(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Usage: oictl files <upload|list|search|count|get|status|content|html-content|named-content|data-content|update-content|rename|delete|delete-all>
`)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	switch action {
	case "upload":
		if err := requireArgs(pos, 1, "oictl files upload <path>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, contentType, err := multipartFileBody(pos[0], flags.values["metadata"])
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		path := appendQuery("/api/v1/files/", map[string]string{"process": flags.values["process"], "process_in_background": flags.values["process-in-background"]})
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: path, body: body, contentType: contentType, authRequired: true})
	case "list":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: appendQuery("/api/v1/files/", map[string]string{"content": flags.values["content"], "page": flags.values["page"]}), authRequired: true})
	case "search":
		filename := firstNonEmpty(flags.values["filename"], flags.values["query"], "*")
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: appendQuery("/api/v1/files/search", map[string]string{"filename": filename, "skip": flags.values["skip"], "limit": flags.values["limit"], "content": flags.values["content"]}), authRequired: true})
	case "count":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/files/count", authRequired: true})
	case "get", "status", "content", "html-content", "data-content", "delete":
		if err := requireArgs(pos, 1, "oictl files "+action+" <file-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		paths := map[string]string{"get": "/api/v1/files/" + url.PathEscape(pos[0]), "status": "/api/v1/files/" + url.PathEscape(pos[0]) + "/process/status", "content": "/api/v1/files/" + url.PathEscape(pos[0]) + "/content", "html-content": "/api/v1/files/" + url.PathEscape(pos[0]) + "/content/html", "data-content": "/api/v1/files/" + url.PathEscape(pos[0]) + "/data/content", "delete": "/api/v1/files/" + url.PathEscape(pos[0])}
		method := http.MethodGet
		if action == "delete" {
			method = http.MethodDelete
		}
		return a.execute(ctx, opts, requestSpec{method: method, path: paths[action], authRequired: true})
	case "named-content":
		if err := requireArgs(pos, 2, "oictl files named-content <file-id> <file-name>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/files/" + url.PathEscape(pos[0]) + "/content/" + url.PathEscape(pos[1]), authRequired: true})
	case "update-content":
		if err := requireArgs(pos, 1, "oictl files update-content <file-id> --file content.json"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/files/" + url.PathEscape(pos[0]) + "/data/content/update", body: body, authRequired: true})
	case "rename":
		if err := requireArgs(pos, 1, "oictl files rename <file-id> --name <name>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if strings.TrimSpace(flags.values["name"]) == "" {
			fmt.Fprintln(a.err, "files rename requires --name")
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/files/" + url.PathEscape(pos[0]) + "/rename", body: jsonBody(map[string]string{"filename": flags.values["name"]}), authRequired: true})
	case "delete-all":
		return a.execute(ctx, opts, requestSpec{method: http.MethodDelete, path: "/api/v1/files/all", authRequired: true})
	}
	fmt.Fprintf(a.err, "unknown files command %q\n", action)
	return 1
}

func multipartFileBody(path string, metadata string) (io.Reader, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return nil, "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, "", err
	}
	if metadata != "" {
		_ = writer.WriteField("metadata", metadata)
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &body, writer.FormDataContentType(), nil
}

func (a *App) runKnowledge(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Usage: oictl knowledge <list|search|search-files|create|get|update|access-update|reset|delete|export|files|dirs|reindex|metadata-reindex|sync>

Examples:
  oictl knowledge export <knowledge-id> --out knowledge.zip
`)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	switch action {
	case "list":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: appendQuery("/api/v1/knowledge/", map[string]string{"page": flags.values["page"]}), authRequired: true})
	case "search":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: appendQuery("/api/v1/knowledge/search", map[string]string{"query": flags.values["query"], "view_option": flags.values["view-option"], "source": flags.values["source"], "page": flags.values["page"]}), authRequired: true})
	case "search-files":
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: appendQuery("/api/v1/knowledge/search/files", map[string]string{"query": flags.values["query"], "include_content": flags.values["include-content"], "page": flags.values["page"]}), authRequired: true})
	case "create", "reindex", "metadata-reindex":
		paths := map[string]string{"create": "/api/v1/knowledge/create", "reindex": "/api/v1/knowledge/reindex", "metadata-reindex": "/api/v1/knowledge/metadata/reindex"}
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: paths[action], body: body, authRequired: true})
	case "get", "export", "delete", "reset", "update", "access-update":
		if err := requireArgs(pos, 1, "oictl knowledge "+action+" <knowledge-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		id := url.PathEscape(pos[0])
		paths := map[string]string{"get": "/api/v1/knowledge/" + id, "export": "/api/v1/knowledge/" + id + "/export", "delete": "/api/v1/knowledge/" + id + "/delete", "reset": "/api/v1/knowledge/" + id + "/reset", "update": "/api/v1/knowledge/" + id + "/update", "access-update": "/api/v1/knowledge/" + id + "/access/update"}
		method := http.MethodGet
		if action == "delete" {
			method = http.MethodDelete
		} else if action != "get" && action != "export" {
			method = http.MethodPost
		}
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: method, path: paths[action], body: body, authRequired: true})
	case "files":
		return a.runKnowledgeFiles(ctx, opts, pos, flags)
	case "dirs":
		return a.runKnowledgeDirs(ctx, opts, pos, flags)
	case "sync":
		return a.runKnowledgeSync(ctx, opts, pos, flags)
	}
	fmt.Fprintf(a.err, "unknown knowledge command %q\n", action)
	return 1
}

func (a *App) runKnowledgeFiles(ctx context.Context, opts globalOptions, args []string, flags commandFlags) int {
	if len(args) < 2 {
		fmt.Fprintln(a.err, "usage: oictl knowledge files <list|pending|add|update|remove|batch-add|move> <knowledge-id> [file-id]")
		return 1
	}
	action, id := args[0], url.PathEscape(args[1])
	switch action {
	case "list", "pending":
		suffix := "/files"
		if action == "pending" {
			suffix = "/files/pending"
		}
		path := "/api/v1/knowledge/" + id + suffix
		if action == "list" {
			path = appendQuery(path, map[string]string{"page": flags.values["page"], "query": firstNonEmpty(flags.values["query"], flags.values["q"]), "view_option": flags.values["view-option"], "order_by": flags.values["order-by"], "direction": flags.values["direction"], "include_content": flags.values["include-content"], "limit": flags.values["limit"]})
			if directory, present := flags.values["directory-id"]; present {
				parsed, _ := url.Parse(path)
				query := parsed.Query()
				query.Set("directory_id", directory)
				parsed.RawQuery = query.Encode()
				path = parsed.String()
			}
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodGet, path: path, authRequired: true})
	case "add":
		if len(args) != 3 {
			fmt.Fprintln(a.err, "usage: oictl knowledge files add <knowledge-id> <file-id>")
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/knowledge/" + id + "/file/add", body: jsonBody(map[string]string{"file_id": args[2]}), authRequired: true})
	case "update", "remove", "move":
		paths := map[string]string{"update": "/file/update", "remove": "/file/remove", "move": "/file/move"}
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/knowledge/" + id + paths[action], body: body, authRequired: true})
	case "batch-add":
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/knowledge/" + id + "/files/batch/add", body: body, authRequired: true})
	}
	fmt.Fprintln(a.err, "unknown knowledge files command")
	return 1
}

func (a *App) runKnowledgeDirs(ctx context.Context, opts globalOptions, args []string, flags commandFlags) int {
	if len(args) < 2 {
		fmt.Fprintln(a.err, "usage: oictl knowledge dirs <create|update|delete> <knowledge-id> [dir-id]")
		return 1
	}
	action, id := args[0], url.PathEscape(args[1])
	body, err := requestBody(flags)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	switch action {
	case "create":
		return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/knowledge/" + id + "/dirs/create", body: body, authRequired: true})
	case "update", "delete":
		if len(args) != 3 {
			fmt.Fprintln(a.err, "directory id is required")
			return 1
		}
		method := http.MethodPost
		if action == "delete" {
			method = http.MethodDelete
		}
		return a.execute(ctx, opts, requestSpec{method: method, path: "/api/v1/knowledge/" + id + "/dirs/" + url.PathEscape(args[2]) + "/" + action, body: body, authRequired: true})
	}
	fmt.Fprintln(a.err, "unknown knowledge dirs command")
	return 1
}

func (a *App) runKnowledgeSync(ctx context.Context, opts globalOptions, args []string, flags commandFlags) int {
	if len(args) != 2 {
		fmt.Fprintln(a.err, "usage: oictl knowledge sync <diff|cleanup> <knowledge-id>")
		return 1
	}
	body, err := requestBody(flags)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	return a.execute(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/knowledge/" + url.PathEscape(args[1]) + "/sync/" + args[0], body: body, authRequired: true})
}
