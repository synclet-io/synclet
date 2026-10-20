package pipelineexecdocker

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/moby/moby/api/types/container"
	"github.com/samber/lo"
)

// OrphanJobChecker checks if a sync job is still active in the database.
type OrphanJobChecker interface {
	IsJobActive(ctx context.Context, jobID string) (bool, error)
}

// orphanCleanerRunner is the subset of ContainerRunner methods used by OrphanCleaner.
type orphanCleanerRunner interface {
	ListByLabel(ctx context.Context, labels ...string) ([]container.Summary, error)
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
}

// OrphanCleaner finds and removes Docker containers that were created
// for sync jobs but are no longer associated with an active job. It touches
// only containers whose OwnerLabel matches its own owner id.
type OrphanCleaner struct {
	runner      orphanCleanerRunner
	checker     OrphanJobChecker
	ownerID     string
	gracePeriod time.Duration
	stopTimeout time.Duration
	logger      *logging.Logger
}

// NewOrphanCleaner creates a new OrphanCleaner.
func NewOrphanCleaner(runner *ContainerRunner, checker OrphanJobChecker, logger *logging.Logger) *OrphanCleaner {
	return &OrphanCleaner{
		runner:      runner,
		checker:     checker,
		ownerID:     runner.OwnerID(),
		gracePeriod: 15 * time.Minute,
		stopTimeout: 30 * time.Second,
		logger:      logger.Named("orphan-cleanup"),
	}
}

// Cleanup lists all Synclet-managed containers, checks if their associated
// job is still active, and stops+removes orphaned containers.
func (oc *OrphanCleaner) Cleanup(ctx context.Context) error {
	containers, err := oc.runner.ListByLabel(ctx, "synclet.io/managed=true", OwnerLabel+"="+oc.ownerID)
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

	if !oc.ownsContainer(ctr) {
		return false, nil
	}

	logger := oc.logger.WithField("container_id", lo.Substring(ctr.ID, 0, 12))

	jobID := ctr.Labels["synclet.io/job-id"]
	if jobID == "" {
		// Container has managed label but no job-id -- should not happen, but clean up.
		logger.Warn(ctx, "orphan cleanup: managed container without job-id label")

		return true, oc.stopAndRemove(ctx, ctr.ID)
	}

	logger = logger.WithField("job_id", jobID)

	active, err := oc.checker.IsJobActive(ctx, jobID)
	if err != nil {
		return false, fmt.Errorf("checking job %s: %w", jobID, err)
	}

	if !active {
		logger.Info(ctx, "orphan cleanup: removing orphaned container")

		return true, oc.stopAndRemove(ctx, ctr.ID)
	}

	return false, nil
}

// CleanupAll lists all Synclet-managed containers and removes orphaned ones
// WITHOUT applying the grace period. Use this on startup when no containers
// from the current process exist yet (any managed container is from a previous run).
func (oc *OrphanCleaner) CleanupAll(ctx context.Context) error {
	containers, err := oc.runner.ListByLabel(ctx, "synclet.io/managed=true", OwnerLabel+"="+oc.ownerID)
	if err != nil {
		return fmt.Errorf("listing managed containers: %w", err)
	}

	if len(containers) == 0 {
		return nil
	}

	cleaned := 0

	for _, ctr := range containers {
		if !oc.ownsContainer(ctr) {
			continue
		}

		jobID := ctr.Labels["synclet.io/job-id"]
		if jobID == "" {
			oc.logger.WithField("container_id", ctr.ID[:12]).Warn(ctx, "startup cleanup: managed container without job-id label")

			if err := oc.stopAndRemove(ctx, ctr.ID); err != nil {
				oc.logger.WithError(err).WithField("container_id", ctr.ID[:12]).Error(ctx, "startup cleanup: failed to remove container, the interval cleanup retries it")

				continue
			}

			cleaned++

			continue
		}

		active, err := oc.checker.IsJobActive(ctx, jobID)
		if err != nil {
			oc.logger.WithError(err).WithField("job_id", jobID).Error(ctx, "startup cleanup: failed to check job")

			continue
		}

		if !active {
			logger := oc.logger.WithFields(map[string]interface{}{"container_id": ctr.ID[:12], "job_id": jobID})
			logger.Info(ctx, "startup cleanup: removing orphaned container")

			if err := oc.stopAndRemove(ctx, ctr.ID); err != nil {
				logger.WithError(err).Error(ctx, "startup cleanup: failed to remove container, the interval cleanup retries it")

				continue
			}

			cleaned++
		}
	}

	if cleaned > 0 {
		oc.logger.WithField("count", cleaned).Info(ctx, "startup orphan cleanup: cleaned containers")
	}

	return nil
}

func (oc *OrphanCleaner) ownsContainer(ctr container.Summary) bool {
	return ctr.Labels[OwnerLabel] == oc.ownerID
}

// stopAndRemove stops and removes a container within stopTimeout. A container
// left behind when the limit passes stays labelled and is picked up by the
// next cleanup run.
func (oc *OrphanCleaner) stopAndRemove(ctx context.Context, containerID string) error {
	ctx, cancel := context.WithTimeout(ctx, oc.stopTimeout)
	defer cancel()

	if err := oc.runner.Stop(ctx, containerID); err != nil && ctx.Err() != nil {
		return fmt.Errorf("stopping container within %s: %w", oc.stopTimeout, err)
	}

	return oc.runner.Remove(ctx, containerID)
}
