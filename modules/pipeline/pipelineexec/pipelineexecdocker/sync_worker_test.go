package pipelineexecdocker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/jobutil"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// --- Test doubles ---

// mockBackend implements ExecutorBackend for testing.
type mockBackend struct {
	claimResult      *pipelineexec.ClaimJobResult
	claimErr         error
	updateErr        error
	heartbeatResult  *pipelineexec.HeartbeatResult
	heartbeatErr     error
	updateCalled     atomic.Int32
	lastUpdateParams pipelineexec.UpdateJobStatusParams
	completionCalled atomic.Int32
	lastCompletion   pipelineexec.ReportCompletionParams
	markRunningCalls []uuid.UUID
}

func (m *mockBackend) ClaimJob(_ context.Context, _ string) (*pipelineexec.ClaimJobResult, error) {
	return m.claimResult, m.claimErr
}

func (m *mockBackend) UpdateJobStatus(_ context.Context, params pipelineexec.UpdateJobStatusParams) error {
	m.updateCalled.Add(1)
	m.lastUpdateParams = params

	return m.updateErr
}

func (m *mockBackend) MarkJobRunning(_ context.Context, jobID uuid.UUID) error {
	m.markRunningCalls = append(m.markRunningCalls, jobID)

	return nil
}

func (m *mockBackend) SetK8sJobName(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockBackend) ExecutorHeartbeat(_ context.Context, _ pipelineexec.ExecutorInfo) error {
	return nil
}

func (m *mockBackend) Heartbeat(_ context.Context, _ uuid.UUID, _, _ int64) (*pipelineexec.HeartbeatResult, error) {
	if m.heartbeatResult != nil {
		return m.heartbeatResult, m.heartbeatErr
	}

	return &pipelineexec.HeartbeatResult{}, m.heartbeatErr
}

func (m *mockBackend) ReportState(_ context.Context, _, _ uuid.UUID, _ *protocol.AirbyteStateMessage) error {
	return nil
}

