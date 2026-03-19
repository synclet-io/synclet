package pipelinejobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// QueueJobParams holds parameters for queuing a sync job.
type QueueJobParams struct {
	ConnectionID uuid.UUID
	JobType      pipelineservice.JobType
	ScheduledAt  time.Time
	MaxAttempts  int
}

// QueueJob creates a pending job for a connection.
type QueueJob struct {
	storage pipelineservice.Storage
}

// NewQueueJob creates a new QueueJob use case.
func NewQueueJob(storage pipelineservice.Storage) *QueueJob {
	return &QueueJob{storage: storage}
}

// Execute creates a new pending job. It reads retry config from the connection
// if MaxAttempts is not provided. It also checks for duplicate pending/running jobs.
func (uc *QueueJob) Execute(ctx context.Context, params QueueJobParams) (*pipelineservice.Job, error) {
	// Load connection to get retry config if not provided.
	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading connection for retry config: %w", err)
	}

	// Enforce one active job per connection (also enforced by DB unique index).
	activeJobs, err := uc.storage.Jobs().Find(ctx, &pipelineservice.JobFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
		Status:       filter.In(pipelineservice.JobStatusScheduled, pipelineservice.JobStatusStarting, pipelineservice.JobStatusRunning),
	})
	if err != nil {
		return nil, fmt.Errorf("checking active jobs: %w", err)
	}
	if len(activeJobs) > 0 {
		return nil, &pipelineservice.ValidationError{Field: "connection_id", Message: "connection already has an active job"}
	}

	maxAttempts := params.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = conn.MaxAttempts
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	scheduledAt := params.ScheduledAt
	if scheduledAt.IsZero() {
		scheduledAt = time.Now()
	}

	job := &pipelineservice.Job{
		ID:           uuid.New(),
		ConnectionID: params.ConnectionID,
		Status:       pipelineservice.JobStatusScheduled,
		JobType:      params.JobType,
		ScheduledAt:  scheduledAt,
		Attempt:      1,
		MaxAttempts:  maxAttempts,
		CreatedAt:    time.Now(),
	}

	created, err := uc.storage.Jobs().Create(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("creating job: %w", err)
	}

	return created, nil
}
