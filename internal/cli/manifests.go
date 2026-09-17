package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const manifestAPIVersion = "oictl.openwebui/v1"

type manifestMetadata struct {
	Name string `json:"name" yaml:"name"`
}

type accessGrant struct {
	PrincipalType  string `json:"principal_type" yaml:"principal_type"`
	PrincipalID    string `json:"principal_id" yaml:"principal_id"`
	PrincipalRef   string `json:"principal_ref,omitempty" yaml:"principal_ref,omitempty"`
	PrincipalEmail string `json:"principal_email,omitempty" yaml:"principal_email,omitempty"`
	PrincipalName  string `json:"principal_name,omitempty" yaml:"principal_name,omitempty"`
	Permission     string `json:"permission" yaml:"permission"`
}

type principalUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type principalGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type principalResolver struct {
	app        *App
	opts       globalOptions
	userCache  map[string][]principalUser
	groupCache []principalGroup
	groupsRead bool
}

type manifestDocument struct {
	Source          string           `json:"-" yaml:"-"`
	APIVersion      string           `json:"apiVersion" yaml:"apiVersion"`
	Kind            string           `json:"kind" yaml:"kind"`
	Metadata        manifestMetadata `json:"metadata" yaml:"metadata"`
	Spec            map[string]any   `json:"spec" yaml:"spec"`
	AccessGrants    []accessGrant    `json:"access_grants,omitempty" yaml:"access_grants,omitempty"`
	HasAccessGrants bool             `json:"-" yaml:"-"`
	raw             map[string]json.RawMessage
}

func (m *manifestDocument) UnmarshalJSON(data []byte) error {
	type alias manifestDocument
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*m = manifestDocument(decoded)
	m.raw = raw
	_, m.HasAccessGrants = raw["access_grants"]
	return nil
}

type manifestOptions struct {
	files       []string
	directories []string
	output      string
	dryRun      bool
	scope       string
	yes         bool
}

type resourceState struct {
	Kind         string
	Name         string
	ID           string
	Spec         map[string]any
	AccessGrants []accessGrant
}

type resourceHandler interface {
	Kind() string
	SupportsAccessGrants() bool
	List(context.Context, *App, globalOptions) ([]resourceState, error)
	Lookup(context.Context, *App, globalOptions, manifestDocument) (*resourceState, error)
	Fetch(context.Context, *App, globalOptions, string) (*resourceState, error)
	Create(context.Context, *App, globalOptions, manifestDocument) (*resourceState, error)
	Update(context.Context, *App, globalOptions, resourceState, manifestDocument) (*resourceState, error)
	Delete(context.Context, *App, globalOptions, resourceState) error
	UpdateAccessGrants(context.Context, *App, globalOptions, resourceState, []accessGrant) (*resourceState, error)
}

type endpointHandler struct {
	kind           string
	listPath       string
	getPath        string
	createPath     string
	updatePath     string
	deletePath     string
	accessPath     string
	identityFields []string
	managedFields  []string
}

type modelHandler struct{}
type channelHandler struct{ endpointHandler }
type functionHandler struct{ endpointHandler }
type terminalServerConnectionHandler struct{}
type valveHandler struct {
	kind     string
	resource string
}

func (h endpointHandler) Kind() string { return h.kind }

func (h endpointHandler) SupportsAccessGrants() bool { return h.accessPath != "" }

func (h endpointHandler) List(ctx context.Context, a *App, opts globalOptions) ([]resourceState, error) {
	objects, err := a.listAllResources(ctx, opts, h.listPath)
	if err != nil {
		return nil, fmt.Errorf("decode %s list: %w", h.kind, err)
	}
	states := make([]resourceState, 0, len(objects))
	for _, object := range objects {
		states = append(states, h.stateFromObject(object))
	}
	return states, nil
}

func (h endpointHandler) Lookup(ctx context.Context, a *App, opts globalOptions, desired manifestDocument) (*resourceState, error) {
	states, err := h.List(ctx, a, opts)
	if err != nil {
		return nil, err
	}
	for _, state := range states {
		if state.ID == desired.Metadata.Name {
			if h.kind == "Tool" {
				return h.Fetch(ctx, a, opts, state.ID)
			}
			return &state, nil
		}
	}
	var matches []resourceState
	for _, state := range states {
		match := state.Name == desired.Metadata.Name
		if h.kind == "Group" {
			match = groupStateMatches(state, desired)
		}
		if h.kind == "Channel" {
			name := strings.ToLower(stringValue(state.Spec["name"]))
			match = name == strings.ToLower(desired.Metadata.Name) || name == strings.ToLower(stringValue(desired.Spec["name"]))
		}
		for _, field := range h.identityFields {
			if stringValue(state.Spec[field]) == desired.Metadata.Name {
				match = true
			}
		}
		if match {
			matches = append(matches, state)
		}
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("manifest %s %s/%s identity is ambiguous", desired.Source, h.kind, desired.Metadata.Name)
	}
	if len(matches) == 1 {
		if h.kind == "Tool" {
			return h.Fetch(ctx, a, opts, matches[0].ID)
		}
		return &matches[0], nil
	}
	return nil, nil
}

func (h endpointHandler) Fetch(ctx context.Context, a *App, opts globalOptions, id string) (*resourceState, error) {
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: fmt.Sprintf(h.getPath, url.PathEscape(id)), authRequired: true})
	if err != nil {
		return nil, err
	}
	return h.decodeState(body)
}

func (h endpointHandler) Create(ctx context.Context, a *App, opts globalOptions, desired manifestDocument) (*resourceState, error) {
	payload, err := h.manifestPayload(desired, nil)
	if err != nil {
		return nil, err
	}
	if h.kind == "Group" {
		delete(payload, "id")
	}
	ensureIdentityFields(payload, desired.Metadata.Name, h.identityFields)
	if desired.HasAccessGrants {
		payload["access_grants"] = desired.AccessGrants
	}
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: h.createPath, body: jsonBody(payload), authRequired: true})
	if err != nil {
		return nil, err
	}
	state, err := h.decodeState(body)
	if err != nil || state == nil || state.ID == "" {
		return state, err
	}
	return h.Fetch(ctx, a, opts, state.ID)
}

func (h endpointHandler) Update(ctx context.Context, a *App, opts globalOptions, current resourceState, desired manifestDocument) (*resourceState, error) {
	payload, err := h.manifestPayload(desired, &current)
	if err != nil {
		return nil, err
	}
	if h.kind == "Skill" {
		if _, managed := desired.Spec["is_active"]; !managed {
			payload["is_active"] = current.Spec["is_active"]
		}
	}
	if h.kind == "Group" {
		delete(payload, "id")
	}
	ensureIdentityFields(payload, desired.Metadata.Name, h.identityFields)
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: fmt.Sprintf(h.updatePath, url.PathEscape(current.ID)), body: jsonBody(payload), authRequired: true})
	if err != nil {
		return nil, err
	}
	state, err := h.decodeState(body)
	if err != nil || state == nil || state.ID == "" {
		return state, err
	}
	return h.Fetch(ctx, a, opts, state.ID)
}

func (h endpointHandler) manifestPayload(desired manifestDocument, current *resourceState) (map[string]any, error) {
	payload := map[string]any{}
	if current != nil && (h.kind == "Tool" || h.kind == "Function" || h.kind == "Prompt" || h.kind == "Skill") {
		payload = copySpec(current.Spec)
	}
	for key, value := range desired.Spec {
		payload[key] = value
	}
	if h.kind == "Tool" || h.kind == "Function" {
		if _, ok := payload["id"]; !ok {
			payload["id"] = desired.Metadata.Name
		}
		if _, ok := payload["meta"]; !ok {
			payload["meta"] = map[string]any{}
		}
	}
	if h.kind == "Tool" || h.kind == "Function" || h.kind == "Prompt" {
		if _, ok := payload["name"]; !ok {
			payload["name"] = desired.Metadata.Name
		}
		if h.kind == "Prompt" {
			if _, ok := payload["command"]; !ok {
				payload["command"] = desired.Metadata.Name
			}
		}
		if _, ok := payload["content"].(string); !ok {
			return nil, fmt.Errorf("%s spec.content must be a string for creation/update", h.kind)
		}
	}
	return payload, nil
}

func (h endpointHandler) Delete(ctx context.Context, a *App, opts globalOptions, current resourceState) error {
	_, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodDelete, path: fmt.Sprintf(h.deletePath, url.PathEscape(current.ID)), authRequired: true})
	return err
}

func (h endpointHandler) UpdateAccessGrants(ctx context.Context, a *App, opts globalOptions, current resourceState, grants []accessGrant) (*resourceState, error) {
	if !h.SupportsAccessGrants() {
		return nil, fmt.Errorf("access grants are unsupported for %s manifests", h.kind)
	}
	payload := map[string]any{"access_grants": grants}
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: fmt.Sprintf(h.accessPath, url.PathEscape(current.ID)), body: jsonBody(payload), authRequired: true})
	if err != nil {
		return nil, err
	}
	state, err := h.decodeState(body)
	if err != nil || state == nil || state.ID == "" {
		return state, err
	}
	return h.Fetch(ctx, a, opts, state.ID)
}

func (h endpointHandler) decodeState(body []byte) (*resourceState, error) {
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	state := h.stateFromObject(object)
	return &state, nil
}

func (h endpointHandler) stateFromObject(object map[string]any) resourceState {
	return resourceState{
		Kind:         h.kind,
		Name:         resourceName(object, h.identityFields),
		ID:           stringValue(object["id"]),
		Spec:         managedSpec(object, h.managedFields),
		AccessGrants: normalizeAccessGrants(accessGrantsFromValue(object["access_grants"])),
	}
}

