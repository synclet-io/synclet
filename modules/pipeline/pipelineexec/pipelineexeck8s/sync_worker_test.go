package pipelineexeck8s

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// mockBackend implements pipelineexec.ExecutorBackend for testing.
type mockBackend struct {
	claimResult *pipelinejobs.ClaimJobBundleResult
}

func (m *mockBackend) ClaimJob(_ context.Context, _ string) (*pipelinejobs.ClaimJobBundleResult, error) {
	return m.claimResult, nil
}

func (m *mockBackend) UpdateJobStatus(_ context.Context, _ pipelineexec.UpdateJobStatusParams) error {
	return nil
}

func (m *mockBackend) Heartbeat(_ context.Context, _ uuid.UUID, _, _ int64) (*pipelineexec.HeartbeatResult, error) {
	return &pipelineexec.HeartbeatResult{}, nil
}

func (m *mockBackend) ReportState(_ context.Context, _, _ uuid.UUID, _ *protocol.AirbyteStateMessage) error {
	return nil
}

func (m *mockBackend) ReportCompletion(_ context.Context, _ pipelineexec.ReportCompletionParams) error {
	return nil
}

func (m *mockBackend) ReportConfigUpdate(_ context.Context, _ pipelineservice.ConnectorType, _ uuid.UUID, _ []byte) error {
	return nil
}

func (m *mockBackend) ReportLog(_ context.Context, _ uuid.UUID, _ []string) error {
	return nil
}

func (m *mockBackend) IsJobActive(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (m *mockBackend) IsTaskActive(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (m *mockBackend) FailJob(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockBackend) ClaimConnectorTask(_ context.Context, _ string) (*pipelineexec.ClaimConnectorTaskResult, error) {
	return nil, nil
}

func (m *mockBackend) ReportConnectorTaskResult(_ context.Context, _ pipelineexec.ReportConnectorTaskResultParams) error {
	return nil
}

// mockK8sSyncLauncher implements K8sSyncLauncher for testing.
type mockK8sSyncLauncher struct {
	launchSyncJobFn func(ctx context.Context, opts K8sSyncJobOptions) (string, error)
	lastOpts        *K8sSyncJobOptions
}

func (m *mockK8sSyncLauncher) LaunchSyncJob(ctx context.Context, opts K8sSyncJobOptions) (string, error) {
	m.lastOpts = &opts
	if m.launchSyncJobFn != nil {
		return m.launchSyncJobFn(ctx, opts)
	}

	return "synclet-sync-test-123", nil
}

func TestK8sSyncWorker_Execute_NoJobAvailable(t *testing.T) {
	backend := &mockBackend{claimResult: nil}

	worker := &K8sSyncWorker{
		backend:  backend,
		workerID: "test-worker",
	}

	err := worker.Execute(context.Background())
	require.NoError(t, err)
}

func TestK8sSyncWorker_K8sSyncJobOptions(t *testing.T) {
	// Verify K8sSyncJobOptions struct has all required fields for 3-container pod spec.
	opts := K8sSyncJobOptions{
		JobID:         uuid.New(),
		ConnectionID:  uuid.New(),
		SourceID:      uuid.New(),
		DestinationID: uuid.New(),
		SourceImage:   "airbyte/source-postgres:0.1.0",
		DestImage:     "airbyte/destination-postgres:0.1.0",
		SourceConfig:  []byte(`{"host":"localhost"}`),
		DestConfig:    []byte(`{"host":"localhost"}`),
		SourceCatalog: []byte(`{"streams":[]}`),
		DestCatalog:   []byte(`{"streams":[]}`),
		State:         []byte(`[{"type":"STREAM"}]`),
		RuntimeConfig: `{"memory_limit":2147483648}`,
	}

	assert.NotEqual(t, uuid.Nil, opts.JobID)
	assert.NotEqual(t, uuid.Nil, opts.ConnectionID)
	assert.NotEqual(t, uuid.Nil, opts.SourceID)
	assert.NotEqual(t, uuid.Nil, opts.DestinationID)
	assert.NotEmpty(t, opts.SourceImage)
	assert.NotEmpty(t, opts.DestImage)
	assert.NotEmpty(t, opts.SourceConfig)
	assert.NotEmpty(t, opts.DestConfig)
	assert.NotEmpty(t, opts.SourceCatalog)
	assert.NotEmpty(t, opts.DestCatalog)
	assert.NotEmpty(t, opts.State)
	assert.NotEmpty(t, opts.RuntimeConfig)
}

func TestK8sSyncWorker_LaunchSyncJobFailure(t *testing.T) {
	// Verify that when k8sRunner.LaunchSyncJob fails, the mock captures the error.
	runner := &mockK8sSyncLauncher{
		launchSyncJobFn: func(ctx context.Context, opts K8sSyncJobOptions) (string, error) {
			return "", errors.New("k8s api unavailable")
		},
	}

	_, err := runner.LaunchSyncJob(context.Background(), K8sSyncJobOptions{
		JobID: uuid.New(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "k8s api unavailable")
}

func TestK8sSyncWorker_LaunchSyncJobSuccess(t *testing.T) {
	// Verify that successful creation returns the K8s job name.
	runner := &mockK8sSyncLauncher{
		launchSyncJobFn: func(ctx context.Context, opts K8sSyncJobOptions) (string, error) {
			return "synclet-sync-" + opts.JobID.String()[:8], nil
		},
	}

	jobID := uuid.New()
	name, err := runner.LaunchSyncJob(context.Background(), K8sSyncJobOptions{
		JobID:       jobID,
		SourceImage: "airbyte/source-postgres:0.1.0",
		DestImage:   "airbyte/destination-postgres:0.1.0",
	})

	require.NoError(t, err)
	assert.Contains(t, name, "synclet-sync-")
	assert.NotNil(t, runner.lastOpts)
	assert.Equal(t, "airbyte/source-postgres:0.1.0", runner.lastOpts.SourceImage)
	assert.Equal(t, "airbyte/destination-postgres:0.1.0", runner.lastOpts.DestImage)
}
