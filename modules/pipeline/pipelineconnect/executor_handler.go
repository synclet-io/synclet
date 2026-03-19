package pipelineconnect

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ExecutorHandler implements the ExecutorService ConnectRPC handler.
// It receives reports from executors and serves job claims. All data access
// goes through use cases -- the handler never touches storage directly.
type ExecutorHandler struct {
	reportCompletion   *pipelinejobs.ReportCompletion
	updateHeartbeat    *pipelinejobs.UpdateHeartbeat
	handleConfigUpdate *pipelinejobs.HandleConfigUpdate
	saveSyncState      *pipelinestate.SaveSyncState
	claimJobBundle     *pipelinejobs.ClaimJobBundle
	updateJobStatus    *pipelinejobs.UpdateJobStatus
	checkJobCancelled  *pipelinejobs.CheckJobCancelled
	getJob             *pipelinejobs.GetJob
	batchAppendLogs    *pipelinelogs.BatchAppendJobLogs
	claimTask          *pipelinetasks.ClaimTask
	reportTaskResult   *pipelinetasks.ReportTaskResult
	logger             *logging.Logger
}

// NewExecutorHandler creates a new executor handler.
func NewExecutorHandler(
	reportCompletion *pipelinejobs.ReportCompletion,
	updateHeartbeat *pipelinejobs.UpdateHeartbeat,
	handleConfigUpdate *pipelinejobs.HandleConfigUpdate,
	saveSyncState *pipelinestate.SaveSyncState,
	claimJobBundle *pipelinejobs.ClaimJobBundle,
	updateJobStatus *pipelinejobs.UpdateJobStatus,
	checkJobCancelled *pipelinejobs.CheckJobCancelled,
	getJob *pipelinejobs.GetJob,
	batchAppendLogs *pipelinelogs.BatchAppendJobLogs,
	claimTask *pipelinetasks.ClaimTask,
	reportTaskResult *pipelinetasks.ReportTaskResult,
	logger *logging.Logger,
) *ExecutorHandler {
	return &ExecutorHandler{
		reportCompletion:   reportCompletion,
		updateHeartbeat:    updateHeartbeat,
		handleConfigUpdate: handleConfigUpdate,
		saveSyncState:      saveSyncState,
		claimJobBundle:     claimJobBundle,
		updateJobStatus:    updateJobStatus,
		checkJobCancelled:  checkJobCancelled,
		getJob:             getJob,
		batchAppendLogs:    batchAppendLogs,
		claimTask:          claimTask,
		reportTaskResult:   reportTaskResult,
		logger:             logger.Named("executor-handler"),
	}
}

// Heartbeat updates the heartbeat timestamp and checks for cancellation per D-05.
func (h *ExecutorHandler) Heartbeat(ctx context.Context, req *connect.Request[executorv1.HeartbeatRequest]) (*connect.Response[executorv1.HeartbeatResponse], error) {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid job_id: %w", err))
	}

	// Update heartbeat timestamp.
	if err := h.updateHeartbeat.Execute(ctx, pipelinejobs.UpdateHeartbeatParams{ID: jobID}); err != nil {
		h.logger.WithError(err).Error(ctx, "executor: heartbeat update failed",
			"job_id", req.Msg.JobId)
		return nil, mapError(err)
	}

	// Check cancellation status per D-05.
	cancelled, err := h.checkJobCancelled.Execute(ctx, pipelinejobs.CheckJobCancelledParams{JobID: jobID})
	if err != nil {
		h.logger.WithError(err).Error(ctx, "executor: cancel check failed",
			"job_id", req.Msg.JobId)
		// Non-fatal: return heartbeat success without cancellation info.
		return connect.NewResponse(&executorv1.HeartbeatResponse{}), nil
	}

	return connect.NewResponse(&executorv1.HeartbeatResponse{
		Cancelled: cancelled,
	}), nil
}

// ReportState persists a state checkpoint from the executor.
func (h *ExecutorHandler) ReportState(ctx context.Context, req *connect.Request[executorv1.ReportStateRequest]) (*connect.Response[executorv1.ReportStateResponse], error) {
	connectionID, err := uuid.Parse(req.Msg.ConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection_id: %w", err))
	}

	var stateMsg protocol.AirbyteStateMessage
	if err := json.Unmarshal(req.Msg.StateData, &stateMsg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid state_data: %w", err))
	}

	if err := h.saveSyncState.Execute(ctx, pipelinestate.SaveSyncStateParams{
		ConnectionID: connectionID,
		StateMessage: &stateMsg,
	}); err != nil {
		h.logger.WithError(err).Error(ctx, "executor: state save failed",
			"job_id", req.Msg.JobId)
		return nil, mapError(err)
	}

	return connect.NewResponse(&executorv1.ReportStateResponse{}), nil
}