func (m *mockBackend) ReportCompletion(_ context.Context, params pipelineexec.ReportCompletionParams) error {
	m.completionCalled.Add(1)
	m.lastCompletion = params

	return m.updateErr
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

type mockExecutor struct {
	stats      *pipelineservice.SyncStats
	err        error
	waitCtx    bool
	starts     bool
	onRunning  func()
	called     atomic.Int32
	lastBundle *pipelineexec.SyncBundle
}

func (m *mockExecutor) Execute(ctx context.Context, bundle *pipelineexec.SyncBundle, onStarted func()) (*pipelineservice.SyncStats, error) {
	m.called.Add(1)
	m.lastBundle = bundle

	if m.starts {
		onStarted()

		if m.onRunning != nil {
			m.onRunning()
		}
	}

	if m.waitCtx {
		<-ctx.Done()

		return nil, ctx.Err()
	}

	return m.stats, m.err
}

func newTestWorker(backend *mockBackend, executor *mockExecutor) *DockerSyncWorker {
	return &DockerSyncWorker{
		backend:           backend,
		executor:          executor,
		maxSyncDuration:   10 * time.Minute,
		heartbeatInterval: 10 * time.Millisecond,
		semaphore:         make(chan struct{}, 10),
		workerID:          "test-worker",
	}
}

func newClaimResult() *pipelineexec.ClaimJobResult {
	return &pipelineexec.ClaimJobResult{
		Job: &pipelineservice.Job{
			ID:      uuid.New(),
			JobType: pipelineservice.JobTypeSync,
			Status:  pipelineservice.JobStatusStarting,
		},
		ConfiguredCatalog: []byte(`{"streams":[]}`),
	}
}

// --- Tests ---

func TestDockerSyncWorker_Execute_ClaimsJobAndSpawnsGoroutine(t *testing.T) {
	jobID := uuid.New()
	connID := uuid.New()
	wsID := uuid.New()

	backend := &mockBackend{
		claimResult: &pipelineexec.ClaimJobResult{
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

	manager := jobutil.NewWorkerManager(nil)

	worker := &DockerSyncWorker{
		backend:           backend,
		executor:          executor,
		manager:           manager,
		maxSyncDuration:   10 * time.Minute,
		heartbeatInterval: time.Second,
		semaphore:         make(chan struct{}, 10),
		workerID:          "test-worker",
	}

	err := worker.Execute(context.Background())
	require.NoError(t, err)

	// Wait for goroutine to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = manager.Close(ctx)

	assert.Equal(t, int32(1), executor.called.Load())
	assert.Equal(t, jobID, executor.lastBundle.Job.ID)
	assert.Equal(t, connID, executor.lastBundle.ConnectionID)
	assert.Equal(t, wsID, executor.lastBundle.WorkspaceID)

	// Verify UpdateJobStatus was called (success).
	assert.Equal(t, int32(1), backend.completionCalled.Load())
	assert.True(t, backend.lastCompletion.Success)
}

func TestDockerSyncWorker_Execute_ReportsExecutorFailure(t *testing.T) {
	backend := &mockBackend{claimResult: newClaimResult()}
	executor := &mockExecutor{err: errors.New("source failed: connection refused")}
	worker := newTestWorker(backend, executor)

	worker.executeJob(context.Background(), backend.claimResult)

	require.Equal(t, int32(1), backend.completionCalled.Load())
	assert.False(t, backend.lastCompletion.Success)
	assert.Equal(t, backend.claimResult.Job.ID, backend.lastCompletion.JobID)
	assert.Equal(t, "source failed: connection refused", backend.lastCompletion.ErrorMessage)
}

func TestDockerSyncWorker_Execute_ReportsConnectorExitCodes(t *testing.T) {
	backend := &mockBackend{claimResult: newClaimResult()}
	backend.claimResult.ConnectionID = uuid.New()
	executor := &mockExecutor{err: fmt.Errorf("sync failed: %w", errors.Join(
		fmt.Errorf("reading source: %w", &pipelineservice.ExitCodeError{ExitCode: 1, Role: "source"}),
		fmt.Errorf("writing destination: %w", &pipelineservice.ExitCodeError{ExitCode: 137, Role: "destination", Stderr: "killed"}),
	))}
	worker := newTestWorker(backend, executor)

	worker.executeJob(context.Background(), backend.claimResult)

	require.Equal(t, int32(1), backend.completionCalled.Load())
	assert.False(t, backend.lastCompletion.Success)
	assert.Equal(t, backend.claimResult.ConnectionID, backend.lastCompletion.ConnectionID)
	assert.Equal(t, 1, backend.lastCompletion.SourceExitCode)
	assert.Equal(t, 137, backend.lastCompletion.DestExitCode)
	assert.Equal(t, executor.err.Error(), backend.lastCompletion.ErrorMessage)
}

func TestDockerSyncWorker_Execute_SkipsStatusUpdateWhenCancelledViaHeartbeat(t *testing.T) {
	backend := &mockBackend{
		claimResult:     newClaimResult(),
		heartbeatResult: &pipelineexec.HeartbeatResult{Cancelled: true},
	}
	executor := &mockExecutor{waitCtx: true}
	worker := newTestWorker(backend, executor)

	worker.executeJob(context.Background(), backend.claimResult)

	assert.Equal(t, int32(0), backend.updateCalled.Load())
	assert.Equal(t, int32(0), backend.completionCalled.Load())
}

func TestDockerSyncWorker_Execute_MarksJobRunningOnceConnectorsStart(t *testing.T) {
	backend := &mockBackend{claimResult: newClaimResult()}
	executor := &mockExecutor{starts: true, stats: &pipelineservice.SyncStats{}}

	var runningWhileExecuting []uuid.UUID

	executor.onRunning = func() {
		runningWhileExecuting = append(runningWhileExecuting, backend.markRunningCalls...)
	}

	worker := newTestWorker(backend, executor)

	worker.executeJob(context.Background(), backend.claimResult)

	assert.Equal(t, []uuid.UUID{backend.claimResult.Job.ID}, runningWhileExecuting)
	assert.Equal(t, []uuid.UUID{backend.claimResult.Job.ID}, backend.markRunningCalls)
	assert.True(t, backend.lastCompletion.Success)
}

func TestDockerSyncWorker_Execute_DoesNotMarkRunningWhenConnectorsFailToStart(t *testing.T) {
	backend := &mockBackend{claimResult: newClaimResult()}
	executor := &mockExecutor{err: errors.New("starting source read: image not found")}
	worker := newTestWorker(backend, executor)

	worker.executeJob(context.Background(), backend.claimResult)

	assert.Empty(t, backend.markRunningCalls)
	assert.False(t, backend.lastCompletion.Success)
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
		claimResult: &pipelineexec.ClaimJobResult{
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
