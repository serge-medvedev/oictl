package cli

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestReview2CoreR24InformationalAndProfileOutputFailures(t *testing.T) {
	for _, args := range [][]string{nil, {"version"}, {"--version"}, {"--help"}, {"models"}, {"models", "--help"}, {"models", "create", "--help"}, {"config", "terminal-servers", "--help"}, {"profiles", "set", "example", "--base-url", "http://example.invalid"}, {"profiles", "delete", "example"}} {
		for _, short := range []bool{false, true} {
			t.Run(strings.Join(args, "/")+"/short="+stringValue(short), func(t *testing.T) {
				app, calls := coreReviewApp(t, `true`)
				original := coreReviewFailWriter{short: short}
				app.out = original
				var diagnostics strings.Builder
				app.err = &diagnostics
				if len(args) > 1 && args[0] == "profiles" && args[1] == "delete" {
					if err := app.saveProfile("example", profileConfig{BaseURL: "http://example.invalid"}); err != nil {
						t.Fatal(err)
					}
				}
				if code := app.Run(context.Background(), args); code == 0 || *calls != 0 || !strings.Contains(diagnostics.String(), "write output:") {
					t.Errorf("code=%d calls=%d stderr=%q", code, *calls, diagnostics.String())
				}
				if app.out != original {
					t.Fatal("Run did not restore output writer")
				}
				if len(args) > 1 && args[0] == "profiles" && args[1] == "set" {
					p, err := app.loadProfile("example")
					if err != nil || p.BaseURL != "http://example.invalid" {
						t.Errorf("persisted profile=%+v err=%v", p, err)
					}
				}
				if len(args) > 1 && args[0] == "profiles" && args[1] == "delete" {
					path, err := app.profilePath("example")
					if err != nil {
						t.Fatal(err)
					}
					if _, err := os.Stat(path); !os.IsNotExist(err) {
						t.Errorf("profile deletion not persisted: %v", err)
					}
				}
				app.out = io.Discard
				if code := app.Run(context.Background(), []string{"version"}); code != 0 {
					t.Errorf("previous failure poisoned later Run: %d", code)
				}
			})
		}
	}
}

func TestReview2CoreR24NestedHelpDoesNotDuplicateDiagnostic(t *testing.T) {
	app, _ := coreReviewApp(t, `true`)
	app.out = coreReviewFailWriter{}
	var diagnostics strings.Builder
	app.err = &diagnostics
	if code := app.Run(context.Background(), []string{"models", "--help"}); code == 0 || strings.Count(diagnostics.String(), "write output:") != 1 {
		t.Errorf("code=%d stderr=%q", code, diagnostics.String())
	}
}

func TestReview2CoreR24FullDeviceAndStderrFailure(t *testing.T) {
	full, err := os.OpenFile("/dev/full", os.O_WRONLY, 0)
	if err != nil {
		t.Skip(err)
	}
	defer full.Close()
	app, _ := coreReviewApp(t, `true`)
	app.out = full
	if code := app.Run(context.Background(), []string{"version"}); code == 0 {
		t.Fatal("/dev/full reported success")
	}
	app, _ = coreReviewApp(t, `true`)
	app.err = coreReviewFailWriter{short: true}
	app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { return nil, io.ErrClosedPipe })
	if code := app.Run(context.Background(), []string{"api", "/error"}); code == 0 {
		t.Fatal("failed error output reported success")
	}
}
