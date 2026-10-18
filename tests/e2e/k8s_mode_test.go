//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"
)

type k8sModeRunner struct {
	connectionSyncer
	server *syncletServer
}

func setupK8sMode(t *testing.T) modeRunner {
	t.Helper()
	if os.Getenv("E2E_K8S") != "1" {
		t.Skip("K8s mode requires E2E_K8S=1")
	}
	if kindKubeconfig == "" {
		t.Fatal("kind cluster not initialized")
	}

	seedTestConnectors(t)

	// Start synclet server with K8s executor pointing at kind cluster.
	server := startSyncletServer(t, map[string]string{
		"KUBECONFIG": kindKubeconfig,
	})
	client := newAPIClient(server.baseURL)
	ctx := context.Background()
	client.RegisterAndLogin(ctx, t)

	t.Cleanup(func() {
		server.Stop()
	})

	return &k8sModeRunner{
		connectionSyncer: connectionSyncer{client: client, timeout: 5 * time.Minute},
		server:           server,
	}
}

func (k *k8sModeRunner) Cleanup(t *testing.T) {
	t.Helper()
	truncateTestTables(t)
}
