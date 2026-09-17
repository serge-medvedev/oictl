package cli

import (
	"io"
	"os"
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
