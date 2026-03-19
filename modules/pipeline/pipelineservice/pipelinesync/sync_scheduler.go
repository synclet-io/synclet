package pipelinesync

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
// The limit parameter controls the maximum number of concurrent jobs.
func NewSyncScheduler(storage pipelineservice.Storage, limit int, logger *logging.Logger) *SyncScheduler {
	return &SyncScheduler{
		storage: storage,
		limit:   limit,
		logger:  logger,
	}
}

// Execute runs one scheduling cycle inside a transaction.
// 1. Acquires advisory lock (returns nil if another instance holds it).
// 2. Counts active jobs and checks against limit.
// 3. Finds due connections ordered by oldest last-sync completion.
// 4. Creates a scheduled job for each due connection.
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
			job := &pipelineservice.Job{
				ID:           uuid.New(),
				ConnectionID: conn.ConnectionID,
				Status:       pipelineservice.JobStatusScheduled,
				JobType:      pipelineservice.JobTypeSync,
				ScheduledAt:  now,
				MaxAttempts:  conn.MaxAttempts,
			}
			if _, err := tx.Jobs().Create(ctx, job); err != nil {
				if s.logger != nil {
					s.logger.WithError(err).WithField("connection_id", conn.ConnectionID.String()).Error(ctx, "creating scheduled job")
				}
				continue
			}

			// Advance next_scheduled_at to next cron tick after job creation.
			connRecord := connByID[conn.ConnectionID]
			if connRecord != nil {
				pipelineservice.RecomputeNextScheduledAt(connRecord, now)
				if _, err := tx.Connections().Update(ctx, connRecord); err != nil {
					if s.logger != nil {
						s.logger.WithError(err).WithField("connection_id", conn.ConnectionID.String()).Error(ctx, "advancing next_scheduled_at")
					}
				}
			}

			created++
		}

		if created > 0 && s.logger != nil {
			s.logger.WithField("count", created).Info(ctx, "scheduled jobs")
		}

		return nil
	})
}
