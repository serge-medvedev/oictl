package cli

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// Contracts from analytics.py at f8ae8a6c328946dc62cd0712961a0632574581bf:
// models:59, users:86, messages:126, summary:164, daily:200, tokens:238,
// model chats:288, model overview:346. Messages preserves chat > model > user
// server precedence; filters are not promised to be intersected by the client.
func TestReview2SurfacesR12AnalyticsEndpointFlags(t *testing.T) {
	reports := []struct{ command, allowed string }{
		{"models", "start-date end-date group-id"},
		{"users", "start-date end-date group-id limit"},
		{"messages", "start-date end-date user-id model-id chat-id skip limit"},
		{"summary", "start-date end-date group-id"},
		{"daily", "start-date end-date group-id granularity"},
		{"tokens", "start-date end-date group-id"},
		{"models chats model-a", "start-date end-date skip limit order-by direction"},
		{"models overview model-a", "days"},
	}
	for _, report := range reports {
		for _, flag := range strings.Fields("start-date end-date group-id user-id model-id chat-id skip limit days granularity order-by direction") {
			t.Run(report.command+"/"+flag, func(t *testing.T) {
				allowed := false
				for _, name := range strings.Fields(report.allowed) {
					if name == flag {
						allowed = true
					}
				}
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					key := strings.ReplaceAll(flag, "-", "_")
					if r.URL.Query().Get(key) != "value" || len(r.URL.Query()) != 1 {
						t.Errorf("query=%s", r.URL.RawQuery)
					}
					io.WriteString(w, `{}`)
				}))
				defer server.Close()
				app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "synthetic"})
				args := append([]string{"analytics"}, strings.Fields(report.command)...)
				args = append(args, "--"+flag, "value", "--output", "json")
				code := app.Run(context.Background(), args)
				if allowed {
					if code != 0 || calls != 1 {
						t.Fatalf("allowed: code=%d calls=%d stderr=%s", code, calls, errOut)
					}
				} else {
					if code == 0 || calls != 0 || !strings.Contains(errOut.String(), "not supported") {
						t.Fatalf("unsupported: code=%d calls=%d stderr=%s", code, calls, errOut)
					}
				}
			})
		}
	}
}

// chats.py:295,784,927,989,1123 and automations.py:163 use pages,
// whereas automation run history explicitly accepts skip/limit.
func TestReview2SurfacesR13PaginationRejectsIgnoredFlags(t *testing.T) {
	for _, command := range [][]string{
		{"chats", "list"}, {"chats", "list", "--user-id", "user-a"},
		{"chats", "list", "--archived", "true"}, {"chats", "list", "--folder-id", "folder-a"},
		{"chats", "search", "--query", "needle"}, {"automations", "list"},
	} {
		for _, flag := range []string{"limit", "skip"} {
			t.Run(strings.Join(command, " ")+"/"+flag, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; io.WriteString(w, `[]`) }))
				defer server.Close()
				app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "synthetic"})
				args := append(append([]string{}, command...), "--"+flag, "1")
				if code := app.Run(context.Background(), args); code == 0 || calls != 0 {
					t.Fatalf("code=%d calls=%d stderr=%s", code, calls, errOut)
				}
			})
		}
	}
}

func TestReview2SurfacesR13PageAndRunOffsetControls(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		path  string
		query url.Values
	}{
		{[]string{"chats", "list", "--page", "2"}, "/api/v1/chats/list", url.Values{"page": {"2"}}},
		{[]string{"chats", "search", "--q", "needle", "--page", "2"}, "/api/v1/chats/search", url.Values{"text": {"needle"}, "page": {"2"}}},
		{[]string{"automations", "list", "--page", "2", "--status", "active"}, "/api/v1/automations/list", url.Values{"page": {"2"}, "status": {"active"}}},
		{[]string{"automations", "runs", "list", "automation-a", "--skip", "30", "--limit", "1"}, "/api/v1/automations/automation-a/runs", url.Values{"skip": {"30"}, "limit": {"1"}}},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != tc.path || !reflect.DeepEqual(r.URL.Query(), tc.query) {
					t.Errorf("request=%s expected=%s %v", r.URL, tc.path, tc.query)
				}
				io.WriteString(w, `[]`)
			}))
			defer server.Close()
			app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "synthetic"})
			if code := app.Run(context.Background(), tc.args); code != 0 || calls != 1 {
				t.Fatalf("code=%d calls=%d stderr=%s", code, calls, errOut)
			}
		})
	}
}

func TestReview2SurfacesR22SearchRequiresQuery(t *testing.T) {
	for _, flags := range [][]string{nil, {"--query", ""}, {"--q", ""}} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; io.WriteString(w, `[]`) }))
		app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "synthetic"})
		if code := app.Run(context.Background(), append([]string{"chats", "search"}, flags...)); code == 0 || calls != 0 || out.Len() != 0 || !strings.Contains(errOut.String(), "required") {
			t.Errorf("flags=%v code=%d calls=%d stdout=%s stderr=%s", flags, code, calls, out, errOut)
		}
		server.Close()
	}
}

func TestReview2SurfacesR12MessagesPreserveServerPrecedence(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		want := url.Values{"chat_id": {"chat-a"}, "model_id": {"model-a"}, "user_id": {"user-a"}, "start_date": {"1"}, "end_date": {"2"}, "skip": {"3"}, "limit": {"4"}}
		if !reflect.DeepEqual(r.URL.Query(), want) {
			t.Errorf("query=%v want=%v", r.URL.Query(), want)
		}
		// Pinned handler chooses chat_id first, ignoring the other branch inputs.
		io.WriteString(w, `[{"chat_id":"chat-a","model_id":"other-model","user_id":"other-user"}]`)
	}))
	defer server.Close()
	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "synthetic"})
	args := []string{"analytics", "messages", "--chat-id", "chat-a", "--model-id", "model-a", "--user-id", "user-a", "--start-date", "1", "--end-date", "2", "--skip", "3", "--limit", "4", "--output", "json"}
	if code := app.Run(context.Background(), args); code != 0 || calls != 1 || !strings.Contains(out.String(), "other-user") {
		t.Fatalf("code=%d calls=%d stdout=%s stderr=%s", code, calls, out, errOut)
	}
}
