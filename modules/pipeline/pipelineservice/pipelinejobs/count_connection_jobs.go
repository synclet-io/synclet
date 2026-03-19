package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// CountConnectionJobsParams holds parameters for counting jobs for a connection.
type CountConnectionJobsParams struct {
	ConnectionID uuid.UUID
}

// CountConnectionJobs counts total jobs for a given connection.
// Used to populate sync_id metadata for Airbyte CDK connectors.
type CountConnectionJobs struct {
	storage pipelineservice.Storage
}

// NewCountConnectionJobs creates a new CountConnectionJobs use case.
func NewCountConnectionJobs(storage pipelineservice.Storage) *CountConnectionJobs {
	return &CountConnectionJobs{storage: storage}
}

// Execute returns the total number of jobs for the given connection.
func (uc *CountConnectionJobs) Execute(ctx context.Context, params CountConnectionJobsParams) (int64, error) {
	count, err := uc.storage.Jobs().Count(ctx, &pipelineservice.JobFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		return 0, fmt.Errorf("counting connection jobs: %w", err)
	}

	return int64(count), nil
}
