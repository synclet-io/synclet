package pipelinejobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

const (
	stepErrorJobMissing     = "the step's job no longer exists"
	stepErrorJobCancelled   = "the step's job was cancelled"
	stepErrorJobFailed      = "the step's job failed"
	stepErrorUpstreamFailed = "skipped: an upstream step did not succeed"
)

// AdvanceWorkflowRuns moves workflow runs forward: it settles the steps whose
// job reached a final state, skips what lies downstream of a failed or
// skipped step, enqueues sync jobs for the steps that became ready and
// closes runs that have nothing left to do. It runs under the scheduler lock
// and enqueues no more jobs than the concurrency cap leaves free.
type AdvanceWorkflowRuns struct {
	storage pipelineservice.Storage
	limit   int
	logger  *logging.Logger
}

// NewAdvanceWorkflowRuns creates the use case.
func NewAdvanceWorkflowRuns(storage pipelineservice.Storage, config pipelineservice.Config, logger *logging.Logger) *AdvanceWorkflowRuns {
	return &AdvanceWorkflowRuns{storage: storage, limit: config.MaxConcurrentJobs, logger: logger}
}

// Execute advances the given runs in one transaction.
func (uc *AdvanceWorkflowRuns) Execute(ctx context.Context, runIDs ...uuid.UUID) error {
	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		return uc.AdvanceInTx(ctx, tx, runIDs...)
	})
}

// ExecuteForJob advances the runs a job's end can move: the run of the step
// the job belongs to, and every running run with a step waiting on the job's
// connection.
func (uc *AdvanceWorkflowRuns) ExecuteForJob(ctx context.Context, job *pipelineservice.Job) error {
	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		if err := tx.WithAdvisoryLock(ctx, SchedulerLockScope, 1); err != nil {
			return fmt.Errorf("acquiring scheduler lock: %w", err)
		}

		runIDs, err := runsAffectedByJob(ctx, tx, job)
		if err != nil {
			return err
		}

		if len(runIDs) == 0 {
			return nil
		}

		budget, err := jobBudget(ctx, tx, uc.limit)
		if err != nil {
			return err
		}

		return advanceWorkflowRuns(ctx, tx, runIDs, &budget)
	})
}

func runsAffectedByJob(ctx context.Context, tx pipelineservice.Storage, job *pipelineservice.Job) ([]uuid.UUID, error) {
	var runIDs []uuid.UUID

	seen := map[uuid.UUID]bool{}
	add := func(id uuid.UUID) {
		if !seen[id] {
			seen[id] = true
			runIDs = append(runIDs, id)
		}
	}

	if job.WorkflowStepRunID != nil {
		stepRun, err := tx.WorkflowStepRuns().First(ctx, &pipelineservice.WorkflowStepRunFilter{
			ID: filter.Equals(*job.WorkflowStepRunID),
		})

		switch {
		case errors.Is(err, pipelineservice.ErrWorkflowStepRunNotFound):
		case err != nil:
			return nil, fmt.Errorf("getting step run of job %s: %w", job.ID, err)
		default:
			add(stepRun.RunID)
		}
	}

	waiting, err := tx.WorkflowStepRuns().Find(ctx, &pipelineservice.WorkflowStepRunFilter{
		ConnectionID: filter.Equals(job.ConnectionID),
		Status:       filter.Equals(pipelineservice.WorkflowStepRunStatusPending),
	})
	if err != nil {
		return nil, fmt.Errorf("finding steps waiting on connection %s: %w", job.ConnectionID, err)
	}

	for _, stepRun := range waiting {
		add(stepRun.RunID)
	}

	return runIDs, nil
}

func jobBudget(ctx context.Context, tx pipelineservice.Storage, limit int) (int, error) {
	active, err := tx.Jobs().CountActiveJobs(ctx)
	if err != nil {
		return 0, fmt.Errorf("counting active jobs: %w", err)
	}

	return max(limit-active, 0), nil
}

// advanceWorkflowRuns advances each run inside the caller's transaction,
// which must hold the scheduler lock. budget is the number of jobs it may
// still create, and is decreased for each one.
func advanceWorkflowRuns(ctx context.Context, tx pipelineservice.Storage, runIDs []uuid.UUID, budget *int) error {
	for _, runID := range runIDs {
		if err := advanceWorkflowRun(ctx, tx, runID, budget); err != nil {
			return fmt.Errorf("advancing workflow run %s: %w", runID, err)
		}
	}

	return nil
}

