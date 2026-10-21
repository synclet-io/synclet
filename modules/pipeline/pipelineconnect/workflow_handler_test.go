package pipelineconnect

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/internal/pipelinefake"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineworkflows"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

func newTestWorkflowHandler(store *pipelinefake.Storage) *WorkflowHandler {
	audit := pipelineservice.NoopAuditRecorder{}

	return NewWorkflowHandler(
		pipelineworkflows.NewCreateWorkflow(store, audit),
		pipelineworkflows.NewUpdateWorkflow(store, audit),
		pipelineworkflows.NewDeleteWorkflow(store, audit),
		pipelineworkflows.NewGetWorkflow(store),
		pipelineworkflows.NewListWorkflows(store),
		pipelineworkflows.NewStartWorkflowRun(store, audit),
		pipelineworkflows.NewRerunWorkflowRun(store, audit),
		pipelineworkflows.NewListWorkflowRuns(store),
		pipelineworkflows.NewGetWorkflowRun(store),
	)
}

func TestWorkflowHandler(t *testing.T) {
	store := pipelinefake.New()
	workspaceID := uuid.New()
	ctx := connectutil.ContextWithWorkspaceID(context.Background(), workspaceID)

	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		store.ConnectionsTable.Insert(&pipelineservice.Connection{ID: id, WorkspaceID: workspaceID, Status: pipelineservice.ConnectionStatusActive})
	}

	foreign := uuid.New()
	store.ConnectionsTable.Insert(&pipelineservice.Connection{ID: foreign, WorkspaceID: uuid.New(), Status: pipelineservice.ConnectionStatusActive})

	handler := newTestWorkflowHandler(store)

	_, err := handler.CreateWorkflow(ctx, connect.NewRequest(&pipelinev1.CreateWorkflowRequest{
		Name: "cycle",
		Steps: []*pipelinev1.WorkflowStepInput{
			{ConnectionId: a.String(), AfterConnectionIds: []string{b.String()}},
			{ConnectionId: b.String(), AfterConnectionIds: []string{a.String()}},
		},
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "a cycle is rejected")

	_, err = handler.CreateWorkflow(ctx, connect.NewRequest(&pipelinev1.CreateWorkflowRequest{
		Name:  "foreign",
		Steps: []*pipelinev1.WorkflowStepInput{{ConnectionId: foreign.String()}},
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "a step from another workspace is rejected")

	_, err = handler.CreateWorkflow(ctx, connect.NewRequest(&pipelinev1.CreateWorkflowRequest{
		Name:  "twice",
		Steps: []*pipelinev1.WorkflowStepInput{{ConnectionId: a.String()}, {ConnectionId: a.String()}},
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "a connection listed twice is rejected")

	schedule := "0 3 * * *"
	created, err := handler.CreateWorkflow(ctx, connect.NewRequest(&pipelinev1.CreateWorkflowRequest{
		Name:     "nightly",
		Schedule: &schedule,
		Steps: []*pipelinev1.WorkflowStepInput{
			{ConnectionId: a.String()},
			{ConnectionId: b.String(), AfterConnectionIds: []string{a.String()}},
		},
	}))
	require.NoError(t, err)

	workflow := created.Msg.GetWorkflow()
	assert.Equal(t, "nightly", workflow.GetName())
	assert.Equal(t, schedule, workflow.GetSchedule())
	assert.NotNil(t, workflow.GetNextScheduledAt())
	require.Len(t, workflow.GetSteps(), 2)

	for _, step := range workflow.GetSteps() {
		if step.GetConnectionId() == b.String() {
			assert.Equal(t, []string{a.String()}, step.GetAfterConnectionIds())
		}
	}

	run, err := handler.RunWorkflow(ctx, connect.NewRequest(&pipelinev1.RunWorkflowRequest{Id: workflow.GetId()}))
	require.NoError(t, err)
	assert.Equal(t, pipelinev1.WorkflowRunTrigger_WORKFLOW_RUN_TRIGGER_MANUAL, run.Msg.GetRun().GetTrigger())
	assert.Equal(t, pipelinev1.WorkflowRunStatus_WORKFLOW_RUN_STATUS_RUNNING, run.Msg.GetRun().GetStatus())
	require.Len(t, run.Msg.GetRun().GetStepRuns(), 2)

	_, err = handler.RunWorkflow(ctx, connect.NewRequest(&pipelinev1.RunWorkflowRequest{Id: workflow.GetId()}))
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "a start during an active run is rejected")

	_, err = handler.DeleteWorkflow(ctx, connect.NewRequest(&pipelinev1.DeleteWorkflowRequest{Id: workflow.GetId()}))
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "a delete during an active run is rejected")

	_, err = handler.RerunWorkflowRun(ctx, connect.NewRequest(&pipelinev1.RerunWorkflowRunRequest{RunId: run.Msg.GetRun().GetId()}))
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "only a failed run can be rerun")

	runs, err := handler.ListWorkflowRuns(ctx, connect.NewRequest(&pipelinev1.ListWorkflowRunsRequest{WorkflowId: workflow.GetId()}))
	require.NoError(t, err)
	assert.Len(t, runs.Msg.GetRuns(), 1)

	list, err := handler.ListWorkflows(ctx, connect.NewRequest(&pipelinev1.ListWorkflowsRequest{}))
	require.NoError(t, err)
	assert.Len(t, list.Msg.GetWorkflows(), 1)

	otherCtx := connectutil.ContextWithWorkspaceID(context.Background(), uuid.New())
	_, err = handler.GetWorkflow(otherCtx, connect.NewRequest(&pipelinev1.GetWorkflowRequest{Id: workflow.GetId()}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "another workspace does not see the workflow")

	_, err = handler.GetWorkflow(ctx, connect.NewRequest(&pipelinev1.GetWorkflowRequest{Id: "not-a-uuid"}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
