package pipelinelogs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetJobLogParams holds parameters for retrieving job logs.
type GetJobLogParams struct {
	WorkspaceID uuid.UUID
	JobID       uuid.UUID
	AfterID     int64 // cursor: return logs with id > AfterID (0 = from beginning)
	Limit       int   // max lines to return (0 = all)
}

// GetJobLogResult holds the result of a job log query.
type GetJobLogResult struct {
	Lines   []string
	LastID  int64
	HasMore bool
}

// GetJobLog retrieves log lines for a job with cursor-based pagination.
type GetJobLog struct {
	storage pipelineservice.Storage
}

// NewGetJobLog creates a new GetJobLog use case.
func NewGetJobLog(storage pipelineservice.Storage) *GetJobLog {
	return &GetJobLog{storage: storage}
}

// Execute returns log lines for the given job, supporting cursor-based pagination.
func (uc *GetJobLog) Execute(ctx context.Context, params GetJobLogParams) (*GetJobLogResult, error) {
	// Verify the job belongs to the caller's workspace.
	job, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{
		ID: filter.Equals(params.JobID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting job: %w", err)
	}

	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID:          filter.Equals(job.ConnectionID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil || conn == nil {
		return nil, pipelineservice.ErrJobNotFound
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 0 // no limit
	}

	// Fetch limit+1 to detect hasMore.
	fetchLimit := limit
	if fetchLimit > 0 {
		fetchLimit = limit + 1
	}

	logs, err := uc.storage.JobLogs().GetLogs(ctx, params.JobID, params.AfterID, fetchLimit)
	if err != nil {
		return nil, err
	}

	hasMore := false
	if fetchLimit > 0 && len(logs) > limit {
		hasMore = true
		logs = logs[:limit]
	}

	lines := make([]string, len(logs))
	var lastID int64

	for i, l := range logs {
		lines[i] = l.LogLine
		if l.ID > lastID {
			lastID = l.ID
		}
	}

	return &GetJobLogResult{
		Lines:   lines,
		LastID:  lastID,
		HasMore: hasMore,
	}, nil
}