// ReportCompletion marks a job as completed or failed.
func (h *ExecutorHandler) ReportCompletion(ctx context.Context, req *connect.Request[executorv1.ReportCompletionRequest]) (*connect.Response[executorv1.ReportCompletionResponse], error) {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid job_id: %w", err))
	}

	connectionID, err := uuid.Parse(req.Msg.ConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection_id: %w", err))
	}

	if err := h.reportCompletion.Execute(ctx, pipelinejobs.ReportCompletionParams{
		JobID:        jobID,
		ConnectionID: connectionID,
		Success:      req.Msg.Success,
		ErrorMessage: req.Msg.ErrorMessage,
		RecordsRead:  req.Msg.RecordsRead,
		BytesSynced:  req.Msg.BytesSynced,
		DurationMs:   req.Msg.DurationMs,
	}); err != nil {
		h.logger.WithError(err).Error(ctx, "executor: completion failed",
			"job_id", req.Msg.JobId)
		return nil, mapError(err)
	}

	return connect.NewResponse(&executorv1.ReportCompletionResponse{}), nil
}

// ReportConfigUpdate handles config update reports from executors.
func (h *ExecutorHandler) ReportConfigUpdate(ctx context.Context, req *connect.Request[executorv1.ReportConfigUpdateRequest]) (*connect.Response[executorv1.ReportConfigUpdateResponse], error) {
	connectorID, err := uuid.Parse(req.Msg.ConnectorId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connector_id: %w", err))
	}

	if err := h.handleConfigUpdate.Execute(ctx, pipelinejobs.HandleConfigUpdateParams{
		ConnectorType: protoConnectorTypeToDomain(req.Msg.ConnectorType),
		ConnectorID:   connectorID,
		Config:        req.Msg.Config,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&executorv1.ReportConfigUpdateResponse{}), nil
}

// ReportLog receives batched log lines from executors and persists them.
func (h *ExecutorHandler) ReportLog(ctx context.Context, req *connect.Request[executorv1.ReportLogRequest]) (*connect.Response[executorv1.ReportLogResponse], error) {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.batchAppendLogs.Execute(ctx, pipelinelogs.BatchAppendJobLogsParams{
		JobID:    jobID,
		LogLines: req.Msg.LogLines,
	}); err != nil {
		h.logger.WithError(err).Error(ctx, "executor: failed to batch append logs",
			"job_id", req.Msg.JobId)
		// Non-fatal: return success anyway so executor doesn't retry.
	}

	return connect.NewResponse(&executorv1.ReportLogResponse{}), nil
}

// ClaimJob claims the next available job and returns the full executor bundle per D-02.
func (h *ExecutorHandler) ClaimJob(ctx context.Context, req *connect.Request[executorv1.ClaimJobRequest]) (*connect.Response[executorv1.ClaimJobResponse], error) {
	result, err := h.claimJobBundle.Execute(ctx, pipelinejobs.ClaimJobParams{
		WorkerID: req.Msg.WorkerId,
	})
	if err != nil {
		h.logger.WithError(err).Error(ctx, "executor: claim job failed")
		return nil, mapError(err)
	}

	if result == nil {
		return connect.NewResponse(&executorv1.ClaimJobResponse{
			HasJob: false,
		}), nil
	}

	return connect.NewResponse(&executorv1.ClaimJobResponse{
		HasJob:                true,
		JobId:                 result.Job.ID.String(),
		ConnectionId:          result.ConnectionID.String(),
		JobType:               domainJobTypeToProto(result.Job.JobType),
		SourceId:              result.SourceID.String(),
		DestinationId:         result.DestinationID.String(),
		SourceImage:           result.SourceImage,
		SourceConfig:          result.SourceConfig,
		DestImage:             result.DestImage,
		DestConfig:            result.DestConfig,
		ConfiguredCatalog:     result.ConfiguredCatalog,
		StateBlob:             result.StateBlob,
		RuntimeConfig:         result.SourceRuntimeConfig,
		WorkspaceId:           result.WorkspaceID.String(),
		NamespaceDefinition:   domainNamespaceDefToProto(result.NamespaceDefinition),
		CustomNamespaceFormat: result.CustomNamespaceFormat,
		StreamPrefix:          result.StreamPrefix,
		MaxAttempts:           int32(result.MaxAttempts),
		SourceRuntimeConfig:   result.SourceRuntimeConfig,
		DestRuntimeConfig:     result.DestRuntimeConfig,
	}), nil
}

// UpdateJobStatus reports job completion or failure per D-03.
func (h *ExecutorHandler) UpdateJobStatus(ctx context.Context, req *connect.Request[executorv1.UpdateJobStatusRequest]) (*connect.Response[executorv1.UpdateJobStatusResponse], error) {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid job_id: %w", err))
	}

	// Build SyncStats from request.
	stats := &pipelineservice.SyncStats{
		RecordsRead: req.Msg.RecordsRead,
		BytesSynced: req.Msg.BytesSynced,
	}

	var syncErr error
	if !req.Msg.Success {
		syncErr = fmt.Errorf("%s", req.Msg.ErrorMessage)
	}

	if err := h.updateJobStatus.Execute(ctx, pipelinejobs.UpdateJobStatusParams{
		ID:        jobID,
		SyncErr:   syncErr,
		SyncStats: stats,
	}); err != nil {
		h.logger.WithError(err).Error(ctx, "executor: update job status failed",
			"job_id", req.Msg.JobId)
		return nil, mapError(err)
	}

	return connect.NewResponse(&executorv1.UpdateJobStatusResponse{}), nil
}

