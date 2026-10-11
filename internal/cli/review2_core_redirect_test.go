package cli

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReview2CoreR02CustomCredentialRedirectFence(t *testing.T) {
	for _, tc := range []struct {
		name              string
		crossHost, bearer bool
	}{
		{"custom-cross-host", true, false}, {"bearer-cross-host", true, true}, {"custom-cross-port", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			credential := ""
			dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				header := "X-Api-Key"
				if tc.bearer {
					header = "Authorization"
				}
				credential = r.Header.Get(header)
				_, _ = io.WriteString(w, `true`)
			}))
			defer dest.Close()
			redirect := dest.URL
			if tc.crossHost {
				redirect = strings.Replace(redirect, "127.0.0.1", "localhost", 1)
			}
			src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, redirect, http.StatusFound) }))
			defer src.Close()
			app, _ := coreReviewApp(t, `true`)
			app.httpClient = src.Client()
			args := []string{"--base-url", src.URL, "api", "/redirect"}
			if !tc.bearer {
				args = append(args, "--api-key-header", "X-Api-Key")
			}
			if code := app.Run(context.Background(), args); code != 0 {
				t.Fatalf("code=%d", code)
			}
			if credential != "" {
				t.Fatalf("redirect disclosed credential %q", credential)
			}
		})
	}
}

func TestReview2CoreR02SameOriginAndExplicitURLKeepCredential(t *testing.T) {
	var got string
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		got = r.Header.Get("X-Api-Key")
		_, _ = io.WriteString(w, `true`)
	}))
	defer src.Close()
	for _, path := range []string{"/redirect", src.URL + "/final"} {
		app, _ := coreReviewApp(t, `true`)
		app.httpClient = src.Client()
		if code := app.Run(context.Background(), []string{"--base-url", src.URL, "api", path, "--api-key-header", "X-Api-Key"}); code != 0 || got != "synthetic" {
			t.Errorf("path=%s code=%d credential=%q", path, code, got)
		}
	}
}

func TestReview2CoreR02DowngradeAndReturnChainStayRedacted(t *testing.T) {
	var seen []string
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("X-Api-Key"))
		_, _ = io.WriteString(w, `true`)
	}))
	defer final.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer source.Close()
	app, _ := coreReviewApp(t, `true`)
	app.httpClient = source.Client()
	if code := app.Run(context.Background(), []string{"--base-url", source.URL, "api", "/redirect", "--api-key-header", "X-Api-Key"}); code != 0 || len(seen) != 1 || seen[0] != "" {
		t.Fatalf("downgrade code=%d headers=%q", code, seen)
	}
	seen = nil
	var origin string
	middle := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("X-Api-Key"))
		http.Redirect(w, r, origin+"/final", http.StatusFound)
	}))
	defer middle.Close()
	start := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/final" {
			seen = append(seen, r.Header.Get("X-Api-Key"))
			_, _ = io.WriteString(w, `true`)
			return
		}
		http.Redirect(w, r, middle.URL, http.StatusFound)
	}))
	defer start.Close()
	origin = start.URL
	app, _ = coreReviewApp(t, `true`)
	app.httpClient = start.Client()
	if code := app.Run(context.Background(), []string{"--base-url", start.URL, "api", "/redirect", "--api-key-header", "X-Api-Key"}); code != 0 || len(seen) != 2 || seen[0] != "" || seen[1] != "" {
		t.Fatalf("chain code=%d headers=%q", code, seen)
	}
}

func TestReview2CoreR02PreservesClientRedirectPolicy(t *testing.T) {
	calls := 0
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/again", http.StatusFound) }))
	defer src.Close()
	app, _ := coreReviewApp(t, `true`)
	client := src.Client()
	client.CheckRedirect = func(r *http.Request, via []*http.Request) error { calls++; return http.ErrUseLastResponse }
	app.httpClient = client
	if code := app.Run(context.Background(), []string{"--base-url", src.URL, "api", "/redirect", "--api-key-header", "X-Api-Key"}); code == 0 || calls != 1 {
		t.Fatalf("code=%d callbacks=%d", code, calls)
	}
	if client.CheckRedirect == nil {
		t.Fatal("client policy changed")
	}
}
