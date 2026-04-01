package k8s

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const orphanGracePeriod = 15 * time.Minute

// OrphanCleaner finds and removes K8s Jobs that were created for sync jobs
// but are no longer associated with an active job in the database.
type OrphanCleaner struct {
	client    kubernetes.Interface
	namespace string
	provider  StaleJobProvider
	logger    *zap.Logger
}

// NewOrphanCleaner creates a new K8s OrphanCleaner.
func NewOrphanCleaner(client kubernetes.Interface, namespace string, provider StaleJobProvider, logger *zap.Logger) *OrphanCleaner {
	return &OrphanCleaner{
		client:    client,
		namespace: namespace,
		provider:  provider,
		logger:    logger.Named("k8s-orphan-cleanup"),
	}
}

// Cleanup lists all K8s Jobs with Synclet labels and cleans up orphaned ones,
// skipping jobs younger than the grace period (15 minutes).
func (oc *OrphanCleaner) Cleanup(ctx context.Context) error {
	cleaned := 0

	// Clean orphaned sync jobs.
	syncCleaned, err := oc.cleanupSyncJobs(ctx, true)
	if err != nil {
		return err
	}

	cleaned += syncCleaned

	// Clean orphaned task jobs.
	taskCleaned, err := oc.cleanupTaskJobs(ctx, true)
	if err != nil {
		return err
	}

	cleaned += taskCleaned

	// Clean orphaned sync secrets.
	syncSecretsCleaned, err := oc.cleanupSyncSecrets(ctx, true)
	if err != nil {
		return err
	}

	cleaned += syncSecretsCleaned

	// Clean orphaned task secrets.
	taskSecretsCleaned, err := oc.cleanupTaskSecrets(ctx, true)
	if err != nil {
		return err
	}

	cleaned += taskSecretsCleaned

	if cleaned > 0 {
		oc.logger.Info("k8s orphan cleanup: cleaned up resources", zap.Int("count", cleaned))
	}

	return nil
}

// CleanupAll lists all K8s Jobs with Synclet labels and removes orphaned ones
// WITHOUT applying the grace period. Use this on startup when all managed K8s
// jobs from the previous process are potential orphans.
func (oc *OrphanCleaner) CleanupAll(ctx context.Context) error {
	cleaned := 0

	syncCleaned, err := oc.cleanupSyncJobs(ctx, false)
	if err != nil {
		return err
	}

	cleaned += syncCleaned

	taskCleaned, err := oc.cleanupTaskJobs(ctx, false)
	if err != nil {
		return err
	}

	cleaned += taskCleaned

	syncSecretsCleaned, err := oc.cleanupSyncSecrets(ctx, false)
	if err != nil {
		return err
	}

	cleaned += syncSecretsCleaned

	taskSecretsCleaned, err := oc.cleanupTaskSecrets(ctx, false)
	if err != nil {
		return err
	}

	cleaned += taskSecretsCleaned

	if cleaned > 0 {
		oc.logger.Info("k8s startup orphan cleanup: cleaned resources", zap.Int("count", cleaned))
	}

	return nil
}

func (oc *OrphanCleaner) cleanupSyncJobs(ctx context.Context, applyGrace bool) (int, error) {
	jobs, err := oc.client.BatchV1().Jobs(oc.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "synclet.io/sync-job",
	})
	if err != nil {
		return 0, fmt.Errorf("listing managed K8s sync jobs: %w", err)
	}

	cleaned := 0

	for _, job := range jobs.Items {
		syncJobID := job.Labels["synclet.io/sync-job"]
		if syncJobID == "" {
			continue
		}

		if applyGrace && time.Since(job.CreationTimestamp.Time) < orphanGracePeriod {
			continue
		}

		active, err := oc.provider.IsJobActive(ctx, syncJobID)
		if err != nil {
			oc.logger.Warn("k8s orphan cleanup: failed to check sync job status",
				zap.String("sync_job_id", syncJobID), zap.Error(err))

			continue
		}

		if !active {
			oc.logger.Info("k8s orphan cleanup: removing orphaned sync job",
				zap.String("k8s_job", job.Name), zap.String("sync_job_id", syncJobID))
			oc.deleteK8sJob(ctx, job.Name)

			cleaned++
		}
	}

	return cleaned, nil
}

