package pipelineexeck8s

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const orphanGracePeriod = 15 * time.Minute

// ResourceChecker checks job/task status via API and reports failures.
type ResourceChecker interface {
	IsJobActive(ctx context.Context, jobID string) (bool, error)
	IsTaskActive(ctx context.Context, taskID string) (bool, error)
	FailJob(ctx context.Context, jobID string, reason string) error
}

// ResourceCleaner iterates K8s resources and reconciles them against the API.
// It handles:
// 1. Pod failure detection (CrashLoopBackOff, ImagePullBackOff, PodFailed) → fail job + delete K8s resources
// 2. Orphan cleanup for sync jobs, task jobs, and secrets → delete K8s resources
type ResourceCleaner struct {
	client    kubernetes.Interface
	namespace string
	checker   ResourceChecker
	logger    *logging.Logger
}

// NewReconciler creates a new K8s reconciler.
func NewReconciler(
	client kubernetes.Interface,
	namespace string,
	checker ResourceChecker,
	logger *logging.Logger,
) *ResourceCleaner {
	return &ResourceCleaner{
		client:    client,
		namespace: namespace,
		checker:   checker,
		logger:    logger.Named("k8s-reconciler"),
	}
}

// Reconcile iterates K8s resources and reconciles them against the API.
// When applyGrace is true, recently created resources are skipped.
func (r *ResourceCleaner) Reconcile(ctx context.Context, applyGrace bool) {
	cleaned := 0

	// Reconcile sync jobs (pod status checks + orphan cleanup).
	n, err := r.reconcileSyncJobs(ctx, applyGrace)
	if err != nil {
		r.logger.WithError(err).Error(ctx, "failed to reconcile sync jobs")
	}

	cleaned += n

	// Clean orphaned task jobs.
	n, err = r.cleanupTaskJobs(ctx, applyGrace)
	if err != nil {
		r.logger.WithError(err).Error(ctx, "failed to cleanup task jobs")
	}

	cleaned += n

	// Clean orphaned sync secrets.
	n, err = r.cleanupSecrets(ctx, "synclet.io/sync-job", applyGrace, func(ctx context.Context, id string) (bool, error) {
		return r.checker.IsJobActive(ctx, id)
	})
	if err != nil {
		r.logger.WithError(err).Error(ctx, "failed to cleanup sync secrets")
	}

	cleaned += n

	// Clean orphaned task secrets.
	n, err = r.cleanupSecrets(ctx, "synclet.io/task", applyGrace, func(ctx context.Context, id string) (bool, error) {
		return r.checker.IsTaskActive(ctx, id)
	})
	if err != nil {
		r.logger.WithError(err).Error(ctx, "failed to cleanup task secrets")
	}

	cleaned += n

	if cleaned > 0 {
		r.logger.WithField("count", cleaned).Info(ctx, "reconciled resources")
	}
}