func advanceWorkflowRun(ctx context.Context, tx pipelineservice.Storage, runID uuid.UUID, budget *int) error {
	run, err := tx.WorkflowRuns().First(ctx, &pipelineservice.WorkflowRunFilter{ID: filter.Equals(runID)})
	if errors.Is(err, pipelineservice.ErrWorkflowRunNotFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("getting run: %w", err)
	}

	if run.Status != pipelineservice.WorkflowRunStatusRunning {
		return nil
	}

	stepRuns, err := tx.WorkflowStepRuns().Find(ctx, &pipelineservice.WorkflowStepRunFilter{RunID: filter.Equals(run.ID)})
	if err != nil {
		return fmt.Errorf("listing step runs: %w", err)
	}

	deps, err := tx.WorkflowStepDependencies().Find(ctx, &pipelineservice.WorkflowStepDependencyFilter{
		WorkflowID: filter.Equals(run.WorkflowID),
	})
	if err != nil {
		return fmt.Errorf("listing step dependencies: %w", err)
	}

	state := &runState{
		byStep:  make(map[uuid.UUID]*pipelineservice.WorkflowStepRun, len(stepRuns)),
		changed: map[uuid.UUID]bool{},
		deps:    deps,
		now:     time.Now(),
	}
	for _, stepRun := range stepRuns {
		state.byStep[stepRun.StepID] = stepRun
	}

	if err := state.settleQueued(ctx, tx); err != nil {
		return err
	}

	if err := state.enqueueReady(ctx, tx, budget); err != nil {
		return err
	}

	for _, stepRun := range stepRuns {
		if !state.changed[stepRun.StepID] {
			continue
		}

		if _, err := tx.WorkflowStepRuns().Update(ctx, stepRun); err != nil {
			return fmt.Errorf("updating step run %s: %w", stepRun.ID, err)
		}
	}

	outcome := pipelineservice.RunOutcome(state.statuses())
	if outcome == pipelineservice.WorkflowRunStatusRunning {
		return nil
	}

	run.Status = outcome
	run.CompletedAt = &state.now

	if _, err := tx.WorkflowRuns().Update(ctx, run); err != nil {
		return fmt.Errorf("completing run: %w", err)
	}

	return publishRunFinished(ctx, tx, run, stepRuns)
}

// publishRunFinished sends workflow_run.finished for a run that reached its
// final status. Error is the error text of the first failed step that has
// one.
func publishRunFinished(
	ctx context.Context,
	tx pipelineservice.Storage,
	run *pipelineservice.WorkflowRun,
	stepRuns []*pipelineservice.WorkflowStepRun,
) error {
	workflow, err := tx.Workflows().First(ctx, &pipelineservice.WorkflowFilter{ID: filter.Equals(run.WorkflowID)})
	if err != nil {
		return fmt.Errorf("getting workflow %s: %w", run.WorkflowID, err)
	}

	connectionIDs := make([]uuid.UUID, 0, len(stepRuns))

	var firstFailed *pipelineservice.WorkflowStepRun

	for _, stepRun := range stepRuns {
		connectionIDs = append(connectionIDs, stepRun.ConnectionID)

		if stepRun.Status != pipelineservice.WorkflowStepRunStatusFailed || stepRun.Error == nil || *stepRun.Error == "" {
			continue
		}

		if firstFailed == nil || completedBefore(stepRun, firstFailed) {
			firstFailed = stepRun
		}
	}

	var runError *string
	if firstFailed != nil {
		runError = firstFailed.Error
	}

	if err := tx.PipelineEvents().Send(ctx, &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeWorkflowRunFinished,
		Data: &pipelineservice.WorkflowRunFinishedEventData{
			WorkspaceID:   workflow.WorkspaceID,
			WorkflowID:    workflow.ID,
			WorkflowName:  workflow.Name,
			RunID:         run.ID,
			Status:        run.Status,
			Error:         runError,
			ConnectionIDs: connectionIDs,
		},
	}); err != nil {
		return fmt.Errorf("publishing workflow_run.finished event for run %s: %w", run.ID, err)
	}

	return nil
}

func completedBefore(a, b *pipelineservice.WorkflowStepRun) bool {
	if a.CompletedAt == nil || b.CompletedAt == nil {
		return a.CompletedAt != nil
	}

	return a.CompletedAt.Before(*b.CompletedAt)
}

type runState struct {
	byStep  map[uuid.UUID]*pipelineservice.WorkflowStepRun
	changed map[uuid.UUID]bool
	deps    []*pipelineservice.WorkflowStepDependency
	now     time.Time
}

