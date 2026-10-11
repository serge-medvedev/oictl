package cli

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestReview2CoreR18LocalScalarGlobalSpellings(t *testing.T) {
	values := []string{"--profile", "--base-url", "--url", "--token", "--output", "--timeout", "--out", "--profile=other", "--base-url=http://example.invalid/changed", "--url=http://example.invalid/changed", "--token=other", "--output=table", "--timeout=1ns", "--out=unwanted.json"}
	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			for _, command := range [][]string{{"users", "list", "--query"}, {"api", "POST", "/capture", "--data"}} {
				app, _ := coreReviewApp(t, `{"ok":true}`)
				calls := 0
				app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.URL.Host != "example.invalid" || r.Header.Get("Authorization") != "Bearer synthetic" {
						t.Errorf("target/auth changed: %s %q", r.URL, r.Header.Get("Authorization"))
					}
					if command[0] == "api" {
						b, _ := io.ReadAll(r.Body)
						if string(b) != value || r.URL.Path != "/capture" {
							t.Errorf("body/path = %q %s", b, r.URL.Path)
						}
					} else if r.URL.Query().Get("query") != value {
						t.Errorf("query = %q", r.URL.Query().Get("query"))
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
				})
				args := append(append([]string{}, command...), value)
				if code := app.Run(context.Background(), args); code != 0 || calls != 1 {
					t.Errorf("args=%q code=%d calls=%d", args, code, calls)
				}
			}
		})
	}
}

func TestReview2CoreR18MalformedExtraOperandCannotRetarget(t *testing.T) {
	for _, args := range [][]string{
		{"users", "list", "--query", "--token=other", "extra"},
		{"api", "POST", "/capture", "--data", "--base-url=http://example.invalid/changed", "extra"},
	} {
		app, calls := coreReviewApp(t, `true`)
		if code := app.Run(context.Background(), args); code == 0 || *calls != 0 {
			t.Errorf("args=%q code=%d calls=%d", args, code, *calls)
		}
	}
}

func TestReview2CoreR18BooleanAndManifestBoundaries(t *testing.T) {
	for _, tc := range []struct {
		args, remaining []string
		token           string
	}{
		{[]string{"models", "sync", "--yes", "--token", "chosen", "--data", "[]"}, []string{"models", "sync", "--yes", "--data", "[]"}, "chosen"},
		{[]string{"tools", "export", "--manifest", "--token", "chosen"}, []string{"tools", "export", "--manifest"}, "chosen"},
		{[]string{"skills", "create", "--manifest", "--token=literal"}, []string{"skills", "create", "--manifest", "--token=literal"}, ""},
		{[]string{"api", "/capture", "--header", "X-Example: literal", "--token", "chosen"}, []string{"api", "/capture", "--header", "X-Example: literal"}, "chosen"},
	} {
		opts, remaining, err := parseGlobalFlags(tc.args)
		if err != nil || opts.token != tc.token || !reflect.DeepEqual(remaining, tc.remaining) {
			t.Errorf("args=%q opts=%+v remaining=%q err=%v", tc.args, opts, remaining, err)
		}
	}
}
