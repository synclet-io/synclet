package pipelineworkflows

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

const stepErrorStepRemoved = "skipped: the step was removed from the workflow"

// RerunWorkflowRunParams holds parameters for rerunning a workflow run.
type RerunWorkflowRunParams struct {
	RunID       uuid.UUID
	WorkspaceID uuid.UUID
}

// RerunWorkflowRun reruns a failed run from its failed steps.
type RerunWorkflowRun struct {
	storage pipelineservice.Storage
	audit   pipelineservice.AuditRecorder
}

// NewRerunWorkflowRun creates a new RerunWorkflowRun use case.
func NewRerunWorkflowRun(storage pipelineservice.Storage, audit pipelineservice.AuditRecorder) *RerunWorkflowRun {
	return &RerunWorkflowRun{storage: storage, audit: audit}
}

// Execute puts the run's failed and skipped steps back to pending, reopens
// the run and, in the same transaction, sends a workflow_run.started event;
// the jobs process enqueues the steps that are ready when it consumes the
// event. Steps that succeeded keep
// their result and are not synced again. A step removed from the workflow
// since the run is skipped; a step added since joins the run as pending. It
// returns ErrWorkflowRunNotFailed for a run that did not fail,
// ErrWorkflowRunActive while another run of the workflow is running, and a
// *NoConfiguredStreamsError when a step the rerun syncs has a connection with
// no configured stream.
func (uc *RerunWorkflowRun) Execute(ctx context.Context, params RerunWorkflowRunParams) (*WorkflowRunDetail, error) {
	var (
		detail       *WorkflowRunDetail
		workflowName string
	)

	err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		if err := lockWorkflowRuns(ctx, tx); err != nil {
			return err
		}

		run, workflow, err := getRun(ctx, tx, params.RunID, params.WorkspaceID)
		if err != nil {
			return err
		}

		workflowName = workflow.Name

		if run.Status != pipelineservice.WorkflowRunStatusFailed {
			return pipelineservice.ErrWorkflowRunNotFailed
		}

		active, err := hasActiveRun(ctx, tx, run.WorkflowID)
		if err != nil {
			return err
		}

		if active {
			return pipelineservice.ErrWorkflowRunActive
		}

		if err := requireRerunnableSteps(ctx, tx, run); err != nil {
			return err
		}

		if err := resetStepRuns(ctx, tx, run); err != nil {
			return err
		}

		run.Status = pipelineservice.WorkflowRunStatusRunning
		run.CompletedAt = nil

		if _, err := tx.WorkflowRuns().Update(ctx, run); err != nil {
			return fmt.Errorf("reopening run: %w", err)
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
		Action:        "rerun",
		ResourceType:  "workflow",
		ResourceID:    detail.Run.WorkflowID,
		ResourceLabel: workflowName,
		After:         map[string]any{"run_id": detail.Run.ID.String()},
	})

	return detail, nil
}

// requireRerunnableSteps checks the connections of the steps a rerun syncs:
// the workflow's steps whose step run in the run did not succeed.
func requireRerunnableSteps(ctx context.Context, tx pipelineservice.Storage, run *pipelineservice.WorkflowRun) error {
	steps, err := tx.WorkflowSteps().Find(ctx, &pipelineservice.WorkflowStepFilter{WorkflowID: filter.Equals(run.WorkflowID)})
	if err != nil {
		return fmt.Errorf("listing steps: %w", err)
	}

	stepRuns, err := tx.WorkflowStepRuns().Find(ctx, &pipelineservice.WorkflowStepRunFilter{RunID: filter.Equals(run.ID)})
	if err != nil {
		return fmt.Errorf("listing step runs: %w", err)
	}

	succeeded := make(map[uuid.UUID]bool, len(stepRuns))
	for _, stepRun := range stepRuns {
		if stepRun.Status == pipelineservice.WorkflowStepRunStatusSucceeded {
			succeeded[stepRun.StepID] = true
		}
	}

	toSync := make([]*pipelineservice.WorkflowStep, 0, len(steps))
	for _, step := range steps {
		if !succeeded[step.ID] {
			toSync = append(toSync, step)
		}
	}

	return requireRunnableSteps(ctx, tx, toSync)
}

func resetStepRuns(ctx context.Context, tx pipelineservice.Storage, run *pipelineservice.WorkflowRun) error {
	steps, err := tx.WorkflowSteps().Find(ctx, &pipelineservice.WorkflowStepFilter{WorkflowID: filter.Equals(run.WorkflowID)})
	if err != nil {
		return fmt.Errorf("listing steps: %w", err)
	}

	current := make(map[uuid.UUID]*pipelineservice.WorkflowStep, len(steps))
	for _, step := range steps {
		current[step.ID] = step
	}

	stepRuns, err := tx.WorkflowStepRuns().Find(ctx, &pipelineservice.WorkflowStepRunFilter{RunID: filter.Equals(run.ID)})
	if err != nil {
		return fmt.Errorf("listing step runs: %w", err)
	}

	now := time.Now()
	inRun := make(map[uuid.UUID]bool, len(stepRuns))

	for _, stepRun := range stepRuns {
		inRun[stepRun.StepID] = true

		if stepRun.Status != pipelineservice.WorkflowStepRunStatusFailed && stepRun.Status != pipelineservice.WorkflowStepRunStatusSkipped {
			continue
		}

		if current[stepRun.StepID] == nil {
			reason := stepErrorStepRemoved
			stepRun.Status = pipelineservice.WorkflowStepRunStatusSkipped
			stepRun.Error = &reason
			stepRun.CompletedAt = &now
		} else {
			stepRun.Status = pipelineservice.WorkflowStepRunStatusPending
			stepRun.JobID = nil
			stepRun.Error = nil
			stepRun.StartedAt = nil
			stepRun.CompletedAt = nil
		}

		if _, err := tx.WorkflowStepRuns().Update(ctx, stepRun); err != nil {
			return fmt.Errorf("resetting step run %s: %w", stepRun.ID, err)
		}
	}

	for _, step := range steps {
		if inRun[step.ID] {
			continue
		}

		if _, err := tx.WorkflowStepRuns().Create(ctx, &pipelineservice.WorkflowStepRun{
			ID:           uuid.New(),
			RunID:        run.ID,
			StepID:       step.ID,
			ConnectionID: step.ConnectionID,
			Status:       pipelineservice.WorkflowStepRunStatusPending,
		}); err != nil {
			return fmt.Errorf("adding step run for step %s: %w", step.ID, err)
		}
	}

	return nil
}

// getRun returns a run with its workflow when the workflow belongs to the
// workspace, or ErrWorkflowRunNotFound.
func getRun(
	ctx context.Context,
	storage pipelineservice.Storage,
	runID, workspaceID uuid.UUID,
) (*pipelineservice.WorkflowRun, *pipelineservice.Workflow, error) {
	run, err := storage.WorkflowRuns().First(ctx, &pipelineservice.WorkflowRunFilter{ID: filter.Equals(runID)})
	if err != nil {
		return nil, nil, fmt.Errorf("getting workflow run: %w", err)
	}

	workflows, err := storage.Workflows().Find(ctx, &pipelineservice.WorkflowFilter{
		ID:          filter.Equals(run.WorkflowID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("checking workflow of run: %w", err)
	}

	if len(workflows) == 0 {
		return nil, nil, pipelineservice.ErrWorkflowRunNotFound
	}

	return run, workflows[0], nil
}
