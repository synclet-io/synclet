package pipelineexeck8s

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	orphanGracePeriod = 15 * time.Minute

	sourceContainerName      = "source"
	destinationContainerName = "destination"
	oomKilledReason          = "OOMKilled"
	oomKilledExitCode        = 137
)

// ResourceChecker checks job/task status via API and reports failures.
type ResourceChecker interface {
	IsJobActive(ctx context.Context, jobID string) (bool, error)
	IsTaskActive(ctx context.Context, taskID string) (bool, error)
	ReportCompletion(ctx context.Context, params pipelineexec.ReportCompletionParams) error
}

// ResourceCleaner iterates the K8s Jobs and Secrets labelled with its owner id
// and reconciles them against the API.
// It handles:
// 1. Sync jobs whose DB job is no longer active (including cancelled) → delete K8s Job, no grace
// 2. Pod failure detection (CrashLoopBackOff, ImagePullBackOff, PodFailed) → fail job + delete K8s resources
// 3. Orphan cleanup for task jobs and secrets → delete K8s resources
type ResourceCleaner struct {
	client    kubernetes.Interface
	namespace string
	ownerID   string
	checker   ResourceChecker
	logger    *logging.Logger
}

// NewReconciler creates a new K8s reconciler.
func NewReconciler(
	client kubernetes.Interface,
	namespace string,
	ownerID string,
	checker ResourceChecker,
	logger *logging.Logger,
) *ResourceCleaner {
	return &ResourceCleaner{
		client:    client,
		namespace: namespace,
		ownerID:   ownerID,
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

// reconcileSyncJobs lists its own K8s Jobs with the sync-job label, checks pod status
// for failures, and cleans up orphans.
func (r *ResourceCleaner) reconcileSyncJobs(ctx context.Context, applyGrace bool) (int, error) {
	jobs, err := r.client.BatchV1().Jobs(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: r.ownedSelector("synclet.io/sync-job"),
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

		log := r.logger.WithFields(map[string]any{
			"k8s_job":     job.Name,
			"sync_job_id": syncJobID,
		})

		// A sync job is claimed (and active) before its K8s Job is created, so an
		// inactive DB job is terminal (completed, failed or cancelled) and its pod
		// is removed regardless of age.
		active, err := r.checker.IsJobActive(ctx, syncJobID)
		if err != nil {
			log.WithError(err).Warn(ctx, "failed to check job status")

			continue
		}

		if !active {
			log.Info(ctx, "removing sync job whose DB job is no longer active")
			r.deleteK8sJob(ctx, job.Name)

			cleaned++

			continue
		}

		if applyGrace && time.Since(job.CreationTimestamp.Time) < orphanGracePeriod {
			continue
		}

		if failure := r.checkPodFailure(ctx, job.Name); failure != nil {
			log.WithField("reason", failure.reason).Warn(ctx, "pod failure detected, failing job")
			r.reportPodFailure(ctx, log, syncJobID, job.Labels["synclet.io/connection"], failure)

			r.deleteK8sJob(ctx, job.Name)

			cleaned++
		}
	}

	return cleaned, nil
}

// cleanupTaskJobs lists its own K8s Jobs with the task label and removes orphans.
func (r *ResourceCleaner) cleanupTaskJobs(ctx context.Context, applyGrace bool) (int, error) {
	jobs, err := r.client.BatchV1().Jobs(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: r.ownedSelector("synclet.io/task"),
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

// cleanupSecrets removes its own orphaned secrets carrying labelKey.
func (r *ResourceCleaner) cleanupSecrets(ctx context.Context, labelKey string, applyGrace bool, isActive func(context.Context, string) (bool, error)) (int, error) {
	secrets, err := r.client.CoreV1().Secrets(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: r.ownedSelector(labelKey),
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

// ownedSelector selects the resources carrying labelKey that this executor owns.
func (r *ResourceCleaner) ownedSelector(labelKey string) string {
	return labelKey + "," + pipelineexec.OwnerLabel + "=" + r.ownerID
}

// podFailure is a terminal failure of a sync pod with the exit codes of its
// connector containers; an exit code is 0 where none is known.
type podFailure struct {
	reason         string
	sourceExitCode int
	destExitCode   int
}

// reportPodFailure fails the sync job through ReportCompletion with the
// connector exit codes read from the pod.
func (r *ResourceCleaner) reportPodFailure(ctx context.Context, log *logging.Logger, syncJobID, connectionLabel string, failure *podFailure) {
	jobID, err := uuid.Parse(syncJobID)
	if err != nil {
		log.WithError(err).Error(ctx, "sync job label is not a job id")

		return
	}

	// A missing connection id only skips the retention cleanup that follows completion.
	connectionID, _ := uuid.Parse(connectionLabel)

	if err := r.checker.ReportCompletion(ctx, pipelineexec.ReportCompletionParams{
		JobID:          jobID,
		ConnectionID:   connectionID,
		Success:        false,
		ErrorMessage:   failure.reason,
		SourceExitCode: failure.sourceExitCode,
		DestExitCode:   failure.destExitCode,
	}); err != nil {
		log.WithError(err).Error(ctx, "failed to mark job as failed")
	}
}

// checkPodFailure checks the pods of a K8s Job and returns the first terminal
// failure it finds, or nil when no pod has failed.
func (r *ResourceCleaner) checkPodFailure(ctx context.Context, k8sJobName string) *podFailure {
	pods, err := r.client.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "synclet.io/job=" + k8sJobName,
	})
	if err != nil {
		r.logger.WithError(err).WithField("k8s_job", k8sJobName).Error(ctx, "failed to list pods")

		return nil
	}

	// No pods yet means the pod may not have been scheduled; the orphan check
	// (IsJobActive) handles truly orphaned jobs.
	for _, pod := range pods.Items {
		if reason := podFailureReason(&pod); reason != "" {
			failure := &podFailure{reason: reason}
			failure.sourceExitCode, failure.destExitCode = connectorExitCodes(&pod)

			return failure
		}
	}

	return nil
}

// podFailureReason returns why the pod has failed for good, or "" when it has not.
func podFailureReason(pod *corev1.Pod) string {
	if pod.Status.Phase == corev1.PodFailed {
		return "pod failed: " + pod.Status.Reason
	}

	for _, status := range podContainerStatuses(pod) {
		if status.State.Waiting != nil {
			waiting := status.State.Waiting.Reason
			switch waiting {
			case "CrashLoopBackOff":
				return fmt.Sprintf("container %s in CrashLoopBackOff", status.Name)
			case "ImagePullBackOff", "ErrImagePull":
				return fmt.Sprintf("container %s: %s", status.Name, waiting)
			}
		}

		// Sync pods never restart a container, so an OOM kill is final.
		if status.State.Terminated != nil && status.State.Terminated.Reason == oomKilledReason {
			return fmt.Sprintf("container %s %s", status.Name, oomKilledReason)
		}
	}

	return ""
}

// connectorExitCodes returns the exit codes of the pod's terminated source and
// destination containers, 0 for a container that has not terminated.
func connectorExitCodes(pod *corev1.Pod) (source, dest int) {
	for _, status := range podContainerStatuses(pod) {
		switch status.Name {
		case sourceContainerName:
			source = containerExitCode(status)
		case destinationContainerName:
			dest = containerExitCode(status)
		}
	}

	return source, dest
}

// containerExitCode returns a terminated container's exit code; an OOM kill
// reports 137 whatever the runtime recorded.
func containerExitCode(status corev1.ContainerStatus) int {
	terminated := status.State.Terminated
	if terminated == nil {
		return 0
	}

	if terminated.Reason == oomKilledReason {
		return oomKilledExitCode
	}

	return int(terminated.ExitCode)
}

// podContainerStatuses returns the statuses of the pod's regular and init containers.
func podContainerStatuses(pod *corev1.Pod) []corev1.ContainerStatus {
	statuses := make([]corev1.ContainerStatus, 0, len(pod.Status.ContainerStatuses)+len(pod.Status.InitContainerStatuses))
	statuses = append(statuses, pod.Status.ContainerStatuses...)

	return append(statuses, pod.Status.InitContainerStatuses...)
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
