package pipelineconnect

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineworkflows"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// WorkflowHandler implements the WorkflowService ConnectRPC handler.
type WorkflowHandler struct {
	pipelinev1connect.UnimplementedWorkflowServiceHandler

	createWorkflow *pipelineworkflows.CreateWorkflow
	updateWorkflow *pipelineworkflows.UpdateWorkflow
	deleteWorkflow *pipelineworkflows.DeleteWorkflow
	getWorkflow    *pipelineworkflows.GetWorkflow
	listWorkflows  *pipelineworkflows.ListWorkflows
	startRun       *pipelineworkflows.StartWorkflowRun
	rerunRun       *pipelineworkflows.RerunWorkflowRun
	listRuns       *pipelineworkflows.ListWorkflowRuns
	getRun         *pipelineworkflows.GetWorkflowRun
}

// NewWorkflowHandler creates a new workflow handler.
func NewWorkflowHandler(
	createWorkflow *pipelineworkflows.CreateWorkflow,
	updateWorkflow *pipelineworkflows.UpdateWorkflow,
	deleteWorkflow *pipelineworkflows.DeleteWorkflow,
	getWorkflow *pipelineworkflows.GetWorkflow,
	listWorkflows *pipelineworkflows.ListWorkflows,
	startRun *pipelineworkflows.StartWorkflowRun,
	rerunRun *pipelineworkflows.RerunWorkflowRun,
	listRuns *pipelineworkflows.ListWorkflowRuns,
	getRun *pipelineworkflows.GetWorkflowRun,
) *WorkflowHandler {
	return &WorkflowHandler{
		createWorkflow: createWorkflow,
		updateWorkflow: updateWorkflow,
		deleteWorkflow: deleteWorkflow,
		getWorkflow:    getWorkflow,
		listWorkflows:  listWorkflows,
		startRun:       startRun,
		rerunRun:       rerunRun,
		listRuns:       listRuns,
		getRun:         getRun,
	}
}

