package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestReview2CoreR19StatusAndDecodedJSONRedaction(t *testing.T) {
	for _, body := range []string{
		`{"detail":"\u0073ynthetic","nested":["prefix synthetic suffix"],"n":9007199254740993}`,
		`"\u0073ynthetic"`, `synthetic plain failure`,
	} {
		app, _ := coreReviewApp(t, `true`)
		var diagnostics strings.Builder
		app.err = &diagnostics
		app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 401, Status: "401 synthetic", Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})
		_, err := app.doRequest(context.Background(), globalOptions{}, requestSpec{path: "/error", authRequired: true})
		if err == nil || strings.Contains(err.Error(), "synthetic") || strings.Contains(diagnostics.String(), "synthetic") || strings.Contains(diagnostics.String(), `\u0073ynthetic`) {
			t.Errorf("body=%q err=%v stderr=%q", body, err, diagnostics.String())
		}
		var status httpStatusError
		if !errors.As(err, &status) || status.status != "401 Unauthorized" || !strings.Contains(status.body, "redacted") {
			t.Errorf("status=%+v err=%v", status, err)
		}
		if strings.Contains(body, "9007199254740993") && !strings.Contains(status.body, "9007199254740993") {
			t.Errorf("numeric diagnostic changed: %s", status.body)
		}
	}
}

func TestReview2CoreR19DecodedStringsPreserveJSONShape(t *testing.T) {
	for _, secret := range []string{"synthetic", `synthetic"quote`, `synthetic\slash`, "synthetic\nline"} {
		encoded, _ := json.Marshal(secret)
		body := ` {"duplicate":` + string(encoded) + `,"duplicate":[` + string(encoded) + `],"number":9007199254740993,"bool":true} `
		got := redactText(body, secret)
		if !json.Valid([]byte(got)) || strings.Contains(got, string(encoded)) || !strings.Contains(got, "9007199254740993") || strings.Count(got, `"duplicate"`) != 2 {
			t.Errorf("secret=%q invalid or lossy diagnostic=%q", secret, got)
		}
	}
}

func TestReview2CoreR19NumericCredentialDoesNotRewriteJSONNumbers(t *testing.T) {
	got := redactText(`{"detail":"123","number":123,"decimal":1.23}`, "123")
	if !json.Valid([]byte(got)) || !strings.Contains(got, `"number":123`) || !strings.Contains(got, `"decimal":1.23`) || strings.Contains(got, `"detail":"123"`) {
		t.Errorf("numeric values corrupted: %q", got)
	}
}

func TestReview2CoreR19CanonicalStatusPreservesTypedError(t *testing.T) {
	app, _ := coreReviewApp(t, `true`)
	app.getenv = func(key string) string {
		if key == "OPEN_WEBUI_URL" {
			return "http://example.invalid"
		}
		if key == "OPEN_WEBUI_API_KEY" {
			return "Unauthorized"
		}
		return ""
	}
	app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 401, Status: "401 untrusted", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"detail":"Unauthorized"}`))}, nil
	})
	_, err := app.doRequest(context.Background(), globalOptions{}, requestSpec{path: "/error", quietErrors: true})
	var status httpStatusError
	if !errors.As(err, &status) || status.status != "401 Unauthorized" || strings.Contains(status.body, "Unauthorized") {
		t.Errorf("trusted canonical status lost type or body leaked: %v", err)
	}
}

func TestReview2CoreR19UnknownStatusIsNumeric(t *testing.T) {
	app, _ := coreReviewApp(t, `true`)
	app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 599, Status: "599 synthetic", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"detail":"synthetic"}`))}, nil
	})
	_, err := app.doRequest(context.Background(), globalOptions{}, requestSpec{path: "/error", quietErrors: true})
	var status httpStatusError
	if !errors.As(err, &status) || status.status != "599" || strings.Contains(status.Error(), "synthetic") {
		t.Errorf("err=%v", err)
	}
}

func TestReview2CoreR19TransportDiagnosticAndRawSuccess(t *testing.T) {
	app, _ := coreReviewApp(t, `true`)
	var diagnostics strings.Builder
	app.err = &diagnostics
	app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { return nil, errors.New("synthetic transport failure") })
	if code := app.Run(context.Background(), []string{"api", "/error"}); code == 0 || strings.Contains(diagnostics.String(), "synthetic") {
		t.Errorf("code=%d stderr=%q", code, diagnostics.String())
	}
	body := " {\"detail\":\"\\u0073ynthetic\",\"n\":9007199254740993}\n"
	app, _ = coreReviewApp(t, body)
	var output strings.Builder
	app.out = &output
	if code := app.Run(context.Background(), []string{"api", "/ok"}); code != 0 || output.String() != body {
		t.Errorf("code=%d raw=%q want=%q", code, output.String(), body)
	}
}
