package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

func (a *App) runChannels(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		a.printChannelsHelp(a.out)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	format := outputFormat(opts, flags, "table")
	simple := func(method, path string, body io.Reader) int {
		return a.executeStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
	}
	channelPath := func(id string, suffix string) string {
		return "/api/v1/channels/" + url.PathEscape(id) + suffix
	}
	switch action {
	case "list":
		if len(pos) != 0 {
			fmt.Fprintln(a.err, "usage: oictl channels list")
			return 1
		}
		return simple(http.MethodGet, "/api/v1/channels/", nil)
	case "get":
		if err := requireArgs(pos, 1, "oictl channels get <channel-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodGet, channelPath(pos[0], ""), nil)
	case "create":
		body, err := requiredBody(flags, "a channel payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/channels/create", body)
	case "update":
		if err := requireArgs(pos, 1, "oictl channels update <channel-id> --file channel.json"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "a channel payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, channelPath(pos[0], "/update"), body)
	case "delete":
		if err := requireArgs(pos, 1, "oictl channels delete <channel-id> --yes"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodDelete, channelPath(pos[0], "/delete"), nil)
	case "members":
		return a.runChannelMembers(ctx, opts, flags, pos, format)
	case "messages":
		return a.runChannelMessages(ctx, opts, flags, pos, format)
	case "pins":
		return a.runChannelPins(ctx, opts, flags, pos, format)
	case "reactions":
		return a.runChannelReactions(ctx, opts, flags, pos, format)
	}
	fmt.Fprintf(a.err, "unknown channels command %q\n\n", action)
	a.printChannelsHelp(a.err)
	return 1
}

func (a *App) printChannelsHelp(w io.Writer) {
	fmt.Fprint(w, `Manage Open WebUI channels, members, messages, pins, and reactions.

Usage:
  oictl channels list
  oictl channels get <channel-id>
  oictl channels create (--data JSON | --file path | --file -)
  oictl channels update <channel-id> (--data JSON | --file path | --file -)
  oictl channels delete <channel-id> --yes
  oictl channels members <list|add|remove|active> <channel-id>
  oictl channels messages <list|post|get|data|thread|update|delete> <channel-id> [message-id]
  oictl channels pins <list|set|unset> <channel-id> [message-id]
  oictl channels reactions <add|remove> <channel-id> <message-id> (--name name | --data JSON | --file path)

Commands:
  list                List accessible channels
  get                 Get one channel by ID
  create              Create a channel from pass-through JSON
  update              Update a channel from pass-through JSON
  delete              Delete a channel; requires --yes
  members list        List channel members with --page, --query, --order-by, and --direction
  members add         Add members using pass-through JSON
  members remove      Remove members using pass-through JSON
  members active      Update caller active state using pass-through JSON
  messages list       List messages with --skip and --limit
  messages post       Post a message using pass-through JSON
  messages get        Get one message
  messages data       Get one message data object
  messages thread     List thread replies with --skip and --limit
  messages update     Update a message using pass-through JSON
  messages delete     Delete a message; requires --yes
  pins list           List pinned messages with --page
  pins set            Pin a message
  pins unset          Unpin a message
  reactions add       Add a reaction; --name creates {"name":...} unless JSON is supplied
  reactions remove    Remove a reaction; --name creates {"name":...} unless JSON is supplied
`)
}

func (a *App) runChannelMembers(ctx context.Context, opts globalOptions, flags commandFlags, args []string, format string) int {
	if len(args) < 2 {
		fmt.Fprintln(a.err, "usage: oictl channels members <list|add|remove|active> <channel-id>")
		return 1
	}
	action, channelID := args[0], args[1]
	if len(args) != 2 {
		fmt.Fprintln(a.err, "unexpected channels members argument")
		return 1
	}
	base := "/api/v1/channels/" + url.PathEscape(channelID)
	spec := requestSpec{authRequired: true, outPath: flags.values["out"]}
	switch action {
	case "list":
		spec.method = http.MethodGet
		spec.path = appendQuery(base+"/members", map[string]string{"page": flags.values["page"], "query": firstNonEmpty(flags.values["query"], flags.values["q"]), "order_by": flags.values["order-by"], "direction": flags.values["direction"]})
	case "add", "remove":
		body, err := requiredBody(flags, "a member payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		spec.method = http.MethodPost
		spec.path = base + "/update/members/" + action
		spec.body = body
	case "active":
		body, err := requiredBody(flags, "an active-state payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err = normalizeChannelMemberActiveBody(body)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		spec.method = http.MethodPost
		spec.path = base + "/members/active"
		spec.body = body
	default:
		fmt.Fprintf(a.err, "unknown channels members command %q\n", action)
		return 1
	}
	return a.executeStructured(ctx, opts, spec, format)
}

func (a *App) runChannelMessages(ctx context.Context, opts globalOptions, flags commandFlags, args []string, format string) int {
	if len(args) < 2 {
		fmt.Fprintln(a.err, "usage: oictl channels messages <list|post|get|data|thread|update|delete> <channel-id> [message-id]")
		return 1
	}
	action, channelID := args[0], args[1]
	base := "/api/v1/channels/" + url.PathEscape(channelID) + "/messages"
	spec := requestSpec{authRequired: true, outPath: flags.values["out"]}
	switch action {
	case "list":
		if len(args) != 2 {
			fmt.Fprintln(a.err, "usage: oictl channels messages list <channel-id>")
			return 1
		}
		spec.method = http.MethodGet
		spec.path = appendQuery(base, map[string]string{"skip": flags.values["skip"], "limit": flags.values["limit"]})
	case "post":
		if len(args) != 2 {
			fmt.Fprintln(a.err, "usage: oictl channels messages post <channel-id> --file message.json")
			return 1
		}
		body, err := requiredBody(flags, "a message payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		spec.method = http.MethodPost
		spec.path = base + "/post"
		spec.body = body
	case "get", "data", "thread":
		if len(args) != 3 {
			fmt.Fprintf(a.err, "usage: oictl channels messages %s <channel-id> <message-id>\n", action)
			return 1
		}
		spec.method = http.MethodGet
		spec.path = base + "/" + url.PathEscape(args[2])
		if action != "get" {
			spec.path += "/" + action
		}
		if action == "thread" {
			spec.path = appendQuery(spec.path, map[string]string{"skip": flags.values["skip"], "limit": flags.values["limit"]})
		}
	case "update":
		if len(args) != 3 {
			fmt.Fprintln(a.err, "usage: oictl channels messages update <channel-id> <message-id> --file message.json")
			return 1
		}
		body, err := requiredBody(flags, "a message payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		spec.method = http.MethodPost
		spec.path = base + "/" + url.PathEscape(args[2]) + "/update"
		spec.body = body
	case "delete":
		if len(args) != 3 {
			fmt.Fprintln(a.err, "usage: oictl channels messages delete <channel-id> <message-id> --yes")
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		spec.method = http.MethodDelete
		spec.path = base + "/" + url.PathEscape(args[2]) + "/delete"
	default:
		fmt.Fprintf(a.err, "unknown channels messages command %q\n", action)
		return 1
	}
	return a.executeStructured(ctx, opts, spec, format)
}

func (a *App) runChannelPins(ctx context.Context, opts globalOptions, flags commandFlags, args []string, format string) int {
	if len(args) < 2 {
		fmt.Fprintln(a.err, "usage: oictl channels pins <list|set|unset> <channel-id> [message-id]")
		return 1
	}
	action, channelID := args[0], args[1]
	base := "/api/v1/channels/" + url.PathEscape(channelID) + "/messages"
	spec := requestSpec{authRequired: true, outPath: flags.values["out"]}
	switch action {
	case "list":
		if len(args) != 2 {
			fmt.Fprintln(a.err, "usage: oictl channels pins list <channel-id>")
			return 1
		}
		spec.method = http.MethodGet
		spec.path = appendQuery(base+"/pinned", map[string]string{"page": flags.values["page"]})
	case "set", "unset":
		if len(args) != 3 {
			fmt.Fprintf(a.err, "usage: oictl channels pins %s <channel-id> <message-id>\n", action)
			return 1
		}
		spec.method = http.MethodPost
		spec.path = base + "/" + url.PathEscape(args[2]) + "/pin"
		spec.body = jsonBody(map[string]bool{"is_pinned": action == "set"})
	default:
		fmt.Fprintf(a.err, "unknown channels pins command %q\n", action)
		return 1
	}
	return a.executeStructured(ctx, opts, spec, format)
}

func (a *App) runChannelReactions(ctx context.Context, opts globalOptions, flags commandFlags, args []string, format string) int {
	if len(args) != 3 {
		fmt.Fprintln(a.err, "usage: oictl channels reactions <add|remove> <channel-id> <message-id> (--name name | --data JSON | --file path)")
		return 1
	}
	action := args[0]
	if action != "add" && action != "remove" {
		fmt.Fprintf(a.err, "unknown channels reactions command %q\n", action)
		return 1
	}
	body, err := requestBody(flags)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if body == nil {
		name := strings.TrimSpace(flags.values["name"])
		if name == "" {
			fmt.Fprintln(a.err, "a reaction payload or --name is required")
			return 1
		}
		body = jsonBody(map[string]string{"name": name})
	}
	path := "/api/v1/channels/" + url.PathEscape(args[1]) + "/messages/" + url.PathEscape(args[2]) + "/reactions/" + action
	return a.executeStructured(ctx, opts, requestSpec{method: http.MethodPost, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
}

func (a *App) runWebhooks(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		a.printWebhooksHelp(a.out)
		return 0
	}
	subdomain := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	format := outputFormat(opts, flags, "table")
	switch subdomain {
	case "channels":
		return a.runWebhookChannels(ctx, opts, flags, pos, format)
	case "events":
		return a.runWebhookEvents(ctx, opts, flags, pos, format)
	}
	fmt.Fprintf(a.err, "unknown webhooks command %q\n\n", subdomain)
	a.printWebhooksHelp(a.err)
	return 1
}

func (a *App) printWebhooksHelp(w io.Writer) {
	fmt.Fprint(w, `Manage Open WebUI channel incoming webhooks and global event webhooks.

Usage:
  oictl webhooks channels <list|get|create|ensure|update|delete|url> <channel-id> [webhook-id]
  oictl webhooks events <catalog|list|get|create|update|enable|disable|delete> [webhook-id]

Channel Commands:
  channels list <channel-id>                              List incoming webhooks with token values redacted
  channels get <channel-id> <webhook-id>                  Get incoming webhook metadata with token values redacted
  channels create <channel-id> --name name [--profile-image-url URL]
  channels ensure <channel-id> --name name [--profile-image-url URL] [--show-url | --out path | --verify-url [--expected-url URL | --expected-url-env NAME]]
  channels update <channel-id> <webhook-id> [--name name] [--profile-image-url URL]
  channels delete <channel-id> <webhook-id> --yes          Delete an incoming webhook
  channels url <channel-id> <webhook-id> (--show-url | --out path)

Event Commands:
  events catalog                                           List event catalog entries
  events list                                              List event webhooks with destination URLs redacted
  events get <webhook-id>                                  Get event webhook metadata with destination URLs redacted
  events create (--data JSON | --file path | --file -)      Create an event webhook
  events update <webhook-id> (--data JSON | --file path | --file -)
  events enable <webhook-id>                               Enable an event webhook while preserving other fields
  events disable <webhook-id>                              Disable an event webhook while preserving other fields
  events delete <webhook-id> --yes                         Delete an event webhook

Channel create requires a non-empty name, supplied by --name or JSON. Flag-based channel updates fetch current metadata and preserve omitted name/image fields; explicit JSON is a replacement form.
JSON output is the complete server response and may include webhook tokens or destination URLs. Use default table output for redacted human-readable views. Use channels url or channels ensure URL options only when you intend to reveal, write, or verify the full incoming webhook URL.
`)
}

func (a *App) runWebhookChannels(ctx context.Context, opts globalOptions, flags commandFlags, args []string, format string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.err, "usage: oictl webhooks channels <list|get|create|ensure|update|delete|url> <channel-id> [webhook-id]")
		return 1
	}
	action := args[0]
	channelPath := func(channelID string) string {
		return "/api/v1/channels/" + url.PathEscape(channelID) + "/webhooks"
	}
	webhookPath := func(channelID, webhookID, suffix string) string {
		return channelPath(channelID) + "/" + url.PathEscape(webhookID) + suffix
	}
	simple := func(method, path string, body io.Reader) int {
		return a.executeWebhookStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
	}
	switch action {
	case "list":
		if err := requireArgs(args[1:], 1, "oictl webhooks channels list <channel-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodGet, channelPath(args[1]), nil)
	case "get":
		if err := requireArgs(args[1:], 2, "oictl webhooks channels get <channel-id> <webhook-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.getListedWebhook(ctx, opts, flags, channelPath(args[1]), args[2], format)
	case "create":
		if err := requireArgs(args[1:], 1, "oictl webhooks channels create <channel-id> --name name [--profile-image-url URL]"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := channelWebhookMutationBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, channelPath(args[1])+"/create", body)
	case "ensure":
		if err := requireArgs(args[1:], 1, "oictl webhooks channels ensure <channel-id> --name name [--show-url | --out path | --verify-url [--expected-url URL | --expected-url-env NAME]]"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if strings.TrimSpace(flags.values["name"]) == "" {
			fmt.Fprintln(a.err, "--name is required")
			return 1
		}
		return a.ensureChannelWebhook(ctx, opts, flags, channelPath(args[1]), args[1], format)
	case "update":
		if err := requireArgs(args[1:], 2, "oictl webhooks channels update <channel-id> <webhook-id> [--name name] [--profile-image-url URL]"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := a.channelWebhookUpdateBody(ctx, opts, flags, channelPath(args[1]), args[2])
		if err != nil {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
			return 1
		}
		return simple(http.MethodPost, webhookPath(args[1], args[2], "/update"), body)
	case "delete":
		if err := requireArgs(args[1:], 2, "oictl webhooks channels delete <channel-id> <webhook-id> --yes"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodDelete, webhookPath(args[1], args[2], "/delete"), nil)
	case "url":
		if err := requireArgs(args[1:], 2, "oictl webhooks channels url <channel-id> <webhook-id> (--show-url | --out path)"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.revealChannelWebhookURL(ctx, opts, flags, channelPath(args[1]), args[2])
	}
	fmt.Fprintf(a.err, "unknown webhooks channels command %q\n", action)
	return 1
}

func channelWebhookMutationBody(flags commandFlags) (io.Reader, error) {
	body, err := requestBody(flags)
	if err != nil {
		return nil, err
	}
	if body == nil {
		payload := map[string]string{}
		if name, ok := flags.values["name"]; ok {
			payload["name"] = strings.TrimSpace(name)
		}
		if image, ok := flags.values["profile-image-url"]; ok {
			payload["profile_image_url"] = strings.TrimSpace(image)
		}
		body = jsonBody(payload)
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, errors.New("channel webhook payload must be a JSON object with a name")
	}
	if name, ok := payload["name"].(string); !ok || strings.TrimSpace(name) == "" {
		return nil, errors.New("a non-empty webhook name is required; supply --name or a complete JSON payload")
	}
	return bytes.NewReader(data), nil
}

func (a *App) channelWebhookUpdateBody(ctx context.Context, opts globalOptions, flags commandFlags, listPath, webhookID string) (io.Reader, error) {
	body, err := requestBody(flags)
	if err != nil || body != nil {
		// Explicit JSON is a replacement form, never a convenience-field merge.
		return body, err
	}
	name, hasName := flags.values["name"]
	image, hasImage := flags.values["profile-image-url"]
	if !hasName && !hasImage {
		return nil, errors.New("a webhook payload, --name, or --profile-image-url is required")
	}
	if hasName && strings.TrimSpace(name) == "" {
		return nil, errors.New("webhook name must be non-empty")
	}
	current, err := a.fetchListedWebhook(ctx, opts, listPath, webhookID)
	if err != nil {
		return nil, err
	}
	var existing map[string]any
	if err := json.Unmarshal(current, &existing); err != nil {
		return nil, fmt.Errorf("decode channel webhook: %w", err)
	}
	payload := map[string]any{"name": existing["name"], "profile_image_url": existing["profile_image_url"]}
	if hasName {
		payload["name"] = strings.TrimSpace(name)
	}
	if hasImage {
		payload["profile_image_url"] = strings.TrimSpace(image)
	}
	if currentName, ok := payload["name"].(string); !ok || strings.TrimSpace(currentName) == "" {
		return nil, errors.New("current webhook has no name; supply --name")
	}
	return jsonBody(payload), nil
}

func (a *App) ensureChannelWebhook(ctx context.Context, opts globalOptions, flags commandFlags, listPath string, channelID string, format string) int {
	outPath := firstNonEmpty(flags.values["out"], opts.outPath)
	expectedURL, hasExpectedURL := flags.values["expected-url"]
	expectedURLEnvName, hasExpectedURLEnv := flags.values["expected-url-env"]
	if hasExpectedURL && hasExpectedURLEnv {
		fmt.Fprintln(a.err, "use only one of --expected-url or --expected-url-env")
		return 1
	}
	if hasExpectedURLEnv {
		expectedURLEnvName = strings.TrimSpace(expectedURLEnvName)
		if expectedURLEnvName == "" {
			fmt.Fprintln(a.err, "--expected-url-env requires a non-empty environment variable name")
			return 1
		}
		expectedURL = a.getenv(expectedURLEnvName)
		if strings.TrimSpace(expectedURL) == "" {
			fmt.Fprintf(a.err, "expected URL environment variable %s is required and must be non-empty\n", expectedURLEnvName)
			return 1
		}
	}
	verifyURL := flags.bools["verify-url"] || hasExpectedURL || hasExpectedURLEnv
	urlModes := 0
	if flags.bools["show-url"] {
		urlModes++
	}
	if outPath != "" {
		urlModes++
	}
	if verifyURL {
		urlModes++
	}
	if urlModes > 1 {
		fmt.Fprintln(a.err, "use only one of --show-url, --out, or --verify-url/expected URL comparison")
		return 1
	}

	body, err := a.ensureChannelWebhookByName(ctx, opts, flags, listPath, channelID)
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		}
		return 1
	}

	if urlModes == 0 {
		return a.writeWebhookStructured(opts, "", body, format)
	}
	webhookURL, webhookID, err := a.channelWebhookURLFromBody(opts, body, "")
	if err != nil {
		fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		return 1
	}
	if verifyURL {
		if hasExpectedURL || hasExpectedURLEnv {
			if webhookURL != expectedURL {
				fmt.Fprintln(a.err, "ensured webhook URL did not match expected URL")
				return 1
			}
			return a.writeLocalOutput([]byte(fmt.Sprintf("Webhook URL matched expected URL for channel webhook %s\n", webhookID)), "")
		}
		return a.writeLocalOutput([]byte(fmt.Sprintf("Webhook URL verified for channel webhook %s\n", webhookID)), "")
	}
	if outPath != "" {
		if err := writeSecureFile(outPath, []byte(webhookURL+"\n")); err != nil {
			fmt.Fprintf(a.err, "write output: %v\n", err)
			return 1
		}
		if flags.values["output"] != "" || opts.output != "" {
			return a.writeWebhookStructured(globalOptions{}, "", webhookMetadataJSON(body), format)
		}
		return 0
	}
	return a.writeLocalOutput([]byte(webhookURL+"\n"), "")
}

func (a *App) ensureChannelWebhookByName(ctx context.Context, opts globalOptions, flags commandFlags, listPath string, channelID string) ([]byte, error) {
	name := strings.TrimSpace(flags.values["name"])
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: listPath, authRequired: true})
	if err != nil {
		return nil, err
	}
	var webhooks []map[string]any
	if err := json.Unmarshal(body, &webhooks); err != nil {
		return nil, fmt.Errorf("decode webhook list response: %w", err)
	}
	matches := make([]map[string]any, 0, 1)
	for _, webhook := range webhooks {
		if strings.TrimSpace(webhookStringField(webhook, "name")) == name {
			matches = append(matches, webhook)
		}
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("channel webhook name %q is ambiguous in channel %q (%d matches)", name, channelID, len(matches))
	}
	if len(matches) == 1 {
		return json.Marshal(matches[0])
	}
	createBody, err := channelWebhookMutationBody(flags)
	if err != nil {
		return nil, err
	}
	return a.doRequest(ctx, opts, requestSpec{method: http.MethodPost, path: listPath + "/create", body: createBody, authRequired: true})
}

func (a *App) revealChannelWebhookURL(ctx context.Context, opts globalOptions, flags commandFlags, listPath string, webhookID string) int {
	outPath := firstNonEmpty(flags.values["out"], opts.outPath)
	if flags.bools["show-url"] && outPath != "" {
		fmt.Fprintln(a.err, "use only one of --show-url or --out")
		return 1
	}
	if !flags.bools["show-url"] && outPath == "" {
		fmt.Fprintln(a.err, "revealing a channel webhook URL requires --show-url or --out")
		return 1
	}
	body, err := a.fetchListedWebhook(ctx, opts, listPath, webhookID)
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		}
		return 1
	}
	webhookURL, _, err := a.channelWebhookURLFromBody(opts, body, webhookID)
	if err != nil {
		fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		return 1
	}
	if outPath != "" {
		if err := writeSecureFile(outPath, []byte(webhookURL+"\n")); err != nil {
			fmt.Fprintf(a.err, "write output: %v\n", err)
			return 1
		}
		return 0
	}
	return a.writeLocalOutput([]byte(webhookURL+"\n"), "")
}

func (a *App) channelWebhookURLFromBody(opts globalOptions, body []byte, fallbackWebhookID string) (string, string, error) {
	var webhook map[string]any
	if err := json.Unmarshal(body, &webhook); err != nil {
		return "", "", fmt.Errorf("decode webhook response: %w", err)
	}
	webhookID := firstNonEmpty(webhookStringField(webhook, "id"), fallbackWebhookID)
	token := firstNonEmpty(webhookStringField(webhook, "token"), webhookStringField(webhook, "webhook_token"))
	if webhookID == "" || token == "" {
		return "", "", errors.New("webhook response did not include an id and token")
	}
	target, err := a.resolveTarget(opts, true)
	if err != nil {
		return "", "", err
	}
	webhookURL, err := joinURL(target.baseURL, "/api/v1/channels/webhooks/"+url.PathEscape(webhookID)+"/"+url.PathEscape(token))
	if err != nil {
		return "", "", err
	}
	return webhookURL, webhookID, nil
}

func webhookStringField(object map[string]any, key string) string {
	if value, ok := object[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func (a *App) runWebhookEvents(ctx context.Context, opts globalOptions, flags commandFlags, args []string, format string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.err, "usage: oictl webhooks events <catalog|list|get|create|update|enable|disable|delete> [webhook-id]")
		return 1
	}
	action := args[0]
	webhookPath := func(id string) string {
		return "/api/events/webhooks/" + url.PathEscape(id)
	}
	simple := func(method, path string, body io.Reader) int {
		return a.executeWebhookStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
	}
	switch action {
	case "catalog":
		if err := requireArgs(args[1:], 0, "oictl webhooks events catalog"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodGet, "/api/events", nil)
	case "list":
		if err := requireArgs(args[1:], 0, "oictl webhooks events list"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodGet, "/api/events/webhooks", nil)
	case "get":
		if err := requireArgs(args[1:], 1, "oictl webhooks events get <webhook-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.getListedWebhook(ctx, opts, flags, "/api/events/webhooks", args[1], format)
	case "create":
		if err := requireArgs(args[1:], 0, "oictl webhooks events create (--data JSON | --file path | --file -)"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "an event webhook payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/events/webhooks", body)
	case "update":
		if err := requireArgs(args[1:], 1, "oictl webhooks events update <webhook-id> (--data JSON | --file path | --file -)"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "an event webhook payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPut, webhookPath(args[1]), body)
	case "enable", "disable":
		if err := requireArgs(args[1:], 1, "oictl webhooks events "+action+" <webhook-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.setEventWebhookEnabled(ctx, opts, flags, args[1], action == "enable", format)
	case "delete":
		if err := requireArgs(args[1:], 1, "oictl webhooks events delete <webhook-id> --yes"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodDelete, webhookPath(args[1]), nil)
	}
	fmt.Fprintf(a.err, "unknown webhooks events command %q\n", action)
	return 1
}

func (a *App) setEventWebhookEnabled(ctx context.Context, opts globalOptions, flags commandFlags, webhookID string, enabled bool, format string) int {
	path := "/api/events/webhooks/" + url.PathEscape(webhookID)
	body, err := a.fetchListedWebhook(ctx, opts, "/api/events/webhooks", webhookID)
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		}
		return 1
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		fmt.Fprintf(a.err, "decode event webhook response: %v\n", err)
		return 1
	}
	payload["enabled"] = enabled
	if _, ok := payload["id"]; !ok {
		payload["id"] = webhookID
	}
	return a.executeWebhookStructured(ctx, opts, requestSpec{method: http.MethodPut, path: path, body: jsonBody(payload), authRequired: true, outPath: flags.values["out"]}, format)
}

func (a *App) executeWebhookStructured(ctx context.Context, opts globalOptions, spec requestSpec, format string) int {
	body, err := a.doRequest(ctx, opts, spec)
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token, spec.bearerToken))
		}
		return 1
	}
	return a.writeWebhookStructured(opts, firstNonEmpty(spec.outPath, opts.outPath), body, format)
}

func (a *App) getListedWebhook(ctx context.Context, opts globalOptions, flags commandFlags, listPath string, webhookID string, format string) int {
	body, err := a.fetchListedWebhook(ctx, opts, listPath, webhookID)
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		}
		return 1
	}
	return a.writeWebhookStructured(opts, firstNonEmpty(flags.values["out"], opts.outPath), body, format)
}

