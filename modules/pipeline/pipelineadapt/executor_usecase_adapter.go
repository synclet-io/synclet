package pipelineadapt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// UseCaseExecutorBackend implements pipelineexec.ExecutorBackend by calling
// use cases directly in-process. Used in standalone mode (per D-15).
type UseCaseExecutorBackend struct {
	claimJobBundle     *pipelinejobs.ClaimJobBundle
	updateJobStatus    *pipelinejobs.UpdateJobStatus
	markJobRunning     *pipelinejobs.MarkJobRunning
	updateHeartbeat    *pipelinejobs.UpdateHeartbeat
	checkJobCancelled  *pipelinejobs.CheckJobCancelled
	reportCompletion   *pipelinejobs.ReportCompletion
	handleConfigUpdate *pipelinejobs.HandleConfigUpdate
	saveSyncState      *pipelinestate.SaveSyncState
	batchAppendLogs    *pipelinelogs.BatchAppendJobLogs
	getJob             *pipelinejobs.GetJob
	claimTask          *pipelinetasks.ClaimTask
	reportTaskResult   *pipelinetasks.ReportTaskResult
	isTaskActive       *pipelinejobs.IsTaskActive
	setK8sJobName      *pipelinejobs.SetK8sJobName
	registerExecutor   *pipelinejobs.RegisterExecutor
}

// NewUseCaseExecutorBackend creates a new UseCaseExecutorBackend.
func NewUseCaseExecutorBackend(
	claimJobBundle *pipelinejobs.ClaimJobBundle,
	updateJobStatus *pipelinejobs.UpdateJobStatus,
	markJobRunning *pipelinejobs.MarkJobRunning,
	updateHeartbeat *pipelinejobs.UpdateHeartbeat,
	checkJobCancelled *pipelinejobs.CheckJobCancelled,
	reportCompletion *pipelinejobs.ReportCompletion,
	handleConfigUpdate *pipelinejobs.HandleConfigUpdate,
	saveSyncState *pipelinestate.SaveSyncState,
	batchAppendLogs *pipelinelogs.BatchAppendJobLogs,
	getJob *pipelinejobs.GetJob,
	claimTask *pipelinetasks.ClaimTask,
	reportTaskResult *pipelinetasks.ReportTaskResult,
	isTaskActive *pipelinejobs.IsTaskActive,
	setK8sJobName *pipelinejobs.SetK8sJobName,
	registerExecutor *pipelinejobs.RegisterExecutor,
) *UseCaseExecutorBackend {
	return &UseCaseExecutorBackend{
		claimJobBundle:     claimJobBundle,
		updateJobStatus:    updateJobStatus,
		markJobRunning:     markJobRunning,
		updateHeartbeat:    updateHeartbeat,
		checkJobCancelled:  checkJobCancelled,
		reportCompletion:   reportCompletion,
		handleConfigUpdate: handleConfigUpdate,
		saveSyncState:      saveSyncState,
		batchAppendLogs:    batchAppendLogs,
		getJob:             getJob,
		claimTask:          claimTask,
		reportTaskResult:   reportTaskResult,
		isTaskActive:       isTaskActive,
		setK8sJobName:      setK8sJobName,
		registerExecutor:   registerExecutor,
	}
}

