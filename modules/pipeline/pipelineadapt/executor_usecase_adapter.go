package pipelineadapt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesync"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// UseCaseExecutorBackend implements pipelinesync.ExecutorBackend by calling
// use cases directly in-process. Used in standalone mode (per D-15).
type UseCaseExecutorBackend struct {
	claimJobBundle     *pipelinejobs.ClaimJobBundle
	updateJobStatus    *pipelinejobs.UpdateJobStatus
	updateHeartbeat    *pipelinejobs.UpdateHeartbeat
	checkJobCancelled  *pipelinejobs.CheckJobCancelled
	reportCompletion   *pipelinejobs.ReportCompletion
	handleConfigUpdate *pipelinejobs.HandleConfigUpdate
	saveSyncState      *pipelinestate.SaveSyncState
	batchAppendLogs    *pipelinelogs.BatchAppendJobLogs
	getJob             *pipelinejobs.GetJob
	claimTask          *pipelinetasks.ClaimTask
	reportTaskResult   *pipelinetasks.ReportTaskResult
}

// NewUseCaseExecutorBackend creates a new UseCaseExecutorBackend.
func NewUseCaseExecutorBackend(
	claimJobBundle *pipelinejobs.ClaimJobBundle,
	updateJobStatus *pipelinejobs.UpdateJobStatus,
	updateHeartbeat *pipelinejobs.UpdateHeartbeat,
	checkJobCancelled *pipelinejobs.CheckJobCancelled,
	reportCompletion *pipelinejobs.ReportCompletion,
	handleConfigUpdate *pipelinejobs.HandleConfigUpdate,
	saveSyncState *pipelinestate.SaveSyncState,
	batchAppendLogs *pipelinelogs.BatchAppendJobLogs,
	getJob *pipelinejobs.GetJob,
	claimTask *pipelinetasks.ClaimTask,
	reportTaskResult *pipelinetasks.ReportTaskResult,
) *UseCaseExecutorBackend {
	return &UseCaseExecutorBackend{
		claimJobBundle:     claimJobBundle,
		updateJobStatus:    updateJobStatus,
		updateHeartbeat:    updateHeartbeat,
		checkJobCancelled:  checkJobCancelled,
		reportCompletion:   reportCompletion,
		handleConfigUpdate: handleConfigUpdate,
		saveSyncState:      saveSyncState,
		batchAppendLogs:    batchAppendLogs,
		getJob:             getJob,
		claimTask:          claimTask,
		reportTaskResult:   reportTaskResult,
	}
}

// ClaimJob claims the next available job and returns the full executor bundle.
func (a *UseCaseExecutorBackend) ClaimJob(ctx context.Context, workerID string) (*pipelinejobs.ClaimJobBundleResult, error) {
	result, err := a.claimJobBundle.Execute(ctx, pipelinejobs.ClaimJobParams{WorkerID: workerID})
	if err != nil {
		return nil, fmt.Errorf("claiming job bundle: %w", err)
	}

	return result, nil
}

// UpdateJobStatus reports job completion or failure.
func (a *UseCaseExecutorBackend) UpdateJobStatus(ctx context.Context, params pipelinesync.UpdateJobStatusParams) error {
	var syncErr error
	if !params.Success {
		syncErr = fmt.Errorf("%s", params.ErrorMessage)
	}

	stats := &pipelineservice.SyncStats{
		RecordsRead: params.RecordsRead,
		BytesSynced: params.BytesSynced,
		Duration:    time.Duration(params.DurationMs) * time.Millisecond,
	}

	return a.updateJobStatus.Execute(ctx, pipelinejobs.UpdateJobStatusParams{
		ID:        params.JobID,
		SyncErr:   syncErr,
		SyncStats: stats,
	})
}

// Heartbeat updates the heartbeat timestamp and checks for cancellation.
func (a *UseCaseExecutorBackend) Heartbeat(ctx context.Context, jobID uuid.UUID, recordsRead, bytesSynced int64) (*pipelinesync.HeartbeatResult, error) {
	// Update heartbeat timestamp; non-fatal if it fails.
	if err := a.updateHeartbeat.Execute(ctx, pipelinejobs.UpdateHeartbeatParams{ID: jobID}); err != nil {
		slog.Error("usecase backend: heartbeat update failed", "job_id", jobID.String(), "error", err)
	}

	// Check cancellation regardless of heartbeat success.
	cancelled, err := a.checkJobCancelled.Execute(ctx, pipelinejobs.CheckJobCancelledParams{JobID: jobID})
	if err != nil {
		slog.Error("usecase backend: check cancelled failed", "job_id", jobID.String(), "error", err)

		return &pipelinesync.HeartbeatResult{Cancelled: false}, nil
	}

	return &pipelinesync.HeartbeatResult{Cancelled: cancelled}, nil
}

