package pipelineexecdocker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-pnp/go-pnp/logging"
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

// CreateLocalDir creates an empty directory under rootDir for a connector's
// /local mount, where Airbyte connectors write their files. It is
// world-writable because connector images run as users the host does not
// know. rootDir follows the rules of CreateTempDir.
func CreateLocalDir(rootDir string) (string, error) {
	if rootDir != "" {
		if err := os.MkdirAll(rootDir, 0o750); err != nil {
			return "", fmt.Errorf("ensuring temp dir root %s: %w", rootDir, err)
		}
	}

	dir, err := os.MkdirTemp(rootDir, "synclet-local-*")
	if err != nil {
		return "", fmt.Errorf("creating local dir: %w", err)
	}

	//nolint:gosec // G302: connector containers write here under an arbitrary uid.
	if err := os.Chmod(dir, 0o777); err != nil {
		_ = os.RemoveAll(dir)

		return "", fmt.Errorf("making local dir writable: %w", err)
	}

	return dir, nil
}

// localRootMode is the mode of a local root the executor creates: writable by
// a connector of any uid, and sticky like /tmp, so a user cannot remove or
// replace another user's files.
const localRootMode = os.ModeSticky | 0o777

// EnsureLocalRoot makes sure the persistent /local root exists. A missing
// root is created with mode 1777; an existing one is used as it is, its
// permissions and contents untouched.
func EnsureLocalRoot(root string) error {
	_, err := statLocalRoot(root)

	return err
}

// PrepareLocalRoot ensures the local root like EnsureLocalRoot and logs a
// warning when the root is not writable by others, so connector images that
// run as another user cannot write their files.
func PrepareLocalRoot(ctx context.Context, root string, logger *logging.Logger) error {
	mode, err := statLocalRoot(root)
	if err != nil {
		return err
	}

	if mode.Perm()&0o002 == 0 {
		logger.WithFields(map[string]interface{}{
			"path": root,
			"mode": mode.String(),
		}).Warn(ctx, "local root is not writable by others; connectors running as another user cannot write to /local")
	}

	return nil
}

func statLocalRoot(root string) (os.FileMode, error) {
	info, err := os.Stat(root)
	if err == nil {
		if !info.IsDir() {
			return 0, fmt.Errorf("local root %s is not a directory", root)
		}

		return info.Mode(), nil
	}

	if !os.IsNotExist(err) {
		return 0, fmt.Errorf("checking local root %s: %w", root, err)
	}

	//nolint:gosec // G301: connector containers write here under an arbitrary uid.
	if err := os.MkdirAll(root, 0o777); err != nil {
		return 0, fmt.Errorf("creating local root %s: %w", root, err)
	}

	if err := os.Chmod(root, localRootMode); err != nil {
		return 0, fmt.Errorf("making local root %s writable: %w", root, err)
	}

	return localRootMode, nil
}

// acquireLocalDir returns the host directory to mount at /local and the
// function that releases it. With localRoot set it is that root, shared by
// every container and never removed; otherwise it is a new directory under
// tempDirRoot that release removes.
func acquireLocalDir(tempDirRoot, localRoot string) (dir string, release func(), err error) {
	if localRoot != "" {
		if err := EnsureLocalRoot(localRoot); err != nil {
			return "", nil, err
		}

		return localRoot, func() {}, nil
	}

	dir, err = CreateLocalDir(tempDirRoot)
	if err != nil {
		return "", nil, err
	}

	return dir, func() { _ = os.RemoveAll(dir) }, nil
}