func (a *App) fetchListedWebhook(ctx context.Context, opts globalOptions, listPath string, webhookID string) ([]byte, error) {
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: listPath, authRequired: true})
	if err != nil {
		return nil, err
	}
	var webhooks []map[string]any
	if err := json.Unmarshal(body, &webhooks); err != nil {
		return nil, fmt.Errorf("decode webhook list response: %w", err)
	}
	for _, webhook := range webhooks {
		if webhookStringField(webhook, "id") == webhookID {
			return json.Marshal(webhook)
		}
	}
	return nil, fmt.Errorf("webhook %q not found", webhookID)
}

func (a *App) writeWebhookStructured(opts globalOptions, outPath string, body []byte, format string) int {
	if outPath != "" {
		if err := writeSecureFile(outPath, body); err != nil {
			fmt.Fprintf(a.err, "write output: %v\n", err)
			return 1
		}
		return 0
	}
	if format == "table" {
		redacted := redactWebhookJSON(body)
		if rendered, ok := renderTable(redacted); ok {
			return a.writeLocalOutput(rendered, "")
		}
	}
	if err := writeAll(a.out, body); err != nil {
		fmt.Fprintf(a.err, "write output: %v\n", err)
		return 1
	}
	return 0
}

func redactWebhookJSON(body []byte) []byte {
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return body
	}
	redacted := redactWebhookValue("", decoded)
	encoded, err := json.Marshal(redacted)
	if err != nil {
		return body
	}
	return encoded
}

