package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// FindStaleJobs retrieves all currently running jobs for stale heartbeat detection.
type FindStaleJobs struct {
	storage pipelineservice.Storage
}

// NewFindStaleJobs creates a new FindStaleJobs use case.
func NewFindStaleJobs(storage pipelineservice.Storage) *FindStaleJobs {
	return &FindStaleJobs{storage: storage}
}

// Execute returns all jobs with running status.
// The caller is responsible for filtering by heartbeat cutoff and K8sJobName.
func (uc *FindStaleJobs) Execute(ctx context.Context) ([]*pipelineservice.Job, error) {
	jobs, err := uc.storage.Jobs().Find(ctx, &pipelineservice.JobFilter{
		Status: filter.Equals(pipelineservice.JobStatusRunning),
	})
	if err != nil {
		return nil, fmt.Errorf("finding running jobs: %w", err)
	}

	return jobs, nil
}
