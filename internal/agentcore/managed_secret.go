package agentcore

import (
	"fmt"
	"os"
	"path/filepath"
)

// Replace a managed secret through a mode-0600 temporary file. Writing an
// existing file directly can retain an older, weaker permission mode.
func writeSecretFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create secret directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".labtether-secret-*")
	if err != nil {
		return fmt.Errorf("create temporary secret file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary secret file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary secret file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary secret file: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replace secret file: %w", err)
	}
	return nil
}
