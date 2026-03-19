package k8s

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

// StaleJob represents a job that may need reconciliation.
type StaleJob struct {
	JobID      string
	K8sJobName string
}

// StaleJobProvider retrieves stale jobs from the database.
type StaleJobProvider interface {
	GetStaleJobs(ctx context.Context, heartbeatTimeout time.Duration) ([]StaleJob, error)
	FailJob(ctx context.Context, jobID string, reason string) error
	IsJobActive(ctx context.Context, jobID string) (bool, error)
	IsTaskActive(ctx context.Context, taskID string) (bool, error)
}

// Reconciler detects and cleans up orphaned/stale K8s jobs.
type Reconciler struct {
	client    kubernetes.Interface
	namespace string
	provider  StaleJobProvider
	logger    *logging.Logger
	interval  time.Duration
	timeout   time.Duration
}

// NewReconciler creates a new K8s job reconciler.
func NewReconciler(
	client kubernetes.Interface,
	namespace string,
	provider StaleJobProvider,
	logger *logging.Logger,
) *Reconciler {
	return &Reconciler{
		client:    client,
		namespace: namespace,
		provider:  provider,
		logger:    logger.Named("k8s-reconciler"),
		interval:  5 * time.Minute,
		timeout:   5 * time.Minute,
	}
}

// Run starts the reconciler loop. It runs immediately on start, then every 5 minutes.
func (r *Reconciler) Run(ctx context.Context) {
	r.logger.Info(ctx, "starting")

	// Run immediately on startup.
	r.reconcile(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info(ctx, "stopping")
			return
		case <-ticker.C:
			r.reconcile(ctx)
		}
	}
}

func (r *Reconciler) reconcile(ctx context.Context) {
	staleJobs, err := r.provider.GetStaleJobs(ctx, r.timeout)
	if err != nil {
		r.logger.WithError(err).Error(ctx, "failed to get stale jobs")
		return
	}

	if len(staleJobs) == 0 {
		return
	}

	r.logger.WithField("count", len(staleJobs)).Info(ctx, "found stale jobs")

	for _, sj := range staleJobs {
		r.reconcileJob(ctx, sj)
	}
}

func (r *Reconciler) reconcileJob(ctx context.Context, sj StaleJob) {
	log := r.logger.WithFields(
		map[string]any{
			"job_id":  sj.JobID,
			"k8s_job": sj.K8sJobName,
		},
	)

	if sj.K8sJobName == "" {
		// No K8s job name recorded — mark as failed.
		log.Warn(ctx, "orphaned job with no K8s job name")
		if err := r.provider.FailJob(ctx, sj.JobID, "orphaned — no K8s job name recorded"); err != nil {
			log.WithError(err).Error(ctx, "failed to mark job as failed")
		}
		return
	}

	// Check if K8s Job still exists.
	k8sJob, err := r.client.BatchV1().Jobs(r.namespace).Get(ctx, sj.K8sJobName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Warn(ctx, "K8s job not found, marking as failed")
			if err := r.provider.FailJob(ctx, sj.JobID, "orphaned — K8s job not found"); err != nil {
				log.WithError(err).Error(ctx, "failed to mark job as failed")
			}
			return
		}
		log.WithError(err).Error(ctx, "failed to get K8s job")
		return
	}

	// Check pod status.
	pods, err := r.client.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("synclet.io/job=%s", sj.K8sJobName),
	})
	if err != nil {
		log.WithError(err).Error(ctx, "failed to list pods")
		return
	}

	if len(pods.Items) == 0 {
		log.Warn(ctx, "no pods found for K8s job, marking as failed")
		r.deleteK8sJob(ctx, sj.K8sJobName)
		if err := r.provider.FailJob(ctx, sj.JobID, "orphaned — no pods found"); err != nil {
			log.WithError(err).Error(ctx, "failed to mark job as failed")
		}
		return
	}

	pod := pods.Items[0]

	// Check for failed/crashloopbackoff pods.
	if pod.Status.Phase == corev1.PodFailed {
		log.Warn(ctx, "pod failed, cleaning up")
		r.deleteK8sJob(ctx, sj.K8sJobName)
		if err := r.provider.FailJob(ctx, sj.JobID, fmt.Sprintf("pod failed: %s", pod.Status.Reason)); err != nil {
			log.WithError(err).Error(ctx, "failed to mark job as failed")
		}
		return
	}

	// Check both regular and init container statuses for failure states.
	allStatuses := make([]corev1.ContainerStatus, 0, len(pod.Status.ContainerStatuses)+len(pod.Status.InitContainerStatuses))
	allStatuses = append(allStatuses, pod.Status.ContainerStatuses...)
	allStatuses = append(allStatuses, pod.Status.InitContainerStatuses...)
	for _, cs := range allStatuses {
		if cs.State.Waiting != nil {
			reason := cs.State.Waiting.Reason
			switch reason {
			case "CrashLoopBackOff":
				log.WithField("container", cs.Name).Warn(ctx, "container in CrashLoopBackOff, cleaning up")
				r.deleteK8sJob(ctx, sj.K8sJobName)
				if err := r.provider.FailJob(ctx, sj.JobID, fmt.Sprintf("container %s in CrashLoopBackOff", cs.Name)); err != nil {
					log.WithError(err).Error(ctx, "failed to mark job as failed")
				}
				return
			case "ImagePullBackOff", "ErrImagePull":
				log.WithFields(map[string]any{
					"container": cs.Name,
					"reason":    reason,
				}).Warn(ctx, "container image pull failed")
				r.deleteK8sJob(ctx, sj.K8sJobName)
				if err := r.provider.FailJob(ctx, sj.JobID, fmt.Sprintf("container %s: %s", cs.Name, reason)); err != nil {
					log.WithError(err).Error(ctx, "failed to mark job as failed")
				}
				return
			}
		}
	}

	// Pod is still running — leave it alone (orchestrator may reconnect).
	_ = k8sJob
	log.Info(ctx, "K8s job still running, leaving it")
}

func (r *Reconciler) deleteK8sJob(ctx context.Context, jobName string) {
	propagation := metav1.DeletePropagationBackground
	if err := r.client.BatchV1().Jobs(r.namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}); err != nil && !errors.IsNotFound(err) {
		r.logger.WithError(err).WithField("k8s_job", jobName).Error(ctx, "failed to delete K8s job")
	}
}
