package pipelinejobs

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// FailUnclaimedJobs acts on scheduled sync jobs no executor claimed within
// the claim timeout. A job whose group has no live executor fails for good,
// with no retry and no attempt row. A job whose group has live executors
// stays scheduled with a busy queue reason, and the first time it is seen
// stalled a JobQueueStalled event is published.
type FailUnclaimedJobs struct {
	storage        pipelineservice.Storage
	logger         *logging.Logger
	claimTimeout   time.Duration
	silenceTimeout time.Duration
	now            func() time.Time
}

// NewFailUnclaimedJobs creates a new FailUnclaimedJobs use case.
func NewFailUnclaimedJobs(storage pipelineservice.Storage, cfg pipelineservice.Config, logger *logging.Logger) *FailUnclaimedJobs {
	return &FailUnclaimedJobs{
		storage:        storage,
		logger:         logger,
		claimTimeout:   cfg.JobClaimTimeout,
		silenceTimeout: cfg.ExecutorSilenceTimeout,
		now:            time.Now,
	}
}

// Execute handles every unclaimed job past the timeout and returns how many
// it failed. A job that fails to update is logged and skipped.
func (uc *FailUnclaimedJobs) Execute(ctx context.Context) (int, error) {
	now := uc.now()
	cutoff := now.Add(-uc.claimTimeout)

	scheduled, err := uc.storage.Jobs().Find(ctx, &pipelineservice.JobFilter{
		Status:  filter.Equals(pipelineservice.JobStatusScheduled),
		JobType: filter.Equals(pipelineservice.JobTypeSync),
	})
	if err != nil {
		return 0, fmt.Errorf("listing scheduled jobs: %w", err)
	}

	liveGroups := map[string]bool{}
	failed := 0

	for _, job := range scheduled {
		if !job.ScheduledAt.Before(cutoff) {
			continue
		}

		live, known := liveGroups[job.ExecutorGroup]
		if !known {
			live, err = pipelineservice.HasLiveExecutor(ctx, uc.storage, job.ExecutorGroup, uc.silenceTimeout, now)
			if err != nil {
				return failed, err
			}

			liveGroups[job.ExecutorGroup] = live
		}

		if live {
			err = uc.markBusy(ctx, job, now)
		} else {
			err = uc.fail(ctx, job, now)
			if err == nil {
				failed++
			}
		}

		if err != nil && uc.logger != nil {
			uc.logger.WithError(err).WithField("job_id", job.ID.String()).Error(ctx, "handling unclaimed job")
		}
	}

	return failed, nil
}

// fail marks the job failed with a NoLiveExecutorError message, schedules
// the connection's next run and publishes job.updated.
func (uc *FailUnclaimedJobs) fail(ctx context.Context, job *pipelineservice.Job, now time.Time) error {
	oldJob := *job

	message := (&pipelineservice.NoLiveExecutorError{Group: job.ExecutorGroup}).Error()
	job.Status = pipelineservice.JobStatusFailed
	job.Error = &message
	job.CompletedAt = &now
	job.QueueReason = nil

	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		updated, err := tx.Jobs().UpdateIfStatus(ctx, job, pipelineservice.JobStatusScheduled)
		if err != nil {
			return fmt.Errorf("failing unclaimed job: %w", err)
		}

		if !updated {
			return nil
		}

		conn, err := tx.Connections().First(ctx, &pipelineservice.ConnectionFilter{ID: filter.Equals(job.ConnectionID)})
		if err != nil {
			return fmt.Errorf("getting connection of unclaimed job: %w", err)
		}

		pipelineservice.RecomputeNextScheduledAt(conn, now)

		if _, err := tx.Connections().Update(ctx, conn); err != nil {
			return fmt.Errorf("updating connection next_scheduled_at: %w", err)
		}

		newJob := *job

		if err := tx.PipelineEvents().Send(ctx, &pipelineservice.PipelineEvent{
			EventType: pipelineservice.PipelineEventTypeJobUpdated,
			Data: &pipelineservice.JobUpdatedEventData{
				WorkspaceID: conn.WorkspaceID,
				OldData:     &oldJob,
				NewData:     &newJob,
			},
		}); err != nil {
			return fmt.Errorf("publishing job.updated event: %w", err)
		}

		return nil
	})
}

// markBusy sets the busy queue reason and, on the first stall of the job,
// stamps stalled_at and publishes JobQueueStalled. Status, attempt and queue
// position stay as they are.
func (uc *FailUnclaimedJobs) markBusy(ctx context.Context, job *pipelineservice.Job, now time.Time) error {
	reason := pipelineservice.ExecutorsBusyReason(job.ExecutorGroup)
	firstStall := job.StalledAt == nil

	if !firstStall && job.QueueReason != nil && *job.QueueReason == reason {
		return nil
	}

	job.QueueReason = &reason
	if firstStall {
		job.StalledAt = &now
	}

	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		updated, err := tx.Jobs().UpdateIfStatus(ctx, job, pipelineservice.JobStatusScheduled)
		if err != nil {
			return fmt.Errorf("marking unclaimed job busy: %w", err)
		}

		if !updated || !firstStall {
			return nil
		}

		conn, err := tx.Connections().First(ctx, &pipelineservice.ConnectionFilter{ID: filter.Equals(job.ConnectionID)})
		if err != nil {
			return fmt.Errorf("getting connection of stalled job: %w", err)
		}

		stalled := *job

		if err := tx.PipelineEvents().Send(ctx, &pipelineservice.PipelineEvent{
			EventType: pipelineservice.PipelineEventTypeJobQueueStalled,
			Data: &pipelineservice.JobQueueStalledEventData{
				WorkspaceID: conn.WorkspaceID,
				Job:         &stalled,
				Group:       job.ExecutorGroup,
				WaitedMs:    now.Sub(job.ScheduledAt).Milliseconds(),
			},
		}); err != nil {
			return fmt.Errorf("publishing job queue stalled event: %w", err)
		}

		return nil
	})
}
