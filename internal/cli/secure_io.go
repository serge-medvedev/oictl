package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
)

// openSecureOutput restricts an existing destination before truncating it or
// exposing any new bytes. Creation permissions alone do not protect reused files.
func openSecureOutput(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err = file.Chmod(0o600); err == nil {
		err = file.Truncate(0)
	}
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

// redactText decodes JSON string tokens before redacting credentials. Rewriting
// only changed strings preserves numeric precision, duplicate keys and layout.
// Non-JSON diagnostics still receive literal and webhook-URL redaction.
func redactText(text string, secrets ...string) string {
	literal := func(value string) string {
		for _, secret := range secrets {
			if secret = strings.TrimSpace(secret); secret != "" {
				value = strings.ReplaceAll(value, secret, "<redacted>")
			}
		}
		return channelWebhookURLPattern.ReplaceAllString(value, "<redacted>")
	}
	if json.Valid([]byte(text)) {
		var result strings.Builder
		for i := 0; i < len(text); {
			if text[i] != '"' {
				result.WriteByte(text[i])
				i++
				continue
			}
			start := i
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '"' {
					i++
					break
				}
				i++
			}
			raw := text[start:i]
			var decoded string
			_ = json.Unmarshal([]byte(raw), &decoded)
			if sanitized := literal(decoded); sanitized != decoded {
				var encoded bytes.Buffer
				encoder := json.NewEncoder(&encoded)
				encoder.SetEscapeHTML(false)
				_ = encoder.Encode(sanitized)
				result.Write(bytes.TrimSuffix(encoded.Bytes(), []byte{'\n'}))
			} else {
				result.WriteString(raw)
			}
		}
		text = result.String()
	} else {
		text = literal(text)
	}
	text = webhookTokenFieldPattern.ReplaceAllString(text, "$1<redacted>$2")
	return webhookURLFieldPattern.ReplaceAllString(text, "$1<redacted>$2")
}

// checkedWriter remembers ignored fmt write failures at the Run boundary and
// converts nil-error short writes into the standard writer failure.
// It never retries: an acknowledgement failure must not repeat an operation.
type checkedWriter struct {
	writer io.Writer
	err    error
}

func (w *checkedWriter) Write(data []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err := w.writer.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	w.err = err
	return n, err
}

func writeAll(w io.Writer, data []byte) error {
	n, err := w.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

func writeSecureFile(path string, data []byte) error {
	file, err := openSecureOutput(path)
	if err != nil {
		return err
	}
	err = writeAll(file, data)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}