// reconcileSyncJobs lists all K8s Jobs with sync-job label, checks pod status
// for failures, and cleans up orphans.
func (r *ResourceCleaner) reconcileSyncJobs(ctx context.Context, applyGrace bool) (int, error) {
	jobs, err := r.client.BatchV1().Jobs(r.namespace).List(ctx, metav1.ListOptions{
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

		log := r.logger.WithFields(map[string]any{
			"k8s_job":     job.Name,
			"sync_job_id": syncJobID,
		})

		// Check pod status for terminal failures.
		if failed, reason := r.checkPodFailure(ctx, job.Name); failed {
			log.WithField("reason", reason).Warn(ctx, "pod failure detected, failing job")

			if err := r.checker.FailJob(ctx, syncJobID, reason); err != nil {
				log.WithError(err).Error(ctx, "failed to mark job as failed")
			}

			r.deleteK8sJob(ctx, job.Name)

			cleaned++

			continue
		}

		// No pod failure — check if job is still active in DB.
		active, err := r.checker.IsJobActive(ctx, syncJobID)
		if err != nil {
			log.WithError(err).Warn(ctx, "failed to check job status")

			continue
		}

		if !active {
			log.Info(ctx, "removing orphaned sync job")
			r.deleteK8sJob(ctx, job.Name)

			cleaned++
		}
	}

	return cleaned, nil
}

// cleanupTaskJobs lists all K8s Jobs with task label and removes orphans.
func (r *ResourceCleaner) cleanupTaskJobs(ctx context.Context, applyGrace bool) (int, error) {
	jobs, err := r.client.BatchV1().Jobs(r.namespace).List(ctx, metav1.ListOptions{
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

		active, err := r.checker.IsTaskActive(ctx, taskID)
		if err != nil {
			r.logger.WithError(err).WithField("task_id", taskID).Warn(ctx, "failed to check task status")

			continue
		}

		if !active {
			r.logger.WithFields(map[string]any{
				"k8s_job": job.Name,
				"task_id": taskID,
			}).Info(ctx, "removing orphaned task job")
			r.deleteK8sJob(ctx, job.Name)

			cleaned++
		}
	}

	return cleaned, nil
}

// cleanupSecrets removes orphaned secrets matching a label selector.
func (r *ResourceCleaner) cleanupSecrets(ctx context.Context, labelKey string, applyGrace bool, isActive func(context.Context, string) (bool, error)) (int, error) {
	secrets, err := r.client.CoreV1().Secrets(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelKey,
	})
	if err != nil {
		return 0, fmt.Errorf("listing managed K8s secrets (%s): %w", labelKey, err)
	}

	cleaned := 0

	for _, secret := range secrets.Items {
		id := secret.Labels[labelKey]
		if id == "" {
			continue
		}

		if applyGrace && time.Since(secret.CreationTimestamp.Time) < orphanGracePeriod {
			continue
		}

		active, err := isActive(ctx, id)
		if err != nil {
			r.logger.WithError(err).WithField("id", id).Warn(ctx, "failed to check status for secret")

			continue
		}

		if !active {
			r.logger.WithFields(map[string]any{
				"secret": secret.Name,
				"id":     id,
			}).Info(ctx, "removing orphaned secret")
			r.deleteSecret(ctx, secret.Name)

			cleaned++
		}
	}

	return cleaned, nil
}

// checkPodFailure checks pods for a K8s Job and returns whether any pod is in a
// terminal failure state. Returns (failed, reason).
func (r *ResourceCleaner) checkPodFailure(ctx context.Context, k8sJobName string) (failed bool, reason string) {
	pods, err := r.client.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "synclet.io/job=" + k8sJobName,
	})
	if err != nil {
		r.logger.WithError(err).WithField("k8s_job", k8sJobName).Error(ctx, "failed to list pods")

		return false, ""
	}

	if len(pods.Items) == 0 {
		// No pods yet — pod may not have been scheduled. Don't treat as failure;
		// the orphan check (IsJobActive) handles truly orphaned jobs.
		return false, ""
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodFailed {
			return true, "pod failed: " + pod.Status.Reason
		}

		// Check both regular and init container statuses for failure states.
		allStatuses := make([]corev1.ContainerStatus, 0, len(pod.Status.ContainerStatuses)+len(pod.Status.InitContainerStatuses))
		allStatuses = append(allStatuses, pod.Status.ContainerStatuses...)
		allStatuses = append(allStatuses, pod.Status.InitContainerStatuses...)

		for _, status := range allStatuses {
			if status.State.Waiting != nil {
				waiting := status.State.Waiting.Reason
				switch waiting {
				case "CrashLoopBackOff":
					return true, fmt.Sprintf("container %s in CrashLoopBackOff", status.Name)
				case "ImagePullBackOff", "ErrImagePull":
					return true, fmt.Sprintf("container %s: %s", status.Name, waiting)
				}
			}
		}
	}

	return false, ""
}

func (r *ResourceCleaner) deleteK8sJob(ctx context.Context, jobName string) {
	propagation := metav1.DeletePropagationBackground
	if err := r.client.BatchV1().Jobs(r.namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}); err != nil && !errors.IsNotFound(err) {
		r.logger.WithError(err).WithField("k8s_job", jobName).Error(ctx, "failed to delete K8s job")
	}
}

func (r *ResourceCleaner) deleteSecret(ctx context.Context, name string) {
	if err := r.client.CoreV1().Secrets(r.namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		r.logger.WithError(err).WithField("secret", name).Error(ctx, "failed to delete secret")
	}
}