func manifestHandlers() map[string]resourceHandler {
	return map[string]resourceHandler{
		"Channel":                  channelHandler{endpointHandler{kind: "Channel", listPath: "/api/v1/channels/list", getPath: "/api/v1/channels/%s", createPath: "/api/v1/channels/create", updatePath: "/api/v1/channels/%s/update", deletePath: "/api/v1/channels/%s/delete", identityFields: []string{"name", "id"}, managedFields: []string{"id", "name", "description", "is_private", "meta", "data", "access_grants"}}},
		"Function":                 functionHandler{endpointHandler{kind: "Function", listPath: "/api/v1/functions/list", getPath: "/api/v1/functions/id/%s", createPath: "/api/v1/functions/create", updatePath: "/api/v1/functions/id/%s/update", deletePath: "/api/v1/functions/id/%s/delete", identityFields: []string{"id", "name"}, managedFields: []string{"id", "name", "content", "meta", "type", "is_active", "is_global"}}},
		"FunctionValve":            valveHandler{kind: "FunctionValve", resource: "functions"},
		"Group":                    endpointHandler{kind: "Group", listPath: "/api/v1/groups/", getPath: "/api/v1/groups/id/%s", createPath: "/api/v1/groups/create", updatePath: "/api/v1/groups/id/%s/update", deletePath: "/api/v1/groups/id/%s/delete", identityFields: []string{"id", "name"}, managedFields: []string{"id", "name", "description", "permissions", "user_ids"}},
		"Knowledge":                endpointHandler{kind: "Knowledge", listPath: "/api/v1/knowledge/", getPath: "/api/v1/knowledge/%s", createPath: "/api/v1/knowledge/create", updatePath: "/api/v1/knowledge/%s/update", deletePath: "/api/v1/knowledge/%s/delete", accessPath: "/api/v1/knowledge/%s/access/update", identityFields: []string{"name", "id"}},
		"Model":                    modelHandler{},
		"Prompt":                   endpointHandler{kind: "Prompt", listPath: "/api/v1/prompts/", getPath: "/api/v1/prompts/id/%s", createPath: "/api/v1/prompts/create", updatePath: "/api/v1/prompts/id/%s/update", deletePath: "/api/v1/prompts/id/%s/delete", accessPath: "/api/v1/prompts/id/%s/access/update", identityFields: []string{"command", "name", "id"}},
		"Skill":                    endpointHandler{kind: "Skill", listPath: "/api/v1/skills/list", getPath: "/api/v1/skills/id/%s", createPath: "/api/v1/skills/create", updatePath: "/api/v1/skills/id/%s/update", deletePath: "/api/v1/skills/id/%s/delete", accessPath: "/api/v1/skills/id/%s/access/update", identityFields: []string{"id", "name"}, managedFields: []string{"id", "name", "description", "content", "meta", "is_active"}},
		"TerminalServerConnection": terminalServerConnectionHandler{},
		"Tool":                     endpointHandler{kind: "Tool", listPath: "/api/v1/tools/list", getPath: "/api/v1/tools/id/%s", createPath: "/api/v1/tools/create", updatePath: "/api/v1/tools/id/%s/update", deletePath: "/api/v1/tools/id/%s/delete", accessPath: "/api/v1/tools/id/%s/access/update", identityFields: []string{"name", "id"}},
		"ToolValve":                valveHandler{kind: "ToolValve", resource: "tools"},
	}
}

func (h functionHandler) Create(ctx context.Context, a *App, opts globalOptions, desired manifestDocument) (*resourceState, error) {
	state, err := h.endpointHandler.Create(ctx, a, opts, desired)
	if err != nil || state == nil {
		return state, err
	}
	return h.reconcileState(ctx, a, opts, *state, desired)
}

func (h functionHandler) Lookup(ctx context.Context, a *App, opts globalOptions, desired manifestDocument) (*resourceState, error) {
	state, err := h.endpointHandler.Lookup(ctx, a, opts, desired)
	if err != nil || state == nil {
		return state, err
	}
	return h.Fetch(ctx, a, opts, state.ID)
}

func (h functionHandler) Update(ctx context.Context, a *App, opts globalOptions, current resourceState, desired manifestDocument) (*resourceState, error) {
	state, err := h.endpointHandler.Update(ctx, a, opts, current, desired)
	if err != nil || state == nil {
		return state, err
	}
	return h.reconcileState(ctx, a, opts, *state, desired)
}

func (h functionHandler) reconcileState(ctx context.Context, a *App, opts globalOptions, current resourceState, desired manifestDocument) (*resourceState, error) {
	state, err := h.reconcileActive(ctx, a, opts, current, desired)
	if err != nil {
		return nil, err
	}
	value, managed := desired.Spec["is_global"]
	if !managed {
		return state, nil
	}
	want, ok := value.(bool)
	if !ok {
		return nil, errors.New("Function spec.is_global must be a boolean")
	}
	actual, ok := state.Spec["is_global"].(bool)
	if !ok {
		return nil, errors.New("Function response is missing boolean is_global")
	}
	if actual == want {
		return state, nil
	}
	if _, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: fmt.Sprintf("/api/v1/functions/id/%s/toggle/global", url.PathEscape(state.ID)), authRequired: true}); err != nil {
		return nil, err
	}
	updated, err := h.Fetch(ctx, a, opts, state.ID)
	if err != nil {
		return nil, err
	}
	actual, ok = updated.Spec["is_global"].(bool)
	if !ok || actual != want {
		return nil, fmt.Errorf("Function global state did not converge: want is_global=%t", want)
	}
	return updated, nil
}

func (h functionHandler) reconcileActive(ctx context.Context, a *App, opts globalOptions, current resourceState, desired manifestDocument) (*resourceState, error) {
	desiredValue, managed := desired.Spec["is_active"]
	if !managed {
		return &current, nil
	}
	desiredActive, ok := desiredValue.(bool)
	if !ok {
		return nil, errors.New("Function spec.is_active must be a boolean")
	}
	currentActive, ok := current.Spec["is_active"].(bool)
	if !ok {
		return nil, errors.New("Function response is missing boolean is_active")
	}
	if currentActive == desiredActive {
		return &current, nil
	}
	if _, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: fmt.Sprintf("/api/v1/functions/id/%s/toggle", url.PathEscape(current.ID)), authRequired: true}); err != nil {
		return nil, err
	}
	updated, err := h.Fetch(ctx, a, opts, current.ID)
	if err != nil {
		return nil, err
	}
	updatedActive, ok := updated.Spec["is_active"].(bool)
	if !ok {
		return nil, errors.New("Function response is missing boolean is_active")
	}
	if updatedActive != desiredActive {
		return nil, fmt.Errorf("Function activation did not converge: got is_active=%t, want %t", updatedActive, desiredActive)
	}
	return updated, nil
}

func (h valveHandler) Kind() string { return h.kind }

func (h valveHandler) SupportsAccessGrants() bool { return false }

func (h valveHandler) List(context.Context, *App, globalOptions) ([]resourceState, error) {
	return nil, fmt.Errorf("%s manifests do not support list or sync pruning", h.kind)
}

func (h valveHandler) Lookup(ctx context.Context, a *App, opts globalOptions, desired manifestDocument) (*resourceState, error) {
	return h.Fetch(ctx, a, opts, desired.Metadata.Name)
}

func (h valveHandler) Fetch(ctx context.Context, a *App, opts globalOptions, id string) (*resourceState, error) {
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: h.valvesPath(id), authRequired: true})
	if err != nil {
		return nil, err
	}
	return h.decodeState(id, body)
}

func (h valveHandler) Create(ctx context.Context, a *App, opts globalOptions, desired manifestDocument) (*resourceState, error) {
	return h.updateValves(ctx, a, opts, desired.Metadata.Name, desired.Spec)
}

func (h valveHandler) Update(ctx context.Context, a *App, opts globalOptions, current resourceState, desired manifestDocument) (*resourceState, error) {
	return h.updateValves(ctx, a, opts, firstNonEmpty(current.ID, desired.Metadata.Name), desired.Spec)
}

func (h valveHandler) Delete(context.Context, *App, globalOptions, resourceState) error {
	return fmt.Errorf("%s manifests do not support delete or sync pruning", h.kind)
}

func (h valveHandler) UpdateAccessGrants(context.Context, *App, globalOptions, resourceState, []accessGrant) (*resourceState, error) {
	return nil, fmt.Errorf("access grants are unsupported for %s manifests", h.kind)
}

func (h valveHandler) updateValves(ctx context.Context, a *App, opts globalOptions, id string, spec map[string]any) (*resourceState, error) {
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: h.valvesPath(id) + "/update", body: jsonBody(spec), authRequired: true})
	if err != nil {
		return nil, err
	}
	return h.decodeState(id, body)
}

func (h valveHandler) decodeState(id string, body []byte) (*resourceState, error) {
	if bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
		return nil, nil
	}
	var spec map[string]any
	if err := json.Unmarshal(body, &spec); err != nil {
		return nil, err
	}
	return &resourceState{Kind: h.kind, Name: id, ID: id, Spec: spec}, nil
}

func (h valveHandler) valvesPath(id string) string {
	return fmt.Sprintf("/api/v1/%s/id/%s/valves", h.resource, url.PathEscape(id))
}

func (h terminalServerConnectionHandler) Kind() string { return "TerminalServerConnection" }

func (h terminalServerConnectionHandler) SupportsAccessGrants() bool { return true }

func (h terminalServerConnectionHandler) List(ctx context.Context, a *App, opts globalOptions) ([]resourceState, error) {
	cfg, err := a.fetchTerminalServerConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	states := make([]resourceState, 0, len(cfg.connections))
	for _, conn := range cfg.connections {
		states = append(states, h.stateFromConnection(conn))
	}
	return states, nil
}

func (h terminalServerConnectionHandler) Lookup(ctx context.Context, a *App, opts globalOptions, desired manifestDocument) (*resourceState, error) {
	cfg, err := a.fetchTerminalServerConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	_, conn, err := lookupTerminalServerConnection(cfg.connections, desired.Metadata.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}
	state := h.stateFromConnection(conn)
	return &state, nil
}

func (h terminalServerConnectionHandler) Fetch(ctx context.Context, a *App, opts globalOptions, id string) (*resourceState, error) {
	cfg, err := a.fetchTerminalServerConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	_, conn, err := lookupTerminalServerConnection(cfg.connections, id)
	if err != nil {
		return nil, err
	}
	state := h.stateFromConnection(conn)
	return &state, nil
}

func (h terminalServerConnectionHandler) Create(ctx context.Context, a *App, opts globalOptions, desired manifestDocument) (*resourceState, error) {
	cfg, err := a.fetchTerminalServerConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	conn, err := h.connectionFromManifest(nil, desired)
	if err != nil {
		return nil, err
	}
	cfg.connections = append(cfg.connections, conn)
	return h.postAndFetch(ctx, a, opts, cfg, desired.Metadata.Name)
}

func (h terminalServerConnectionHandler) Update(ctx context.Context, a *App, opts globalOptions, current resourceState, desired manifestDocument) (*resourceState, error) {
	cfg, err := a.fetchTerminalServerConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	idx, currentConn, err := lookupTerminalServerConnection(cfg.connections, firstNonEmpty(current.ID, current.Name, desired.Metadata.Name))
	if err != nil {
		return nil, err
	}
	conn, err := h.connectionFromManifest(currentConn, desired)
	if err != nil {
		return nil, err
	}
	cfg.connections[idx] = conn
	return h.postAndFetch(ctx, a, opts, cfg, desired.Metadata.Name)
}