// IsJobActive checks whether a job is still running or starting per D-04.
// Returns active=true only for Running or Starting statuses.
// Returns active=false for all other statuses and on any error (including not found).
func (h *ExecutorHandler) IsJobActive(ctx context.Context, req *connect.Request[executorv1.IsJobActiveRequest]) (*connect.Response[executorv1.IsJobActiveResponse], error) {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return connect.NewResponse(&executorv1.IsJobActiveResponse{Active: false}), nil //nolint:nilerr // invalid UUID means not active
	}

	job, err := h.getJob.Execute(ctx, pipelinejobs.GetJobParams{ID: jobID})
	if err != nil {
		return connect.NewResponse(&executorv1.IsJobActiveResponse{Active: false}), nil //nolint:nilerr // not-found is expected, return not active
	}

	active := job.Status == pipelineservice.JobStatusRunning || job.Status == pipelineservice.JobStatusStarting

	return connect.NewResponse(&executorv1.IsJobActiveResponse{Active: active}), nil
}

// ClaimConnectorTask claims the next pending connector task for the given worker (D-15).
func (h *ExecutorHandler) ClaimConnectorTask(ctx context.Context, req *connect.Request[executorv1.ClaimConnectorTaskRequest]) (*connect.Response[executorv1.ClaimConnectorTaskResponse], error) {
	result, err := h.claimTask.Execute(ctx, req.Msg.WorkerId)
	if err != nil {
		h.logger.WithError(err).Error(ctx, "executor: claim connector task failed")
		return nil, mapError(err)
	}

	if result == nil {
		return connect.NewResponse(&executorv1.ClaimConnectorTaskResponse{
			HasTask: false,
		}), nil
	}

	return connect.NewResponse(&executorv1.ClaimConnectorTaskResponse{
		HasTask:     true,
		TaskId:      result.TaskID.String(),
		TaskType:    domainTaskTypeToProto(result.TaskType),
		Image:       result.Image,
		Config:      result.Config,
		WorkspaceId: result.WorkspaceID.String(),
	}), nil
}

// ReportConnectorTaskResult reports the result of a connector task (D-18/D-19).
func (h *ExecutorHandler) ReportConnectorTaskResult(ctx context.Context, req *connect.Request[executorv1.ReportConnectorTaskResultRequest]) (*connect.Response[executorv1.ReportConnectorTaskResultResponse], error) {
	taskID, err := uuid.Parse(req.Msg.TaskId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid task_id: %w", err))
	}

	if err := h.reportTaskResult.Execute(ctx, pipelinetasks.ReportTaskResultParams{
		TaskID:       taskID,
		Success:      req.Msg.Success,
		ErrorMessage: req.Msg.ErrorMessage,
		Result:       req.Msg.Result,
	}); err != nil {
		h.logger.WithError(err).Error(ctx, "executor: report connector task result failed",
			"task_id", req.Msg.TaskId)
		return nil, mapError(err)
	}

	return connect.NewResponse(&executorv1.ReportConnectorTaskResultResponse{}), nil
}