func webhookMetadataJSON(body []byte) []byte {
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return body
	}
	metadata := webhookMetadataValue("", decoded)
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return body
	}
	return encoded
}

func webhookMetadataValue(key string, value any) any {
	switch typed := value.(type) {
	case map[string]any:
		metadata := make(map[string]any, len(typed))
		for childKey, childValue := range typed {
			if isWebhookSecretKey(childKey) || isWebhookURLKey(childKey) {
				continue
			}
			metadata[childKey] = webhookMetadataValue(childKey, childValue)
		}
		return metadata
	case []any:
		metadata := make([]any, 0, len(typed))
		for _, item := range typed {
			metadata = append(metadata, webhookMetadataValue("", item))
		}
		return metadata
	case string:
		if isWebhookSecretKey(key) || isWebhookURLKey(key) || strings.Contains(typed, "/api/v1/channels/webhooks/") {
			return ""
		}
	}
	return value
}

func redactWebhookValue(key string, value any) any {
	switch typed := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for childKey, childValue := range typed {
			redacted[childKey] = redactWebhookValue(childKey, childValue)
		}
		return redacted
	case []any:
		redacted := make([]any, 0, len(typed))
		for _, item := range typed {
			redacted = append(redacted, redactWebhookValue("", item))
		}
		return redacted
	case string:
		if isWebhookSecretKey(key) || isWebhookURLKey(key) || strings.Contains(typed, "/api/v1/channels/webhooks/") {
			return "<redacted>"
		}
	}
	return value
}

func isWebhookSecretKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	return normalized == "token" || normalized == "webhook_token" || normalized == "webhook_token_hash"
}

func isWebhookURLKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	return normalized == "url" || normalized == "webhook_url" || normalized == "destination_url" || normalized == "callback_url" || normalized == "target_url"
}

func (a *App) runGroups(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Manage native Open WebUI groups.

Usage:
  oictl groups <list|create|get|info|export|update|delete|preview|users>
  oictl groups users <list|add|remove> <group-id> [user-id...]

Commands:
  list          List native groups; supports --share true|false
  create        Create a native group from --data, --file, or --file - JSON
  get           Get one native group by ID
  info          Get native group info
  export        Export a native group to stdout or --out
  update        Update a native group from --data, --file, or --file - JSON
  delete        Delete a native group; requires --yes
  preview       Preview access controlled by a native group
  users list    List users in a native group
  users add     Add users to a native group
  users remove  Remove users from a native group

Native groups use normal Open WebUI API authentication. SCIM provisioning groups remain under oictl scim groups.
`)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	format := outputFormat(opts, flags, "table")
	simple := func(method, path string, body io.Reader) int {
		return a.executeStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
	}
	groupPath := func(id string, suffix string) string {
		return "/api/v1/groups/id/" + url.PathEscape(id) + suffix
	}
	switch action {
	case "list":
		return simple(http.MethodGet, appendQuery("/api/v1/groups/", map[string]string{"share": flags.values["share"]}), nil)
	case "create":
		body, err := requiredBody(flags, "a group payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/groups/create", body)
	case "get", "info", "preview":
		if err := requireArgs(pos, 1, "oictl groups "+action+" <group-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		suffix := ""
		if action != "get" {
			suffix = "/" + action
		}
		return simple(http.MethodGet, groupPath(pos[0], suffix), nil)
	case "export":
		if err := requireArgs(pos, 1, "oictl groups export <group-id> [--out path]"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.executeRaw(ctx, opts, requestSpec{method: http.MethodGet, path: groupPath(pos[0], "/export"), authRequired: true, outPath: flags.values["out"]})
	case "update":
		if err := requireArgs(pos, 1, "oictl groups update <group-id> --file group.json"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "a group payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, groupPath(pos[0], "/update"), body)
	case "delete":
		if err := requireArgs(pos, 1, "oictl groups delete <group-id> --yes"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodDelete, groupPath(pos[0], "/delete"), nil)
	case "users":
		return a.runGroupUsers(ctx, opts, flags, pos, format)
	}
	fmt.Fprintf(a.err, "unknown groups command %q\n", action)
	return 1
}

func (a *App) runUsers(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Manage Open WebUI users and user settings.

Usage:
  oictl users list [--query text] [--page n] [--order-by field] [--direction asc|desc]
  oictl users search --query text [--page n] [--order-by field] [--direction asc|desc]
  oictl users get <user-id>
  oictl users create (--data JSON | --file path | --file -)
  oictl users update <user-id> (--data JSON | --file path | --file -)
  oictl users delete <user-id> (--yes | --confirm)
  oictl users settings get
  oictl users settings update (--data JSON | --file path | --file -) [--allow-sensitive-ui-keys]
  oictl users ui-settings patch <user-id> (--data JSON | --file path | --file -) [--allow-sensitive-ui-keys]
  oictl users ui-settings bulk-patch [user-id...] [--user-id id ...] [--users-file path] (--data JSON | --file path | --file -) [--dry-run] [--allow-sensitive-ui-keys]
  oictl users ui-settings bulk-patch (--all | --query text) (--data JSON | --file path | --file -) [--dry-run | --yes | --confirm] [--allow-sensitive-ui-keys]

Commands:
  list                    List users with query, pagination, and ordering filters
  search                  Search users with query, pagination, and ordering filters
  get                     Get one user by ID
  create                  Create a user from pass-through JSON
  update                  Update a user from pass-through JSON
  delete                  Delete a user; requires --yes or --confirm
  settings get       Fetch the authenticated user's settings
  settings update    Update the authenticated user's settings from JSON
  ui-settings patch  Patch another user's settings.ui map as an admin
  ui-settings bulk-patch  Patch settings.ui for multiple users sequentially

Directory-derived non-dry-run bulk-patch operations require --yes or --confirm; use --dry-run to audit discovered targets first.
Cross-user ui-settings commands require --allow-ui-settings-extension, including previews: the route is a deployment extension absent from the original API. Current-user settings use a different route and are never substituted.
Sensitive UI keys such as toolServers require --allow-sensitive-ui-keys.
`)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	raw := func(method, path string, body io.Reader) int {
		return a.executeRaw(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]})
	}
	structured := func(method, path string, body io.Reader) int {
		return a.executeStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, outputFormat(opts, flags, "json"))
	}
	switch action {
	case "list":
		if len(pos) != 0 {
			fmt.Fprintln(a.err, "usage: oictl users list [--query text] [--page n] [--order-by field] [--direction asc|desc]")
			return 1
		}
		return structured(http.MethodGet, appendQuery("/api/v1/users/", map[string]string{"query": firstNonEmpty(flags.values["query"], flags.values["q"]), "page": flags.values["page"], "order_by": flags.values["order-by"], "direction": flags.values["direction"]}), nil)
	case "search":
		if len(pos) != 0 {
			fmt.Fprintln(a.err, "usage: oictl users search [--query text] [--page n] [--order-by field] [--direction asc|desc]")
			return 1
		}
		return structured(http.MethodGet, appendQuery("/api/v1/users/search", map[string]string{"query": firstNonEmpty(flags.values["query"], flags.values["q"]), "page": flags.values["page"], "order_by": flags.values["order-by"], "direction": flags.values["direction"]}), nil)
	case "get":
		if err := requireArgs(pos, 1, "oictl users get <user-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return structured(http.MethodGet, "/api/v1/users/"+url.PathEscape(pos[0]), nil)
	case "create":
		if err := requireArgs(pos, 0, "oictl users create (--data JSON | --file path | --file -)"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "a user payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return structured(http.MethodPost, "/api/v1/auths/add", body)
	case "update":
		if err := requireArgs(pos, 1, "oictl users update <user-id> (--data JSON | --file path | --file -)"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "a user payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return structured(http.MethodPost, "/api/v1/users/"+url.PathEscape(pos[0])+"/update", body)
	case "delete":
		if err := requireArgs(pos, 1, "oictl users delete <user-id> (--yes | --confirm)"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return structured(http.MethodDelete, "/api/v1/users/"+url.PathEscape(pos[0]), nil)
	case "settings":
		if len(pos) != 1 {
			fmt.Fprintln(a.err, "usage: oictl users settings <get|update>")
			return 1
		}
		switch pos[0] {
		case "get":
			return raw(http.MethodGet, "/api/v1/users/user/settings", nil)
		case "update":
			body, err := userSettingsMutationBody(flags, "a settings payload is required", true)
			if err != nil {
				fmt.Fprintln(a.err, err)
				return 1
			}
			return raw(http.MethodPost, "/api/v1/users/user/settings/update", body)
		default:
			fmt.Fprintf(a.err, "unknown users settings command %q\n", pos[0])
			return 1
		}
	case "ui-settings":
		if !flags.bools["allow-ui-settings-extension"] {
			fmt.Fprintln(a.err, "cross-user UI settings require a deployment extension absent from the original API; acknowledge support with --allow-ui-settings-extension (also required for previews)")
			return 1
		}
		if len(pos) == 0 {
			fmt.Fprintln(a.err, "usage: oictl users ui-settings <patch|bulk-patch>")
			return 1
		}
		switch pos[0] {
		case "patch":
			if len(pos) != 2 {
				fmt.Fprintln(a.err, "usage: oictl users ui-settings patch <user-id>")
				return 1
			}
			body, err := userSettingsMutationBody(flags, "a UI settings payload is required", false)
			if err != nil {
				fmt.Fprintln(a.err, err)
				return 1
			}
			return raw(http.MethodPatch, "/api/v1/users/"+url.PathEscape(pos[1])+"/settings/ui", body)
		case "bulk-patch":
			return a.runUsersUISettingsBulkPatch(ctx, opts, flags, pos[1:])
		default:
			fmt.Fprintf(a.err, "unknown users ui-settings command %q\n", pos[0])
			return 1
		}
	}
	fmt.Fprintf(a.err, "unknown users command %q\n", action)
	return 1
}

type bulkUISettingsResult struct {
	UserID       string `json:"user_id"`
	Action       string `json:"action"`
	Status       string `json:"status"`
	TargetSource string `json:"target_source,omitempty"`
	Query        string `json:"query,omitempty"`
	Error        string `json:"error,omitempty"`
}

type bulkUISettingsTargetSource struct {
	mode  string
	query string
}

func (a *App) runUsersUISettingsBulkPatch(ctx context.Context, opts globalOptions, flags commandFlags, targets []string) int {
	source, err := bulkUISettingsTargetSourceFor(targets, flags)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if source.mode != "" && !flags.bools["dry-run"] && !flags.bools["yes"] && !flags.bools["confirm"] {
		fmt.Fprintln(a.err, "directory-derived bulk UI settings patches require --yes or --dry-run")
		return 1
	}
	body, err := userSettingsMutationBody(flags, "a UI settings payload is required", false)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	payload, err := io.ReadAll(body)
	if err != nil {
		fmt.Fprintf(a.err, "read UI settings payload: %v\n", err)
		return 1
	}
	userIDs, err := a.collectBulkUISettingsUserIDs(ctx, opts, targets, flags, source)
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		}
		return 1
	}
	results := make([]bulkUISettingsResult, 0, len(userIDs))
	if flags.bools["dry-run"] {
		for _, userID := range userIDs {
			results = append(results, bulkUISettingsResult{UserID: userID, Action: "patch_ui_settings", Status: "planned", TargetSource: source.mode, Query: source.query})
		}
		return a.writeBulkUISettingsResults(opts, flags, results)
	}
	failed := false
	for _, userID := range userIDs {
		result := bulkUISettingsResult{UserID: userID, Action: "patch_ui_settings", Status: "success", TargetSource: source.mode, Query: source.query}
		_, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodPatch, path: "/api/v1/users/" + url.PathEscape(userID) + "/settings/ui", body: bytes.NewReader(payload), authRequired: true})
		if err != nil {
			failed = true
			result.Status = "failed"
			var statusErr httpStatusError
			if errors.As(err, &statusErr) {
				result.Error = strings.TrimSpace(statusErr.status + " " + statusErr.body)
			} else {
				result.Error = redactSecrets(err.Error(), opts.token)
			}
		}
		results = append(results, result)
	}
	if code := a.writeBulkUISettingsResults(opts, flags, results); code != 0 {
		return code
	}
	if failed {
		return 1
	}
	return 0
}

