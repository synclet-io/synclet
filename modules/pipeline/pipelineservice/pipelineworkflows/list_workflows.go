package pipelineworkflows

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListWorkflowsParams holds parameters for listing workflows.
type ListWorkflowsParams struct {
	WorkspaceID uuid.UUID
}

// ListWorkflows lists the workflows of a workspace by name.
type ListWorkflows struct {
	storage pipelineservice.Storage
}

// NewListWorkflows creates a new ListWorkflows use case.
func NewListWorkflows(storage pipelineservice.Storage) *ListWorkflows {
	return &ListWorkflows{storage: storage}
}

// Execute returns the workspace's workflows with their steps.
func (uc *ListWorkflows) Execute(ctx context.Context, params ListWorkflowsParams) ([]*WorkflowDetail, error) {
	workflows, err := uc.storage.Workflows().Find(ctx, &pipelineservice.WorkflowFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}, dbutil.WithOrder(pipelineservice.WorkflowFieldName, dbutil.OrderDirAsc))
	if err != nil {
		return nil, fmt.Errorf("listing workflows: %w", err)
	}

	return loadDetails(ctx, uc.storage, workflows)
}
