package pipelineworkflows

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
)

// StartWorkflowRunParams holds parameters for starting a workflow run.
type StartWorkflowRunParams struct {
	WorkflowID  uuid.UUID
	WorkspaceID uuid.UUID
}

// StartWorkflowRun starts a manual run of a workflow, from the UI or the API.
type StartWorkflowRun struct {
	storage pipelineservice.Storage
	audit   pipelineservice.AuditRecorder
}

// NewStartWorkflowRun creates a new StartWorkflowRun use case.
func NewStartWorkflowRun(storage pipelineservice.Storage, audit pipelineservice.AuditRecorder) *StartWorkflowRun {
	return &StartWorkflowRun{storage: storage, audit: audit}
}

// Execute creates the run with every step pending and, in the same
// transaction, sends a workflow_run.started event; the jobs process enqueues
// the run's first steps when it consumes the event. It returns
// ErrWorkflowRunActive while the workflow has a running run.
func (uc *StartWorkflowRun) Execute(ctx context.Context, params StartWorkflowRunParams) (*WorkflowRunDetail, error) {
	var (
		detail       *WorkflowRunDetail
		workflowName string
	)

	err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		if err := lockWorkflowRuns(ctx, tx); err != nil {
			return err
		}

		workflow, err := getWorkflow(ctx, tx, params.WorkflowID, params.WorkspaceID)
		if err != nil {
			return err
		}

		workflowName = workflow.Name

		active, err := hasActiveRun(ctx, tx, workflow.ID)
		if err != nil {
			return err
		}

		if active {
			return pipelineservice.ErrWorkflowRunActive
		}

		run, err := pipelinejobs.CreateWorkflowRun(ctx, tx, workflow.ID, pipelineservice.WorkflowRunTriggerManual, time.Now())
		if err != nil {
			return err
		}

		if err := sendRunStarted(ctx, tx, workflow.WorkspaceID, run); err != nil {
			return err
		}

		detail, err = loadRunDetail(ctx, tx, run.ID)

		return err
	})
	if err != nil {
		return nil, err
	}

	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   params.WorkspaceID,
		Action:        "run",
		ResourceType:  "workflow",
		ResourceID:    detail.Run.WorkflowID,
		ResourceLabel: workflowName,
		After:         map[string]any{"run_id": detail.Run.ID.String()},
	})

	return detail, nil
}
