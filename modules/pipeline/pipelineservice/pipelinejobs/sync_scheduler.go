package pipelinejobs

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// SchedulerLockScope names the advisory lock under which jobs are enqueued by
// the scheduler and the workflow advancer.
const SchedulerLockScope = "sync_scheduler"

// SyncScheduler creates scheduled jobs for connections that are due for a sync.
// It acquires an advisory lock to ensure only one instance runs at a time,
// counts active jobs to respect the concurrency limit, and prioritizes
// connections by oldest last-sync completion time.
type SyncScheduler struct {
	storage pipelineservice.Storage
	limit   int
	logger  *logging.Logger
}

// NewSyncScheduler creates a new SyncScheduler.
func NewSyncScheduler(storage pipelineservice.Storage, config pipelineservice.Config, logger *logging.Logger) *SyncScheduler {
	return &SyncScheduler{
		storage: storage,
		limit:   config.MaxConcurrentJobs,
		logger:  logger,
	}
}

// Execute runs one scheduling cycle inside a transaction.
//  1. Acquires advisory lock (blocks while another instance holds it).
//  2. Counts active jobs to get the free slots under the concurrency limit.
//  3. Creates a scheduled job for each due connection, oldest last-sync first.
//  4. Starts due workflows (skipping those with an active run) and advances
//     every running workflow run with the slots left.
//
// Any failed statement aborts the Postgres transaction, so the first error
// rolls back the whole cycle and is returned; the next cycle retries.
func (s *SyncScheduler) Execute(ctx context.Context) error {
	return s.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		err := tx.WithAdvisoryLock(ctx, SchedulerLockScope, 1)
		if err != nil {
			return fmt.Errorf("acquiring scheduler lock: %w", err)
		}

		budget, err := jobBudget(ctx, tx, s.limit)
		if err != nil {
			return err
		}

		created, err := s.scheduleConnections(ctx, tx, budget)
		if err != nil {
			return err
		}

		budget -= created

		if err := startDueWorkflows(ctx, tx, time.Now()); err != nil {
			return err
		}

		running, err := tx.WorkflowRuns().Find(ctx, &pipelineservice.WorkflowRunFilter{
			Status: filter.Equals(pipelineservice.WorkflowRunStatusRunning),
		})
		if err != nil {
			return fmt.Errorf("listing running workflow runs: %w", err)
		}

		runIDs := make([]uuid.UUID, 0, len(running))
		for _, run := range running {
			runIDs = append(runIDs, run.ID)
		}

		return advanceWorkflowRuns(ctx, tx, runIDs, &budget)
	})
}

// startDueWorkflows starts a scheduled run of each workflow whose next
// scheduled time has come, unless the workflow has a run still active, and
// moves next_scheduled_at to the next cron tick either way.
func startDueWorkflows(ctx context.Context, tx pipelineservice.Storage, now time.Time) error {
	due, err := tx.Workflows().Find(ctx, &pipelineservice.WorkflowFilter{
		NextScheduledAt: filter.LessOrEquals(&now),
	})
	if err != nil {
		return fmt.Errorf("finding due workflows: %w", err)
	}

	for _, workflow := range due {
		active, err := tx.WorkflowRuns().Count(ctx, &pipelineservice.WorkflowRunFilter{
			WorkflowID: filter.Equals(workflow.ID),
			Status:     filter.Equals(pipelineservice.WorkflowRunStatusRunning),
		})
		if err != nil {
			return fmt.Errorf("counting active runs of workflow %s: %w", workflow.ID, err)
		}

		if active == 0 {
			if _, err := CreateWorkflowRun(ctx, tx, workflow.ID, pipelineservice.WorkflowRunTriggerSchedule, now); err != nil {
				return err
			}
		}

		workflow.NextScheduledAt = pipelineservice.ComputeNextScheduledAt(workflow.Schedule, now)

		if _, err := tx.Workflows().Update(ctx, workflow); err != nil {
			return fmt.Errorf("advancing next_scheduled_at for workflow %s: %w", workflow.ID, err)
		}
	}

	return nil
}

// scheduleConnections creates a job for each connection due for a sync, up
// to remaining, and returns how many it created.
func (s *SyncScheduler) scheduleConnections(ctx context.Context, tx pipelineservice.Storage, remaining int) (int, error) {
	if remaining <= 0 {
		return 0, nil
	}

	dueConnections, err := tx.Connections().FindDueConnections(ctx, remaining)
	if err != nil {
		return 0, fmt.Errorf("finding due connections: %w", err)
	}

	if len(dueConnections) == 0 {
		return 0, nil
	}

	// Batch-load full connection records to avoid N+1 queries per scheduled connection.
	orFilters := make([]*pipelineservice.ConnectionFilter, len(dueConnections))
	for i, dc := range dueConnections {
		orFilters[i] = &pipelineservice.ConnectionFilter{
			ID: filter.Equals(dc.ConnectionID),
		}
	}

	connRecords, err := tx.Connections().Find(ctx, &pipelineservice.ConnectionFilter{
		Or: orFilters,
	})
	if err != nil {
		return 0, fmt.Errorf("batch loading connections: %w", err)
	}

	connByID := make(map[uuid.UUID]*pipelineservice.Connection, len(connRecords))
	for _, c := range connRecords {
		connByID[c.ID] = c
	}

	now := time.Now()
	created := 0

	for _, conn := range dueConnections {
		connRecord := connByID[conn.ConnectionID]
		if connRecord == nil {
			return 0, fmt.Errorf("due connection %s not found", conn.ConnectionID)
		}

		jobConn := *connRecord
		jobConn.MaxAttempts = conn.MaxAttempts

		if _, err := createScheduledSyncJob(ctx, tx, &jobConn, nil, now); err != nil {
			return 0, err
		}

		pipelineservice.RecomputeNextScheduledAt(connRecord, now)

		if _, err := tx.Connections().Update(ctx, connRecord); err != nil {
			return 0, fmt.Errorf("advancing next_scheduled_at for connection %s: %w", conn.ConnectionID, err)
		}

		created++
	}

	if created > 0 {
		s.logger.WithField("count", created).Info(ctx, "scheduled jobs")
	}

	return created, nil
}
