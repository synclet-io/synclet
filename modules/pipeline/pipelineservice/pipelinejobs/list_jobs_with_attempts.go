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
// A nil ConnectionID lists jobs across every connection in the workspace.
type ListJobsWithAttemptsParams struct {
	WorkspaceID  uuid.UUID
	ConnectionID *uuid.UUID
	Statuses     []pipelineservice.JobStatus
	Limit        int
	Offset       int
}

// JobWithAttempts pairs a job with its attempts.
type JobWithAttempts struct {
	Job      *pipelineservice.Job
	Attempts []*pipelineservice.JobAttempt
}

// ListJobsWithAttemptsResult holds one page of jobs and the number of jobs matching the filter.
type ListJobsWithAttemptsResult struct {
	Jobs  []*JobWithAttempts
	Total int
}

// ListJobsWithAttempts retrieves a workspace-scoped page of jobs with their attempts pre-loaded.
type ListJobsWithAttempts struct {
	getConnection   *pipelineconnections.GetConnection
	listConnections *pipelineconnections.ListConnections
	listJobs        *ListJobs
	storage         pipelineservice.Storage
}

// NewListJobsWithAttempts creates a new ListJobsWithAttempts use case.
func NewListJobsWithAttempts(
	getConnection *pipelineconnections.GetConnection,
	listConnections *pipelineconnections.ListConnections,
	listJobs *ListJobs,
	storage pipelineservice.Storage,
) *ListJobsWithAttempts {
	return &ListJobsWithAttempts{
		getConnection:   getConnection,
		listConnections: listConnections,
		listJobs:        listJobs,
		storage:         storage,
	}
}

// Execute resolves the workspace's connections, lists one page of jobs, and batch-loads attempts for that page.
func (uc *ListJobsWithAttempts) Execute(ctx context.Context, params ListJobsWithAttemptsParams) (*ListJobsWithAttemptsResult, error) {
	connectionIDs, err := uc.connectionIDs(ctx, params)
	if err != nil {
		return nil, err
	}

	page, err := uc.listJobs.Execute(ctx, ListJobsParams{
		ConnectionIDs: connectionIDs,
		Statuses:      params.Statuses,
		Limit:         params.Limit,
		Offset:        params.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing jobs: %w", err)
	}

	result := &ListJobsWithAttemptsResult{Total: page.Total}
	if len(page.Jobs) == 0 {
		return result, nil
	}

	jobIDs := make([]uuid.UUID, len(page.Jobs))
	for i, j := range page.Jobs {
		jobIDs[i] = j.ID
	}

	allAttempts, err := uc.storage.JobAttempts().Find(ctx, &pipelineservice.JobAttemptFilter{
		JobID: filter.In(jobIDs...),
	})
	if err != nil {
		return nil, fmt.Errorf("batch loading attempts: %w", err)
	}

	attemptMap := make(map[uuid.UUID][]*pipelineservice.JobAttempt)
	for _, a := range allAttempts {
		attemptMap[a.JobID] = append(attemptMap[a.JobID], a)
	}

	result.Jobs = make([]*JobWithAttempts, len(page.Jobs))
	for i, j := range page.Jobs {
		result.Jobs[i] = &JobWithAttempts{Job: j, Attempts: attemptMap[j.ID]}
	}

	return result, nil
}

func (uc *ListJobsWithAttempts) connectionIDs(ctx context.Context, params ListJobsWithAttemptsParams) ([]uuid.UUID, error) {
	if params.ConnectionID != nil {
		if _, err := uc.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
			ID:          *params.ConnectionID,
			WorkspaceID: params.WorkspaceID,
		}); err != nil {
			return nil, fmt.Errorf("connection not found in workspace: %w", err)
		}

		return []uuid.UUID{*params.ConnectionID}, nil
	}

	conns, err := uc.listConnections.Execute(ctx, pipelineconnections.ListConnectionsParams{
		WorkspaceID: params.WorkspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing workspace connections: %w", err)
	}

	ids := make([]uuid.UUID, len(conns))
	for i, c := range conns {
		ids[i] = c.ID
	}

	return ids, nil
}
