package pipelineworkflows

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// DeleteWorkflowParams holds parameters for deleting a workflow.
type DeleteWorkflowParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// DeleteWorkflow deletes a workflow with its steps and runs.
type DeleteWorkflow struct {
	storage pipelineservice.Storage
	audit   pipelineservice.AuditRecorder
}

// NewDeleteWorkflow creates a new DeleteWorkflow use case.
func NewDeleteWorkflow(storage pipelineservice.Storage, audit pipelineservice.AuditRecorder) *DeleteWorkflow {
	return &DeleteWorkflow{storage: storage, audit: audit}
}

// Execute deletes the workflow. It returns ErrWorkflowRunActive while the
// workflow has a running run.
func (uc *DeleteWorkflow) Execute(ctx context.Context, params DeleteWorkflowParams) error {
	var deleted *WorkflowDetail

	err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		if err := lockWorkflowRuns(ctx, tx); err != nil {
			return err
		}

		workflow, err := getWorkflow(ctx, tx, params.ID, params.WorkspaceID)
		if err != nil {
			return err
		}

		active, err := hasActiveRun(ctx, tx, workflow.ID)
		if err != nil {
			return err
		}

		if active {
			return pipelineservice.ErrWorkflowRunActive
		}

		deleted, err = loadDetail(ctx, tx, workflow)
		if err != nil {
			return err
		}

		if err := tx.WorkflowStepDependencies().Delete(ctx, &pipelineservice.WorkflowStepDependencyFilter{
			WorkflowID: filter.Equals(workflow.ID),
		}); err != nil {
			return fmt.Errorf("deleting step dependencies: %w", err)
		}

		if err := tx.WorkflowSteps().Delete(ctx, &pipelineservice.WorkflowStepFilter{WorkflowID: filter.Equals(workflow.ID)}); err != nil {
			return fmt.Errorf("deleting steps: %w", err)
		}

		if err := tx.Workflows().Delete(ctx, &pipelineservice.WorkflowFilter{
			ID:          filter.Equals(workflow.ID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		}); err != nil {
			return fmt.Errorf("deleting workflow: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "delete",
		ResourceType:  "workflow",
		ResourceID:    deleted.Workflow.ID,
		ResourceLabel: deleted.Workflow.Name,
		Before:        workflowToAuditPayload(deleted),
	})

	return nil
}