func bulkUISettingsTargetSourceFor(positionals []string, flags commandFlags) (bulkUISettingsTargetSource, error) {
	query, hasQuery := flags.values["query"]
	query = strings.TrimSpace(query)
	all := flags.bools["all"]
	if all && hasQuery {
		return bulkUISettingsTargetSource{}, errors.New("--all and --query cannot be combined")
	}
	if hasQuery && query == "" {
		return bulkUISettingsTargetSource{}, errors.New("--query requires non-empty text")
	}
	_, hasUsersFile := flags.values["users-file"]
	hasExplicitTargets := len(positionals) > 0 || len(flags.multiValues["user-id"]) > 0 || hasUsersFile
	if (all || hasQuery) && hasExplicitTargets {
		return bulkUISettingsTargetSource{}, errors.New("directory discovery cannot be combined with positional targets, --user-id, or --users-file")
	}
	if all {
		return bulkUISettingsTargetSource{mode: "all"}, nil
	}
	if hasQuery {
		return bulkUISettingsTargetSource{mode: "query", query: query}, nil
	}
	return bulkUISettingsTargetSource{}, nil
}

func (a *App) collectBulkUISettingsUserIDs(ctx context.Context, opts globalOptions, positionals []string, flags commandFlags, source bulkUISettingsTargetSource) ([]string, error) {
	if source.mode != "" {
		return a.discoverBulkUISettingsUserIDs(ctx, opts, source)
	}
	return collectBulkUserIDs(positionals, flags)
}

func collectBulkUserIDs(positionals []string, flags commandFlags) ([]string, error) {
	seen := map[string]bool{}
	userIDs := []string{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			userIDs = append(userIDs, value)
		}
	}
	for _, userID := range positionals {
		add(userID)
	}
	for _, userID := range flags.multiValues["user-id"] {
		add(userID)
	}
	if path := strings.TrimSpace(flags.values["users-file"]); path != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read users file: %w", err)
		}
		for _, line := range strings.Split(string(body), "\n") {
			add(line)
		}
	}
	if len(userIDs) == 0 {
		return nil, errors.New("at least one target user or discovery mode is required")
	}
	return userIDs, nil
}

func (a *App) discoverBulkUISettingsUserIDs(ctx context.Context, opts globalOptions, source bulkUISettingsTargetSource) ([]string, error) {
	type directoryUser struct {
		ID string `json:"id"`
	}
	type directoryResponse struct {
		Users []directoryUser `json:"users"`
		Total int             `json:"total"`
	}

	seen := map[string]bool{}
	userIDs := []string{}
	collected := 0
	for page := 1; ; page++ {
		path := appendQuery("/api/v1/users/", map[string]string{"page": fmt.Sprint(page), "query": source.query})
		body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: path, authRequired: true})
		if err != nil {
			return nil, err
		}
		var response directoryResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("decode users directory response: %w", err)
		}
		if len(response.Users) == 0 {
			break
		}
		before := len(userIDs)
		for _, user := range response.Users {
			collected++
			userID := strings.TrimSpace(user.ID)
			if userID != "" && !seen[userID] {
				seen[userID] = true
				userIDs = append(userIDs, userID)
			}
		}
		if len(userIDs) == before {
			break
		}
		if response.Total > 0 && collected >= response.Total {
			break
		}
	}
	if len(userIDs) == 0 {
		return nil, errors.New("no target user IDs were discovered")
	}
	return userIDs, nil
}

func (a *App) writeBulkUISettingsResults(opts globalOptions, flags commandFlags, results []bulkUISettingsResult) int {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		fmt.Fprintf(a.err, "encode bulk UI settings results: %v\n", err)
		return 1
	}
	body := out.Bytes()
	outPath := firstNonEmpty(flags.values["out"], opts.outPath)
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

func userSettingsMutationBody(flags commandFlags, requiredMessage string, nestedUI bool) (io.Reader, error) {
	body, err := requiredBody(flags, requiredMessage)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New(requiredMessage)
	}
	keys, err := sensitiveUISettingKeys(data, nestedUI)
	if err != nil {
		return nil, err
	}
	if len(keys) > 0 && !flags.bools["allow-sensitive-ui-keys"] {
		return nil, fmt.Errorf("sensitive UI setting key %s requires --allow-sensitive-ui-keys", strings.Join(keys, ", "))
	}
	return bytes.NewReader(data), nil
}

