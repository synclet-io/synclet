package pipelineworkflows

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListConnectionWorkflowsParams holds parameters for listing the workflows of
// connections.
type ListConnectionWorkflowsParams struct {
	WorkspaceID   uuid.UUID
	ConnectionIDs []uuid.UUID
}

// ListConnectionWorkflows lists, per connection, the workflows it is a step of.
type ListConnectionWorkflows struct {
	storage pipelineservice.Storage
}

// NewListConnectionWorkflows creates a new ListConnectionWorkflows use case.
func NewListConnectionWorkflows(storage pipelineservice.Storage) *ListConnectionWorkflows {
	return &ListConnectionWorkflows{storage: storage}
}

// Execute returns the workflows of the workspace each connection is a step
// of, keyed by connection id. A connection in no workflow has no entry.
func (uc *ListConnectionWorkflows) Execute(ctx context.Context, params ListConnectionWorkflowsParams) (map[uuid.UUID][]*pipelineservice.Workflow, error) {
	result := make(map[uuid.UUID][]*pipelineservice.Workflow)
	if len(params.ConnectionIDs) == 0 {
		return result, nil
	}

	steps, err := uc.storage.WorkflowSteps().Find(ctx, &pipelineservice.WorkflowStepFilter{
		ConnectionID: filter.In(params.ConnectionIDs...),
	})
	if err != nil {
		return nil, fmt.Errorf("listing workflow steps of connections: %w", err)
	}

	if len(steps) == 0 {
		return result, nil
	}

	workflowIDs := lo.Uniq(lo.Map(steps, func(step *pipelineservice.WorkflowStep, _ int) uuid.UUID { return step.WorkflowID }))

	workflows, err := uc.storage.Workflows().Find(ctx, &pipelineservice.WorkflowFilter{
		ID:          filter.In(workflowIDs...),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing workflows of connections: %w", err)
	}

	byID := lo.KeyBy(workflows, func(workflow *pipelineservice.Workflow) uuid.UUID { return workflow.ID })

	for _, step := range steps {
		if workflow, ok := byID[step.WorkflowID]; ok {
			result[step.ConnectionID] = append(result[step.ConnectionID], workflow)
		}
	}

	return result, nil
}
