package pipelineworkflows

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// CreateWorkflowParams holds parameters for creating a workflow.
type CreateWorkflowParams struct {
	WorkspaceID uuid.UUID
	Name        string
	Schedule    *string
	Steps       []pipelineservice.WorkflowGraphStep
}

// CreateWorkflow creates a workflow with its steps.
type CreateWorkflow struct {
	storage pipelineservice.Storage
	audit   pipelineservice.AuditRecorder
}

// NewCreateWorkflow creates a new CreateWorkflow use case.
func NewCreateWorkflow(storage pipelineservice.Storage, audit pipelineservice.AuditRecorder) *CreateWorkflow {
	return &CreateWorkflow{storage: storage, audit: audit}
}

// Execute validates the definition and stores the workflow. A cycle, a
// connection listed twice or a connection of another workspace is a
// *ValidationError.
func (uc *CreateWorkflow) Execute(ctx context.Context, params CreateWorkflowParams) (*WorkflowDetail, error) {
	var detail *WorkflowDetail

	err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		if err := validateDefinition(ctx, tx, params.WorkspaceID, params.Name, params.Schedule, params.Steps); err != nil {
			return err
		}

		now := time.Now()
		schedule := normalizeSchedule(params.Schedule)

		workflow, err := tx.Workflows().Create(ctx, &pipelineservice.Workflow{
			ID:              uuid.New(),
			WorkspaceID:     params.WorkspaceID,
			Name:            params.Name,
			Schedule:        schedule,
			NextScheduledAt: pipelineservice.ComputeNextScheduledAt(schedule, now),
			CreatedAt:       now,
			UpdatedAt:       now,
		})
		if err != nil {
			return fmt.Errorf("creating workflow: %w", err)
		}

		if err := saveSteps(ctx, tx, workflow.ID, params.Steps); err != nil {
			return err
		}

		detail, err = loadDetail(ctx, tx, workflow)

		return err
	})
	if err != nil {
		return nil, err
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "create",
		ResourceType:  "workflow",
		ResourceID:    detail.Workflow.ID,
		ResourceLabel: detail.Workflow.Name,
		After:         workflowToAuditPayload(detail),
	})

	return detail, nil
}
