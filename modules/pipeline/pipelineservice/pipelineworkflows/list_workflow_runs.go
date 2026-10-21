package pipelineworkflows

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListWorkflowRunsParams holds parameters for listing a workflow's runs.
type ListWorkflowRunsParams struct {
	WorkflowID  uuid.UUID
	WorkspaceID uuid.UUID
}

// ListWorkflowRuns lists a workflow's runs, newest first, with their step runs.
type ListWorkflowRuns struct {
	storage pipelineservice.Storage
}

// NewListWorkflowRuns creates a new ListWorkflowRuns use case.
func NewListWorkflowRuns(storage pipelineservice.Storage) *ListWorkflowRuns {
	return &ListWorkflowRuns{storage: storage}
}

// Execute returns the runs, or ErrWorkflowNotFound.
func (uc *ListWorkflowRuns) Execute(ctx context.Context, params ListWorkflowRunsParams) ([]*WorkflowRunDetail, error) {
	workflow, err := getWorkflow(ctx, uc.storage, params.WorkflowID, params.WorkspaceID)
	if err != nil {
		return nil, err
	}

	runs, err := uc.storage.WorkflowRuns().Find(ctx, &pipelineservice.WorkflowRunFilter{
		WorkflowID: filter.Equals(workflow.ID),
	}, dbutil.WithOrder(pipelineservice.WorkflowRunFieldStartedAt, dbutil.OrderDirDesc))
	if err != nil {
		return nil, fmt.Errorf("listing workflow runs: %w", err)
	}

	if len(runs) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(runs))
	details := make(map[uuid.UUID]*WorkflowRunDetail, len(runs))
	result := make([]*WorkflowRunDetail, 0, len(runs))

	for _, run := range runs {
		ids = append(ids, run.ID)
		detail := &WorkflowRunDetail{Run: run}
		details[run.ID] = detail
		result = append(result, detail)
	}

	stepRuns, err := uc.storage.WorkflowStepRuns().Find(ctx, &pipelineservice.WorkflowStepRunFilter{RunID: filter.In(ids...)})
	if err != nil {
		return nil, fmt.Errorf("listing step runs: %w", err)
	}

	for _, stepRun := range stepRuns {
		details[stepRun.RunID].StepRuns = append(details[stepRun.RunID].StepRuns, stepRun)
	}

	return result, nil
}