func sensitiveUISettingKeys(data []byte, nestedUI bool) ([]string, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, errors.New("settings payload must be a JSON object")
	}
	if nestedUI {
		raw, ok := object["ui"]
		if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil, nil
		}
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, nil
		}
	}
	sensitive := map[string]bool{"toolServers": true}
	keys := make([]string, 0, len(sensitive))
	for key := range sensitive {
		if _, ok := object[key]; ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (a *App) runGroupUsers(ctx context.Context, opts globalOptions, flags commandFlags, args []string, format string) int {
	if len(args) < 2 {
		fmt.Fprintln(a.err, "usage: oictl groups users <list|add|remove> <group-id> [user-id...]")
		return 1
	}
	action := args[0]
	groupID := args[1]
	path := "/api/v1/groups/id/" + url.PathEscape(groupID) + "/users"
	if action == "add" || action == "remove" {
		path += "/" + action
	}
	spec := requestSpec{method: http.MethodPost, path: path, authRequired: true, outPath: flags.values["out"]}
	switch action {
	case "list":
		if len(args) != 2 {
			fmt.Fprintln(a.err, "usage: oictl groups users list <group-id>")
			return 1
		}
	case "add", "remove":
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		userIDs := args[2:]
		if body == nil {
			if len(userIDs) == 0 {
				fmt.Fprintln(a.err, "at least one user id is required")
				return 1
			}
			body = jsonBody(map[string][]string{"user_ids": userIDs})
		}
		spec.body = body
	default:
		fmt.Fprintf(a.err, "unknown groups users command %q\n", action)
		return 1
	}
	return a.executeStructured(ctx, opts, spec, format)
}

func (a *App) runFunctions(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Manage Open WebUI functions and filters.

Usage:
  oictl functions <list|get|create|update|delete|export|load-url|sync|toggle|toggle-global|valves>

Commands:
  list          List functions; use --type filter to show filters only
  get           Get one function by ID as JSON, preserving source and metadata
  create        Create a function from --data, --file, or --file - JSON
  update        Update a function from --data, --file, or --file - JSON
  delete        Delete a function; requires --yes
  export        Export functions to stdout or --out; --include-valves requests valve values
  load-url      Ask Open WebUI to fetch Python source from a trusted URL
  sync          Reconcile functions from JSON; requires --yes and can remove omitted remote functions
  toggle        Toggle a function active state
  toggle-global Toggle a function global state
  valves        Get, inspect, and update global or user-scoped valves

Filters are managed as function records whose server-returned type is "filter"; no separate filters command is used.
Function source is arbitrary Python loaded by Open WebUI. Only create, update, load, or sync code from trusted sources.
Sync requires an explicit functions array or a bare-array shorthand; an empty array with --yes intentionally removes the remote inventory.

Examples:
  oictl functions list --type filter
  oictl functions create --file function.json
  oictl functions sync --file functions.json --yes
  oictl functions valves get my_function
`)
		return 0
	}
	action := args[0]
	if action == "valves" {
		return a.runValves(ctx, opts, "functions", "function-id", args[1:])
	}
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	jsonFormat := outputFormat(opts, flags, "json")
	simple := func(method, path string, body io.Reader) int {
		return a.executeStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, jsonFormat)
	}
	functionPath := func(id string, suffix string) string {
		return "/api/v1/functions/id/" + url.PathEscape(id) + suffix
	}
	switch action {
	case "list":
		return a.listFunctions(ctx, opts, flags, outputFormat(opts, flags, "table"))
	case "get":
		if err := requireArgs(pos, 1, "oictl functions get <function-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodGet, functionPath(pos[0], ""), nil)
	case "create":
		body, err := requiredBody(flags, "a function payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/functions/create", body)
	case "update":
		if err := requireArgs(pos, 1, "oictl functions update <function-id> --file function.json"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "a function payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, functionPath(pos[0], "/update"), body)
	case "delete":
		if err := requireArgs(pos, 1, "oictl functions delete <function-id> --yes"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodDelete, functionPath(pos[0], "/delete"), nil)
	case "export":
		path := "/api/v1/functions/export"
		if flags.bools["include-valves"] {
			path = appendQuery(path, map[string]string{"include_valves": "true"})
		}
		return simple(http.MethodGet, path, nil)
	case "load-url":
		if err := requireArgs(pos, 1, "oictl functions load-url <url>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/functions/load/url", jsonBody(map[string]string{"url": pos[0]}))
	case "sync":
		if err := requireFunctionSyncConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := functionSyncBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/functions/sync", body)
	case "toggle", "toggle-global":
		if err := requireArgs(pos, 1, "oictl functions "+action+" <function-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		suffix := "/toggle"
		if action == "toggle-global" {
			suffix = "/toggle/global"
		}
		return simple(http.MethodPost, functionPath(pos[0], suffix), nil)
	}
	fmt.Fprintf(a.err, "unknown functions command %q\n", action)
	return 1
}

func (a *App) listFunctions(ctx context.Context, opts globalOptions, flags commandFlags, format string) int {
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/functions/list", authRequired: true})
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		}
		return 1
	}
	if flags.values["type"] != "" {
		body, err = filterFunctionList(body, flags.values["type"])
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
	}
	outPath := firstNonEmpty(flags.values["out"], opts.outPath)
	if outPath != "" {
		if err := writeSecureFile(outPath, body); err != nil {
			fmt.Fprintf(a.err, "write output: %v\n", err)
			return 1
		}
		return 0
	}
	if format == "table" {
		if rendered, ok := renderFunctionTable(body); ok {
			return a.writeLocalOutput(rendered, "")
		}
	}
	if err := writeAll(a.out, body); err != nil {
		fmt.Fprintf(a.err, "write output: %v\n", err)
		return 1
	}
	return 0
}

func filterFunctionList(body []byte, functionType string) ([]byte, error) {
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode function list: %w", err)
	}
	filtered := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if stringValue(row["type"]) == functionType {
			filtered = append(filtered, row)
		}
	}
	return json.Marshal(filtered)
}

func renderFunctionTable(body []byte) ([]byte, bool) {
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, false
	}
	if len(rows) == 0 {
		return []byte("No results\n"), true
	}
	columns := []string{"id", "name", "type", "active", "global", "updated_at", "created_at"}
	normalized := make([]map[string]string, 0, len(rows))
	widths := map[string]int{}
	for _, column := range columns {
		widths[column] = len(strings.ToUpper(column))
	}
	for _, row := range rows {
		normalizedRow := map[string]string{
			"id":         tableValue(row["id"]),
			"name":       tableValue(row["name"]),
			"type":       tableValue(row["type"]),
			"active":     tableValue(firstPresent(row, "is_active", "active")),
			"global":     tableValue(firstPresent(row, "is_global", "global")),
			"updated_at": tableValue(row["updated_at"]),
			"created_at": tableValue(row["created_at"]),
		}
		for _, column := range columns {
			if len(normalizedRow[column]) > widths[column] {
				widths[column] = len(normalizedRow[column])
			}
		}
		normalized = append(normalized, normalizedRow)
	}
	var out bytes.Buffer
	for i, column := range columns {
		if i > 0 {
			out.WriteByte(' ')
		}
		fmt.Fprintf(&out, "%-*s", widths[column], strings.ToUpper(column))
	}
	out.WriteByte('\n')
	for _, row := range normalized {
		for i, column := range columns {
			if i > 0 {
				out.WriteByte(' ')
			}
			fmt.Fprintf(&out, "%-*s", widths[column], row[column])
		}
		out.WriteByte('\n')
	}
	return out.Bytes(), true
}

func firstPresent(row map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			return value
		}
	}
	return nil
}

func functionSyncBody(flags commandFlags) (io.Reader, error) {
	body, err := requiredBody(flags, "a function sync payload is required")
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, errors.New("a function sync payload is required")
	}
	if !json.Valid(trimmed) {
		return nil, errors.New("function sync payload must be valid JSON")
	}
	if trimmed[0] == '[' {
		wrapped := append([]byte(`{"functions":`), trimmed...)
		wrapped = append(wrapped, '}')
		return bytes.NewReader(wrapped), nil
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &envelope); err != nil {
		return nil, errors.New("function sync payload must contain an explicit functions array")
	}
	inventory := bytes.TrimSpace(envelope["functions"])
	if len(inventory) == 0 || inventory[0] != '[' {
		return nil, errors.New("function sync payload must contain an explicit functions array")
	}
	return bytes.NewReader(data), nil
}

func requireFunctionSyncConfirmation(flags commandFlags) error {
	if flags.bools["yes"] || flags.bools["confirm"] {
		return nil
	}
	return errors.New("destructive function sync can remove remote functions omitted from the input; use --yes")
}

func (a *App) runSkills(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Manage Open WebUI skills.

Usage:
  oictl skills <list|get|create|update|access-update|toggle|delete|export>

Commands:
  list          List skills with --query, --view-option, and --page filters
  get           Get one skill by ID
  create        Create a skill from --data, --file, or --manifest
  update        Update a skill from --data, --file, or --manifest
  access-update Replace access grants from --data or --file JSON
  toggle        Toggle a skill active state
  delete        Delete a skill; requires --yes
  export        Export all skills as JSON, one skill as JSON, or one skill as --format manifest --out path

Markdown manifests support frontmatter keys id, name, description, is_active, and simple tags. Use JSON for complex fields such as access_grants.
`)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	format := outputFormat(opts, flags, "table")
	simple := func(method, path string, body io.Reader) int {
		return a.executeStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
	}
	skillPath := func(id string, suffix string) string {
		return "/api/v1/skills/id/" + url.PathEscape(id) + suffix
	}
	switch action {
	case "list":
		return simple(http.MethodGet, appendQuery("/api/v1/skills/list", map[string]string{"query": flags.values["query"], "view_option": flags.values["view-option"], "page": flags.values["page"]}), nil)
	case "get":
		if err := requireArgs(pos, 1, "oictl skills get <skill-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodGet, skillPath(pos[0], ""), nil)
	case "create":
		body, err := skillPayloadBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if body == nil {
			fmt.Fprintln(a.err, "a skill payload is required")
			return 1
		}
		return simple(http.MethodPost, "/api/v1/skills/create", body)
	case "update":
		if err := requireArgs(pos, 1, "oictl skills update <skill-id> --file skill.json"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := skillPayloadBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if body == nil {
			fmt.Fprintln(a.err, "a skill payload is required")
			return 1
		}
		return simple(http.MethodPost, skillPath(pos[0], "/update"), body)
	case "access-update":
		if err := requireArgs(pos, 1, "oictl skills access-update <skill-id> --file access.json"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "an access-grants payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, skillPath(pos[0], "/access/update"), body)
	case "toggle":
		if err := requireArgs(pos, 1, "oictl skills toggle <skill-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, skillPath(pos[0], "/toggle"), nil)
	case "delete":
		if err := requireArgs(pos, 1, "oictl skills delete <skill-id> --yes"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodDelete, skillPath(pos[0], "/delete"), nil)
	case "export":
		format = outputFormat(opts, flags, "json")
		if flags.values["format"] == "manifest" {
			return a.exportSkillManifest(ctx, opts, flags, pos)
		}
		if len(pos) == 0 {
			return simple(http.MethodGet, "/api/v1/skills/export", nil)
		}
		if err := requireArgs(pos, 1, "oictl skills export [skill-id]"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodGet, skillPath(pos[0], ""), nil)
	}
	fmt.Fprintf(a.err, "unknown skills command %q\n", action)
	return 1
}

func (a *App) runTools(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Manage Open WebUI tools.

Usage:
  oictl tools <list|get|create|update|delete|export|load-url|access-update|valves>

Commands:
  list          List tools
  get           Get one tool by ID
  create        Create a tool from --data or --file JSON
  update        Update a tool from --data or --file JSON
  delete        Delete a tool by ID
  export        Export raw server records, or JSON Tool manifests with --manifest
  load-url      Ask Open WebUI to fetch tool source from a URL payload
  access-update Replace access grants from --data or --file JSON
  valves        Get, inspect, and update global or user-scoped valves

Examples:
  oictl tools list
  oictl tools create --file tool.json
  oictl tools update weather_tool --data '{"name":"Weather","content":"..."}'
  oictl tools valves spec weather_tool
  oictl tools export --out tools.json
  oictl tools export weather_tool --manifest --out weather-tool.json
  oictl tools export --manifest --directory manifests/tools
`)
		return 0
	}
	action := args[0]
	if action == "valves" {
		return a.runValves(ctx, opts, "tools", "tool-id", args[1:])
	}
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	format := outputFormat(opts, flags, "table")
	simple := func(method, path string, body io.Reader) int {
		return a.executeStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
	}
	toolPath := func(id string, suffix string) string {
		return "/api/v1/tools/id/" + url.PathEscape(id) + suffix
	}
	switch action {
	case "list":
		return simple(http.MethodGet, "/api/v1/tools/list", nil)
	case "get":
		id, err := requireToolID(pos, "oictl tools get <tool-id>")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodGet, toolPath(id, ""), nil)
	case "create":
		body, err := requiredBody(flags, "a JSON payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/tools/create", body)
	case "update":
		id, err := requireToolID(pos, "oictl tools update <tool-id> --file tool.json")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "a JSON payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, toolPath(id, "/update"), body)
	case "delete":
		id, err := requireToolID(pos, "oictl tools delete <tool-id>")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodDelete, toolPath(id, "/delete"), nil)
	case "access-update":
		id, err := requireToolID(pos, "oictl tools access-update <tool-id> --file access.json")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "a JSON payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, toolPath(id, "/access/update"), body)
	case "load-url":
		body, err := requiredBody(flags, "a JSON payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/tools/load/url", body)
	case "export":
		if flags.bools["manifest"] {
			return a.exportToolManifests(ctx, opts, flags, pos)
		}
		if len(pos) != 0 {
			fmt.Fprintln(a.err, "usage: oictl tools export [--out path] or oictl tools export [tool-id] --manifest")
			return 1
		}
		return a.executeRaw(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/tools/export", authRequired: true, outPath: flags.values["out"]})
	}
	fmt.Fprintf(a.err, "unknown tools command %q\n", action)
	return 1
}

func (a *App) runValves(ctx context.Context, opts globalOptions, resource string, idName string, args []string) int {
	if isHelp(args) || hasHelpFlag(args) {
		a.printValvesHelp(resource, idName)
		return 0
	}
	flags, pos, err := parseCommandFlags(args)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if len(pos) == 0 {
		fmt.Fprintf(a.err, "usage: oictl %s valves <get|spec|update|user>\n", resource)
		return 1
	}
	scope := ""
	action := pos[0]
	if action == "user" {
		scope = "/user"
		if len(pos) < 2 {
			fmt.Fprintf(a.err, "usage: oictl %s valves user <get|spec|update> <%s>\n", resource, idName)
			return 1
		}
		action = pos[1]
		pos = pos[2:]
	} else {
		pos = pos[1:]
	}
	if action != "get" && action != "spec" && action != "update" {
		fmt.Fprintf(a.err, "unknown %s valves command %q\n", resource, action)
		return 1
	}
	if err := requireArgs(pos, 1, valveUsage(resource, idName, scope, action)); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	var body io.Reader
	method := http.MethodGet
	suffix := "/valves" + scope
	if action == "spec" {
		suffix += "/spec"
	} else if action == "update" {
		method = http.MethodPost
		suffix += "/update"
		body, err = requiredBody(flags, "a valve payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
	}
	path := "/api/v1/" + resource + "/id/" + url.PathEscape(pos[0]) + suffix
	format := outputFormat(opts, flags, "json")
	return a.executeStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
}

func (a *App) printValvesHelp(resource string, idName string) {
	fmt.Fprintf(a.out, `Manage Open WebUI %s valves.

Usage:
  oictl %s valves get <%s>
  oictl %s valves spec <%s>
  oictl %s valves update <%s> (--data JSON | --file path | --file -)
  oictl %s valves user get <%s>
  oictl %s valves user spec <%s>
  oictl %s valves user update <%s> (--data JSON | --file path | --file -)

Commands:
  get          Get global valve values
  spec         Get the global valve JSON schema
  update       Update global valve values from JSON
  user get     Get user-scoped valve values
  user spec    Get the user-scoped valve JSON schema
  user update  Update user-scoped valve values from JSON
`, resource, resource, idName, resource, idName, resource, idName, resource, idName, resource, idName, resource, idName)
}

func hasHelpFlag(args []string) bool {
	_, help := jsonHelpPositionals(args)
	return help || (len(args) > 0 && args[0] == "help")
}

func valveUsage(resource string, idName string, scope string, action string) string {
	parts := []string{"oictl", resource, "valves"}
	if scope == "/user" {
		parts = append(parts, "user")
	}
	parts = append(parts, action, "<"+idName+">")
	if action == "update" {
		parts = append(parts, "(--data JSON | --file path | --file -)")
	}
	return strings.Join(parts, " ")
}

func requireToolID(args []string, usage string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("tool id is required")
	}
	if len(args) != 1 {
		return "", fmt.Errorf("usage: %s", usage)
	}
	return args[0], nil
}

type exportedToolManifest struct {
	APIVersion   string           `json:"apiVersion"`
	Kind         string           `json:"kind"`
	Metadata     manifestMetadata `json:"metadata"`
	Spec         map[string]any   `json:"spec"`
	AccessGrants *[]accessGrant   `json:"access_grants,omitempty"`
}

func (a *App) exportToolManifests(ctx context.Context, opts globalOptions, flags commandFlags, args []string) int {
	if len(args) > 1 {
		fmt.Fprintln(a.err, "usage: oictl tools export [tool-id] --manifest [--out path]")
		return 1
	}
	if len(args) == 1 {
		body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/tools/id/" + url.PathEscape(args[0]), authRequired: true})
		if err != nil {
			var statusErr httpStatusError
			if !errors.As(err, &statusErr) {
				fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
			}
			return 1
		}
		manifest, _, err := toolManifestFromJSON(body)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return a.writeLocalOutput(manifest, firstNonEmpty(flags.values["out"], opts.outPath))
	}

	dir := firstNonEmpty(flags.values["directory"], flags.values["dir"])
	if dir == "" {
		fmt.Fprintln(a.err, "--directory is required for multi-tool manifest export")
		return 1
	}
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/tools/export", authRequired: true})
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		}
		return 1
	}
	tools, err := decodeToolExport(body)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	sort.SliceStable(tools, func(i, j int) bool {
		return stringValue(tools[i]["id"]) < stringValue(tools[j]["id"])
	})
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(a.err, "create manifest directory: %v\n", err)
		return 1
	}
	seen := map[string]string{}
	for _, tool := range tools {
		manifest, id, err := toolManifestFromObject(tool)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		name := toolManifestFileName(id)
		if previous, ok := seen[name]; ok {
			fmt.Fprintf(a.err, "tool ids %q and %q map to the same manifest file %q\n", previous, id, name)
			return 1
		}
		seen[name] = id
		if err := writeSecureFile(filepath.Join(dir, name), manifest); err != nil {
			fmt.Fprintf(a.err, "write manifest: %v\n", err)
			return 1
		}
	}
	return 0
}

func (a *App) writeLocalOutput(body []byte, outPath string) int {
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

func toolManifestFromJSON(body []byte) ([]byte, string, error) {
	var tool map[string]any
	if err := json.Unmarshal(body, &tool); err != nil {
		return nil, "", fmt.Errorf("decode tool response: %w", err)
	}
	return toolManifestFromObject(tool)
}

func toolManifestFromObject(tool map[string]any) ([]byte, string, error) {
	id := strings.TrimSpace(stringValue(tool["id"]))
	if id == "" {
		return nil, "", errors.New("tool export is missing id")
	}
	spec := map[string]any{"id": id}
	for _, field := range []string{"name", "content", "meta"} {
		if value, ok := tool[field]; ok {
			spec[field] = value
		}
	}
	manifest := exportedToolManifest{
		APIVersion: manifestAPIVersion,
		Kind:       "Tool",
		Metadata:   manifestMetadata{Name: id},
		Spec:       spec,
	}
	if value, ok := tool["access_grants"]; ok {
		grants, err := decodeAccessGrantsValue(value)
		if err != nil {
			return nil, "", err
		}
		grants = normalizeAccessGrants(grants)
		manifest.AccessGrants = &grants
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, "", err
	}
	body = append(body, '\n')
	return body, id, nil
}

func decodeAccessGrantsValue(value any) ([]accessGrant, error) {
	body, err := json.Marshal(value)
	if err != nil || bytes.Equal(body, []byte("null")) {
		return nil, err
	}
	var grants []accessGrant
	if err := json.Unmarshal(body, &grants); err != nil {
		return nil, fmt.Errorf("decode tool access_grants: %w", err)
	}
	return grants, nil
}

func decodeToolExport(body []byte) ([]map[string]any, error) {
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode tools export: %w", err)
	}
	switch value := decoded.(type) {
	case []any:
		return rowsFromArray(value), nil
	case map[string]any:
		for _, key := range []string{"tools", "items", "data", "results"} {
			if array, ok := value[key].([]any); ok {
				return rowsFromArray(array), nil
			}
		}
		return []map[string]any{value}, nil
	default:
		return nil, errors.New("tools export response must be a JSON object or array")
	}
}

func toolManifestFileName(id string) string {
	var name strings.Builder
	for _, r := range id {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' {
			name.WriteRune(r)
		} else {
			name.WriteByte('_')
		}
	}
	base := name.String()
	if base == "" || base == "." || base == ".." {
		base = "tool"
	}
	return base + ".json"
}

func skillPayloadBody(flags commandFlags) (io.Reader, error) {
	manifest := flags.values["manifest"]
	if manifest == "" {
		return requestBody(flags)
	}
	if flags.values["data"] != "" || flags.values["file"] != "" || flags.values["data-file"] != "" {
		return nil, errors.New("use only one of --manifest, --data, or --file")
	}
	body, err := os.ReadFile(manifest)
	if err != nil {
		return nil, fmt.Errorf("read skill manifest: %w", err)
	}
	payload, err := parseSkillManifest(body)
	if err != nil {
		return nil, err
	}
	return jsonBody(payload), nil
}

func parseSkillManifest(body []byte) (map[string]any, error) {
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	frontmatter := ""
	content := text
	if strings.HasPrefix(text, "---\n") {
		end := strings.Index(text[4:], "\n---")
		if end < 0 {
			return nil, errors.New("skill manifest frontmatter is missing closing ---")
		}
		frontmatter = text[4 : 4+end]
		content = strings.TrimPrefix(text[4+end+4:], "\n")
	}
	payload := map[string]any{"content": content}
	if strings.TrimSpace(frontmatter) == "" {
		return payload, nil
	}
	lines := strings.Split(frontmatter, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, raw, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("unsupported skill manifest frontmatter line %q", line)
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)
		switch key {
		case "id", "name", "description":
			value, err := parseManifestScalar(raw)
			if err != nil {
				return nil, fmt.Errorf("%s must be a scalar string", key)
			}
			payload[key] = value
		case "is_active":
			value := strings.ToLower(strings.Trim(raw, `"'`))
			if value != "true" && value != "false" {
				return nil, errors.New("is_active must be true or false")
			}
			payload[key] = value == "true"
		case "tags":
			tags, next, err := parseManifestTags(raw, lines, i)
			if err != nil {
				return nil, err
			}
			i = next
			payload["meta"] = map[string]any{"tags": tags}
		case "access_grants":
			return nil, errors.New("skill manifest does not support access_grants; use JSON --file or --data")
		default:
			return nil, fmt.Errorf("unsupported skill manifest frontmatter key %q", key)
		}
	}
	return payload, nil
}

func parseManifestScalar(raw string) (string, error) {
	if raw == "" || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
		return "", errors.New("not scalar")
	}
	if strings.HasPrefix(raw, `"`) {
		var value string
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			return "", fmt.Errorf("invalid double-quoted scalar: %w", err)
		}
		return value, nil
	}
	if strings.HasPrefix(raw, `'`) {
		if len(raw) < 2 || raw[len(raw)-1] != '\'' {
			return "", errors.New("unterminated quoted scalar")
		}
		inner := raw[1 : len(raw)-1]
		if strings.Contains(strings.ReplaceAll(inner, "''", ""), "'") {
			return "", errors.New("single quotes within a single-quoted scalar must be doubled")
		}
		return strings.ReplaceAll(inner, "''", "'"), nil
	}
	return raw, nil
}

func parseManifestTags(raw string, lines []string, index int) ([]string, int, error) {
	if raw != "" {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			trimmed = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
		}
		if trimmed == "" {
			return nil, index, nil
		}
		parts := strings.Split(trimmed, ",")
		tags := make([]string, 0, len(parts))
		for _, part := range parts {
			value, err := parseManifestScalar(strings.TrimSpace(part))
			if err != nil {
				return nil, index, errors.New("tags must be a simple scalar list")
			}
			if value != "" {
				tags = append(tags, value)
			}
		}
		return tags, index, nil
	}
	tags := []string{}
	next := index
	for j := index + 1; j < len(lines); j++ {
		line := lines[j]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}
		if !strings.HasPrefix(trimmed, "-") {
			return nil, index, errors.New("tags must be a simple scalar list")
		}
		value, err := parseManifestScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
		if err != nil {
			return nil, index, errors.New("tags must be a simple scalar list")
		}
		tags = append(tags, value)
		next = j
	}
	return tags, next, nil
}