func (h terminalServerConnectionHandler) Delete(context.Context, *App, globalOptions, resourceState) error {
	return errors.New("TerminalServerConnection manifests do not support delete or sync pruning")
}

func (h terminalServerConnectionHandler) UpdateAccessGrants(ctx context.Context, a *App, opts globalOptions, current resourceState, grants []accessGrant) (*resourceState, error) {
	cfg, err := a.fetchTerminalServerConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	idx, conn, err := lookupTerminalServerConnection(cfg.connections, firstNonEmpty(current.ID, current.Name))
	if err != nil {
		return nil, err
	}
	if err := setTerminalServerAccessGrants(conn, grants); err != nil {
		return nil, err
	}
	cfg.connections[idx] = conn
	return h.postAndFetch(ctx, a, opts, cfg, firstNonEmpty(current.ID, current.Name))
}

func (h terminalServerConnectionHandler) postAndFetch(ctx context.Context, a *App, opts globalOptions, cfg *terminalServerConfig, ref string) (*resourceState, error) {
	body, err := cfg.body()
	if err != nil {
		return nil, err
	}
	response, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: terminalServersConfigPath, body: bytes.NewReader(body), authRequired: true})
	if err != nil {
		return nil, err
	}
	posted := cfg
	if decoded, err := decodeTerminalServerConfig(response); err == nil {
		posted = decoded
	}
	_, conn, err := lookupTerminalServerConnection(posted.connections, ref)
	if err != nil {
		return nil, err
	}
	state := h.stateFromConnection(conn)
	return &state, nil
}

func (h terminalServerConnectionHandler) connectionFromManifest(current map[string]any, desired manifestDocument) (map[string]any, error) {
	conn := copySpec(current)
	for key, value := range desired.Spec {
		conn[key] = value
	}
	if strings.TrimSpace(stringValue(conn["id"])) == "" {
		conn["id"] = desired.Metadata.Name
	}
	if strings.TrimSpace(stringValue(conn["name"])) == "" {
		conn["name"] = desired.Metadata.Name
	}
	grants := desired.AccessGrants
	if !desired.HasAccessGrants {
		grants = terminalServerAccessGrants(current)
	}
	if desired.HasAccessGrants || len(grants) > 0 {
		if err := setTerminalServerAccessGrants(conn, grants); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

func (h terminalServerConnectionHandler) stateFromConnection(conn map[string]any) resourceState {
	id := stringValue(conn["id"])
	return resourceState{
		Kind:         h.Kind(),
		Name:         firstNonEmpty(id, stringValue(conn["name"])),
		ID:           id,
		Spec:         terminalServerManifestSpec(conn),
		AccessGrants: terminalServerAccessGrants(conn),
	}
}

func terminalServerManifestSpec(conn map[string]any) map[string]any {
	spec := copySpec(conn)
	config := terminalServerConfigObject(conn)
	if config == nil {
		return spec
	}
	configCopy := copySpec(config)
	delete(configCopy, "access_grants")
	if len(configCopy) == 0 {
		delete(spec, "config")
	} else {
		spec["config"] = configCopy
	}
	return spec
}

func (h modelHandler) Kind() string { return "Model" }

func (h modelHandler) SupportsAccessGrants() bool { return true }

func (h channelHandler) SupportsAccessGrants() bool { return true }

func (h channelHandler) Update(ctx context.Context, a *App, opts globalOptions, current resourceState, desired manifestDocument) (*resourceState, error) {
	latest, err := h.Fetch(ctx, a, opts, current.ID)
	if err != nil {
		return nil, err
	}
	payload := copySpec(latest.Spec)
	for key, value := range desired.Spec {
		payload[key] = value
	}
	desired.Spec = payload
	return h.endpointHandler.Update(ctx, a, opts, *latest, desired)
}

func (h channelHandler) UpdateAccessGrants(ctx context.Context, a *App, opts globalOptions, current resourceState, grants []accessGrant) (*resourceState, error) {
	latest, err := h.Fetch(ctx, a, opts, current.ID)
	if err != nil {
		return nil, err
	}
	payload := copySpec(latest.Spec)
	payload["access_grants"] = grants
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: fmt.Sprintf(h.updatePath, url.PathEscape(current.ID)), body: jsonBody(payload), authRequired: true})
	if err != nil {
		return nil, err
	}
	state, err := h.decodeState(body)
	if err != nil || state == nil || state.ID == "" {
		return state, err
	}
	return h.Fetch(ctx, a, opts, state.ID)
}

func (h modelHandler) List(ctx context.Context, a *App, opts globalOptions) ([]resourceState, error) {
	objects, err := a.listAllResources(ctx, opts, "/api/v1/models/list")
	if err != nil {
		return nil, err
	}
	var states []resourceState
	for _, object := range objects {
		states = append(states, h.stateFromObject(object))
	}
	return states, nil
}

func (h modelHandler) Lookup(ctx context.Context, a *App, opts globalOptions, desired manifestDocument) (*resourceState, error) {
	states, err := h.List(ctx, a, opts)
	if err != nil {
		return nil, err
	}
	for _, state := range states {
		if state.ID == desired.Metadata.Name {
			return &state, nil
		}
	}
	return nil, nil
}

func (h modelHandler) Fetch(ctx context.Context, a *App, opts globalOptions, id string) (*resourceState, error) {
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: appendQuery("/api/v1/models/model", map[string]string{"id": id}), authRequired: true})
	if err != nil {
		return nil, err
	}
	return h.decodeState(body)
}

func (h modelHandler) Create(ctx context.Context, a *App, opts globalOptions, desired manifestDocument) (*resourceState, error) {
	payload := modelPayload(desired.Spec)
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/models/create", body: jsonBody(payload), authRequired: true})
	if err != nil {
		return nil, err
	}
	state, err := h.decodeState(body)
	if err != nil || state == nil || state.ID == "" {
		return state, err
	}
	return h.Fetch(ctx, a, opts, state.ID)
}

func (h modelHandler) Update(ctx context.Context, a *App, opts globalOptions, current resourceState, desired manifestDocument) (*resourceState, error) {
	payload := modelPayload(desired.Spec)
	payload["id"] = current.ID
	if _, managed := desired.Spec["is_active"]; !managed {
		payload["is_active"] = current.Spec["is_active"]
	}
	payload["access_grants"] = normalizeAccessGrants(current.AccessGrants)
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/models/model/update", body: jsonBody(payload), authRequired: true})
	if err != nil {
		return nil, err
	}
	state, err := h.decodeState(body)
	if err != nil || state == nil || state.ID == "" {
		return state, err
	}
	return h.Fetch(ctx, a, opts, state.ID)
}

func (h modelHandler) Delete(ctx context.Context, a *App, opts globalOptions, current resourceState) error {
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/models/model/delete", body: jsonBody(map[string]string{"id": current.ID}), authRequired: true})
	if err != nil {
		return err
	}
	var deleted bool
	if err := json.Unmarshal(body, &deleted); err != nil {
		return fmt.Errorf("decode Model deletion result: %w", err)
	}
	if !deleted {
		return fmt.Errorf("server did not delete Model/%s", current.ID)
	}
	states, err := h.List(ctx, a, opts)
	if err != nil {
		return fmt.Errorf("verify Model deletion: %w", err)
	}
	for _, state := range states {
		if state.ID == current.ID {
			return fmt.Errorf("Model/%s still exists after deletion", current.ID)
		}
	}
	return nil
}

func (h modelHandler) UpdateAccessGrants(ctx context.Context, a *App, opts globalOptions, current resourceState, grants []accessGrant) (*resourceState, error) {
	payload := map[string]any{"id": current.ID, "name": firstNonEmpty(current.Name, current.ID), "access_grants": grants}
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: "/api/v1/models/model/access/update", body: jsonBody(payload), authRequired: true})
	if err != nil {
		return nil, err
	}
	state, err := h.decodeState(body)
	if err != nil || state == nil || state.ID == "" {
		return state, err
	}
	return h.Fetch(ctx, a, opts, state.ID)
}

func (h modelHandler) decodeState(body []byte) (*resourceState, error) {
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	state := h.stateFromObject(object)
	return &state, nil
}

func (h modelHandler) stateFromObject(object map[string]any) resourceState {
	return resourceState{
		Kind:         h.Kind(),
		Name:         stringValue(object["id"]),
		ID:           stringValue(object["id"]),
		Spec:         modelManagedSpec(object),
		AccessGrants: normalizeAccessGrants(accessGrantsFromValue(object["access_grants"])),
	}
}

func (a *App) runManifests(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		a.printManifestsHelp(a.out)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if len(pos) != 0 {
		fmt.Fprintf(a.err, "unexpected manifest argument %q\n", pos[0])
		return 1
	}
	manifestOpts := manifestOptions{
		files:       splitCSV(flags.values["file"]),
		directories: splitCSV(firstNonEmpty(flags.values["directory"], flags.values["dir"])),
		output:      firstNonEmpty(flags.values["output"], opts.output, "table"),
		dryRun:      flags.bools["dry-run"],
		scope:       flags.values["scope"],
		yes:         flags.bools["yes"] || flags.bools["confirm"],
	}

	switch action {
	case "diff", "apply", "sync":
		return a.runManifestWorkflow(ctx, opts, action, manifestOpts)
	case "help", "-h", "--help":
		a.printManifestsHelp(a.out)
		return 0
	default:
		fmt.Fprintf(a.err, "unknown manifests command %q\n", action)
		return 1
	}
}

func (a *App) printManifestsHelp(w io.Writer) {
	fmt.Fprint(w, `Manage declarative Open WebUI resource manifests.

Usage:
  oictl manifests diff --file resource.json [--output table|json]
  oictl manifests apply (--file resource.yaml | --directory manifests/) [--dry-run] [--output table|json]
  oictl manifests sync --scope knowledge|models|prompts|tools|skills|functions|groups|channels|all (--file resource.json | --directory manifests/) [--dry-run] [--yes]

Workflows:
  diff   Preview create, update, delete, access-grant, unchanged, and unsupported actions without mutation
  apply  Create or update declared resources without pruning omitted remote resources
  sync   Reconcile a scoped remote set to manifests and prune omitted resources after confirmation

Flags:
  --file path        JSON or YAML manifest file to load; comma-separated paths are accepted
  --directory path   Directory of JSON/YAML manifests to load deterministically; --dir is an alias
  --output string    Output format: table or json (default table)
  --dry-run          Produce a plan for apply or sync without sending mutation requests
  --scope string     Sync scope: knowledge, models, prompts, tools, skills, functions, groups, channels, or all
  --yes              Confirm destructive sync delete actions
  --confirm          Alias for --yes

Supported manifest kinds include ToolValve, FunctionValve, and TerminalServerConnection. Valve manifests and terminal server connections are not pruned by sync.
`)
}

