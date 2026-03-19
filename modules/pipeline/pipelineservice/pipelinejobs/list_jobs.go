package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListJobsParams holds parameters for listing jobs.
type ListJobsParams struct {
	ConnectionID uuid.UUID
}

// ListJobs retrieves all jobs for a connection, ordered by creation time descending.
type ListJobs struct {
	storage pipelineservice.Storage
}

// NewListJobs creates a new ListJobs use case.
func NewListJobs(storage pipelineservice.Storage) *ListJobs {
	return &ListJobs{storage: storage}
}

// Execute returns all jobs for the given connection.
func (uc *ListJobs) Execute(ctx context.Context, params ListJobsParams) ([]*pipelineservice.Job, error) {
	jobs, err := uc.storage.Jobs().Find(ctx, &pipelineservice.JobFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
	}, dbutil.WithOrder(pipelineservice.JobFieldCreatedAt, dbutil.OrderDirDesc))
	if err != nil {
		return nil, fmt.Errorf("listing jobs: %w", err)
	}

	return jobs, nil
}
