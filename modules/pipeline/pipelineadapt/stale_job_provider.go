package pipelineadapt

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/pkg/k8s"
)

// StaleJobProviderAdapter adapts pipeline use cases to k8s.StaleJobProvider.
type StaleJobProviderAdapter struct {
	findStaleJobs   *pipelinejobs.FindStaleJobs
	isJobActive     *pipelinejobs.IsJobActive
	isTaskActive    *pipelinejobs.IsTaskActive
	updateJobStatus *pipelinejobs.UpdateJobStatus
}

// NewStaleJobProviderAdapter creates a new stale job provider adapter.
func NewStaleJobProviderAdapter(
	findStaleJobs *pipelinejobs.FindStaleJobs,
	isJobActive *pipelinejobs.IsJobActive,
	isTaskActive *pipelinejobs.IsTaskActive,
	updateJobStatus *pipelinejobs.UpdateJobStatus,
) *StaleJobProviderAdapter {
	return &StaleJobProviderAdapter{
		findStaleJobs:   findStaleJobs,
		isJobActive:     isJobActive,
		isTaskActive:    isTaskActive,
		updateJobStatus: updateJobStatus,
	}
}

// GetStaleJobs returns running K8s jobs with stale heartbeats.
func (a *StaleJobProviderAdapter) GetStaleJobs(ctx context.Context, timeout time.Duration) ([]k8s.StaleJob, error) {
	cutoff := time.Now().Add(-timeout)

	jobs, err := a.findStaleJobs.Execute(ctx)
	if err != nil {
		return nil, err
	}

	var result []k8s.StaleJob
	for _, job := range jobs {
		if job.HeartbeatAt != nil && job.HeartbeatAt.Before(cutoff) && job.K8sJobName != nil {
			result = append(result, k8s.StaleJob{
				JobID:      job.ID.String(),
				K8sJobName: *job.K8sJobName,
			})
		}
	}

	return result, nil
}

// FailJob marks a job as failed.
func (a *StaleJobProviderAdapter) FailJob(ctx context.Context, jobID, reason string) error {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return fmt.Errorf("parsing job ID: %w", err)
	}

	return a.updateJobStatus.Execute(ctx, pipelinejobs.UpdateJobStatusParams{
		ID:      id,
		SyncErr: fmt.Errorf("%s", reason),
	})
}

// IsJobActive checks if a job is still in running status.
func (a *StaleJobProviderAdapter) IsJobActive(ctx context.Context, jobID string) (bool, error) {
	return a.isJobActive.ExecuteByString(ctx, jobID)
}

// IsTaskActive checks if a connector task is still in pending or running status.
func (a *StaleJobProviderAdapter) IsTaskActive(ctx context.Context, taskID string) (bool, error) {
	return a.isTaskActive.ExecuteByString(ctx, taskID)
}
