package pipelineworkflows

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// UpdateWorkflowParams holds parameters for updating a workflow.
type UpdateWorkflowParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Schedule    *string
	Steps       []pipelineservice.WorkflowGraphStep
}

// UpdateWorkflow replaces a workflow's name, schedule and steps.
type UpdateWorkflow struct {
	storage pipelineservice.Storage
	audit   pipelineservice.AuditRecorder
}

// NewUpdateWorkflow creates a new UpdateWorkflow use case.
func NewUpdateWorkflow(storage pipelineservice.Storage, audit pipelineservice.AuditRecorder) *UpdateWorkflow {
	return &UpdateWorkflow{storage: storage, audit: audit}
}

// Execute validates the definition as CreateWorkflow does and stores it. The
// step of a connection that stays keeps its id. It returns
// ErrWorkflowRunActive while the workflow has a running run.
func (uc *UpdateWorkflow) Execute(ctx context.Context, params UpdateWorkflowParams) (*WorkflowDetail, error) {
	var before, detail *WorkflowDetail

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

		if err := validateDefinition(ctx, tx, params.WorkspaceID, params.Name, params.Schedule, params.Steps); err != nil {
			return err
		}

		before, err = loadDetail(ctx, tx, workflow)
		if err != nil {
			return err
		}

		previous := *workflow
		before.Workflow = &previous

		now := time.Now()
		workflow.Name = params.Name
		workflow.Schedule = normalizeSchedule(params.Schedule)
		workflow.NextScheduledAt = pipelineservice.ComputeNextScheduledAt(workflow.Schedule, now)
		workflow.UpdatedAt = now

		if _, err := tx.Workflows().Update(ctx, workflow); err != nil {
			return fmt.Errorf("updating workflow: %w", err)
		}

		if err := handOverSchedules(ctx, tx, params.WorkspaceID, params.Steps); err != nil {
			return err
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
		Action:        "update",
		ResourceType:  "workflow",
		ResourceID:    detail.Workflow.ID,
		ResourceLabel: detail.Workflow.Name,
		Before:        workflowToAuditPayload(before),
		After:         workflowToAuditPayload(detail),
	})

	return detail, nil
}
