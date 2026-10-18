package pipelinejobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// UpdateJobStatusParams holds parameters for updating a job's status.
type UpdateJobStatusParams struct {
	ID        uuid.UUID
	SyncErr   error
	SyncStats *pipelineservice.SyncStats
	// SkipRetry fails the job for good instead of scheduling another attempt.
	SkipRetry bool
}

// UpdateJobStatus completes a job by setting its status based on whether
// an error occurred, creating a JobAttempt record, and handling retries.
type UpdateJobStatus struct {
	storage pipelineservice.Storage
}

// NewUpdateJobStatus creates a new UpdateJobStatus use case.
func NewUpdateJobStatus(storage pipelineservice.Storage) *UpdateJobStatus {
	return &UpdateJobStatus{storage: storage}
}

// Execute updates the job status after execution. It handles success, failure,
// and retry logic including linear backoff based on connection retry config.
// Publishes a JobUpdated event to the outbox in the same transaction.
// Only starting or running jobs are updated; a report for a job that already
// reached another status (cancelled, completed, failed or rescheduled) is a no-op.
func (uc *UpdateJobStatus) Execute(ctx context.Context, params UpdateJobStatusParams) error {
	prevJob, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	if !isJobInFlight(prevJob.Status) {
		return nil
	}

	// Snapshot the previous state for the event before mutating.
	oldJob := *prevJob

	job := prevJob

	now := time.Now()
	job.CompletedAt = &now

	// Record the attempt.
	statsJSON := "{}"

	if params.SyncStats != nil {
		b, err := json.Marshal(params.SyncStats)
		if err != nil {
			return fmt.Errorf("marshaling sync stats: %w", err)
		}

		statsJSON = string(b)
	}

	attempt := &pipelineservice.JobAttempt{
		ID:            uuid.New(),
		JobID:         params.ID,
		AttemptNumber: job.Attempt,
		StartedAt:     derefTime(job.StartedAt),
		CompletedAt:   &now,
		SyncStatsJSON: statsJSON,
	}

	if params.SyncErr != nil {
		errStr := params.SyncErr.Error()
		job.Error = &errStr
		attempt.Error = &errStr

		// Classify exit code if available for retry decisions.
		skipRetry := params.SkipRetry

		// The first permanent (OOM, segfault) or intentional (SIGTERM) exit names
		// the failure and stops retries; without one, the first exit names it.
		for i, exitErr := range pipelineservice.ExitCodeErrors(params.SyncErr) {
			category, reason := ClassifyExitCode(exitErr.ExitCode)
			if category == FailurePermanent || category == FailureIntentional {
				job.FailureReason = &reason
				skipRetry = true

				break
			}

			if i == 0 {
				job.FailureReason = &reason
			}
		}

		// Check if we should retry.
		if !skipRetry && job.Attempt < job.MaxAttempts {
			job.Status = pipelineservice.JobStatusScheduled
			job.Attempt++
			job.CompletedAt = nil

			scheduledAt := time.Now()
			job.ScheduledAt = scheduledAt
			job.StartedAt = nil
			job.WorkerID = nil
		} else {
			job.Status = pipelineservice.JobStatusFailed
		}
	} else {
		job.Status = pipelineservice.JobStatusCompleted
	}

	return uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		updated, err := tx.Jobs().UpdateIfStatus(ctx, job, pipelineservice.JobStatusStarting, pipelineservice.JobStatusRunning)
		if err != nil {
			return fmt.Errorf("updating job: %w", err)
		}

		if !updated {
			return nil
		}

		if _, err := tx.JobAttempts().Create(ctx, attempt); err != nil {
			return fmt.Errorf("creating job attempt: %w", err)
		}

		conn, connErr := tx.Connections().First(ctx, &pipelineservice.ConnectionFilter{
			ID: filter.Equals(job.ConnectionID),
		})
		if connErr != nil {
			return fmt.Errorf("getting connection for event emission: %w", connErr)
		}

		// Recompute next_scheduled_at when job reaches final status (completed or permanent failure).
		// Skip if job is being retried (status == Scheduled means retry in progress).
		if job.Status == pipelineservice.JobStatusCompleted || job.Status == pipelineservice.JobStatusFailed {
			pipelineservice.RecomputeNextScheduledAt(conn, time.Now())

			if _, updateErr := tx.Connections().Update(ctx, conn); updateErr != nil {
				return fmt.Errorf("updating connection next_scheduled_at: %w", updateErr)
			}
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

// UpdateHeartbeatParams holds parameters for updating a heartbeat.
type UpdateHeartbeatParams struct {
	ID uuid.UUID
}

// UpdateHeartbeat updates the heartbeat timestamp for a running job.
type UpdateHeartbeat struct {
	storage pipelineservice.Storage
}

// NewUpdateHeartbeat creates a new UpdateHeartbeat use case.
func NewUpdateHeartbeat(storage pipelineservice.Storage) *UpdateHeartbeat {
	return &UpdateHeartbeat{storage: storage}
}

// Execute updates the heartbeat timestamp on the given job. Heartbeats for a
// job that is no longer starting or running are ignored so they never revive it.
func (uc *UpdateHeartbeat) Execute(ctx context.Context, params UpdateHeartbeatParams) error {
	if _, err := uc.storage.Jobs().TouchHeartbeat(ctx, params.ID, time.Now()); err != nil {
		return fmt.Errorf("updating heartbeat: %w", err)
	}

	return nil
}

// FailStaleJobs finds running jobs with stale heartbeats and marks them as
// failed so they can be retried.
type FailStaleJobs struct {
	logger           *logging.Logger
	storage          pipelineservice.Storage
	updateJobStatus  *UpdateJobStatus
	heartbeatTimeout time.Duration
}

// NewFailStaleJobs creates a new FailStaleJobs use case.
func NewFailStaleJobs(
	logger *logging.Logger,
	storage pipelineservice.Storage,
	updateJobStatus *UpdateJobStatus,
	config pipelineservice.Config,
) *FailStaleJobs {
	return &FailStaleJobs{
		logger:           logger,
		storage:          storage,
		updateJobStatus:  updateJobStatus,
		heartbeatTimeout: config.HeartbeatTimeout,
	}
}

// Execute finds stale running jobs and marks them as failed, returning the count.
// Uses DB-level HeartbeatAt filter (per D-05) with explicit NULL heartbeat handling
// for jobs that crash before sending their first heartbeat (per D-04).
func (uc *FailStaleJobs) Execute(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-uc.heartbeatTimeout)

	// DB-level filter for running/starting jobs with stale heartbeats (per D-05)
	staleJobs, err := uc.storage.Jobs().Find(ctx, &pipelineservice.JobFilter{
		Status:      filter.In(pipelineservice.JobStatusRunning, pipelineservice.JobStatusStarting),
		HeartbeatAt: filter.Less(&cutoff),
	})
	if err != nil {
		return 0, fmt.Errorf("listing stale jobs: %w", err)
	}

	// Also find running/starting jobs with NULL heartbeat that started before cutoff
	// (D-04: fail ALL stale jobs, including those that crash before first heartbeat)
	nullHeartbeatJobs, err := uc.storage.Jobs().Find(ctx, &pipelineservice.JobFilter{
		Status:      filter.In(pipelineservice.JobStatusRunning, pipelineservice.JobStatusStarting),
		StartedAt:   filter.Less(&cutoff),
		HeartbeatAt: filter.IsNull[*time.Time](),
	})
	if err != nil {
		return 0, fmt.Errorf("listing null-heartbeat stale jobs: %w", err)
	}

	// Merge: add null-heartbeat jobs that aren't already in staleJobs
	seen := make(map[uuid.UUID]struct{}, len(staleJobs))
	for _, j := range staleJobs {
		seen[j.ID] = struct{}{}
	}

	for _, j := range nullHeartbeatJobs {
		if _, ok := seen[j.ID]; !ok {
			staleJobs = append(staleJobs, j)
		}
	}

	failed := 0

	for _, job := range staleJobs {
		var errMsg string
		if job.HeartbeatAt != nil {
			errMsg = fmt.Sprintf("worker heartbeat timeout (last heartbeat: %s)", job.HeartbeatAt.Format(time.RFC3339))
		} else {
			errMsg = fmt.Sprintf("worker never sent heartbeat (started: %s)", derefTime(job.StartedAt).Format(time.RFC3339))
		}

		if err := uc.updateJobStatus.Execute(ctx, UpdateJobStatusParams{
			ID:      job.ID,
			SyncErr: fmt.Errorf("%s", errMsg),
		}); err != nil {
			uc.logger.WithError(err).WithField("job_id", job.ID).Error(ctx, "failed to mark stale job as failed")

			continue
		}

		failed++
	}

	return failed, nil
}

// SetK8sJobNameParams holds parameters for setting a K8s job name.
type SetK8sJobNameParams struct {
	ID         uuid.UUID
	K8sJobName string
}

// SetK8sJobName stores the K8s job name on a job record.
type SetK8sJobName struct {
	storage pipelineservice.Storage
}

// NewSetK8sJobName creates a new SetK8sJobName use case.
func NewSetK8sJobName(storage pipelineservice.Storage) *SetK8sJobName {
	return &SetK8sJobName{storage: storage}
}

// Execute sets the K8s job name on the given job while it is still active.
func (uc *SetK8sJobName) Execute(ctx context.Context, params SetK8sJobNameParams) error {
	job, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{
		ID: filter.Equals(params.ID),
	})
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	job.K8sJobName = &params.K8sJobName

	if _, err := uc.storage.Jobs().UpdateIfStatus(ctx, job,
		pipelineservice.JobStatusScheduled, pipelineservice.JobStatusStarting, pipelineservice.JobStatusRunning); err != nil {
		return fmt.Errorf("updating job k8s name: %w", err)
	}

	return nil
}

func isJobInFlight(status pipelineservice.JobStatus) bool {
	return status == pipelineservice.JobStatusStarting || status == pipelineservice.JobStatusRunning
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}

	return *t
}