// ClaimJob claims the next available job and returns the full executor bundle.
func (a *UseCaseExecutorBackend) ClaimJob(ctx context.Context, workerID string) (*pipelineexec.ClaimJobResult, error) {
	result, err := a.claimJobBundle.Execute(ctx, pipelinejobs.ClaimJobParams{WorkerID: workerID})
	if err != nil {
		return nil, fmt.Errorf("claiming job bundle: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	return &pipelineexec.ClaimJobResult{
		Job:                   result.Job,
		ConnectionID:          result.ConnectionID,
		WorkspaceID:           result.WorkspaceID,
		SourceID:              result.SourceID,
		DestinationID:         result.DestinationID,
		SourceImage:           result.SourceImage,
		SourceConfig:          result.SourceConfig,
		DestImage:             result.DestImage,
		DestConfig:            result.DestConfig,
		ConfiguredCatalog:     result.ConfiguredCatalog,
		StateBlob:             result.StateBlob,
		SourceRuntimeConfig:   result.SourceRuntimeConfig,
		DestRuntimeConfig:     result.DestRuntimeConfig,
		NamespaceDefinition:   result.NamespaceDefinition,
		CustomNamespaceFormat: result.CustomNamespaceFormat,
		StreamPrefix:          result.StreamPrefix,
		MapperSalt:            result.MapperSalt,
		MaxAttempts:           result.MaxAttempts,
	}, nil
}

// UpdateJobStatus reports job completion or failure.
func (a *UseCaseExecutorBackend) UpdateJobStatus(ctx context.Context, params pipelineexec.UpdateJobStatusParams) error {
	// ReportCompletionParams requires the job's connection id.
	job, err := a.getJob.Execute(ctx, pipelinejobs.GetJobParams{ID: params.JobID})
	if err != nil {
		return fmt.Errorf("getting job for completion report: %w", err)
	}

	return a.reportCompletion.Execute(ctx, pipelinejobs.ReportCompletionParams{
		JobID:           params.JobID,
		ConnectionID:    job.ConnectionID,
		Success:         params.Success,
		ErrorMessage:    params.ErrorMessage,
		RecordsRead:     params.RecordsRead,
		RecordsFiltered: params.RecordsFiltered,
		BytesSynced:     params.BytesSynced,
		DurationMs:      params.DurationMs,
	})
}

// MarkJobRunning moves a starting job to running.
func (a *UseCaseExecutorBackend) MarkJobRunning(ctx context.Context, jobID uuid.UUID) error {
	if err := a.markJobRunning.Execute(ctx, pipelinejobs.MarkJobRunningParams{ID: jobID}); err != nil {
		return fmt.Errorf("marking job running: %w", err)
	}

	return nil
}

// Heartbeat updates the heartbeat timestamp and checks for cancellation.
func (a *UseCaseExecutorBackend) Heartbeat(ctx context.Context, jobID uuid.UUID, recordsRead, bytesSynced int64) (*pipelineexec.HeartbeatResult, error) {
	// Update heartbeat timestamp; non-fatal if it fails.
	if err := a.updateHeartbeat.Execute(ctx, pipelinejobs.UpdateHeartbeatParams{ID: jobID}); err != nil {
		slog.Error("usecase backend: heartbeat update failed", "job_id", jobID.String(), "error", err)
	}

	// Check cancellation regardless of heartbeat success.
	cancelled, err := a.checkJobCancelled.Execute(ctx, pipelinejobs.CheckJobCancelledParams{JobID: jobID})
	if err != nil {
		slog.Error("usecase backend: check cancelled failed", "job_id", jobID.String(), "error", err)

		return &pipelineexec.HeartbeatResult{Cancelled: false}, nil
	}

	return &pipelineexec.HeartbeatResult{Cancelled: cancelled}, nil
}

// ReportState saves a sync state message for the given connection.
func (a *UseCaseExecutorBackend) ReportState(ctx context.Context, connectionID, jobID uuid.UUID, stateMsg *protocol.AirbyteStateMessage) error {
	return a.saveSyncState.Execute(ctx, pipelinestate.SaveSyncStateParams{
		ConnectionID: connectionID,
		StateMessage: stateMsg,
	})
}

// ReportCompletion reports job completion with full stats and triggers events.
func (a *UseCaseExecutorBackend) ReportCompletion(ctx context.Context, params pipelineexec.ReportCompletionParams) error {
	return a.reportCompletion.Execute(ctx, pipelinejobs.ReportCompletionParams{
		JobID:           params.JobID,
		ConnectionID:    params.ConnectionID,
		Success:         params.Success,
		ErrorMessage:    params.ErrorMessage,
		RecordsRead:     params.RecordsRead,
		RecordsFiltered: params.RecordsFiltered,
		BytesSynced:     params.BytesSynced,
		DurationMs:      params.DurationMs,
		SourceExitCode:  params.SourceExitCode,
		DestExitCode:    params.DestExitCode,
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

// ReportLog appends log lines for a job.
func (a *UseCaseExecutorBackend) ReportLog(ctx context.Context, jobID uuid.UUID, lines []string) error {
	if err := a.batchAppendLogs.Execute(ctx, pipelinelogs.BatchAppendJobLogsParams{
		JobID:    jobID,
		LogLines: lines,
	}); err != nil {
		return fmt.Errorf("appending job logs: %w", err)
	}

	return nil
}

// ClaimConnectorTask claims the next pending connector task (D-15).
func (a *UseCaseExecutorBackend) ClaimConnectorTask(ctx context.Context, workerID string) (*pipelineexec.ClaimConnectorTaskResult, error) {
	result, err := a.claimTask.Execute(ctx, workerID)
	if err != nil {
		return nil, fmt.Errorf("claiming connector task: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	return &pipelineexec.ClaimConnectorTaskResult{
		TaskID:      result.TaskID,
		TaskType:    result.TaskType,
		Image:       result.Image,
		Config:      result.Config,
		WorkspaceID: result.WorkspaceID,

		ServiceAccountName: result.ServiceAccountName,
	}, nil
}

// ReportConnectorTaskResult reports the result of a connector task (D-18/D-19).
func (a *UseCaseExecutorBackend) ReportConnectorTaskResult(ctx context.Context, params pipelineexec.ReportConnectorTaskResultParams) error {
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
		var notFound pipelineservice.NotFoundError
		if errors.As(err, &notFound) {
			return false, nil
		}

		return false, fmt.Errorf("getting job: %w", err)
	}

	return job.Status == pipelineservice.JobStatusRunning || job.Status == pipelineservice.JobStatusStarting, nil
}

// IsTaskActive checks whether a connector task is in pending or running status.
func (a *UseCaseExecutorBackend) IsTaskActive(ctx context.Context, taskID string) (bool, error) {
	return a.isTaskActive.ExecuteByString(ctx, taskID)
}

// FailJob marks a job as failed with the given reason.
func (a *UseCaseExecutorBackend) FailJob(ctx context.Context, jobID, reason string) error {
	parsed, err := uuid.Parse(jobID)
	if err != nil {
		return fmt.Errorf("parsing job ID: %w", err)
	}

	return a.updateJobStatus.Execute(ctx, pipelinejobs.UpdateJobStatusParams{
		ID:      parsed,
		SyncErr: errors.New(reason),
	})
}

// SetK8sJobName stores the Kubernetes Job name launched for a job.
func (a *UseCaseExecutorBackend) SetK8sJobName(ctx context.Context, jobID uuid.UUID, name string) error {
	return a.setK8sJobName.Execute(ctx, pipelinejobs.SetK8sJobNameParams{
		ID:         jobID,
		K8sJobName: name,
	})
}

// ExecutorHeartbeat registers the in-process executor in the executor registry.
func (a *UseCaseExecutorBackend) ExecutorHeartbeat(ctx context.Context, info pipelineexec.ExecutorInfo) error {
	return a.registerExecutor.Execute(ctx, pipelinejobs.RegisterExecutorParams{
		InstanceID:       info.ID,
		Type:             info.Type,
		Version:          info.Version,
		Capacity:         info.Capacity,
		Capabilities:     info.Capabilities,
		ResourceDefaults: info.ResourceDefaults,
	})
}