func (s *runState) statuses() map[uuid.UUID]pipelineservice.WorkflowStepRunStatus {
	statuses := make(map[uuid.UUID]pipelineservice.WorkflowStepRunStatus, len(s.byStep))
	for stepID, stepRun := range s.byStep {
		statuses[stepID] = stepRun.Status
	}

	return statuses
}

func (s *runState) finish(stepRun *pipelineservice.WorkflowStepRun, status pipelineservice.WorkflowStepRunStatus, reason *string, at time.Time) {
	stepRun.Status = status
	stepRun.Error = reason
	stepRun.CompletedAt = &at
	s.changed[stepRun.StepID] = true

	if status == pipelineservice.WorkflowStepRunStatusSucceeded {
		return
	}

	for _, downstreamID := range pipelineservice.Downstream(stepRun.StepID, s.deps) {
		downstream := s.byStep[downstreamID]
		if downstream == nil || downstream.Status != pipelineservice.WorkflowStepRunStatusPending {
			continue
		}

		skipReason := stepErrorUpstreamFailed
		downstream.Status = pipelineservice.WorkflowStepRunStatusSkipped
		downstream.Error = &skipReason
		downstream.CompletedAt = &s.now
		s.changed[downstreamID] = true
	}
}

// settleQueued gives each queued step the outcome of its job once the job
// is final; a job between attempts keeps its step queued.
func (s *runState) settleQueued(ctx context.Context, tx pipelineservice.Storage) error {
	for _, stepRun := range s.byStep {
		if stepRun.Status != pipelineservice.WorkflowStepRunStatusQueued || stepRun.JobID == nil {
			continue
		}

		job, err := tx.Jobs().First(ctx, &pipelineservice.JobFilter{ID: filter.Equals(*stepRun.JobID)})
		if errors.Is(err, pipelineservice.ErrJobNotFound) {
			reason := stepErrorJobMissing
			s.finish(stepRun, pipelineservice.WorkflowStepRunStatusFailed, &reason, s.now)

			continue
		}

		if err != nil {
			return fmt.Errorf("getting job of step run %s: %w", stepRun.ID, err)
		}

		completedAt := s.now
		if job.CompletedAt != nil {
			completedAt = *job.CompletedAt
		}

		switch job.Status {
		case pipelineservice.JobStatusCompleted:
			s.finish(stepRun, pipelineservice.WorkflowStepRunStatusSucceeded, nil, completedAt)
		case pipelineservice.JobStatusFailed:
			reason := stepErrorJobFailed
			if job.Error != nil && *job.Error != "" {
				reason = *job.Error
			}

			s.finish(stepRun, pipelineservice.WorkflowStepRunStatusFailed, &reason, completedAt)
		case pipelineservice.JobStatusCancelled:
			reason := stepErrorJobCancelled
			s.finish(stepRun, pipelineservice.WorkflowStepRunStatusFailed, &reason, completedAt)
		case pipelineservice.JobStatusScheduled, pipelineservice.JobStatusStarting, pipelineservice.JobStatusRunning:
		}
	}

	return nil
}

// enqueueReady creates a sync job for each ready step whose connection is
// active and has no active job, while budget lasts. A step of a connection
// that is not active is skipped with everything downstream of it; a step of
// a connection with no configured stream fails, and what lies downstream of
// it is skipped.
func (s *runState) enqueueReady(ctx context.Context, tx pipelineservice.Storage, budget *int) error {
	for _, stepID := range pipelineservice.ReadySteps(s.statuses(), s.deps) {
		stepRun := s.byStep[stepID]

		conn, err := tx.Connections().First(ctx, &pipelineservice.ConnectionFilter{ID: filter.Equals(stepRun.ConnectionID)})
		if err != nil {
			return fmt.Errorf("getting connection of step run %s: %w", stepRun.ID, err)
		}

		if conn.Status != pipelineservice.ConnectionStatusActive {
			reason := "skipped: connection is " + conn.Status.String()
			s.finish(stepRun, pipelineservice.WorkflowStepRunStatusSkipped, &reason, s.now)

			continue
		}

		active, err := tx.Jobs().Count(ctx, &pipelineservice.JobFilter{
			ConnectionID: filter.Equals(conn.ID),
			Status: filter.In(
				pipelineservice.JobStatusScheduled,
				pipelineservice.JobStatusStarting,
				pipelineservice.JobStatusRunning,
			),
		})
		if err != nil {
			return fmt.Errorf("counting active jobs of connection %s: %w", conn.ID, err)
		}

		if active > 0 || *budget <= 0 {
			continue
		}

		if err := pipelineservice.RequireConfiguredStreams(ctx, tx, conn); err != nil {
			var noStreams *pipelineservice.NoConfiguredStreamsError
			if !errors.As(err, &noStreams) {
				return err
			}

			reason := noStreams.Error()
			s.finish(stepRun, pipelineservice.WorkflowStepRunStatusFailed, &reason, s.now)

			continue
		}

		job, err := createScheduledSyncJob(ctx, tx, conn, &stepRun.ID, s.now)
		if err != nil {
			return err
		}

		*budget--
		stepRun.Status = pipelineservice.WorkflowStepRunStatusQueued
		stepRun.JobID = &job.ID
		stepRun.Error = nil
		stepRun.StartedAt = &s.now
		stepRun.CompletedAt = nil
		s.changed[stepID] = true
	}

	return nil
}

