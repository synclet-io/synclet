package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

// ListJobsWithAttemptsParams holds parameters for listing jobs with their attempts.
type ListJobsWithAttemptsParams struct {
	ConnectionID uuid.UUID
	WorkspaceID  uuid.UUID
}

// JobWithAttempts pairs a job with its attempts.
type JobWithAttempts struct {
	Job      *pipelineservice.Job
	Attempts []*pipelineservice.JobAttempt
}

// ListJobsWithAttempts retrieves all jobs for a connection with their attempts pre-loaded.
// Verifies workspace ownership and uses batch attempt loading (2 queries total).
type ListJobsWithAttempts struct {
	getConnection *pipelineconnections.GetConnection
	listJobs      *ListJobs
	storage       pipelineservice.Storage
}

// NewListJobsWithAttempts creates a new ListJobsWithAttempts use case.
func NewListJobsWithAttempts(
	getConnection *pipelineconnections.GetConnection,
	listJobs *ListJobs,
	storage pipelineservice.Storage,
) *ListJobsWithAttempts {
	return &ListJobsWithAttempts{
		getConnection: getConnection,
		listJobs:      listJobs,
		storage:       storage,
	}
}

// Execute verifies workspace ownership, lists jobs, and batch-loads attempts for all jobs.
func (uc *ListJobsWithAttempts) Execute(ctx context.Context, params ListJobsWithAttemptsParams) ([]*JobWithAttempts, error) {
	// Verify workspace ownership.
	if _, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          params.ConnectionID,
		WorkspaceID: params.WorkspaceID,
	}); err != nil {
		return nil, fmt.Errorf("connection not found in workspace: %w", err)
	}

	jobs, err := uc.listJobs.Execute(ctx, ListJobsParams{
		ConnectionID: params.ConnectionID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing jobs: %w", err)
	}

	if len(jobs) == 0 {
		return nil, nil
	}

	// Collect all job IDs for batch attempt query.
	jobIDs := make([]uuid.UUID, len(jobs))
	for i, j := range jobs {
		jobIDs[i] = j.ID
	}

	// Single batch query for all attempts instead of per-job N+1 queries.
	allAttempts, err := uc.storage.JobAttempts().Find(ctx, &pipelineservice.JobAttemptFilter{
		JobID: filter.In(jobIDs...),
	})
	if err != nil {
		return nil, fmt.Errorf("batch loading attempts: %w", err)
	}

	// Group attempts by job ID.
	attemptMap := make(map[uuid.UUID][]*pipelineservice.JobAttempt)
	for _, a := range allAttempts {
		attemptMap[a.JobID] = append(attemptMap[a.JobID], a)
	}

	// Assemble result with pre-loaded attempts.
	result := make([]*JobWithAttempts, len(jobs))
	for i, j := range jobs {
		result[i] = &JobWithAttempts{Job: j, Attempts: attemptMap[j.ID]}
	}

	return result, nil
}
