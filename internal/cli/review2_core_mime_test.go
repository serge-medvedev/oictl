package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReview2CoreR20MultipartMediaTypes(t *testing.T) {
	for _, tc := range []struct {
		name, contentType string
		data              []byte
	}{
		{"sound.wav", "audio/", []byte("RIFF\x04\x00\x00\x00WAVEfmt ")},
		{"sound.mp3", "audio/", []byte("ID3\x04\x00\x00\x00\x00\x00\x00")},
		{"image.png", "image/png", []byte("\x89PNG\r\n\x1a\nimage")},
		{"movie.mp4", "video/mp4", []byte("\x00\x00\x00\x18ftypmp42fixture")},
		{"document.pdf", "application/pdf", []byte("%PDF-1.7\nfixture")},
		{"notes.txt", "text/plain", []byte("plain text\n")},
		{"unknown.unknownreview2", "application/octet-stream", []byte{0, 1, 2, 3, 4, 5}},
		{"detected.unknownreview2", "image/png", []byte("\x89PNG\r\n\x1a\nimage")},
		{"quoted\"name.txt", "text/plain", []byte("literal filename")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tc.name)
			if err := os.WriteFile(path, tc.data, 0600); err != nil {
				t.Fatal(err)
			}
			app, calls := coreReviewApp(t, `true`)
			app.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				*calls++
				reader, err := r.MultipartReader()
				if err != nil {
					t.Fatal(err)
				}
				part, err := reader.NextPart()
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(part.Header.Get("Content-Type"), tc.contentType) || part.FormName() != "file" || part.FileName() != tc.name {
					t.Errorf("part=%v filename=%q", part.Header, part.FileName())
				}
				got, err := io.ReadAll(part)
				if err != nil || !bytes.Equal(got, tc.data) {
					t.Errorf("payload=%q err=%v", got, err)
				}
				meta, err := reader.NextPart()
				if err != nil {
					t.Fatal(err)
				}
				got, err = io.ReadAll(meta)
				if err != nil || meta.FormName() != "metadata" || string(got) != `{"label":"example"}` {
					t.Errorf("metadata=%q err=%v", got, err)
				}
				if _, err := reader.NextPart(); err != io.EOF {
					t.Errorf("extra part: %v", err)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`true`)), Header: make(http.Header)}, nil
			})
			if code := app.Run(context.Background(), []string{"files", "upload", path, "--metadata", `{"label":"example"}`}); code != 0 || *calls != 1 {
				t.Errorf("code=%d calls=%d", code, *calls)
			}
		})
	}
}
