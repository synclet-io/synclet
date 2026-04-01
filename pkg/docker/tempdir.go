package docker

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateTempDir creates a temporary directory and writes the provided files into it.
// The keys of the files map are filenames (e.g., "config.json"), and the values are file contents.
// Returns the path to the created directory.
func CreateTempDir(files map[string][]byte) (string, error) {
	dir, err := os.MkdirTemp("", "synclet-docker-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o600); err != nil {
			// Clean up on error.
			_ = os.RemoveAll(dir)

			return "", fmt.Errorf("writing temp file %s: %w", name, err)
		}
	}

	return dir, nil
}

// CleanupTempDir removes the temporary directory and all its contents.
func CleanupTempDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("removing temp dir %s: %w", path, err)
	}

	return nil
}
