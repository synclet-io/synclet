//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"
)

type dockerModeRunner struct {
	server *syncletServer
	client *apiClient
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

	return &dockerModeRunner{server: server, client: client}
}

func (d *dockerModeRunner) RunSync(t *testing.T, ctx context.Context, opts syncOpts) *syncResult {
	t.Helper()

	// 1. CreateSource with opts.SourceConfig.
	sourceID := d.client.CreateSource(ctx, t, t.Name()+"-src", opts.SourceConfig)

	// 2. Discover source schema (required before ConfigureStreams validation).
	d.client.DiscoverSourceSchema(ctx, t, sourceID)

	// 3. CreateDestination with opts.DestConfig.
	destConfig := opts.DestConfig
	if destConfig == nil {
		destConfig = map[string]any{"output_dir": "/tmp/output"}
	}
	destID := d.client.CreateDestination(ctx, t, t.Name()+"-dst", destConfig)

	// 4. CreateConnection.
	connID := d.client.CreateConnection(ctx, t, t.Name(), sourceID, destID)

	// 5. ConfigureStreams.
	d.client.ConfigureStreams(ctx, t, connID, opts.Streams)

	// 6. TriggerSync.
	jobID := d.client.TriggerSync(ctx, t, connID)

	// 7. WaitForJob (poll until terminal).
	result := d.client.WaitForJob(ctx, t, jobID, 3*time.Minute)

	// 8. Get stream states for incremental verification.
	result.StreamStates, result.StateType = d.client.ListStreamStates(ctx, t, connID)

	return result
}

func (d *dockerModeRunner) Cleanup(t *testing.T) {
	t.Helper()
	truncateTestTables(t)
}