func (h *WorkflowHandler) CreateWorkflow(ctx context.Context, req *connect.Request[pipelinev1.CreateWorkflowRequest]) (*connect.Response[pipelinev1.CreateWorkflowResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	steps, err := workflowStepsFromProto(req.Msg.GetSteps())
	if err != nil {
		return nil, err
	}

	detail, err := h.createWorkflow.Execute(ctx, pipelineworkflows.CreateWorkflowParams{
		WorkspaceID: workspaceID,
		Name:        req.Msg.GetName(),
		Schedule:    req.Msg.Schedule,
		Steps:       steps,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.CreateWorkflowResponse{Workflow: workflowToProto(detail)}), nil
}

func (h *WorkflowHandler) UpdateWorkflow(ctx context.Context, req *connect.Request[pipelinev1.UpdateWorkflowRequest]) (*connect.Response[pipelinev1.UpdateWorkflowResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := parseID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	steps, err := workflowStepsFromProto(req.Msg.GetSteps())
	if err != nil {
		return nil, err
	}

	detail, err := h.updateWorkflow.Execute(ctx, pipelineworkflows.UpdateWorkflowParams{
		ID:          id,
		WorkspaceID: workspaceID,
		Name:        req.Msg.GetName(),
		Schedule:    req.Msg.Schedule,
		Steps:       steps,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.UpdateWorkflowResponse{Workflow: workflowToProto(detail)}), nil
}

func (h *WorkflowHandler) DeleteWorkflow(ctx context.Context, req *connect.Request[pipelinev1.DeleteWorkflowRequest]) (*connect.Response[pipelinev1.DeleteWorkflowResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := parseID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	if err := h.deleteWorkflow.Execute(ctx, pipelineworkflows.DeleteWorkflowParams{ID: id, WorkspaceID: workspaceID}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.DeleteWorkflowResponse{}), nil
}

func (h *WorkflowHandler) GetWorkflow(ctx context.Context, req *connect.Request[pipelinev1.GetWorkflowRequest]) (*connect.Response[pipelinev1.GetWorkflowResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := parseID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	detail, err := h.getWorkflow.Execute(ctx, pipelineworkflows.GetWorkflowParams{ID: id, WorkspaceID: workspaceID})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.GetWorkflowResponse{Workflow: workflowToProto(detail)}), nil
}

func (h *WorkflowHandler) ListWorkflows(ctx context.Context, _ *connect.Request[pipelinev1.ListWorkflowsRequest]) (*connect.Response[pipelinev1.ListWorkflowsResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	details, err := h.listWorkflows.Execute(ctx, pipelineworkflows.ListWorkflowsParams{WorkspaceID: workspaceID})
	if err != nil {
		return nil, mapError(err)
	}

	workflows := make([]*pipelinev1.Workflow, 0, len(details))
	for _, detail := range details {
		workflows = append(workflows, workflowToProto(detail))
	}

	return connect.NewResponse(&pipelinev1.ListWorkflowsResponse{Workflows: workflows}), nil
}

func (h *WorkflowHandler) RunWorkflow(ctx context.Context, req *connect.Request[pipelinev1.RunWorkflowRequest]) (*connect.Response[pipelinev1.RunWorkflowResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := parseID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	run, err := h.startRun.Execute(ctx, pipelineworkflows.StartWorkflowRunParams{WorkflowID: id, WorkspaceID: workspaceID})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.RunWorkflowResponse{Run: workflowRunToProto(run)}), nil
}

func (h *WorkflowHandler) RerunWorkflowRun(ctx context.Context, req *connect.Request[pipelinev1.RerunWorkflowRunRequest]) (*connect.Response[pipelinev1.RerunWorkflowRunResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	runID, err := parseID(req.Msg.GetRunId(), "run_id")
	if err != nil {
		return nil, err
	}

	run, err := h.rerunRun.Execute(ctx, pipelineworkflows.RerunWorkflowRunParams{RunID: runID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.RerunWorkflowRunResponse{Run: workflowRunToProto(run)}), nil
}

func (h *WorkflowHandler) ListWorkflowRuns(ctx context.Context, req *connect.Request[pipelinev1.ListWorkflowRunsRequest]) (*connect.Response[pipelinev1.ListWorkflowRunsResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	workflowID, err := parseID(req.Msg.GetWorkflowId(), "workflow_id")
	if err != nil {
		return nil, err
	}

	details, err := h.listRuns.Execute(ctx, pipelineworkflows.ListWorkflowRunsParams{WorkflowID: workflowID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, mapError(err)
	}

	runs := make([]*pipelinev1.WorkflowRun, 0, len(details))
	for _, detail := range details {
		runs = append(runs, workflowRunToProto(detail))
	}

	return connect.NewResponse(&pipelinev1.ListWorkflowRunsResponse{Runs: runs}), nil
}

func (h *WorkflowHandler) GetWorkflowRun(ctx context.Context, req *connect.Request[pipelinev1.GetWorkflowRunRequest]) (*connect.Response[pipelinev1.GetWorkflowRunResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	runID, err := parseID(req.Msg.GetRunId(), "run_id")
	if err != nil {
		return nil, err
	}

	run, err := h.getRun.Execute(ctx, pipelineworkflows.GetWorkflowRunParams{RunID: runID, WorkspaceID: workspaceID})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.GetWorkflowRunResponse{Run: workflowRunToProto(run)}), nil
}

func parseID(value, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid %s: %w", field, err))
	}

	return id, nil
}

func workflowStepsFromProto(inputs []*pipelinev1.WorkflowStepInput) ([]pipelineservice.WorkflowGraphStep, error) {
	steps := make([]pipelineservice.WorkflowGraphStep, 0, len(inputs))

	for _, input := range inputs {
		connID, err := parseID(input.GetConnectionId(), "connection_id")
		if err != nil {
			return nil, err
		}

		step := pipelineservice.WorkflowGraphStep{ConnectionID: connID}

		for _, after := range input.GetAfterConnectionIds() {
			upstream, err := parseID(after, "after_connection_ids")
			if err != nil {
				return nil, err
			}

			step.AfterConnectionIDs = append(step.AfterConnectionIDs, upstream)
		}

		steps = append(steps, step)
	}

	return steps, nil
}

func workflowToProto(detail *pipelineworkflows.WorkflowDetail) *pipelinev1.Workflow {
	connByStep := make(map[uuid.UUID]uuid.UUID, len(detail.Steps))
	for _, step := range detail.Steps {
		connByStep[step.ID] = step.ConnectionID
	}

	after := make(map[uuid.UUID][]string, len(detail.Steps))
	for _, dep := range detail.Dependencies {
		after[dep.StepID] = append(after[dep.StepID], connByStep[dep.UpstreamStepID].String())
	}

	steps := make([]*pipelinev1.WorkflowStep, 0, len(detail.Steps))
	for _, step := range detail.Steps {
		steps = append(steps, &pipelinev1.WorkflowStep{
			Id:                 step.ID.String(),
			ConnectionId:       step.ConnectionID.String(),
			AfterConnectionIds: after[step.ID],
		})
	}

	workflow := detail.Workflow

	return &pipelinev1.Workflow{
		Id:              workflow.ID.String(),
		WorkspaceId:     workflow.WorkspaceID.String(),
		Name:            workflow.Name,
		Schedule:        workflow.Schedule,
		NextScheduledAt: optionalTimestamp(workflow.NextScheduledAt),
		Steps:           steps,
		CreatedAt:       timestamppb.New(workflow.CreatedAt),
		UpdatedAt:       timestamppb.New(workflow.UpdatedAt),
	}
}

func workflowRunToProto(detail *pipelineworkflows.WorkflowRunDetail) *pipelinev1.WorkflowRun {
	stepRuns := make([]*pipelinev1.WorkflowStepRun, 0, len(detail.StepRuns))

	for _, stepRun := range detail.StepRuns {
		var jobID *string

		if stepRun.JobID != nil {
			id := stepRun.JobID.String()
			jobID = &id
		}

		stepRuns = append(stepRuns, &pipelinev1.WorkflowStepRun{
			Id:           stepRun.ID.String(),
			StepId:       stepRun.StepID.String(),
			ConnectionId: stepRun.ConnectionID.String(),
			Status:       workflowStepRunStatusToProto(stepRun.Status),
			JobId:        jobID,
			Error:        stepRun.Error,
			StartedAt:    optionalTimestamp(stepRun.StartedAt),
			CompletedAt:  optionalTimestamp(stepRun.CompletedAt),
		})
	}

	run := detail.Run

	return &pipelinev1.WorkflowRun{
		Id:          run.ID.String(),
		WorkflowId:  run.WorkflowID.String(),
		Status:      workflowRunStatusToProto(run.Status),
		Trigger:     workflowRunTriggerToProto(run.Trigger),
		StartedAt:   timestamppb.New(run.StartedAt),
		CompletedAt: optionalTimestamp(run.CompletedAt),
		StepRuns:    stepRuns,
	}
}

func optionalTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}

	return timestamppb.New(*t)
}

func workflowRunStatusToProto(status pipelineservice.WorkflowRunStatus) pipelinev1.WorkflowRunStatus {
	switch status {
	case pipelineservice.WorkflowRunStatusRunning:
		return pipelinev1.WorkflowRunStatus_WORKFLOW_RUN_STATUS_RUNNING
	case pipelineservice.WorkflowRunStatusSucceeded:
		return pipelinev1.WorkflowRunStatus_WORKFLOW_RUN_STATUS_SUCCEEDED
	case pipelineservice.WorkflowRunStatusFailed:
		return pipelinev1.WorkflowRunStatus_WORKFLOW_RUN_STATUS_FAILED
	default:
		return pipelinev1.WorkflowRunStatus_WORKFLOW_RUN_STATUS_UNSPECIFIED
	}
}

func workflowRunTriggerToProto(trigger pipelineservice.WorkflowRunTrigger) pipelinev1.WorkflowRunTrigger {
	switch trigger {
	case pipelineservice.WorkflowRunTriggerSchedule:
		return pipelinev1.WorkflowRunTrigger_WORKFLOW_RUN_TRIGGER_SCHEDULE
	case pipelineservice.WorkflowRunTriggerManual:
		return pipelinev1.WorkflowRunTrigger_WORKFLOW_RUN_TRIGGER_MANUAL
	default:
		return pipelinev1.WorkflowRunTrigger_WORKFLOW_RUN_TRIGGER_UNSPECIFIED
	}
}

func workflowStepRunStatusToProto(status pipelineservice.WorkflowStepRunStatus) pipelinev1.WorkflowStepRunStatus {
	switch status {
	case pipelineservice.WorkflowStepRunStatusPending:
		return pipelinev1.WorkflowStepRunStatus_WORKFLOW_STEP_RUN_STATUS_PENDING
	case pipelineservice.WorkflowStepRunStatusQueued:
		return pipelinev1.WorkflowStepRunStatus_WORKFLOW_STEP_RUN_STATUS_QUEUED
	case pipelineservice.WorkflowStepRunStatusSucceeded:
		return pipelinev1.WorkflowStepRunStatus_WORKFLOW_STEP_RUN_STATUS_SUCCEEDED
	case pipelineservice.WorkflowStepRunStatusFailed:
		return pipelinev1.WorkflowStepRunStatus_WORKFLOW_STEP_RUN_STATUS_FAILED
	case pipelineservice.WorkflowStepRunStatusSkipped:
		return pipelinev1.WorkflowStepRunStatus_WORKFLOW_STEP_RUN_STATUS_SKIPPED
	default:
		return pipelinev1.WorkflowStepRunStatus_WORKFLOW_STEP_RUN_STATUS_UNSPECIFIED
	}
}
