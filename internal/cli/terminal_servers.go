package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const terminalServersConfigPath = "/api/v1/configs/terminal_servers"
const terminalServerConnectionsKey = "TERMINAL_SERVER_CONNECTIONS"

type terminalServerConfig struct {
	raw         map[string]json.RawMessage
	connections []map[string]any
}

func (a *App) runTerminalServerAccessGrants(ctx context.Context, opts globalOptions, args []string, flags commandFlags) int {
	if len(args) < 2 {
		fmt.Fprintln(a.err, "usage: oictl config terminal-servers access-grants <get|diff|set> <connection-id-or-name> [--file grants.json]")
		return 1
	}
	action, ref := args[0], args[1]
	if len(args) != 2 {
		fmt.Fprintln(a.err, "unexpected terminal server access-grants argument")
		return 1
	}

	switch action {
	case "get":
		cfg, err := a.fetchTerminalServerConfig(ctx, opts)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		_, conn, err := lookupTerminalServerConnection(cfg.connections, ref)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return writeJSON(a.out, a.err, terminalServerAccessGrants(conn))
	case "diff":
		desired, err := readDesiredAccessGrants(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		cfg, err := a.fetchTerminalServerConfig(ctx, opts)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		_, conn, err := lookupTerminalServerConnection(cfg.connections, ref)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		added, removed := diffGrants(desired, terminalServerAccessGrants(conn))
		return writeJSON(a.out, a.err, map[string]any{"grants_added": added, "grants_removed": removed})
	case "set":
		desired, err := readDesiredAccessGrants(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		cfg, err := a.fetchTerminalServerConfig(ctx, opts)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		idx, conn, err := lookupTerminalServerConnection(cfg.connections, ref)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := setTerminalServerAccessGrants(conn, desired); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		cfg.connections[idx] = conn
		body, err := cfg.body()
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		response, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: terminalServersConfigPath, body: bytes.NewReader(body), authRequired: true})
		if err != nil {
			var statusErr httpStatusError
			if !errors.As(err, &statusErr) {
				fmt.Fprintln(a.err, err)
			}
			return 1
		}
		returned := desired
		if decoded, err := decodeTerminalServerConfig(response); err == nil {
			if _, returnedConn, err := lookupTerminalServerConnection(decoded.connections, ref); err == nil {
				returned = terminalServerAccessGrants(returnedConn)
			}
		}
		missing, extra := diffGrants(desired, returned)
		result := map[string]any{"connection": ref, "access_grants": returned}
		if len(missing) > 0 || len(extra) > 0 {
			result["server_grants_missing"] = missing
			result["server_grants_extra"] = extra
		}
		return writeJSON(a.out, a.err, result)
	default:
		fmt.Fprintf(a.err, "unknown terminal server access-grants command %q\n", action)
		return 1
	}
}

func (a *App) fetchTerminalServerConfig(ctx context.Context, opts globalOptions) (*terminalServerConfig, error) {
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: terminalServersConfigPath, authRequired: true})
	if err != nil {
		return nil, err
	}
	cfg, err := decodeTerminalServerConfig(body)
	if err != nil {
		return nil, fmt.Errorf("decode terminal server config: %w", err)
	}
	return cfg, nil
}

func decodeTerminalServerConfig(body []byte) (*terminalServerConfig, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	var connections []map[string]any
	if value, ok := raw[terminalServerConnectionsKey]; ok && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		if err := json.Unmarshal(value, &connections); err != nil {
			return nil, fmt.Errorf("decode %s: %w", terminalServerConnectionsKey, err)
		}
	}
	if connections == nil {
		connections = []map[string]any{}
	}
	return &terminalServerConfig{raw: raw, connections: connections}, nil
}

func (c *terminalServerConfig) body() ([]byte, error) {
	if c.raw == nil {
		c.raw = map[string]json.RawMessage{}
	}
	connections, err := json.Marshal(c.connections)
	if err != nil {
		return nil, err
	}
	c.raw[terminalServerConnectionsKey] = connections
	return json.Marshal(c.raw)
}

func lookupTerminalServerConnection(connections []map[string]any, ref string) (int, map[string]any, error) {
	ref = strings.TrimSpace(ref)
	for i, conn := range connections {
		if stringValue(conn["id"]) == ref {
			return i, conn, nil
		}
	}
	match := -1
	for i, conn := range connections {
		if stringValue(conn["name"]) != ref {
			continue
		}
		if match != -1 {
			return -1, nil, fmt.Errorf("terminal server connection name %q is ambiguous", ref)
		}
		match = i
	}
	if match == -1 {
		return -1, nil, fmt.Errorf("terminal server connection %q not found", ref)
	}
	return match, connections[match], nil
}

func readDesiredAccessGrants(flags commandFlags) ([]accessGrant, error) {
	body, err := requestBody(flags)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, errors.New("access grant JSON is required; use --file grants.json or --data JSON")
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return decodeDesiredAccessGrants(data)
}

func decodeDesiredAccessGrants(data []byte) ([]accessGrant, error) {
	var grants []accessGrant
	if err := json.Unmarshal(data, &grants); err == nil {
		return validateAndNormalizeResolvedAccessGrants(grants)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, err
	}
	raw, ok := object["access_grants"]
	if !ok {
		return nil, errors.New("grant file must be a JSON array or an object with access_grants")
	}
	if err := json.Unmarshal(raw, &grants); err != nil {
		return nil, fmt.Errorf("decode access_grants: %w", err)
	}
	return validateAndNormalizeResolvedAccessGrants(grants)
}

func terminalServerAccessGrants(conn map[string]any) []accessGrant {
	config := terminalServerConfigObject(conn)
	if config == nil {
		return []accessGrant{}
	}
	return normalizeAccessGrants(accessGrantsFromValue(config["access_grants"]))
}

func setTerminalServerAccessGrants(conn map[string]any, grants []accessGrant) error {
	config, err := mutableTerminalServerConfigObject(conn)
	if err != nil {
		return err
	}
	config["access_grants"] = grants
	conn["config"] = config
	return nil
}

func terminalServerConfigObject(conn map[string]any) map[string]any {
	config, _ := conn["config"].(map[string]any)
	if config != nil {
		return config
	}
	if conn["config"] == nil {
		return nil
	}
	body, err := json.Marshal(conn["config"])
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(body, &config); err != nil {
		return nil
	}
	return config
}

func mutableTerminalServerConfigObject(conn map[string]any) (map[string]any, error) {
	if conn["config"] == nil {
		return map[string]any{}, nil
	}
	config := terminalServerConfigObject(conn)
	if config == nil {
		return nil, errors.New("terminal server connection config is not an object")
	}
	// Callers may have overlaid a manifest's config map. Never insert stored
	// ACLs into that caller-owned map or later comparisons will drift.
	return copySpec(config), nil
}

func writeJSON(out io.Writer, errOut io.Writer, value any) int {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	return 0
}
