package pipelineworkflows

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetWorkflowRunParams holds parameters for getting a workflow run.
type GetWorkflowRunParams struct {
	RunID       uuid.UUID
	WorkspaceID uuid.UUID
}

// GetWorkflowRun returns a run of a workspace's workflow with its step runs.
type GetWorkflowRun struct {
	storage pipelineservice.Storage
}

// NewGetWorkflowRun creates a new GetWorkflowRun use case.
func NewGetWorkflowRun(storage pipelineservice.Storage) *GetWorkflowRun {
	return &GetWorkflowRun{storage: storage}
}

// Execute returns the run, or ErrWorkflowRunNotFound.
func (uc *GetWorkflowRun) Execute(ctx context.Context, params GetWorkflowRunParams) (*WorkflowRunDetail, error) {
	run, _, err := getRun(ctx, uc.storage, params.RunID, params.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return loadRunDetail(ctx, uc.storage, run.ID)
}