func (a *App) exportSkillManifest(ctx context.Context, opts globalOptions, flags commandFlags, args []string) int {
	if err := requireArgs(args, 1, "oictl skills export <skill-id> --format manifest --out skill.md"); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	outPath := firstNonEmpty(flags.values["out"], opts.outPath)
	if outPath == "" {
		fmt.Fprintln(a.err, "--out is required for manifest export")
		return 1
	}
	body, err := a.doRequest(ctx, opts, requestSpec{method: http.MethodGet, path: "/api/v1/skills/id/" + url.PathEscape(args[0]), authRequired: true})
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		}
		return 1
	}
	manifest, err := skillManifestFromJSON(body)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if err := writeSecureFile(outPath, manifest); err != nil {
		fmt.Fprintf(a.err, "write output: %v\n", err)
		return 1
	}
	return 0
}

func skillManifestFromJSON(body []byte) ([]byte, error) {
	var skill map[string]any
	if err := json.Unmarshal(body, &skill); err != nil {
		return nil, fmt.Errorf("decode skill response: %w", err)
	}
	var out bytes.Buffer
	out.WriteString("---\n")
	for _, key := range []string{"id", "name", "description"} {
		if value, ok := skill[key].(string); ok && value != "" {
			fmt.Fprintf(&out, "%s: %s\n", key, manifestQuoted(value))
		}
	}
	if value, ok := skill["is_active"].(bool); ok {
		fmt.Fprintf(&out, "is_active: %t\n", value)
	}
	if meta, ok := skill["meta"].(map[string]any); ok {
		if values, ok := meta["tags"].([]any); ok && len(values) > 0 {
			out.WriteString("tags:\n")
			for _, value := range values {
				if tag, ok := value.(string); ok {
					fmt.Fprintf(&out, "  - %s\n", manifestQuoted(tag))
				}
			}
		}
	}
	out.WriteString("---\n")
	if content, ok := skill["content"].(string); ok {
		out.WriteString(content)
	}
	return out.Bytes(), nil
}

func manifestQuoted(value string) string {
	body, _ := json.Marshal(value)
	return string(body)
}