func (a *App) runManifestWorkflow(ctx context.Context, opts globalOptions, action string, manifestOpts manifestOptions) int {
	handlers := manifestHandlers()
	docs, err := loadManifestDocuments(manifestOpts, handlers)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	pruneKinds, err := syncScopeKinds(action, manifestOpts.scope)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	resolver := principalResolver{app: a, opts: opts, userCache: map[string][]principalUser{}}
	dependencies := manifestDependencies{}
	if action == "apply" && !manifestOpts.dryRun {
		dependencies, err = preflightManifestDependencies(ctx, docs, handlers, &resolver)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
	}
	if action != "apply" || manifestOpts.dryRun || !dependencies.hasDeferred() {
		if err := resolver.resolveDocuments(ctx, docs); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
	}
	if action == "apply" && !manifestOpts.dryRun && dependencies.hasDeferred() {
		// Validate identities and create payloads across the whole set before
		// any phase mutates it. Valve reads wait for their owners; other
		// resource lookups are safe now.
		var identityDocs []manifestDocument
		for _, doc := range docs {
			if doc.Kind != "ToolValve" && doc.Kind != "FunctionValve" {
				identityDocs = append(identityDocs, doc)
			}
		}
		if _, err := buildManifestPlan(ctx, a, opts, identityDocs, handlers, nil); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		phases, err := dependencies.documentPhases(docs)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		plan := manifestPlan{}
		for _, phaseIndexes := range phases {
			phase := make([]manifestDocument, 0, len(phaseIndexes))
			for _, index := range phaseIndexes {
				phase = append(phase, docs[index])
			}
			if err := ensureResolvedGroupGrants(phase); err != nil {
				fmt.Fprintln(a.err, err)
				return 1
			}
			phasePlan, err := buildManifestPlan(ctx, a, opts, phase, handlers, nil)
			if err != nil {
				fmt.Fprintln(a.err, err)
				return 1
			}
			if err := executeManifestPlan(ctx, a, opts, phasePlan, action); err != nil {
				fmt.Fprintln(a.err, err)
				return 1
			}
			if err := dependencies.resolveAppliedGroups(ctx, a, opts, docs, handlers["Group"]); err != nil {
				fmt.Fprintln(a.err, err)
				return 1
			}
			plan.Actions = append(plan.Actions, phasePlan.Actions...)
		}
		if err := writePlan(a.out, plan, manifestOpts.output); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return 0
	}
	plan, err := buildManifestPlan(ctx, a, opts, docs, handlers, pruneKinds)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if action == "sync" && plan.hasDestructiveActions() && !manifestOpts.dryRun && !manifestOpts.yes {
		_ = writePlan(a.err, plan, "table")
		fmt.Fprintln(a.err, "destructive sync actions require --yes")
		return 1
	}
	if action == "diff" || manifestOpts.dryRun {
		if err := writePlan(a.out, plan, manifestOpts.output); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return 0
	}
	if err := executeManifestPlan(ctx, a, opts, plan, action); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if err := writePlan(a.out, plan, manifestOpts.output); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	return 0
}

func loadManifestDocuments(opts manifestOptions, handlers map[string]resourceHandler) ([]manifestDocument, error) {
	paths, err := discoverManifestPaths(opts)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, errors.New("at least one manifest --file or --directory is required")
	}
	docs := make([]manifestDocument, 0, len(paths))
	seen := map[string]string{}
	for _, path := range paths {
		doc, err := readManifestDocument(path)
		if err != nil {
			return nil, err
		}
		if err := validateManifestDocument(&doc, handlers); err != nil {
			return nil, err
		}
		identity := doc.Kind + "/" + doc.Metadata.Name
		if previous, ok := seen[identity]; ok {
			return nil, fmt.Errorf("duplicate manifest identity %s in %s and %s", identity, previous, doc.Source)
		}
		seen[identity] = doc.Source
		docs = append(docs, doc)
	}
	return docs, nil
}

func discoverManifestPaths(opts manifestOptions) ([]string, error) {
	var paths []string
	paths = append(paths, opts.files...)
	for _, dir := range opts.directories {
		info, err := os.Stat(dir)
		if err != nil {
			return nil, fmt.Errorf("read manifest directory %s: %w", dir, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("manifest directory %s is not a directory", dir)
		}
		if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if isSupportedManifestExtension(path) {
				paths = append(paths, path)
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walk manifest directory %s: %w", dir, err)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func isSupportedManifestExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func readManifestDocument(path string) (manifestDocument, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return manifestDocument{}, fmt.Errorf("read manifest %s: %w", path, err)
	}
	var doc manifestDocument
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(body, &doc); err != nil {
			return manifestDocument{}, fmt.Errorf("parse manifest %s: %w", path, err)
		}
		var raw map[string]any
		if err := yaml.Unmarshal(body, &raw); err != nil {
			return manifestDocument{}, fmt.Errorf("parse manifest %s: %w", path, err)
		}
		_, doc.HasAccessGrants = raw["access_grants"]
	default:
		if err := json.Unmarshal(body, &doc); err != nil {
			return manifestDocument{}, fmt.Errorf("parse manifest %s: %w", path, err)
		}
	}
	doc.Source = path
	if err := normalizeManifestDocument(&doc); err != nil {
		return manifestDocument{}, err
	}
	return doc, nil
}

func normalizeManifestDocument(doc *manifestDocument) error {
	var err error
	doc.APIVersion, err = renderManifestString(doc.Source, doc.APIVersion)
	if err != nil {
		return err
	}
	doc.Kind, err = renderManifestString(doc.Source, doc.Kind)
	if err != nil {
		return err
	}
	doc.Metadata.Name, err = renderManifestString(doc.Source, doc.Metadata.Name)
	if err != nil {
		return err
	}
	renderedSpec, err := renderManifestValue(doc.Source, doc.Spec)
	if err != nil {
		return err
	}
	if renderedSpec == nil {
		doc.Spec = nil
	} else if spec, ok := renderedSpec.(map[string]any); ok {
		doc.Spec = spec
	} else {
		return fmt.Errorf("manifest %s spec must be an object", doc.Source)
	}
	for i := range doc.AccessGrants {
		grant := &doc.AccessGrants[i]
		if grant.PrincipalType, err = renderManifestString(doc.Source, grant.PrincipalType); err != nil {
			return err
		}
		if grant.PrincipalID, err = renderManifestString(doc.Source, grant.PrincipalID); err != nil {
			return err
		}
		if grant.PrincipalRef, err = renderManifestString(doc.Source, grant.PrincipalRef); err != nil {
			return err
		}
		if grant.PrincipalEmail, err = renderManifestString(doc.Source, grant.PrincipalEmail); err != nil {
			return err
		}
		if grant.PrincipalName, err = renderManifestString(doc.Source, grant.PrincipalName); err != nil {
			return err
		}
		if grant.Permission, err = renderManifestString(doc.Source, grant.Permission); err != nil {
			return err
		}
	}
	if err := resolveManifestContentFile(doc); err != nil {
		return err
	}
	return nil
}

func renderManifestValue(source string, value any) (any, error) {
	switch typed := value.(type) {
	case string:
		return renderManifestString(source, typed)
	case map[string]any:
		for key, child := range typed {
			rendered, err := renderManifestValue(source, child)
			if err != nil {
				return nil, err
			}
			typed[key] = rendered
		}
		return typed, nil
	case []any:
		for i, child := range typed {
			rendered, err := renderManifestValue(source, child)
			if err != nil {
				return nil, err
			}
			typed[i] = rendered
		}
		return typed, nil
	default:
		return value, nil
	}
}

func renderManifestString(source string, value string) (string, error) {
	if !strings.Contains(value, "${") {
		return value, nil
	}
	var out strings.Builder
	for {
		start := strings.Index(value, "${")
		if start < 0 {
			out.WriteString(value)
			return out.String(), nil
		}
		out.WriteString(value[:start])
		value = value[start+2:]
		end := strings.IndexByte(value, '}')
		if end < 0 {
			out.WriteString("${")
			out.WriteString(value)
			return out.String(), nil
		}
		name := value[:end]
		if !isManifestEnvName(name) {
			return "", fmt.Errorf("manifest %s contains unsupported environment placeholder ${%s}; use simple ${VAR} placeholders", source, name)
		}
		replacement, ok := os.LookupEnv(name)
		if !ok {
			return "", fmt.Errorf("manifest %s references unset environment variable %s", source, name)
		}
		out.WriteString(replacement)
		value = value[end+1:]
	}
}

func isManifestEnvName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			continue
		}
		if i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}

func resolveManifestContentFile(doc *manifestDocument) error {
	if doc.Spec == nil {
		return nil
	}
	contentFile, hasContentFile := doc.Spec["content_file"]
	if !hasContentFile {
		return nil
	}
	if _, hasContent := doc.Spec["content"]; hasContent {
		return fmt.Errorf("manifest %s %s/%s declares both spec.content and spec.content_file", doc.Source, doc.Kind, doc.Metadata.Name)
	}
	path, ok := contentFile.(string)
	if !ok || strings.TrimSpace(path) == "" {
		return fmt.Errorf("manifest %s %s/%s spec.content_file must be a non-empty string", doc.Source, doc.Kind, doc.Metadata.Name)
	}
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(doc.Source), resolved)
	}
	body, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("manifest %s %s/%s read spec.content_file %s: %w", doc.Source, doc.Kind, doc.Metadata.Name, path, err)
	}
	if !utf8.Valid(body) {
		return fmt.Errorf("manifest %s %s/%s spec.content_file %s is not valid UTF-8", doc.Source, doc.Kind, doc.Metadata.Name, path)
	}
	doc.Spec["content"] = string(body)
	delete(doc.Spec, "content_file")
	return nil
}

