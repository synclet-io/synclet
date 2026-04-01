package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/go-pnp/go-pnp/logging"
)

// OrphanJobChecker checks if a sync job is still active in the database.
type OrphanJobChecker interface {
	IsJobActive(ctx context.Context, jobID string) (bool, error)
}

// orphanCleanerRunner is the subset of ContainerRunner methods used by OrphanCleaner.
type orphanCleanerRunner interface {
	ListByLabel(ctx context.Context, label string) ([]container.Summary, error)
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
}

// OrphanCleaner finds and removes Docker containers that were created
// for sync jobs but are no longer associated with an active job.
type OrphanCleaner struct {
	runner      orphanCleanerRunner
	checker     OrphanJobChecker
	gracePeriod time.Duration
	logger      *logging.Logger
}

// NewOrphanCleaner creates a new OrphanCleaner.
func NewOrphanCleaner(runner *ContainerRunner, checker OrphanJobChecker, logger *logging.Logger) *OrphanCleaner {
	return &OrphanCleaner{
		runner:      runner,
		checker:     checker,
		gracePeriod: 15 * time.Minute,
		logger:      logger.Named("orphan-cleanup"),
	}
}

// Cleanup lists all Synclet-managed containers, checks if their associated
// job is still active, and stops+removes orphaned containers.
func (oc *OrphanCleaner) Cleanup(ctx context.Context) error {
	containers, err := oc.runner.ListByLabel(ctx, "synclet.io/managed=true")
	if err != nil {
		return fmt.Errorf("listing managed containers: %w", err)
	}

	if len(containers) == 0 {
		return nil
	}

	cleaned := 0

	for _, ctr := range containers {
		removed, err := oc.processContainer(ctx, ctr)
		if err != nil {
			oc.logger.WithError(err).WithField("container_id", ctr.ID[:12]).Error(ctx, "orphan cleanup: failed to process container")

			continue
		}

		if removed {
			cleaned++
		}
	}

	if cleaned > 0 {
		oc.logger.WithField("count", cleaned).Info(ctx, "orphan cleanup: cleaned up containers")
	}

	return nil
}

// processContainer checks a single container and removes it if orphaned.
// Returns true if the container was removed.
func (oc *OrphanCleaner) processContainer(ctx context.Context, ctr container.Summary) (bool, error) {
	// Grace period check: skip containers younger than 15 minutes.
	created := time.Unix(ctr.Created, 0)
	if time.Since(created) < oc.gracePeriod {
		return false, nil
	}

	jobID := ctr.Labels["synclet.io/job-id"]
	if jobID == "" {
		// Container has managed label but no job-id -- should not happen, but clean up.
		oc.logger.WithField("container_id", ctr.ID[:12]).Warn(ctx, "orphan cleanup: managed container without job-id label")

		return true, oc.stopAndRemove(ctx, ctr.ID)
	}

	active, err := oc.checker.IsJobActive(ctx, jobID)
	if err != nil {
		return false, fmt.Errorf("checking job %s: %w", jobID, err)
	}

	if !active {
		oc.logger.WithFields(map[string]interface{}{"container_id": ctr.ID[:12], "job_id": jobID}).Info(ctx, "orphan cleanup: removing orphaned container")

		return true, oc.stopAndRemove(ctx, ctr.ID)
	}

	return false, nil
}

// CleanupAll lists all Synclet-managed containers and removes orphaned ones
// WITHOUT applying the grace period. Use this on startup when no containers
// from the current process exist yet (any managed container is from a previous run).
func (oc *OrphanCleaner) CleanupAll(ctx context.Context) error {
	containers, err := oc.runner.ListByLabel(ctx, "synclet.io/managed=true")
	if err != nil {
		return fmt.Errorf("listing managed containers: %w", err)
	}

	if len(containers) == 0 {
		return nil
	}

	cleaned := 0

	for _, ctr := range containers {
		jobID := ctr.Labels["synclet.io/job-id"]
		if jobID == "" {
			oc.logger.WithField("container_id", ctr.ID[:12]).Warn(ctx, "startup cleanup: managed container without job-id label")
			_ = oc.stopAndRemove(ctx, ctr.ID)
			cleaned++

			continue
		}

		active, err := oc.checker.IsJobActive(ctx, jobID)
		if err != nil {
			oc.logger.WithError(err).WithField("job_id", jobID).Error(ctx, "startup cleanup: failed to check job")

			continue
		}

		if !active {
			oc.logger.WithFields(map[string]interface{}{"container_id": ctr.ID[:12], "job_id": jobID}).Info(ctx, "startup cleanup: removing orphaned container")
			_ = oc.stopAndRemove(ctx, ctr.ID)
			cleaned++
		}
	}

	if cleaned > 0 {
		oc.logger.WithField("count", cleaned).Info(ctx, "startup orphan cleanup: cleaned containers")
	}

	return nil
}

func (oc *OrphanCleaner) stopAndRemove(ctx context.Context, containerID string) error {
	// Stop with default timeout (these are already orphaned).
	_ = oc.runner.Stop(ctx, containerID)

	return oc.runner.Remove(ctx, containerID)
}
