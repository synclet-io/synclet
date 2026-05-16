package pipelineexecdocker

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateTempDir creates a temporary directory under rootDir and writes the
// provided files into it. The keys of the files map are filenames (e.g.,
// "config.json"), and the values are file contents.
//
// Pass rootDir == "" to use os.MkdirTemp's default (the OS temp dir). When
// synclet runs inside Docker and dispatches tasks to the host Docker daemon
// via /var/run/docker.sock, rootDir must point at a path that is bind-mounted
// to the same location on the host so the daemon can resolve the bind-mount
// source — see pipelineConfig.DockerTempDirRoot for the wiring.
func CreateTempDir(rootDir string, files map[string][]byte) (string, error) {
	if rootDir != "" {
		if err := os.MkdirAll(rootDir, 0o750); err != nil {
			return "", fmt.Errorf("ensuring temp dir root %s: %w", rootDir, err)
		}
	}

	dir, err := os.MkdirTemp(rootDir, "synclet-docker-*")
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