func (oc *OrphanCleaner) cleanupTaskJobs(ctx context.Context, applyGrace bool) (int, error) {
	jobs, err := oc.client.BatchV1().Jobs(oc.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "synclet.io/task",
	})
	if err != nil {
		return 0, fmt.Errorf("listing managed K8s task jobs: %w", err)
	}

	cleaned := 0

	for _, job := range jobs.Items {
		taskID := job.Labels["synclet.io/task"]
		if taskID == "" {
			continue
		}

		if applyGrace && time.Since(job.CreationTimestamp.Time) < orphanGracePeriod {
			continue
		}

		active, err := oc.provider.IsTaskActive(ctx, taskID)
		if err != nil {
			oc.logger.Warn("k8s orphan cleanup: failed to check task status",
				zap.String("task_id", taskID), zap.Error(err))

			continue
		}

		if !active {
			oc.logger.Info("k8s orphan cleanup: removing orphaned task job",
				zap.String("k8s_job", job.Name), zap.String("task_id", taskID))
			oc.deleteK8sJob(ctx, job.Name)

			cleaned++
		}
	}

	return cleaned, nil
}

func (oc *OrphanCleaner) cleanupSyncSecrets(ctx context.Context, applyGrace bool) (int, error) {
	secrets, err := oc.client.CoreV1().Secrets(oc.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "synclet.io/sync-job",
	})
	if err != nil {
		return 0, fmt.Errorf("listing managed K8s sync secrets: %w", err)
	}

	cleaned := 0

	for _, secret := range secrets.Items {
		syncJobID := secret.Labels["synclet.io/sync-job"]
		if syncJobID == "" {
			continue
		}

		if applyGrace && time.Since(secret.CreationTimestamp.Time) < orphanGracePeriod {
			continue
		}

		active, err := oc.provider.IsJobActive(ctx, syncJobID)
		if err != nil {
			oc.logger.Warn("k8s orphan cleanup: failed to check sync job status for secret",
				zap.String("sync_job_id", syncJobID), zap.Error(err))

			continue
		}

		if !active {
			oc.logger.Info("k8s orphan cleanup: removing orphaned sync secret",
				zap.String("secret", secret.Name), zap.String("sync_job_id", syncJobID))
			oc.deleteSecret(ctx, secret.Name)

			cleaned++
		}
	}

	return cleaned, nil
}

func (oc *OrphanCleaner) cleanupTaskSecrets(ctx context.Context, applyGrace bool) (int, error) {
	secrets, err := oc.client.CoreV1().Secrets(oc.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "synclet.io/task",
	})
	if err != nil {
		return 0, fmt.Errorf("listing managed K8s task secrets: %w", err)
	}

	cleaned := 0

	for _, secret := range secrets.Items {
		taskID := secret.Labels["synclet.io/task"]
		if taskID == "" {
			continue
		}

		if applyGrace && time.Since(secret.CreationTimestamp.Time) < orphanGracePeriod {
			continue
		}

		active, err := oc.provider.IsTaskActive(ctx, taskID)
		if err != nil {
			oc.logger.Warn("k8s orphan cleanup: failed to check task status for secret",
				zap.String("task_id", taskID), zap.Error(err))

			continue
		}

		if !active {
			oc.logger.Info("k8s orphan cleanup: removing orphaned task secret",
				zap.String("secret", secret.Name), zap.String("task_id", taskID))
			oc.deleteSecret(ctx, secret.Name)

			cleaned++
		}
	}

	return cleaned, nil
}

func (oc *OrphanCleaner) deleteSecret(ctx context.Context, name string) {
	if err := oc.client.CoreV1().Secrets(oc.namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		oc.logger.Error("k8s orphan cleanup: failed to delete secret",
			zap.String("secret", name), zap.Error(err))
	}
}

func (oc *OrphanCleaner) deleteK8sJob(ctx context.Context, jobName string) {
	propagation := metav1.DeletePropagationBackground
	if err := oc.client.BatchV1().Jobs(oc.namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}); err != nil && !errors.IsNotFound(err) {
		oc.logger.Error("k8s orphan cleanup: failed to delete K8s job",
			zap.String("k8s_job", jobName),
			zap.Error(err))
	}
}