func (a *App) runChats(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Manage Open WebUI chats and chat workflows.

Usage:
  oictl chats <list|search|get|export|import|share|tags|archive|pin|delete|compact>

Commands:
  list       List chats with pagination and one scope: --user-id, --folder-id, or --archived (scope flags cannot be combined)
  search     Search chats with --query and pagination filters
  get        Get one chat by ID
  export     Export own chats as NDJSON to stdout or --out; --all-users explicitly selects admin database-wide JSON export, --chat-id selects one chat
  import     Import chat records from --data, --file, or --file -
  share      Share a chat
  tags       Get, set, or delete chat tags
  archive    Archive or unarchive a chat
  pin        Pin or unpin a chat
  delete     Delete a chat; requires --yes
  compact    Compact a chat using a JSON payload or --model
`)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	format := outputFormat(opts, flags, "table")
	simple := func(method, path string, body io.Reader) int {
		return a.executeStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
	}
	switch action {
	case "list":
		scopes := 0
		for _, key := range []string{"user-id", "archived", "folder-id"} {
			if _, supplied := flags.values[key]; supplied {
				scopes++
			}
		}
		if scopes > 1 {
			fmt.Fprintln(a.err, "chat scope filters --user-id, --archived, and --folder-id cannot be combined")
			return 1
		}
		path := "/api/v1/chats/list"
		if flags.values["user-id"] != "" {
			path = "/api/v1/chats/list/user/" + url.PathEscape(flags.values["user-id"])
		}
		if flags.values["archived"] == "true" {
			path = "/api/v1/chats/archived"
		}
		if flags.values["folder-id"] != "" {
			path = "/api/v1/chats/folder/" + url.PathEscape(flags.values["folder-id"]) + "/list"
		}
		return simple(http.MethodGet, appendQuery(path, map[string]string{"page": flags.values["page"], "limit": flags.values["limit"], "skip": flags.values["skip"]}), nil)
	case "search":
		return simple(http.MethodGet, appendQuery("/api/v1/chats/search", map[string]string{"query": firstNonEmpty(flags.values["query"], flags.values["q"]), "text": firstNonEmpty(flags.values["query"], flags.values["q"]), "page": flags.values["page"], "limit": flags.values["limit"]}), nil)
	case "get":
		if err := requireArgs(pos, 1, "oictl chats get <chat-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodGet, "/api/v1/chats/"+url.PathEscape(pos[0]), nil)
	case "export":
		if flags.bools["all-users"] && flags.values["chat-id"] != "" {
			fmt.Fprintln(a.err, "--all-users and --chat-id cannot be combined")
			return 1
		}
		path := "/api/v1/chats/all"
		if flags.bools["all-users"] {
			path = "/api/v1/chats/all/db"
		}
		if flags.values["chat-id"] != "" {
			path = "/api/v1/chats/" + url.PathEscape(flags.values["chat-id"])
		}
		// Own-user exports are NDJSON, including the single-record case.
		return a.executeRaw(ctx, opts, requestSpec{method: http.MethodGet, path: path, authRequired: true, outPath: flags.values["out"]})
	case "import":
		body, err := requiredBody(flags, "an import payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/chats/import", body)
	case "share":
		if err := requireArgs(pos, 1, "oictl chats share <chat-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/chats/"+url.PathEscape(pos[0])+"/share", nil)
	case "tags":
		return a.runChatTags(ctx, opts, flags, pos, format)
	case "archive", "pin":
		if err := requireArgs(pos, 1, "oictl chats "+action+" <chat-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/chats/"+url.PathEscape(pos[0])+"/"+action, nil)
	case "compact":
		if err := requireArgs(pos, 1, "oictl chats compact <chat-id> [--model model-id]"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if body == nil && flags.values["model"] != "" {
			body = jsonBody(map[string]string{"model": flags.values["model"]})
		}
		return simple(http.MethodPost, "/api/v1/chats/"+url.PathEscape(pos[0])+"/compact", body)
	case "delete":
		if err := requireArgs(pos, 1, "oictl chats delete <chat-id> --yes"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodDelete, "/api/v1/chats/"+url.PathEscape(pos[0]), nil)
	}
	fmt.Fprintf(a.err, "unknown chats command %q\n", action)
	return 1
}

func (a *App) runChatTags(ctx context.Context, opts globalOptions, flags commandFlags, args []string, format string) int {
	if len(args) < 2 {
		fmt.Fprintln(a.err, "usage: oictl chats tags <get|set|delete> <chat-id>")
		return 1
	}
	action, id := args[0], url.PathEscape(args[1])
	path := "/api/v1/chats/" + id + "/tags"
	spec := requestSpec{path: path, authRequired: true, outPath: flags.values["out"]}
	switch action {
	case "get":
		spec.method = http.MethodGet
	case "set", "delete":
		spec.method = http.MethodPost
		if action == "delete" {
			spec.method = http.MethodDelete
		}
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if body == nil {
			tags := flags.multiValues["tag"]
			if len(tags) != 1 {
				fmt.Fprintf(a.err, "chats tags %s accepts exactly one --tag value; use --data or --file for custom payloads\n", action)
				return 1
			}
			body = jsonBody(map[string]any{"name": tags[0]})
		}
		spec.body = body
	default:
		fmt.Fprintln(a.err, "unknown chats tags command")
		return 1
	}
	return a.executeStructured(ctx, opts, spec, format)
}

func normalizeChannelMemberActiveBody(body io.Reader) (io.Reader, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return bytes.NewReader(data), nil
	}
	if active, ok := payload["active"]; ok {
		if _, hasCanonical := payload["is_active"]; !hasCanonical {
			payload["is_active"] = active
		}
		delete(payload, "active")
	}
	return jsonBody(payload), nil
}

func (a *App) runAnalytics(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Inspect Open WebUI administrative analytics reports.

Usage:
  oictl analytics <models|users|messages|summary|daily|tokens>
  oictl analytics models <chats|overview> <model-id>

Filters:
  --start-date int  Start epoch timestamp
  --end-date int    End epoch timestamp
  --group-id id     Group filter where supported
  --user-id id      User filter where supported
  --model-id id     Model filter where supported
  --chat-id id      Chat filter where supported
  --skip int        Result offset
  --limit int       Result limit
`)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	format := outputFormat(opts, flags, "table")
	query := analyticsQuery(flags)
	path := ""
	switch action {
	case "models":
		if len(pos) == 0 {
			path = appendQuery("/api/v1/analytics/models", query)
		} else if len(pos) == 2 && (pos[0] == "chats" || pos[0] == "overview") {
			path = "/api/v1/analytics/models/" + url.PathEscape(pos[1]) + "/" + pos[0]
			path = appendQuery(path, query)
		} else {
			fmt.Fprintln(a.err, "usage: oictl analytics models [chats|overview <model-id>]")
			return 1
		}
	case "users", "messages", "summary", "daily", "tokens":
		path = appendQuery("/api/v1/analytics/"+action, query)
	default:
		fmt.Fprintf(a.err, "unknown analytics command %q\n", action)
		return 1
	}
	return a.executeStructured(ctx, opts, requestSpec{method: http.MethodGet, path: path, authRequired: true, outPath: flags.values["out"]}, format)
}

func analyticsQuery(flags commandFlags) map[string]string {
	return map[string]string{
		"start_date":  flags.values["start-date"],
		"end_date":    flags.values["end-date"],
		"group_id":    flags.values["group-id"],
		"user_id":     flags.values["user-id"],
		"model_id":    flags.values["model-id"],
		"chat_id":     flags.values["chat-id"],
		"skip":        flags.values["skip"],
		"limit":       flags.values["limit"],
		"days":        flags.values["days"],
		"granularity": flags.values["granularity"],
		"order_by":    flags.values["order-by"],
		"direction":   flags.values["direction"],
	}
}

func (a *App) runAutomations(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Manage Open WebUI scheduled automation workflows.

Usage:
  oictl automations <list|create|get|update|toggle|run|delete>
  oictl automations runs list <automation-id>

Payload commands accept --data, --file, or --file - and rely on server-side schedule validation.
Delete requires --yes.
`)
		return 0
	}
	action := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	format := outputFormat(opts, flags, "table")
	simple := func(method, path string, body io.Reader) int {
		return a.executeStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
	}
	switch action {
	case "list":
		return simple(http.MethodGet, appendQuery("/api/v1/automations/list", map[string]string{"page": flags.values["page"], "status": flags.values["status"], "limit": flags.values["limit"], "skip": flags.values["skip"]}), nil)
	case "create":
		body, err := requiredBody(flags, "an automation payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/automations/create", body)
	case "get", "toggle", "run":
		if err := requireArgs(pos, 1, "oictl automations "+action+" <automation-id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		method, suffix := http.MethodGet, ""
		if action == "toggle" || action == "run" {
			method, suffix = http.MethodPost, "/"+action
		}
		return simple(method, "/api/v1/automations/"+url.PathEscape(pos[0])+suffix, nil)
	case "update":
		if err := requireArgs(pos, 1, "oictl automations update <automation-id> --file automation.json"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		body, err := requiredBody(flags, "an automation payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, "/api/v1/automations/"+url.PathEscape(pos[0])+"/update", body)
	case "delete":
		if err := requireArgs(pos, 1, "oictl automations delete <automation-id> --yes"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodDelete, "/api/v1/automations/"+url.PathEscape(pos[0])+"/delete", nil)
	case "runs":
		if len(pos) != 2 || pos[0] != "list" {
			fmt.Fprintln(a.err, "usage: oictl automations runs list <automation-id>")
			return 1
		}
		return simple(http.MethodGet, appendQuery("/api/v1/automations/"+url.PathEscape(pos[1])+"/runs", map[string]string{"limit": flags.values["limit"], "skip": flags.values["skip"]}), nil)
	}
	fmt.Fprintf(a.err, "unknown automations command %q\n", action)
	return 1
}

func (a *App) runSCIM(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Manage SCIM 2.0 provisioning resources through Open WebUI.

Usage:
  oictl scim service-provider-config
  oictl scim resource-types
  oictl scim schemas
  oictl scim users <list|get|create|replace|patch|delete>
  oictl scim groups <list|get|create|replace|patch|delete>

Authentication uses --scim-token, OPEN_WEBUI_SCIM_TOKEN, or a profile scim_token. Normal Open WebUI API tokens are not used.
`)
		return 0
	}
	flags, pos, err := parseCommandFlags(args)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	token, err := a.resolveSCIMToken(opts, flags)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if len(pos) == 0 {
		fmt.Fprintln(a.err, "usage: oictl scim <service-provider-config|resource-types|schemas|users|groups>")
		return 1
	}
	format := outputFormat(opts, flags, "json")
	base := "/api/v1/scim/v2"
	spec := requestSpec{method: http.MethodGet, authRequired: false, skipAuth: true, bearerToken: token, outPath: flags.values["out"]}
	switch pos[0] {
	case "service-provider-config":
		spec.path = base + "/ServiceProviderConfig"
	case "resource-types":
		spec.path = base + "/ResourceTypes"
	case "schemas":
		spec.path = base + "/Schemas"
	case "users", "groups":
		return a.runSCIMResource(ctx, opts, flags, pos[0], pos[1:], token, format)
	default:
		fmt.Fprintf(a.err, "unknown scim command %q\n", pos[0])
		return 1
	}
	return a.executeStructured(ctx, opts, spec, format)
}