func validateManifestDocument(doc *manifestDocument, handlers map[string]resourceHandler) error {
	if strings.TrimSpace(doc.APIVersion) == "" {
		return fmt.Errorf("manifest %s missing apiVersion", doc.Source)
	}
	if doc.APIVersion != manifestAPIVersion {
		return fmt.Errorf("manifest %s has unsupported apiVersion %q; expected %s", doc.Source, doc.APIVersion, manifestAPIVersion)
	}
	if strings.TrimSpace(doc.Kind) == "" {
		return fmt.Errorf("manifest %s missing kind", doc.Source)
	}
	handler, ok := handlers[doc.Kind]
	if !ok {
		return fmt.Errorf("manifest %s kind %q is unsupported; supported kinds: %s", doc.Source, doc.Kind, strings.Join(supportedManifestKinds(handlers), ", "))
	}
	if strings.TrimSpace(doc.Metadata.Name) == "" {
		return fmt.Errorf("manifest %s missing metadata.name", doc.Source)
	}
	if doc.Spec == nil {
		return fmt.Errorf("manifest %s missing spec", doc.Source)
	}
	if err := validateManifestFields(*doc); err != nil {
		return err
	}
	if doc.Kind == "Model" {
		if err := validateAndNormalizeModelDocument(doc); err != nil {
			return err
		}
	}
	if doc.HasAccessGrants {
		if !handler.SupportsAccessGrants() {
			return fmt.Errorf("manifest %s kind %s does not support access_grants", doc.Source, doc.Kind)
		}
		grants, err := validateAndNormalizeAccessGrants(doc.AccessGrants)
		if err != nil {
			return fmt.Errorf("manifest %s invalid access_grants: %w", doc.Source, err)
		}
		doc.AccessGrants = grants
	}
	return nil
}

func validateManifestFields(doc manifestDocument) error {
	if doc.Kind == "Tool" || doc.Kind == "Function" || doc.Kind == "Prompt" {
		for _, field := range []string{"id", "name", "command", "content"} {
			if value, supplied := doc.Spec[field]; supplied {
				text, ok := value.(string)
				if !ok || (field != "content" && strings.TrimSpace(text) == "") {
					return fmt.Errorf("%s spec.%s must be a non-empty string", doc.Kind, field)
				}
			}
		}
		if value, supplied := doc.Spec["meta"]; supplied && (doc.Kind == "Tool" || doc.Kind == "Function") {
			if object, ok := value.(map[string]any); !ok || object == nil {
				return fmt.Errorf("%s spec.meta must be an object", doc.Kind)
			}
		}
	}
	for _, field := range []string{"is_active", "is_global"} {
		if value, supplied := doc.Spec[field]; supplied && (doc.Kind == "Model" || doc.Kind == "Skill" || doc.Kind == "Function") {
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("%s spec.%s must be a boolean", doc.Kind, field)
			}
		}
	}
	return nil
}

type manifestPlan struct {
	Actions []planAction `json:"actions"`
}

type planAction struct {
	Action        string            `json:"action"`
	Kind          string            `json:"kind"`
	Name          string            `json:"name"`
	ID            string            `json:"id,omitempty"`
	Changes       []fieldChange     `json:"changes,omitempty"`
	GrantsAdded   []accessGrant     `json:"grants_added,omitempty"`
	GrantsRemoved []accessGrant     `json:"grants_removed,omitempty"`
	Message       string            `json:"message,omitempty"`
	Desired       *manifestDocument `json:"-"`
	Current       *resourceState    `json:"-"`
	Handler       resourceHandler   `json:"-"`
}

type fieldChange struct {
	Field   string `json:"field"`
	Current any    `json:"current,omitempty"`
	Desired any    `json:"desired,omitempty"`
}

type manifestDependencies struct {
	requires   map[string]map[string]bool
	groupOwner map[string]string
}

func (d manifestDependencies) hasDeferred() bool { return len(d.requires) > 0 }

func (d *manifestDependencies) require(dependent string, owner string) {
	if d.requires[dependent] == nil {
		d.requires[dependent] = map[string]bool{}
	}
	d.requires[dependent][owner] = true
}

