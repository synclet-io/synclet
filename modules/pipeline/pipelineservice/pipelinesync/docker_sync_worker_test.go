package pipelinesync

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// --- Test doubles ---

// mockBackend implements ExecutorBackend for testing.
type mockBackend struct {
	claimResult      *pipelinejobs.ClaimJobBundleResult
	claimErr         error
	updateErr        error
	heartbeatResult  *HeartbeatResult
	heartbeatErr     error
	updateCalled     atomic.Int32
	lastUpdateParams UpdateJobStatusParams
}

func (m *mockBackend) ClaimJob(_ context.Context, _ string) (*pipelinejobs.ClaimJobBundleResult, error) {
	return m.claimResult, m.claimErr
}

func (m *mockBackend) UpdateJobStatus(_ context.Context, params UpdateJobStatusParams) error {
	m.updateCalled.Add(1)
	m.lastUpdateParams = params

	return m.updateErr
}

func (m *mockBackend) Heartbeat(_ context.Context, _ uuid.UUID, _, _ int64) (*HeartbeatResult, error) {
	if m.heartbeatResult != nil {
		return m.heartbeatResult, m.heartbeatErr
	}

	return &HeartbeatResult{}, m.heartbeatErr
}

func (m *mockBackend) ReportState(_ context.Context, _, _ uuid.UUID, _ *protocol.AirbyteStateMessage) error {
	return nil
}

func (m *mockBackend) ReportCompletion(_ context.Context, _ ReportCompletionParams) error {
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

func (m *mockBackend) ClaimConnectorTask(_ context.Context, _ string) (*ClaimConnectorTaskResult, error) {
	return nil, nil
}

func (m *mockBackend) ReportConnectorTaskResult(_ context.Context, _ ReportConnectorTaskResultParams) error {
	return nil
}

type mockExecutor struct {
	stats      *pipelineservice.SyncStats
	err        error
	called     atomic.Int32
	lastBundle *SyncBundle
}

func (m *mockExecutor) Execute(_ context.Context, bundle *SyncBundle) (*pipelineservice.SyncStats, error) {
	m.called.Add(1)
	m.lastBundle = bundle

	return m.stats, m.err
}

// --- Tests ---

func TestDockerSyncWorker_Execute_ClaimsJobAndSpawnsGoroutine(t *testing.T) {
	jobID := uuid.New()
	connID := uuid.New()
	wsID := uuid.New()

	backend := &mockBackend{
		claimResult: &pipelinejobs.ClaimJobBundleResult{
			Job: &pipelineservice.Job{
				ID:           jobID,
				ConnectionID: connID,
				JobType:      pipelineservice.JobTypeSync,
				Status:       pipelineservice.JobStatusStarting,
			},
			ConnectionID:      connID,
			WorkspaceID:       wsID,
			ConfiguredCatalog: []byte(`{"streams":[]}`),
		},
	}
	executor := &mockExecutor{stats: &pipelineservice.SyncStats{RecordsRead: 10}}

	manager := NewSyncWorkerManager(context.Background(), nil)

	worker := &DockerSyncWorker{
		backend:         backend,
		executor:        executor,
		manager:         manager,
		maxSyncDuration: 10 * time.Minute,
		semaphore:       make(chan struct{}, 10),
		workerID:        "test-worker",
	}

	err := worker.Execute(context.Background())
	require.NoError(t, err)

	// Wait for goroutine to finish.
	_ = manager.Shutdown(5 * time.Second)

	assert.Equal(t, int32(1), executor.called.Load())
	assert.Equal(t, jobID, executor.lastBundle.Job.ID)
	assert.Equal(t, connID, executor.lastBundle.ConnectionID)
	assert.Equal(t, wsID, executor.lastBundle.WorkspaceID)

	// Verify UpdateJobStatus was called (success).
	assert.Equal(t, int32(1), backend.updateCalled.Load())
	assert.True(t, backend.lastUpdateParams.Success)
}

func TestDockerSyncWorker_Execute_ReturnsNilWhenNoJobs(t *testing.T) {
	backend := &mockBackend{claimResult: nil}

	worker := &DockerSyncWorker{
		backend:   backend,
		semaphore: make(chan struct{}, 10),
		workerID:  "test-worker",
	}

	err := worker.Execute(context.Background())
	require.NoError(t, err)
}

func TestDockerSyncWorker_Execute_ReturnsNilWhenSemaphoreFull(t *testing.T) {
	backend := &mockBackend{
		claimResult: &pipelinejobs.ClaimJobBundleResult{
			Job: &pipelineservice.Job{ID: uuid.New()},
		},
	}

	// Create semaphore with capacity 1 and fill it.
	sem := make(chan struct{}, 1)
	sem <- struct{}{}

	worker := &DockerSyncWorker{
		backend:   backend,
		semaphore: sem,
		workerID:  "test-worker",
	}

	err := worker.Execute(context.Background())
	require.NoError(t, err)
	// ClaimJob should NOT have been called since semaphore is full.
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