func (a *App) runSCIMResource(ctx context.Context, opts globalOptions, flags commandFlags, resource string, args []string, token string, format string) int {
	if len(args) == 0 {
		fmt.Fprintf(a.err, "usage: oictl scim %s <list|get|create|replace|patch|delete>\n", resource)
		return 1
	}
	name := map[string]string{"users": "Users", "groups": "Groups"}[resource]
	base := "/api/v1/scim/v2/" + name
	spec := requestSpec{authRequired: false, skipAuth: true, bearerToken: token, outPath: flags.values["out"]}
	switch args[0] {
	case "list":
		spec.method = http.MethodGet
		spec.path = appendQuery(base, map[string]string{"startIndex": firstNonEmpty(flags.values["start-index"], flags.values["startIndex"]), "count": flags.values["count"], "filter": flags.values["filter"]})
	case "get":
		if err := requireArgs(args[1:], 1, "oictl scim "+resource+" get <id>"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		spec.method = http.MethodGet
		spec.path = base + "/" + url.PathEscape(args[1])
	case "create", "replace", "patch":
		body, err := requiredBody(flags, "a SCIM payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		spec.body = body
		if args[0] == "create" {
			spec.method = http.MethodPost
			spec.path = base
		} else {
			if err := requireArgs(args[1:], 1, "oictl scim "+resource+" "+args[0]+" <id>"); err != nil {
				fmt.Fprintln(a.err, err)
				return 1
			}
			spec.method = map[string]string{"replace": http.MethodPut, "patch": http.MethodPatch}[args[0]]
			spec.path = base + "/" + url.PathEscape(args[1])
		}
	case "delete":
		if err := requireArgs(args[1:], 1, "oictl scim "+resource+" delete <id> --yes"); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		if err := requireConfirmation(flags); err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		spec.method = http.MethodDelete
		spec.path = base + "/" + url.PathEscape(args[1])
	default:
		fmt.Fprintf(a.err, "unknown scim %s command %q\n", resource, args[0])
		return 1
	}
	return a.executeStructured(ctx, opts, spec, format)
}

func (a *App) runProviders(ctx context.Context, opts globalOptions, args []string) int {
	if isHelp(args) {
		fmt.Fprint(a.out, `Send guarded OpenAI-compatible and Ollama-compatible requests through Open WebUI.

Usage:
  oictl providers openai <config|verify|models|request>
  oictl providers ollama <config|verify|tags|version|ps|models|request>

Raw request paths are relative passthrough paths routed through Open WebUI; external provider URLs and Authorization header overrides are rejected.
`)
		return 0
	}
	if len(args) == 0 || args[0] != "openai" && args[0] != "ollama" {
		fmt.Fprintln(a.err, "usage: oictl providers <openai|ollama> <command>")
		return 1
	}
	family := args[0]
	flags, pos, err := parseCommandFlags(args[1:])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if len(pos) == 0 {
		fmt.Fprintf(a.err, "usage: oictl providers %s <config|verify|models|request>\n", family)
		return 1
	}
	format := outputFormat(opts, flags, "json")
	prefix := "/" + family
	simple := func(method, path string, body io.Reader) int {
		return a.executeStructured(ctx, opts, requestSpec{method: method, path: path, body: body, authRequired: true, outPath: flags.values["out"]}, format)
	}
	switch pos[0] {
	case "config":
		if len(pos) != 2 || pos[1] != "get" && pos[1] != "update" && pos[1] != "set" {
			fmt.Fprintf(a.err, "usage: oictl providers %s config <get|update>\n", family)
			return 1
		}
		if pos[1] == "get" {
			return simple(http.MethodGet, prefix+"/config", nil)
		}
		body, err := requiredBody(flags, "a provider config payload is required")
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, prefix+"/config/update", body)
	case "verify":
		body, err := requestBody(flags)
		if err != nil {
			fmt.Fprintln(a.err, err)
			return 1
		}
		return simple(http.MethodPost, prefix+"/verify", body)
	case "models":
		path := prefix + "/models"
		if family == "ollama" {
			path = prefix + "/v1/models"
		}
		return simple(http.MethodGet, path, nil)
	case "tags":
		if family != "ollama" {
			fmt.Fprintln(a.err, "tags is only available for ollama")
			return 1
		}
		return simple(http.MethodGet, prefix+"/api/tags", nil)
	case "version", "ps":
		if family != "ollama" {
			fmt.Fprintf(a.err, "%s is only available for ollama\n", pos[0])
			return 1
		}
		return simple(http.MethodGet, prefix+"/api/"+pos[0], nil)
	case "request":
		return a.runProviderRequest(ctx, opts, flags, family, pos[1:])
	}
	fmt.Fprintf(a.err, "unknown providers %s command %q\n", family, pos[0])
	return 1
}

func (a *App) runProviderRequest(ctx context.Context, opts globalOptions, flags commandFlags, family string, args []string) int {
	if err := rejectUnsafeProviderHeaders(flags.headers); err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	if len(args) != 1 {
		fmt.Fprintf(a.err, "usage: oictl providers %s request [--method METHOD] <path>\n", family)
		return 1
	}
	path, err := providerPath(family, args[0])
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	body, err := requestBody(flags)
	if err != nil {
		fmt.Fprintln(a.err, err)
		return 1
	}
	method := strings.ToUpper(firstNonEmpty(flags.values["method"], http.MethodGet))
	if !isHTTPMethod(method) {
		fmt.Fprintf(a.err, "unsupported HTTP method %q\n", method)
		return 1
	}
	spec := requestSpec{method: method, path: path, body: body, headers: flags.headers, authRequired: true, outPath: flags.values["out"]}
	if flags.bools["stream"] {
		return a.executeStream(ctx, opts, spec)
	}
	return a.executeRaw(ctx, opts, spec)
}

func (a *App) executeStructured(ctx context.Context, opts globalOptions, spec requestSpec, format string) int {
	body, err := a.doRequest(ctx, opts, spec)
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token, spec.bearerToken))
		}
		return 1
	}
	outPath := firstNonEmpty(spec.outPath, opts.outPath)
	if outPath != "" {
		if err := writeSecureFile(outPath, body); err != nil {
			fmt.Fprintf(a.err, "write output: %v\n", err)
			return 1
		}
		return 0
	}
	if format == "table" {
		if rendered, ok := renderTable(body); ok {
			return a.writeLocalOutput(rendered, "")
		}
	}
	if err := writeAll(a.out, body); err != nil {
		fmt.Fprintf(a.err, "write output: %v\n", err)
		return 1
	}
	return 0
}

func (a *App) executeRaw(ctx context.Context, opts globalOptions, spec requestSpec) int {
	body, err := a.doRequest(ctx, opts, spec)
	if err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		}
		return 1
	}
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

func (a *App) executeStream(ctx context.Context, opts globalOptions, spec requestSpec) int {
	if err := a.doStream(ctx, opts, spec); err != nil {
		var statusErr httpStatusError
		if !errors.As(err, &statusErr) {
			fmt.Fprintln(a.err, redactSecrets(err.Error(), opts.token))
		}
		return 1
	}
	return 0
}

func (a *App) doStream(ctx context.Context, opts globalOptions, spec requestSpec) (retErr error) {
	target, err := a.resolveTarget(opts, spec.authRequired)
	if err != nil {
		return err
	}
	targetURL, err := joinURL(target.baseURL, spec.path)
	if err != nil {
		return err
	}
	if spec.method == "" {
		spec.method = http.MethodGet
	}
	if opts.timeout != "" {
		duration, err := time.ParseDuration(opts.timeout)
		if err != nil {
			return fmt.Errorf("parse timeout: %w", err)
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, spec.method, targetURL, spec.body)
	if err != nil {
		return err
	}
	if spec.contentType != "" {
		req.Header.Set("Content-Type", spec.contentType)
	} else if spec.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, header := range spec.headers {
		req.Header.Set(header.name, header.value)
	}
	if target.token != "" {
		req.Header.Set("Authorization", "Bearer "+target.token)
	}
	client := a.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, resp.Body.Close()) }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fmt.Fprintln(a.err, resp.Status)
		body, _ := io.ReadAll(resp.Body)
		_, _ = a.err.Write([]byte(redactSecrets(string(body), target.token)))
		return httpStatusError{status: resp.Status}
	}
	out := a.out
	if spec.outPath != "" || opts.outPath != "" {
		file, err := openSecureOutput(firstNonEmpty(spec.outPath, opts.outPath))
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		defer func() { retErr = errors.Join(retErr, file.Close()) }()
		out = file
	}
	_, err = io.Copy(out, resp.Body)
	return err
}

func outputFormat(opts globalOptions, flags commandFlags, fallback string) string {
	return strings.ToLower(firstNonEmpty(flags.values["output"], opts.output, fallback))
}

func requiredBody(flags commandFlags, message string) (io.Reader, error) {
	body, err := requestBody(flags)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, errors.New(message)
	}
	return body, nil
}

func requireConfirmation(flags commandFlags) error {
	if flags.bools["yes"] || flags.bools["confirm"] {
		return nil
	}
	return errors.New("destructive operation requires --yes")
}

func (a *App) resolveSCIMToken(opts globalOptions, flags commandFlags) (string, error) {
	profile, err := a.loadProfile(opts.profile)
	if err != nil {
		return "", err
	}
	token := firstNonEmpty(flags.values["scim-token"], a.getenv("OPEN_WEBUI_SCIM_TOKEN"), profile.SCIMToken)
	if token == "" {
		return "", errors.New("SCIM token is required; set --scim-token, OPEN_WEBUI_SCIM_TOKEN, or profile scim_token")
	}
	return token, nil
}

func providerPath(family string, raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(raw, "//") {
		return "", errors.New("provider passthrough paths must be relative and are routed through Open WebUI")
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	return "/" + family + raw, nil
}

func rejectUnsafeProviderHeaders(headers []headerValue) error {
	for _, header := range headers {
		if strings.EqualFold(header.name, "Authorization") {
			return errors.New("Authorization header overrides are not allowed for provider passthrough commands")
		}
	}
	return nil
}

func redactSecrets(text string, secrets ...string) string {
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "<redacted>")
		}
	}
	text = channelWebhookURLPattern.ReplaceAllString(text, "<redacted>")
	text = webhookTokenFieldPattern.ReplaceAllString(text, "$1<redacted>$2")
	text = webhookURLFieldPattern.ReplaceAllString(text, "$1<redacted>$2")
	return text
}

var (
	channelWebhookURLPattern = regexp.MustCompile(`https?://[^\s"']*/api/v1/channels/webhooks/[^\s"']+`)
	webhookTokenFieldPattern = regexp.MustCompile(`("(?:token|webhook_token|webhookToken|webhook_token_hash)"\s*:\s*")[^"]+(")`)
	webhookURLFieldPattern   = regexp.MustCompile(`("(?:url|webhook_url|destination_url|callback_url|target_url)"\s*:\s*")[^"]+(")`)
)

func renderTable(body []byte) ([]byte, bool) {
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, false
	}
	rows := extractRows(decoded)
	if len(rows) == 0 {
		return []byte("No results\n"), true
	}
	columns := tableColumns(rows)
	if len(columns) == 0 {
		return nil, false
	}
	widths := map[string]int{}
	for _, column := range columns {
		widths[column] = len(strings.ToUpper(column))
	}
	stringRows := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		stringRow := map[string]string{}
		for _, column := range columns {
			value := tableValue(row[column])
			stringRow[column] = value
			if len(value) > widths[column] {
				widths[column] = len(value)
			}
		}
		stringRows = append(stringRows, stringRow)
	}
	var out bytes.Buffer
	for i, column := range columns {
		if i > 0 {
			out.WriteByte(' ')
		}
		fmt.Fprintf(&out, "%-*s", widths[column], strings.ToUpper(column))
	}
	out.WriteByte('\n')
	for _, row := range stringRows {
		for i, column := range columns {
			if i > 0 {
				out.WriteByte(' ')
			}
			fmt.Fprintf(&out, "%-*s", widths[column], row[column])
		}
		out.WriteByte('\n')
	}
	return out.Bytes(), true
}

func extractRows(decoded any) []map[string]any {
	switch value := decoded.(type) {
	case []any:
		return rowsFromArray(value)
	case map[string]any:
		for _, key := range []string{"items", "data", "results", "models", "users", "chats", "Resources", "runs", "automations"} {
			if array, ok := value[key].([]any); ok {
				return rowsFromArray(array)
			}
		}
		return []map[string]any{value}
	default:
		return nil
	}
}

func rowsFromArray(values []any) []map[string]any {
	rows := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if row, ok := value.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func tableColumns(rows []map[string]any) []string {
	preferred := []string{"id", "chat_id", "model_id", "user_id", "name", "email", "title", "status", "count", "total", "total_tokens", "updated_at", "created_at"}
	seen := map[string]bool{}
	columns := []string{}
	for _, column := range preferred {
		for _, row := range rows {
			if _, ok := row[column]; ok {
				columns = append(columns, column)
				seen[column] = true
				break
			}
		}
	}
	if len(columns) >= 4 {
		return columns[:4]
	}
	keys := []string{}
	for _, row := range rows {
		for key := range row {
			if !seen[key] {
				keys = append(keys, key)
				seen[key] = true
			}
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		columns = append(columns, key)
		if len(columns) == 4 {
			break
		}
	}
	return columns
}

func tableValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		if len(typed) > 48 {
			return typed[:45] + "..."
		}
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprint(typed)
	default:
		body, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		text := string(body)
		if len(text) > 48 {
			return text[:45] + "..."
		}
		return text
	}
}
