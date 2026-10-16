package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/saturn4er/boilerplate-go/lib/pagination"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListJobsParams holds parameters for listing jobs.
type ListJobsParams struct {
	ConnectionIDs []uuid.UUID
	Statuses      []pipelineservice.JobStatus
	Limit         int
	Offset        int
}

// ListJobsResult holds one page of jobs and the number of jobs matching the filter.
type ListJobsResult struct {
	Jobs  []*pipelineservice.Job
	Total int
}

// ListJobs retrieves a page of jobs for a set of connections, ordered by creation time descending.
type ListJobs struct {
	storage pipelineservice.Storage
}

// NewListJobs creates a new ListJobs use case.
func NewListJobs(storage pipelineservice.Storage) *ListJobs {
	return &ListJobs{storage: storage}
}

// Execute returns the jobs in [Offset, Offset+Limit) for the given connections and statuses.
func (uc *ListJobs) Execute(ctx context.Context, params ListJobsParams) (*ListJobsResult, error) {
	if len(params.ConnectionIDs) == 0 || params.Limit <= 0 {
		return &ListJobsResult{}, nil
	}

	jobFilter := &pipelineservice.JobFilter{
		ConnectionID: filter.In(params.ConnectionIDs...),
	}
	if len(params.Statuses) > 0 {
		jobFilter.Status = filter.In(params.Statuses...)
	}

	total, err := uc.storage.Jobs().Count(ctx, jobFilter)
	if err != nil {
		return nil, fmt.Errorf("counting jobs: %w", err)
	}

	offset := max(params.Offset, 0)
	if offset >= total {
		return &ListJobsResult{Total: total}, nil
	}

	jobs, err := uc.findPage(ctx, jobFilter, offset, params.Limit)
	if err != nil {
		return nil, err
	}

	return &ListJobsResult{Jobs: jobs, Total: total}, nil
}

// findPage loads the rows in [offset, offset+limit). An offset that is not a multiple of
// limit spans two storage pages, so both are loaded and trimmed.
func (uc *ListJobs) findPage(ctx context.Context, jobFilter *pipelineservice.JobFilter, offset, limit int) ([]*pipelineservice.Job, error) {
	firstPage := offset/limit + 1
	skip := offset % limit

	pages := 1
	if skip > 0 {
		pages = 2
	}

	var jobs []*pipelineservice.Job

	for page := firstPage; page < firstPage+pages; page++ {
		found, err := uc.storage.Jobs().Find(ctx, jobFilter,
			dbutil.WithOrder(pipelineservice.JobFieldCreatedAt, dbutil.OrderDirDesc),
			dbutil.WithOrder(pipelineservice.JobFieldID, dbutil.OrderDirDesc),
			dbutil.WithPagination(&pagination.Pagination{Page: page, PerPage: limit}),
		)
		if err != nil {
			return nil, fmt.Errorf("listing jobs: %w", err)
		}

		jobs = append(jobs, found...)

		if len(found) < limit {
			break
		}
	}

	if skip >= len(jobs) {
		return nil, nil
	}

	return jobs[skip:min(skip+limit, len(jobs))], nil
}
