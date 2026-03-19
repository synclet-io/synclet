//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"
)

type k8sModeRunner struct {
	server *syncletServer
	client *apiClient
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

	return &k8sModeRunner{server: server, client: client}
}

func (k *k8sModeRunner) RunSync(t *testing.T, ctx context.Context, opts syncOpts) *syncResult {
	t.Helper()

	// 1. CreateSource with opts.SourceConfig.
	sourceID := k.client.CreateSource(ctx, t, t.Name()+"-src", opts.SourceConfig)

	// 2. Discover source schema (required before ConfigureStreams validation).
	k.client.DiscoverSourceSchema(ctx, t, sourceID)

	// 3. CreateDestination with opts.DestConfig.
	destConfig := opts.DestConfig
	if destConfig == nil {
		destConfig = map[string]any{"output_dir": "/tmp/output"}
	}
	destID := k.client.CreateDestination(ctx, t, t.Name()+"-dst", destConfig)

	// 4. CreateConnection.
	connID := k.client.CreateConnection(ctx, t, t.Name(), sourceID, destID)

	// 5. ConfigureStreams.
	k.client.ConfigureStreams(ctx, t, connID, opts.Streams)

	// 6. TriggerSync.
	jobID := k.client.TriggerSync(ctx, t, connID)

	// 7. WaitForJob (poll until terminal) -- K8s pods may take longer to schedule.
	result := k.client.WaitForJob(ctx, t, jobID, 5*time.Minute)

	// 8. Get stream states for incremental verification.
	result.StreamStates, result.StateType = k.client.ListStreamStates(ctx, t, connID)

	return result
}

func (k *k8sModeRunner) Cleanup(t *testing.T) {
	t.Helper()
	truncateTestTables(t)
}
