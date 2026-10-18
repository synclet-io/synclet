//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"
)

type dockerModeRunner struct {
	connectionSyncer
	server *syncletServer
}

func setupDockerMode(t *testing.T) modeRunner {
	t.Helper()

	seedTestConnectors(t)

	server := startSyncletServer(t, map[string]string{})
	client := newAPIClient(server.baseURL)
	ctx := context.Background()
	client.RegisterAndLogin(ctx, t)

	t.Cleanup(func() {
		server.Stop()
	})

	return &dockerModeRunner{
		connectionSyncer: connectionSyncer{client: client, timeout: 3 * time.Minute},
		server:           server,
	}
}

func (d *dockerModeRunner) Cleanup(t *testing.T) {
	t.Helper()
	truncateTestTables(t)
}