// ReportState saves a sync state message for the given connection.
func (a *UseCaseExecutorBackend) ReportState(ctx context.Context, connectionID, jobID uuid.UUID, stateMsg *protocol.AirbyteStateMessage) error {
	return a.saveSyncState.Execute(ctx, pipelinestate.SaveSyncStateParams{
		ConnectionID: connectionID,
		StateMessage: stateMsg,
	})
}

// ReportCompletion reports job completion with full stats and triggers events.
func (a *UseCaseExecutorBackend) ReportCompletion(ctx context.Context, params pipelinesync.ReportCompletionParams) error {
	return a.reportCompletion.Execute(ctx, pipelinejobs.ReportCompletionParams{
		JobID:        params.JobID,
		ConnectionID: params.ConnectionID,
		Success:      params.Success,
		ErrorMessage: params.ErrorMessage,
		RecordsRead:  params.RecordsRead,
		BytesSynced:  params.BytesSynced,
		DurationMs:   params.DurationMs,
	})
}

// ReportConfigUpdate handles a connector config update (CONTROL message).
func (a *UseCaseExecutorBackend) ReportConfigUpdate(ctx context.Context, connectorType pipelineservice.ConnectorType, connectorID uuid.UUID, config []byte) error {
	return a.handleConfigUpdate.Execute(ctx, pipelinejobs.HandleConfigUpdateParams{
		ConnectorType: connectorType,
		ConnectorID:   connectorID,
		Config:        json.RawMessage(config),
	})
}

// ReportLog appends log lines for a job. Non-fatal: logs error but returns nil.
func (a *UseCaseExecutorBackend) ReportLog(ctx context.Context, jobID uuid.UUID, lines []string) error {
	if err := a.batchAppendLogs.Execute(ctx, pipelinelogs.BatchAppendJobLogsParams{
		JobID:    jobID,
		LogLines: lines,
	}); err != nil {
		slog.Error("usecase backend: report log failed", "job_id", jobID.String(), "error", err)
	}

	return nil
}

// ClaimConnectorTask claims the next pending connector task (D-15).
func (a *UseCaseExecutorBackend) ClaimConnectorTask(ctx context.Context, workerID string) (*pipelinesync.ClaimConnectorTaskResult, error) {
	result, err := a.claimTask.Execute(ctx, workerID)
	if err != nil {
		return nil, fmt.Errorf("claiming connector task: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	return &pipelinesync.ClaimConnectorTaskResult{
		TaskID:      result.TaskID,
		TaskType:    result.TaskType,
		Image:       result.Image,
		Config:      result.Config,
		WorkspaceID: result.WorkspaceID,
	}, nil
}

// ReportConnectorTaskResult reports the result of a connector task (D-18/D-19).
func (a *UseCaseExecutorBackend) ReportConnectorTaskResult(ctx context.Context, params pipelinesync.ReportConnectorTaskResultParams) error {
	return a.reportTaskResult.Execute(ctx, pipelinetasks.ReportTaskResultParams{
		TaskID:       params.TaskID,
		Success:      params.Success,
		ErrorMessage: params.ErrorMessage,
		Result:       params.Result,
	})
}

// IsJobActive checks whether a job is currently running or starting.
func (a *UseCaseExecutorBackend) IsJobActive(ctx context.Context, jobID string) (bool, error) {
	parsed, err := uuid.Parse(jobID)
	if err != nil {
		return false, nil //nolint:nilerr // invalid UUID means job does not exist
	}

	job, err := a.getJob.Execute(ctx, pipelinejobs.GetJobParams{ID: parsed})
	if err != nil {
		return false, nil //nolint:nilerr // not-found is expected, return false for safe orphan cleanup
	}

	return job.Status == pipelineservice.JobStatusRunning || job.Status == pipelineservice.JobStatusStarting, nil
}