// createScheduledSyncJob enqueues a sync job for the connection and
// publishes job.created. stepRunID ties the job to a workflow step run.
func createScheduledSyncJob(
	ctx context.Context,
	tx pipelineservice.Storage,
	conn *pipelineservice.Connection,
	stepRunID *uuid.UUID,
	now time.Time,
) (*pipelineservice.Job, error) {
	created, err := tx.Jobs().Create(ctx, &pipelineservice.Job{
		ID:                uuid.New(),
		ConnectionID:      conn.ID,
		Status:            pipelineservice.JobStatusScheduled,
		JobType:           pipelineservice.JobTypeSync,
		ScheduledAt:       now,
		Attempt:           1,
		MaxAttempts:       conn.MaxAttempts,
		WorkflowStepRunID: stepRunID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating scheduled job for connection %s: %w", conn.ID, err)
	}

	if err := tx.PipelineEvents().Send(ctx, &pipelineservice.PipelineEvent{
		EventType: pipelineservice.PipelineEventTypeJobCreated,
		Data: &pipelineservice.JobCreatedEventData{
			WorkspaceID: conn.WorkspaceID,
			Job:         created,
		},
	}); err != nil {
		return nil, fmt.Errorf("publishing job.created event for connection %s: %w", conn.ID, err)
	}

	return created, nil
}

// AdvanceInTx advances the given runs inside the caller's transaction.
func (uc *AdvanceWorkflowRuns) AdvanceInTx(ctx context.Context, tx pipelineservice.Storage, runIDs ...uuid.UUID) error {
	if err := tx.WithAdvisoryLock(ctx, SchedulerLockScope, 1); err != nil {
		return fmt.Errorf("acquiring scheduler lock: %w", err)
	}

	budget, err := jobBudget(ctx, tx, uc.limit)
	if err != nil {
		return err
	}

	return advanceWorkflowRuns(ctx, tx, runIDs, &budget)
}

// CreateWorkflowRun starts a run of the workflow with one pending step run
// per step. The caller advances it.
func CreateWorkflowRun(
	ctx context.Context,
	tx pipelineservice.Storage,
	workflowID uuid.UUID,
	trigger pipelineservice.WorkflowRunTrigger,
	now time.Time,
) (*pipelineservice.WorkflowRun, error) {
	steps, err := tx.WorkflowSteps().Find(ctx, &pipelineservice.WorkflowStepFilter{WorkflowID: filter.Equals(workflowID)})
	if err != nil {
		return nil, fmt.Errorf("listing steps of workflow %s: %w", workflowID, err)
	}

	run, err := tx.WorkflowRuns().Create(ctx, &pipelineservice.WorkflowRun{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     pipelineservice.WorkflowRunStatusRunning,
		Trigger:    trigger,
		StartedAt:  now,
		CreatedAt:  now,
	})
	if err != nil {
		return nil, fmt.Errorf("creating run of workflow %s: %w", workflowID, err)
	}

	stepRuns := make([]*pipelineservice.WorkflowStepRun, 0, len(steps))
	for _, step := range steps {
		stepRuns = append(stepRuns, &pipelineservice.WorkflowStepRun{
			ID:           uuid.New(),
			RunID:        run.ID,
			StepID:       step.ID,
			ConnectionID: step.ConnectionID,
			Status:       pipelineservice.WorkflowStepRunStatusPending,
		})
	}

	if len(stepRuns) > 0 {
		if _, err := tx.WorkflowStepRuns().BatchCreate(ctx, stepRuns); err != nil {
			return nil, fmt.Errorf("creating step runs of run %s: %w", run.ID, err)
		}
	}

	return run, nil
}
