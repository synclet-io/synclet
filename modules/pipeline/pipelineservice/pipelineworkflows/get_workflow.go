package pipelineworkflows

import (
	"context"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetWorkflowParams holds parameters for getting a workflow.
type GetWorkflowParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// GetWorkflow returns a workflow of a workspace with its steps.
type GetWorkflow struct {
	storage pipelineservice.Storage
}

// NewGetWorkflow creates a new GetWorkflow use case.
func NewGetWorkflow(storage pipelineservice.Storage) *GetWorkflow {
	return &GetWorkflow{storage: storage}
}

// Execute returns the workflow, or ErrWorkflowNotFound.
func (uc *GetWorkflow) Execute(ctx context.Context, params GetWorkflowParams) (*WorkflowDetail, error) {
	workflow, err := getWorkflow(ctx, uc.storage, params.ID, params.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return loadDetail(ctx, uc.storage, workflow)
}
