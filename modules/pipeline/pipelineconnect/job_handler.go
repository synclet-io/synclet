package pipelineconnect

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// JobHandler implements the JobService ConnectRPC handler.
type JobHandler struct {
	pipelinev1connect.UnimplementedJobServiceHandler

	triggerSync          *pipelinejobs.TriggerSync
	cancelJobForWs       *pipelinejobs.CancelJobForWorkspace
	getJobWithAttempts   *pipelinejobs.GetJobWithAttempts
	listJobsWithAttempts *pipelinejobs.ListJobsWithAttempts
	getJobLog            *pipelinelogs.GetJobLog
}

// NewJobHandler creates a new job handler.
func NewJobHandler(
	triggerSync *pipelinejobs.TriggerSync,
	cancelJobForWs *pipelinejobs.CancelJobForWorkspace,
	getJobWithAttempts *pipelinejobs.GetJobWithAttempts,
	listJobsWithAttempts *pipelinejobs.ListJobsWithAttempts,
	getJobLog *pipelinelogs.GetJobLog,
) *JobHandler {
	return &JobHandler{
		triggerSync:          triggerSync,
		cancelJobForWs:       cancelJobForWs,
		getJobWithAttempts:   getJobWithAttempts,
		listJobsWithAttempts: listJobsWithAttempts,
		getJobLog:            getJobLog,
	}
}

func (h *JobHandler) TriggerSync(ctx context.Context, req *connect.Request[pipelinev1.TriggerSyncRequest]) (*connect.Response[pipelinev1.TriggerSyncResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	connID, err := uuid.Parse(req.Msg.GetConnectionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	job, err := h.triggerSync.Execute(ctx, pipelinejobs.TriggerSyncParams{
		ConnectionID: connID,
		WorkspaceID:  workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.TriggerSyncResponse{
		Job: jobToProto(job, nil),
	}), nil
}

func (h *JobHandler) CancelJob(ctx context.Context, req *connect.Request[pipelinev1.CancelJobRequest]) (*connect.Response[pipelinev1.CancelJobResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	jobID, err := uuid.Parse(req.Msg.GetJobId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.cancelJobForWs.Execute(ctx, pipelinejobs.CancelJobForWorkspaceParams{
		JobID:       jobID,
		WorkspaceID: workspaceID,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.CancelJobResponse{}), nil
}

func (h *JobHandler) GetJob(ctx context.Context, req *connect.Request[pipelinev1.GetJobRequest]) (*connect.Response[pipelinev1.GetJobResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	jobID, err := uuid.Parse(req.Msg.GetJobId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	result, err := h.getJobWithAttempts.Execute(ctx, pipelinejobs.GetJobWithAttemptsParams{
		JobID:       jobID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.GetJobResponse{
		Job: jobToProto(result.Job, result.Attempts),
	}), nil
}

func (h *JobHandler) ListJobs(ctx context.Context, req *connect.Request[pipelinev1.ListJobsRequest]) (*connect.Response[pipelinev1.ListJobsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	connID, err := uuid.Parse(req.Msg.GetConnectionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	results, err := h.listJobsWithAttempts.Execute(ctx, pipelinejobs.ListJobsWithAttemptsParams{
		ConnectionID: connID,
		WorkspaceID:  workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	protoJobs := make([]*pipelinev1.Job, len(results))
	for i, r := range results {
		protoJobs[i] = jobToProto(r.Job, r.Attempts)
	}

	paginated, total := paginateSlice(protoJobs, req.Msg.GetPageSize(), req.Msg.GetOffset())

	return connect.NewResponse(&pipelinev1.ListJobsResponse{
		Jobs:  paginated,
		Total: total,
	}), nil
}

func (h *JobHandler) GetJobLogs(ctx context.Context, req *connect.Request[pipelinev1.GetJobLogsRequest]) (*connect.Response[pipelinev1.GetJobLogsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	jobID, err := uuid.Parse(req.Msg.GetJobId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	result, err := h.getJobLog.Execute(ctx, pipelinelogs.GetJobLogParams{
		WorkspaceID: workspaceID,
		JobID:       jobID,
		AfterID:     req.Msg.GetAfterId(),
		Limit:       int(req.Msg.GetLimit()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.GetJobLogsResponse{
		Lines:   result.Lines,
		LastId:  result.LastID,
		HasMore: result.HasMore,
	}), nil
}

func jobToProto(job *pipelineservice.Job, attempts []*pipelineservice.JobAttempt) *pipelinev1.Job {
	protoJob := &pipelinev1.Job{
		Id:           job.ID.String(),
		ConnectionId: job.ConnectionID.String(),
		Status:       jobStatusToProto(job.Status),
		JobType:      jobTypeToProto(job.JobType),
		ScheduledAt:  timestamppb.New(job.ScheduledAt),
		Attempt:      int32(job.Attempt),
		MaxAttempts:  int32(job.MaxAttempts),
		CreatedAt:    timestamppb.New(job.CreatedAt),
	}

	if job.StartedAt != nil {
		protoJob.StartedAt = timestamppb.New(*job.StartedAt)
	}

	if job.CompletedAt != nil {
		protoJob.CompletedAt = timestamppb.New(*job.CompletedAt)
	}

	if job.Error != nil {
		protoJob.Error = *job.Error
	}

	for _, attempt := range attempts {
		protoAttempt := &pipelinev1.JobAttempt{
			Id:            attempt.ID.String(),
			AttemptNumber: int32(attempt.AttemptNumber),
			StartedAt:     timestamppb.New(attempt.StartedAt),
		}
		if attempt.CompletedAt != nil {
			protoAttempt.CompletedAt = timestamppb.New(*attempt.CompletedAt)
		}

		if attempt.Error != nil {
			protoAttempt.Error = *attempt.Error
		}

		// Parse SyncStats from JSON.
		if attempt.SyncStatsJSON != "" && attempt.SyncStatsJSON != "{}" {
			var stats pipelineservice.SyncStats
			if err := json.Unmarshal([]byte(attempt.SyncStatsJSON), &stats); err == nil {
				protoAttempt.SyncStats = &pipelinev1.SyncStats{
					RecordsRead: stats.RecordsRead,
					BytesSynced: stats.BytesSynced,
					DurationMs:  stats.Duration.Milliseconds(),
				}
			}
		}

		protoJob.Attempts = append(protoJob.Attempts, protoAttempt)
	}

	return protoJob
}
