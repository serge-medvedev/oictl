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

func TestReview2SurfacesR19DecodedDiagnosticRedaction(t *testing.T) {
	secret := "core-synthetic-secret"
	for _, text := range []string{
		`{"detail":"\u0063ore-synthetic-secret","nested":["prefix core-synthetic-secret suffix"],"count":9007199254740993}`,
		`{"\u0063ore-synthetic-secret":"key also contains credential"}`,
		`"\u0063ore-synthetic-secret"`,
		`plain core-synthetic-secret fallback`,
	} {
		got := redactSecrets(text, secret)
		var decoded any
		if json.Valid([]byte(text)) {
			decoder := json.NewDecoder(strings.NewReader(got))
			decoder.UseNumber()
			if err := decoder.Decode(&decoded); err != nil {
				t.Fatalf("invalid redacted JSON %q: %v", got, err)
			}
			body, _ := json.Marshal(decoded)
			if strings.Contains(string(body), secret) || !strings.Contains(got, "redacted") {
				t.Fatalf("leaked decoded credential: %s", got)
			}
			if strings.Contains(text, "9007199254740993") && !strings.Contains(got, "9007199254740993") {
				t.Fatalf("number changed: %s", got)
			}
		} else if strings.Contains(got, secret) || !strings.Contains(got, "<redacted>") {
			t.Fatalf("fallback leaked: %s", got)
		}
	}
}

func TestReview2SurfacesR19StreamTransportAndCloseErrors(t *testing.T) {
	for _, stage := range []string{"transport", "close"} {
		t.Run(stage, func(t *testing.T) {
			app, _, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": "https://example.invalid", "OPEN_WEBUI_API_KEY": "core-synthetic-secret"})
			app.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if stage == "transport" {
					return nil, errors.New("transport core-synthetic-secret")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: review2SurfaceSecretClose{strings.NewReader("successful raw bytes")}}, nil
			})}
			if code := app.executeStream(context.Background(), globalOptions{}, requestSpec{path: "/stream", authRequired: true}); code == 0 || strings.Contains(errOut.String(), "core-synthetic-secret") || !strings.Contains(errOut.String(), "<redacted>") {
				t.Fatalf("code=%d stderr=%s", code, errOut)
			}
		})
	}
}

type review2SurfaceSecretClose struct{ io.Reader }

func (review2SurfaceSecretClose) Close() error { return errors.New("close core-synthetic-secret") }

func TestReview2SurfacesR19StreamStatusAndBody(t *testing.T) {
	app, out, errOut := newTestApp(map[string]string{"OPEN_WEBUI_URL": "https://example.invalid", "OPEN_WEBUI_API_KEY": "core-synthetic-secret"})
	app.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 401, Status: "401 core-synthetic-secret", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"detail":"\u0063ore-synthetic-secret"}`))}, nil
	})}
	if code := app.executeStream(context.Background(), globalOptions{}, requestSpec{path: "/stream", authRequired: true}); code == 0 || out.Len() != 0 {
		t.Fatalf("code=%d stdout=%s", code, out)
	}
	if strings.Contains(errOut.String(), "core-synthetic-secret") || strings.Contains(errOut.String(), `\u0063ore-synthetic-secret`) || !strings.Contains(errOut.String(), "401 Unauthorized") || !strings.Contains(errOut.String(), "redacted") {
		t.Fatalf("unsafe diagnostic=%s", errOut)
	}
}
