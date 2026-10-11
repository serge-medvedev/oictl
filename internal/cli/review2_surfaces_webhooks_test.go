package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReview2SurfacesR14EnsureCreateIdentity(t *testing.T) {
	for _, source := range []string{"data", "file"} {
		t.Run(source, func(t *testing.T) {
			gets, posts := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					gets++
					io.WriteString(w, `[]`)
				} else {
					posts++
					io.WriteString(w, `{"name":"Different"}`)
				}
			}))
			defer server.Close()
			app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "synthetic"})
			value := `{"name":"Different"}`
			if source == "file" {
				path := filepath.Join(t.TempDir(), "webhook.json")
				if err := os.WriteFile(path, []byte(value), 0600); err != nil {
					t.Fatal(err)
				}
				value = path
			}
			args := []string{"webhooks", "channels", "ensure", "channel-a", "--name", "Expected", "--" + source, value, "--output", "json"}
			for i := 0; i < 2; i++ {
				if code := app.Run(context.Background(), args); code == 0 || posts != 0 || gets != i+1 || out.Len() != 0 {
					t.Fatalf("code=%d gets=%d posts=%d stdout=%s stderr=%s", code, gets, posts, out, errOut)
				}
			}
		})
	}
}

func TestReview2SurfacesR14EnsureExistingDoesNotReadCreateBody(t *testing.T) {
	for _, flags := range [][]string{
		{"--data", `{"name":"Different"}`}, {"--data", `invalid JSON`}, {"--file", "/missing/synthetic/webhook.json"},
	} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method != "GET" {
				t.Errorf("unexpected mutation %s", r.Method)
			}
			io.WriteString(w, `[{"id":"hook-a","name":"Expected","token":"synthetic-hook"}]`)
		}))
		app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "synthetic"})
		args := append([]string{"webhooks", "channels", "ensure", "channel-a", "--name", "Expected", "--output", "json"}, flags...)
		if code := app.Run(context.Background(), args); code != 0 || calls != 1 {
			t.Errorf("code=%d calls=%d stderr=%s", code, calls, errOut)
		}
		server.Close()
	}
}

func TestReview2SurfacesR14EnsurePreservesMatchingLiteralBody(t *testing.T) {
	literal := `{"name":" Expected ","profile_image_url":"https://example.invalid/image.png"}`
	var sent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			io.WriteString(w, `[]`)
			return
		}
		data, _ := io.ReadAll(r.Body)
		sent = string(data)
		io.WriteString(w, sent)
	}))
	defer server.Close()
	app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": server.URL, "OPEN_WEBUI_API_KEY": "synthetic"})
	if code := app.Run(context.Background(), []string{"webhooks", "channels", "ensure", "channel-a", "--name", " Expected ", "--data", literal, "--output", "json"}); code != 0 || sent != literal {
		t.Fatalf("code=%d sent=%s stderr=%s", code, sent, errOut)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(sent), &decoded); err != nil || !reflect.DeepEqual(decoded["name"], " Expected ") {
		t.Fatalf("literal identity modified: %v", decoded)
	}
}