func (d manifestDependencies) documentPhases(docs []manifestDocument) ([][]int, error) {
	remaining := make(map[string]int, len(docs))
	for index, doc := range docs {
		remaining[doc.Kind+"/"+doc.Metadata.Name] = index
	}
	completed := map[string]bool{}
	var phases [][]int
	for len(remaining) > 0 {
		var phase []int
		for index, doc := range docs {
			key := doc.Kind + "/" + doc.Metadata.Name
			if _, ok := remaining[key]; !ok {
				continue
			}
			ready := true
			for owner := range d.requires[key] {
				if !completed[owner] {
					ready = false
					break
				}
			}
			if ready {
				phase = append(phase, index)
			}
		}
		if len(phase) == 0 {
			keys := make([]string, 0, len(remaining))
			for key := range remaining {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			return nil, fmt.Errorf("manifest dependency cycle involving %s", keys[0])
		}
		for _, index := range phase {
			key := docs[index].Kind + "/" + docs[index].Metadata.Name
			delete(remaining, key)
			completed[key] = true
		}
		phases = append(phases, phase)
	}
	return phases, nil
}

func (d manifestDependencies) resolveAppliedGroups(ctx context.Context, a *App, opts globalOptions, docs []manifestDocument, handler resourceHandler) error {
	resolved := map[string]resourceState{}
	for docIndex := range docs {
		for grantIndex := range docs[docIndex].AccessGrants {
			grant := docs[docIndex].AccessGrants[grantIndex]
			if grant.PrincipalType != "group" || grant.PrincipalID != "" {
				continue
			}
			selectorKind, selector := grantSelector(grant)
			owner, managed := d.groupOwner[selectorKind+"\x00"+selector]
			if !managed {
				continue
			}
			state, ok := resolved[owner]
			if !ok {
				var ownerDoc *manifestDocument
				for index := range docs {
					if docs[index].Kind+"/"+docs[index].Metadata.Name == owner {
						ownerDoc = &docs[index]
						break
					}
				}
				if ownerDoc == nil {
					return fmt.Errorf("manifest dependency owner %s was not found", owner)
				}
				current, err := handler.Lookup(ctx, a, opts, *ownerDoc)
				if err != nil {
					return err
				}
				if current == nil || current.ID == "" {
					return fmt.Errorf("manifest %s %s/%s invalid access_grants: group principal %q could not be resolved after apply", docs[docIndex].Source, docs[docIndex].Kind, docs[docIndex].Metadata.Name, selector)
				}
				state = *current
				resolved[owner] = state
			}
			grant.PrincipalID = state.ID
			grant.PrincipalRef, grant.PrincipalEmail, grant.PrincipalName = "", "", ""
			docs[docIndex].AccessGrants[grantIndex] = grant
		}
		docs[docIndex].AccessGrants = normalizeAccessGrants(docs[docIndex].AccessGrants)
	}
	return nil
}

func ensureResolvedGroupGrants(docs []manifestDocument) error {
	for _, doc := range docs {
		for _, grant := range doc.AccessGrants {
			if grant.PrincipalType == "group" && grant.PrincipalID == "" {
				_, selector := grantSelector(grant)
				return fmt.Errorf("manifest %s %s/%s invalid access_grants: group principal %q could not be resolved after apply", doc.Source, doc.Kind, doc.Metadata.Name, selector)
			}
		}
	}
	return nil
}

func preflightManifestDependencies(ctx context.Context, docs []manifestDocument, handlers map[string]resourceHandler, resolver *principalResolver) (manifestDependencies, error) {
	dependencies := manifestDependencies{requires: map[string]map[string]bool{}, groupOwner: map[string]string{}}
	ownerStates := map[string][]resourceState{}
	for _, doc := range docs {
		ownerKind := ""
		switch doc.Kind {
		case "FunctionValve":
			ownerKind = "Function"
		case "ToolValve":
			ownerKind = "Tool"
		default:
			continue
		}
		sameSet := matchingOwnerDocuments(docs, ownerKind, doc.Metadata.Name)
		if len(sameSet) > 1 {
			return manifestDependencies{}, fmt.Errorf("manifest %s %s/%s owner %s/%s is ambiguous", doc.Source, doc.Kind, doc.Metadata.Name, ownerKind, doc.Metadata.Name)
		}
		states, ok := ownerStates[ownerKind]
		if !ok {
			var err error
			states, err = handlers[ownerKind].List(ctx, resolver.app, resolver.opts)
			if err != nil {
				return manifestDependencies{}, err
			}
			ownerStates[ownerKind] = states
		}
		remote := matchingOwnerStates(states, doc.Metadata.Name)
		if len(remote) > 1 {
			return manifestDependencies{}, fmt.Errorf("manifest %s %s/%s owner %s/%s is ambiguous", doc.Source, doc.Kind, doc.Metadata.Name, ownerKind, doc.Metadata.Name)
		}
		if len(sameSet) == 0 && len(remote) == 0 {
			return manifestDependencies{}, fmt.Errorf("manifest %s %s/%s owner %s/%s could not be resolved", doc.Source, doc.Kind, doc.Metadata.Name, ownerKind, doc.Metadata.Name)
		}
		if len(sameSet) == 1 {
			dependencies.require(doc.Kind+"/"+doc.Metadata.Name, ownerKind+"/"+sameSet[0].Metadata.Name)
		}
	}

	var groups []principalGroup
	groupsRead := false
	for docIndex := range docs {
		if !docs[docIndex].HasAccessGrants {
			continue
		}
		for grantIndex := range docs[docIndex].AccessGrants {
			grant := docs[docIndex].AccessGrants[grantIndex]
			if grant.PrincipalID != "" || grant.PrincipalType == "user" {
				resolved, err := resolver.resolveGrant(ctx, docs[docIndex], grant)
				if err != nil {
					return manifestDependencies{}, err
				}
				docs[docIndex].AccessGrants[grantIndex] = resolved
				continue
			}
			selectorKind, selector := grantSelector(grant)
			if !groupsRead {
				var err error
				groups, err = resolver.groups(ctx)
				if err != nil {
					return manifestDependencies{}, err
				}
				groupsRead = true
			}
			remote := matchingPrincipalGroups(groups, selectorKind, selector)
			if len(remote) > 1 {
				return manifestDependencies{}, fmt.Errorf("manifest %s %s/%s invalid access_grants: group principal %q is ambiguous", docs[docIndex].Source, docs[docIndex].Kind, docs[docIndex].Metadata.Name, selector)
			}
			sameSet := matchingGroupDocuments(docs, selectorKind, selector)
			if len(sameSet) > 1 {
				return manifestDependencies{}, fmt.Errorf("manifest %s %s/%s invalid access_grants: group principal %q is ambiguous", docs[docIndex].Source, docs[docIndex].Kind, docs[docIndex].Metadata.Name, selector)
			}
			if len(remote) == 1 {
				grant.PrincipalID = remote[0].ID
				grant.PrincipalRef, grant.PrincipalEmail, grant.PrincipalName = "", "", ""
				docs[docIndex].AccessGrants[grantIndex] = grant
				continue
			}
			if len(sameSet) == 0 {
				return manifestDependencies{}, fmt.Errorf("manifest %s %s/%s invalid access_grants: group principal %q could not be resolved", docs[docIndex].Source, docs[docIndex].Kind, docs[docIndex].Metadata.Name, selector)
			}
			owner := "Group/" + sameSet[0].Metadata.Name
			dependencies.require(docs[docIndex].Kind+"/"+docs[docIndex].Metadata.Name, owner)
			dependencies.groupOwner[selectorKind+"\x00"+selector] = owner
		}
		docs[docIndex].AccessGrants = normalizeAccessGrants(docs[docIndex].AccessGrants)
	}
	return dependencies, nil
}

func matchingOwnerDocuments(docs []manifestDocument, kind string, selector string) []manifestDocument {
	var matches []manifestDocument
	for _, doc := range docs {
		if doc.Kind == kind && firstNonEmpty(stringValue(doc.Spec["id"]), doc.Metadata.Name) == selector {
			matches = append(matches, doc)
		}
	}
	return matches
}

func matchingOwnerStates(states []resourceState, selector string) []resourceState {
	var matches []resourceState
	for _, state := range states {
		if state.ID == selector {
			matches = append(matches, state)
		}
	}
	return matches
}

func matchingGroupDocuments(docs []manifestDocument, selectorKind string, selector string) []manifestDocument {
	var matches []manifestDocument
	for _, doc := range docs {
		if doc.Kind != "Group" {
			continue
		}
		name := firstNonEmpty(stringValue(doc.Spec["name"]), doc.Metadata.Name)
		id := stringValue(doc.Spec["id"])
		if selectorKind == "name" && name == selector || selectorKind == "ref" && (doc.Metadata.Name == selector || id == selector || name == selector) {
			matches = append(matches, doc)
		}
	}
	return matches
}

func groupStateMatches(state resourceState, desired manifestDocument) bool {
	desiredAliases := []string{desired.Metadata.Name, stringValue(desired.Spec["id"]), stringValue(desired.Spec["name"])}
	stateAliases := []string{state.ID, state.Name, stringValue(state.Spec["id"]), stringValue(state.Spec["name"])}
	for _, desiredAlias := range desiredAliases {
		if desiredAlias == "" {
			continue
		}
		for _, stateAlias := range stateAliases {
			if desiredAlias == stateAlias {
				return true
			}
		}
	}
	return false
}

func canonicalChannelSpec(spec map[string]any) map[string]any {
	normalized := copySpec(spec)
	if name, ok := normalized["name"].(string); ok {
		normalized["name"] = strings.ToLower(name)
	}
	return normalized
}

func groupSpecForCompare(spec map[string]any) map[string]any {
	normalized := copySpec(spec)
	delete(normalized, "id")
	return normalized
}

func buildManifestPlan(ctx context.Context, a *App, opts globalOptions, docs []manifestDocument, handlers map[string]resourceHandler, pruneKinds map[string]bool) (manifestPlan, error) {
	plan := manifestPlan{}
	resolved := map[string]string{}
	channelCreates := map[string]string{}
	for i := range docs {
		doc := docs[i]
		handler := handlers[doc.Kind]
		current, err := handler.Lookup(ctx, a, opts, doc)
		if err != nil {
			return manifestPlan{}, err
		}
		if current == nil {
			if doc.Kind == "Channel" {
				name := strings.ToLower(firstNonEmpty(stringValue(doc.Spec["name"]), doc.Metadata.Name))
				if owner, exists := channelCreates[name]; exists {
					return manifestPlan{}, fmt.Errorf("Channel manifests %s and %s create the same canonical name %q", owner, doc.Metadata.Name, name)
				}
				channelCreates[name] = doc.Metadata.Name
			}
			plan.Actions = append(plan.Actions, planAction{Action: "create", Kind: doc.Kind, Name: doc.Metadata.Name, Desired: &docs[i], Handler: handler})
			continue
		}
		key := doc.Kind + "/" + current.ID
		if owner, exists := resolved[key]; exists {
			return manifestPlan{}, fmt.Errorf("manifests %s and %s resolve to the same remote %s", owner, doc.Metadata.Name, key)
		}
		resolved[key] = doc.Metadata.Name
		desiredSpec, currentSpec := doc.Spec, current.Spec
		if doc.Kind == "Channel" {
			desiredSpec = canonicalChannelSpec(desiredSpec)
			currentSpec = canonicalChannelSpec(currentSpec)
		}
		if doc.Kind == "Group" {
			desiredSpec = groupSpecForCompare(desiredSpec)
			currentSpec = groupSpecForCompare(currentSpec)
		}
		if doc.Kind == "Function" || doc.Kind == "Tool" {
			desiredSpec, currentSpec = functionSpecsForCompare(desiredSpec, currentSpec)
		}
		changes := diffSpec(desiredSpec, currentSpec)
		if doc.Kind == "Model" {
			changes = diffSpec(normalizeModelSpecForCompare(doc.Spec), normalizeModelSpecForCompare(current.Spec))
		}
		if len(changes) > 0 {
			plan.Actions = append(plan.Actions, planAction{Action: "update", Kind: doc.Kind, Name: doc.Metadata.Name, ID: current.ID, Changes: changes, Desired: &docs[i], Current: current, Handler: handler})
		}
		if doc.HasAccessGrants {
			added, removed := diffGrants(doc.AccessGrants, current.AccessGrants)
			if len(added) > 0 || len(removed) > 0 {
				plan.Actions = append(plan.Actions, planAction{Action: "replace-grants", Kind: doc.Kind, Name: doc.Metadata.Name, ID: current.ID, GrantsAdded: added, GrantsRemoved: removed, Desired: &docs[i], Current: current, Handler: handler})
			}
		}
		if len(changes) == 0 && (!doc.HasAccessGrants || grantsEqual(doc.AccessGrants, current.AccessGrants)) {
			plan.Actions = append(plan.Actions, planAction{Action: "unchanged", Kind: doc.Kind, Name: doc.Metadata.Name, ID: current.ID, Desired: &docs[i], Current: current, Handler: handler})
		}
	}
	for kind := range pruneKinds {
		handler := handlers[kind]
		states, err := handler.List(ctx, a, opts)
		if err != nil {
			return manifestPlan{}, err
		}
		for i := range states {
			state := states[i]
			if _, retained := resolved[kind+"/"+state.ID]; !retained {
				plan.Actions = append(plan.Actions, planAction{Action: "delete", Kind: kind, Name: firstNonEmpty(state.Name, state.ID), ID: state.ID, Current: &states[i], Handler: handler})
			}
		}
	}
	sort.SliceStable(plan.Actions, func(i, j int) bool {
		left, right := plan.Actions[i], plan.Actions[j]
		leftResource := left.Kind + "/" + left.Name
		rightResource := right.Kind + "/" + right.Name
		if leftResource != rightResource {
			return leftResource < rightResource
		}
		if left.Kind == "Model" && left.Action == "update" && right.Action == "replace-grants" {
			return true
		}
		if left.Kind == "Model" && left.Action == "replace-grants" && right.Action == "update" {
			return false
		}
		return left.Action < right.Action
	})
	return plan, validateManifestPlan(plan)
}

func executeManifestPlan(ctx context.Context, a *App, opts globalOptions, plan manifestPlan, workflow string) error {
	if err := validateManifestPlan(plan); err != nil {
		return err
	}
	created := map[string]resourceState{}
	for _, action := range plan.Actions {
		switch action.Action {
		case "unchanged", "unsupported":
			continue
		case "create":
			state, err := action.Handler.Create(ctx, a, opts, *action.Desired)
			if err != nil {
				return fmt.Errorf("create %s/%s: %w", action.Kind, action.Name, err)
			}
			if state != nil {
				created[action.Kind+"/"+action.Name] = *state
				if action.Desired.HasAccessGrants {
					updated, err := action.Handler.UpdateAccessGrants(ctx, a, opts, *state, action.Desired.AccessGrants)
					if err != nil {
						return fmt.Errorf("replace grants for %s/%s: %w", action.Kind, action.Name, err)
					}
					if updated != nil && !grantsEqual(action.Desired.AccessGrants, updated.AccessGrants) {
						return fmt.Errorf("server filtered access grants for %s/%s", action.Kind, action.Name)
					}
				}
			}
		case "update":
			if _, err := action.Handler.Update(ctx, a, opts, *action.Current, *action.Desired); err != nil {
				return fmt.Errorf("update %s/%s: %w", action.Kind, action.Name, err)
			}
		case "replace-grants":
			current := action.Current
			if createdState, ok := created[action.Kind+"/"+action.Name]; ok {
				current = &createdState
			}
			updated, err := action.Handler.UpdateAccessGrants(ctx, a, opts, *current, action.Desired.AccessGrants)
			if err != nil {
				return fmt.Errorf("replace grants for %s/%s: %w", action.Kind, action.Name, err)
			}
			if updated != nil && !grantsEqual(action.Desired.AccessGrants, updated.AccessGrants) {
				return fmt.Errorf("server filtered access grants for %s/%s", action.Kind, action.Name)
			}
		case "delete":
			if workflow != "sync" {
				continue
			}
			if err := action.Handler.Delete(ctx, a, opts, *action.Current); err != nil {
				return fmt.Errorf("delete %s/%s: %w", action.Kind, action.Name, err)
			}
		}
	}
	return nil
}

func validateManifestPlan(plan manifestPlan) error {
	retained, deleted := map[string]bool{}, map[string]bool{}
	for _, action := range plan.Actions {
		if action.Action == "create" && (action.Kind == "Tool" || action.Kind == "Function" || action.Kind == "Prompt") {
			// Use the same payload defaults and requirements as Create, before
			// any action (or dependency phase) is allowed to mutate resources.
			if _, err := (endpointHandler{kind: action.Kind}).manifestPayload(*action.Desired, nil); err != nil {
				return fmt.Errorf("create %s/%s: %w", action.Kind, action.Name, err)
			}
		}
		id := action.ID
		if action.Current != nil {
			id = firstNonEmpty(action.Current.ID, id)
		}
		if id == "" {
			continue
		}
		key := action.Kind + "/" + id
		if action.Action == "delete" {
			deleted[key] = true
		} else {
			retained[key] = true
		}
		if deleted[key] && retained[key] {
			return fmt.Errorf("conflicting manifest plan retains and deletes %s", key)
		}
	}
	return nil
}

func writePlan(w io.Writer, plan manifestPlan, output string) error {
	if output == "json" {
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		return encoder.Encode(plan)
	}
	if output != "" && output != "table" {
		return fmt.Errorf("unsupported output format %q", output)
	}
	if len(plan.Actions) == 0 {
		_, err := fmt.Fprintln(w, "No changes")
		return err
	}
	for _, action := range plan.Actions {
		line := strings.ToUpper(action.Action) + " " + action.Kind + "/" + action.Name
		if action.ID != "" {
			line += " id=" + action.ID
		}
		if len(action.Changes) > 0 {
			fields := make([]string, 0, len(action.Changes))
			for _, change := range action.Changes {
				fields = append(fields, change.Field)
			}
			line += " fields=" + strings.Join(fields, ",")
		}
		if len(action.GrantsAdded) > 0 || len(action.GrantsRemoved) > 0 {
			line += fmt.Sprintf(" grants=+%d,-%d", len(action.GrantsAdded), len(action.GrantsRemoved))
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func syncScopeKinds(action string, scope string) (map[string]bool, error) {
	if action != "sync" {
		return nil, nil
	}
	if strings.TrimSpace(scope) == "" {
		return nil, errors.New("manifests sync requires --scope knowledge|models|prompts|tools|skills|functions|groups|channels|all")
	}
	switch strings.ToLower(scope) {
	case "knowledge":
		return map[string]bool{"Knowledge": true}, nil
	case "model", "models":
		return map[string]bool{"Model": true}, nil
	case "prompt", "prompts":
		return map[string]bool{"Prompt": true}, nil
	case "tool", "tools":
		return map[string]bool{"Tool": true}, nil
	case "skill", "skills":
		return map[string]bool{"Skill": true}, nil
	case "function", "functions":
		return map[string]bool{"Function": true}, nil
	case "group", "groups":
		return map[string]bool{"Group": true}, nil
	case "channel", "channels":
		return map[string]bool{"Channel": true}, nil
	case "all":
		return map[string]bool{"Channel": true, "Function": true, "Group": true, "Knowledge": true, "Model": true, "Prompt": true, "Skill": true, "Tool": true}, nil
	default:
		return nil, fmt.Errorf("unsupported sync scope %q", scope)
	}
}

func supportedManifestKinds(handlers map[string]resourceHandler) []string {
	kinds := make([]string, 0, len(handlers))
	for kind := range handlers {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

func validateAndNormalizeModelDocument(doc *manifestDocument) error {
	id := strings.TrimSpace(stringValue(doc.Spec["id"]))
	if id == "" {
		doc.Spec["id"] = doc.Metadata.Name
	} else if id != doc.Metadata.Name {
		return fmt.Errorf("manifest %s Model identity is inconsistent: metadata.name %q must match spec.id %q", doc.Source, doc.Metadata.Name, id)
	}
	if strings.TrimSpace(stringValue(doc.Spec["name"])) == "" {
		return fmt.Errorf("manifest %s kind Model requires spec.name", doc.Source)
	}
	if doc.Spec["meta"] == nil {
		doc.Spec["meta"] = map[string]any{}
	}
	if doc.Spec["params"] == nil {
		doc.Spec["params"] = map[string]any{}
	}
	return nil
}

func (p manifestPlan) hasDestructiveActions() bool {
	for _, action := range p.Actions {
		if action.Action == "delete" {
			return true
		}
	}
	return false
}

func validateAndNormalizeAccessGrants(grants []accessGrant) ([]accessGrant, error) {
	validated := make([]accessGrant, 0, len(grants))
	for _, grant := range grants {
		grant.PrincipalType = strings.TrimSpace(grant.PrincipalType)
		grant.PrincipalID = strings.TrimSpace(grant.PrincipalID)
		grant.PrincipalRef = strings.TrimSpace(grant.PrincipalRef)
		grant.PrincipalEmail = strings.TrimSpace(grant.PrincipalEmail)
		grant.PrincipalName = strings.TrimSpace(grant.PrincipalName)
		grant.Permission = strings.TrimSpace(grant.Permission)
		if grant.PrincipalType != "user" && grant.PrincipalType != "group" {
			return nil, errors.New("only user and group grant principal types are supported")
		}
		selectors := 0
		for _, selector := range []string{grant.PrincipalID, grant.PrincipalRef, grant.PrincipalEmail, grant.PrincipalName} {
			if selector != "" {
				selectors++
			}
		}
		if selectors == 0 {
			return nil, errors.New("exactly one principal selector is required")
		}
		if selectors > 1 {
			return nil, errors.New("only one principal selector is allowed")
		}
		if grant.PrincipalEmail != "" && grant.PrincipalType != "user" {
			return nil, errors.New("principal_email is only supported for user grants")
		}
		if refType, refValue, ok := splitPrincipalRef(grant.PrincipalRef); ok {
			if refValue == "" {
				return nil, errors.New("principal_ref selector is required")
			}
			if refType != grant.PrincipalType {
				return nil, fmt.Errorf("principal_ref type %q does not match principal_type %q", refType, grant.PrincipalType)
			}
		}
		if grant.Permission != "read" && grant.Permission != "write" {
			return nil, errors.New("only read and write permissions are supported")
		}
		validated = append(validated, grant)
	}
	return normalizeAccessGrants(validated), nil
}

func validateAndNormalizeResolvedAccessGrants(grants []accessGrant) ([]accessGrant, error) {
	validated := make([]accessGrant, 0, len(grants))
	for _, grant := range grants {
		grant.PrincipalType = strings.TrimSpace(grant.PrincipalType)
		grant.PrincipalID = strings.TrimSpace(grant.PrincipalID)
		grant.PrincipalRef = strings.TrimSpace(grant.PrincipalRef)
		grant.PrincipalEmail = strings.TrimSpace(grant.PrincipalEmail)
		grant.PrincipalName = strings.TrimSpace(grant.PrincipalName)
		grant.Permission = strings.TrimSpace(grant.Permission)
		if grant.PrincipalType != "user" && grant.PrincipalType != "group" {
			return nil, errors.New("only user and group grant principal types are supported")
		}
		if grant.PrincipalID == "" {
			return nil, errors.New("principal_id is required")
		}
		if grant.PrincipalRef != "" || grant.PrincipalEmail != "" || grant.PrincipalName != "" {
			return nil, errors.New("principal_ref, principal_email, and principal_name are only supported in manifests")
		}
		if grant.Permission != "read" && grant.Permission != "write" {
			return nil, errors.New("only read and write permissions are supported")
		}
		validated = append(validated, grant)
	}
	return normalizeAccessGrants(validated), nil
}

func normalizeAccessGrants(grants []accessGrant) []accessGrant {
	seen := map[string]accessGrant{}
	var unresolved []accessGrant
	for _, grant := range grants {
		if grant.PrincipalID == "" {
			unresolved = append(unresolved, grant)
			continue
		}
		grant.PrincipalRef = ""
		grant.PrincipalEmail = ""
		grant.PrincipalName = ""
		key := grant.PrincipalType + "\x00" + grant.PrincipalID + "\x00" + grant.Permission
		seen[key] = grant
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]accessGrant, 0, len(keys)+len(unresolved))
	for _, key := range keys {
		result = append(result, seen[key])
	}
	result = append(result, unresolved...)
	return result
}

func (r *principalResolver) resolveDocuments(ctx context.Context, docs []manifestDocument) error {
	for docIndex := range docs {
		if !docs[docIndex].HasAccessGrants {
			continue
		}
		for grantIndex := range docs[docIndex].AccessGrants {
			grant, err := r.resolveGrant(ctx, docs[docIndex], docs[docIndex].AccessGrants[grantIndex])
			if err != nil {
				return err
			}
			docs[docIndex].AccessGrants[grantIndex] = grant
		}
		docs[docIndex].AccessGrants = normalizeAccessGrants(docs[docIndex].AccessGrants)
	}
	return nil
}

func (r *principalResolver) resolveGrant(ctx context.Context, doc manifestDocument, grant accessGrant) (accessGrant, error) {
	if grant.PrincipalID != "" {
		grant.PrincipalRef = ""
		grant.PrincipalEmail = ""
		grant.PrincipalName = ""
		return grant, nil
	}
	selectorKind, selector := grantSelector(grant)
	if selector == "" {
		return grant, fmt.Errorf("manifest %s %s/%s invalid access_grants: exactly one principal selector is required", doc.Source, doc.Kind, doc.Metadata.Name)
	}
	resolvedID, err := r.resolvePrincipalID(ctx, grant.PrincipalType, selectorKind, selector)
	if err != nil {
		return grant, fmt.Errorf("manifest %s %s/%s invalid access_grants: %w", doc.Source, doc.Kind, doc.Metadata.Name, err)
	}
	grant.PrincipalID = resolvedID
	grant.PrincipalRef = ""
	grant.PrincipalEmail = ""
	grant.PrincipalName = ""
	return grant, nil
}

func (r *principalResolver) resolvePrincipalID(ctx context.Context, principalType string, selectorKind string, selector string) (string, error) {
	switch principalType {
	case "user":
		return r.resolveUserID(ctx, selectorKind, selector)
	case "group":
		return r.resolveGroupID(ctx, selectorKind, selector)
	default:
		return "", fmt.Errorf("unsupported principal_type %q", principalType)
	}
}

func (r *principalResolver) resolveUserID(ctx context.Context, selectorKind string, selector string) (string, error) {
	users, err := r.users(ctx, selector)
	if err != nil {
		return "", err
	}
	if selectorKind == "ref" {
		for _, user := range users {
			if user.ID == selector {
				return user.ID, nil
			}
		}
	}
	var matches []principalUser
	for _, user := range users {
		switch selectorKind {
		case "email":
			if user.Email == selector {
				matches = append(matches, user)
			}
		case "name":
			if user.Name == selector {
				matches = append(matches, user)
			}
		case "ref":
			if user.ID == selector || user.Email == selector || user.Name == selector {
				matches = append(matches, user)
			}
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("user principal %q could not be resolved", selector)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("user principal %q is ambiguous", selector)
	}
	return matches[0].ID, nil
}

func (r *principalResolver) resolveGroupID(ctx context.Context, selectorKind string, selector string) (string, error) {
	groups, err := r.groups(ctx)
	if err != nil {
		return "", err
	}
	matches := matchingPrincipalGroups(groups, selectorKind, selector)
	if len(matches) == 0 {
		return "", fmt.Errorf("group principal %q could not be resolved", selector)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("group principal %q is ambiguous", selector)
	}
	return matches[0].ID, nil
}

func matchingPrincipalGroups(groups []principalGroup, selectorKind string, selector string) []principalGroup {
	var matches []principalGroup
	for _, group := range groups {
		switch selectorKind {
		case "name":
			if group.Name == selector {
				matches = append(matches, group)
			}
		case "ref":
			if group.ID == selector || group.Name == selector {
				matches = append(matches, group)
			}
		}
	}
	return matches
}

func (r *principalResolver) users(ctx context.Context, selector string) ([]principalUser, error) {
	if r.userCache == nil {
		r.userCache = map[string][]principalUser{}
	}
	if users, ok := r.userCache[""]; ok {
		return users, nil
	}
	objects, err := r.app.listAllResources(ctx, r.opts, "/api/v1/users/")
	if err != nil {
		return nil, err
	}
	users := make([]principalUser, 0, len(objects))
	for _, object := range objects {
		users = append(users, principalUser{ID: stringValue(object["id"]), Name: stringValue(object["name"]), Email: stringValue(object["email"])})
	}
	r.userCache[""] = users
	return users, nil
}

func (r *principalResolver) groups(ctx context.Context) ([]principalGroup, error) {
	if r.groupsRead {
		return r.groupCache, nil
	}
	objects, err := r.app.listAllResources(ctx, r.opts, "/api/v1/groups/")
	if err != nil {
		return nil, err
	}
	groups := make([]principalGroup, 0, len(objects))
	for _, object := range objects {
		groups = append(groups, principalGroup{ID: stringValue(object["id"]), Name: stringValue(object["name"])})
	}
	r.groupCache = groups
	r.groupsRead = true
	return groups, nil
}

func grantSelector(grant accessGrant) (string, string) {
	if grant.PrincipalEmail != "" {
		return "email", grant.PrincipalEmail
	}
	if grant.PrincipalName != "" {
		return "name", grant.PrincipalName
	}
	if _, value, ok := splitPrincipalRef(grant.PrincipalRef); ok {
		return "ref", value
	}
	if grant.PrincipalRef != "" {
		return "ref", grant.PrincipalRef
	}
	return "", ""
}

func splitPrincipalRef(ref string) (string, string, bool) {
	prefix, value, ok := strings.Cut(ref, ":")
	if !ok {
		return "", ref, false
	}
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	value = strings.TrimSpace(value)
	if prefix != "user" && prefix != "group" {
		return "", ref, false
	}
	return prefix, value, true
}

func decodePrincipalUsers(body []byte) ([]principalUser, error) {
	objects, err := decodeResourceList(body)
	if err != nil {
		return nil, err
	}
	users := make([]principalUser, 0, len(objects))
	for _, object := range objects {
		users = append(users, principalUser{ID: stringValue(object["id"]), Email: stringValue(object["email"]), Name: stringValue(object["name"])})
	}
	return users, nil
}

func decodePrincipalGroups(body []byte) ([]principalGroup, error) {
	objects, err := decodeResourceList(body)
	if err != nil {
		return nil, err
	}
	groups := make([]principalGroup, 0, len(objects))
	for _, object := range objects {
		groups = append(groups, principalGroup{ID: stringValue(object["id"]), Name: stringValue(object["name"])})
	}
	return groups, nil
}

func diffGrants(desired []accessGrant, current []accessGrant) ([]accessGrant, []accessGrant) {
	desired = normalizeAccessGrants(desired)
	current = normalizeAccessGrants(current)
	desiredSet := map[string]accessGrant{}
	currentSet := map[string]accessGrant{}
	for _, grant := range desired {
		desiredSet[grant.PrincipalType+"\x00"+grant.PrincipalID+"\x00"+grant.Permission] = grant
	}
	for _, grant := range current {
		currentSet[grant.PrincipalType+"\x00"+grant.PrincipalID+"\x00"+grant.Permission] = grant
	}
	var added, removed []accessGrant
	for key, grant := range desiredSet {
		if _, ok := currentSet[key]; !ok {
			added = append(added, grant)
		}
	}
	for key, grant := range currentSet {
		if _, ok := desiredSet[key]; !ok {
			removed = append(removed, grant)
		}
	}
	sortAccessGrants(added)
	sortAccessGrants(removed)
	return added, removed
}

func grantsEqual(left []accessGrant, right []accessGrant) bool {
	added, removed := diffGrants(left, right)
	return len(added) == 0 && len(removed) == 0
}

func sortAccessGrants(grants []accessGrant) {
	sort.Slice(grants, func(i, j int) bool {
		left := grants[i].PrincipalType + "/" + grants[i].PrincipalID + "/" + grants[i].Permission
		right := grants[j].PrincipalType + "/" + grants[j].PrincipalID + "/" + grants[j].Permission
		return left < right
	})
}

func accessGrantsFromValue(value any) []accessGrant {
	body, err := json.Marshal(value)
	if err != nil || bytes.Equal(body, []byte("null")) {
		return nil
	}
	var grants []accessGrant
	if err := json.Unmarshal(body, &grants); err != nil {
		return nil
	}
	return grants
}

func diffSpec(desired map[string]any, current map[string]any) []fieldChange {
	changes := make([]fieldChange, 0)
	fields := make([]string, 0, len(desired))
	for field := range desired {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	for _, field := range fields {
		if !jsonEqual(desired[field], current[field]) {
			changes = append(changes, fieldChange{Field: field, Current: current[field], Desired: desired[field]})
		}
	}
	return changes
}

func jsonEqual(left any, right any) bool {
	var leftNorm, rightNorm any
	leftBody, _ := json.Marshal(left)
	rightBody, _ := json.Marshal(right)
	_ = json.Unmarshal(leftBody, &leftNorm)
	_ = json.Unmarshal(rightBody, &rightNorm)
	return reflect.DeepEqual(leftNorm, rightNorm)
}

func decodeResourceList(body []byte) ([]map[string]any, error) {
	var array []map[string]any
	if err := json.Unmarshal(body, &array); err == nil {
		return array, nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	for _, key := range []string{"items", "data", "results", "users"} {
		if raw, ok := object[key]; ok {
			if err := json.Unmarshal(raw, &array); err != nil {
				return nil, err
			}
			return array, nil
		}
	}
	return nil, errors.New("expected JSON array or object with items")
}

func decodeModelList(body []byte) ([]map[string]any, int, error) {
	var array []map[string]any
	if err := json.Unmarshal(body, &array); err == nil {
		return array, 0, nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, 0, err
	}
	if raw, ok := object["items"]; ok {
		if err := json.Unmarshal(raw, &array); err != nil {
			return nil, 0, err
		}
		return array, intFromRawJSON(object["total"]), nil
	}
	return nil, 0, errors.New("expected JSON array or object with items")
}

func intFromRawJSON(raw json.RawMessage) int {
	if raw == nil {
		return 0
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err != nil {
		return 0
	}
	return int(number)
}

func modelManagedSpec(object map[string]any) map[string]any {
	spec := map[string]any{}
	for _, field := range []string{"id", "base_model_id", "name", "meta", "params", "is_active"} {
		if value, ok := object[field]; ok {
			spec[field] = value
		}
	}
	if spec["meta"] == nil {
		spec["meta"] = map[string]any{}
	}
	if spec["params"] == nil {
		spec["params"] = map[string]any{}
	}
	return spec
}

func normalizeModelSpecForCompare(spec map[string]any) map[string]any {
	normalized := copySpec(spec)
	meta, ok := normalized["meta"].(map[string]any)
	if !ok {
		return normalized
	}
	metaCopy := make(map[string]any, len(meta))
	for key, value := range meta {
		metaCopy[key] = value
	}
	for _, key := range []string{"capabilities", "description", "knowledge", "profile_image_url"} {
		if value, ok := metaCopy[key]; ok && value == nil {
			delete(metaCopy, key)
		}
	}
	normalized["meta"] = metaCopy
	return normalized
}

func functionSpecsForCompare(desired map[string]any, current map[string]any) (map[string]any, map[string]any) {
	desiredMeta, ok := desired["meta"].(map[string]any)
	if _, supplied := desired["meta"]; supplied && !ok {
		return desired, current
	}
	currentMeta, ok := current["meta"].(map[string]any)
	if !ok {
		return desired, current
	}
	normalizedCurrent := copySpec(current)
	normalizedMeta := copySpec(currentMeta)
	// Source manifests are server-derived unless explicitly managed. Form
	// defaults must not turn an explicitly empty meta object into perpetual drift.
	if _, managed := desiredMeta["manifest"]; !managed {
		delete(normalizedMeta, "manifest")
	}
	if _, managed := desiredMeta["description"]; !managed && normalizedMeta["description"] == nil {
		delete(normalizedMeta, "description")
	}
	if _, managed := desiredMeta["has_user_valves"]; !managed && normalizedMeta["has_user_valves"] == false {
		delete(normalizedMeta, "has_user_valves")
	}
	normalizedCurrent["meta"] = normalizedMeta
	return desired, normalizedCurrent
}

func managedSpec(object map[string]any, fields []string) map[string]any {
	if len(fields) == 0 {
		return object
	}
	spec := map[string]any{}
	for _, field := range fields {
		if value, ok := object[field]; ok {
			spec[field] = value
		}
	}
	return spec
}

func modelPayload(spec map[string]any) map[string]any {
	payload := modelManagedSpec(spec)
	if _, ok := spec["is_active"]; !ok {
		delete(payload, "is_active")
	}
	return payload
}

func resourceName(object map[string]any, fields []string) string {
	for _, field := range fields {
		if value := stringValue(object[field]); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func copySpec(spec map[string]any) map[string]any {
	copy := make(map[string]any, len(spec))
	for key, value := range spec {
		copy[key] = value
	}
	return copy
}

func ensureIdentityFields(spec map[string]any, name string, fields []string) {
	for _, field := range fields {
		if field == "id" {
			continue
		}
		if strings.TrimSpace(stringValue(spec[field])) != "" {
			return
		}
	}
	if len(fields) > 0 {
		field := fields[0]
		if field == "id" && len(fields) > 1 {
			field = fields[1]
		}
		spec[field] = name
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
