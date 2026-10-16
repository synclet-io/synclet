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
// 1. Acquires advisory lock (returns nil if another instance holds it).
// 2. Counts active jobs and checks against limit.
// 3. Finds due connections ordered by oldest last-sync completion.
// 4. Creates a scheduled job for each due connection.
// Any failed statement aborts the Postgres transaction, so the first error
// rolls back the whole cycle and is returned; the next cycle retries.
func (s *SyncScheduler) Execute(ctx context.Context) error {
	return s.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		err := tx.WithAdvisoryLock(ctx, "sync_scheduler", 1)
		if err != nil {
			return fmt.Errorf("acquiring scheduler lock: %w", err)
		}

		// Count active jobs (scheduled + starting + running).
		activeCount, err := tx.Jobs().CountActiveJobs(ctx)
		if err != nil {
			return fmt.Errorf("counting active jobs: %w", err)
		}

		remaining := s.limit - activeCount
		if remaining <= 0 {
			return nil
		}

		// Find connections due for sync, ordered by oldest completion time.
		dueConnections, err := tx.Connections().FindDueConnections(ctx, remaining)
		if err != nil {
			return fmt.Errorf("finding due connections: %w", err)
		}

		if len(dueConnections) == 0 {
			return nil
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
			return fmt.Errorf("batch loading connections: %w", err)
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
				return fmt.Errorf("due connection %s not found", conn.ConnectionID)
			}

			job := &pipelineservice.Job{
				ID:           uuid.New(),
				ConnectionID: conn.ConnectionID,
				Status:       pipelineservice.JobStatusScheduled,
				JobType:      pipelineservice.JobTypeSync,
				ScheduledAt:  now,
				MaxAttempts:  conn.MaxAttempts,
			}

			createdJob, err := tx.Jobs().Create(ctx, job)
			if err != nil {
				return fmt.Errorf("creating scheduled job for connection %s: %w", conn.ConnectionID, err)
			}

			pipelineservice.RecomputeNextScheduledAt(connRecord, now)

			if _, err := tx.Connections().Update(ctx, connRecord); err != nil {
				return fmt.Errorf("advancing next_scheduled_at for connection %s: %w", conn.ConnectionID, err)
			}

			if err := tx.PipelineEvents().Send(ctx, &pipelineservice.PipelineEvent{
				EventType: pipelineservice.PipelineEventTypeJobCreated,
				Data: &pipelineservice.JobCreatedEventData{
					WorkspaceID: connRecord.WorkspaceID,
					Job:         createdJob,
				},
			}); err != nil {
				return fmt.Errorf("publishing job.created event for connection %s: %w", conn.ConnectionID, err)
			}

			created++
		}

		if created > 0 {
			s.logger.WithField("count", created).Info(ctx, "scheduled jobs")
		}

		return nil
	})
}
