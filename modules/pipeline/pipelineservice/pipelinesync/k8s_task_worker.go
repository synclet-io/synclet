package pipelinesync

import (
	"context"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// K8sConnectorTaskLauncher abstracts the K8s runner for creating connector task jobs.
type K8sConnectorTaskLauncher interface {
	LaunchConnectorTask(ctx context.Context, opts ConnectorTaskOptions) (string, error)
}

// ConnectorTaskOptions contains parameters to launch a connector task K8s Job.
type ConnectorTaskOptions struct {
	TaskID         string
	TaskType       pipelineservice.ConnectorTaskType
	Image          string // Connector image
	Config         []byte // Decrypted config JSON (nil for spec)
	InternalAPIURL string // Server address for gRPC callbacks
}

// K8sConnectorTaskWorker is a fire-and-forget jobber that claims pending connector tasks
// and creates K8s Jobs with 2-container pods (coordinator + connector).
// Per D-17, it does NOT wait for pod completion -- it returns immediately
// after submitting the K8s Job. Uses ExecutorBackend for all server
// communication per D-14.
type K8sConnectorTaskWorker struct {
	backend        ExecutorBackend
	k8sRunner      K8sConnectorTaskLauncher
	internalAPIURL string
	workerID       string
	logger         *logging.Logger
}

// NewK8sConnectorTaskWorker creates a new K8sConnectorTaskWorker.
func NewK8sConnectorTaskWorker(
	backend ExecutorBackend,
	k8sRunner K8sConnectorTaskLauncher,
	internalAPIURL string,
	workerID string,
	logger *logging.Logger,
) *K8sConnectorTaskWorker {
	return &K8sConnectorTaskWorker{
		backend:        backend,
		k8sRunner:      k8sRunner,
		internalAPIURL: internalAPIURL,
		workerID:       workerID,
		logger:         logger,
	}
}

// Execute claims the next pending connector task and creates a K8s Job with a 2-container pod.
// Fire-and-forget per D-09: returns immediately after K8s Job creation.
func (w *K8sConnectorTaskWorker) Execute(ctx context.Context) error {
	result, err := w.backend.ClaimConnectorTask(ctx, w.workerID)
	if err != nil {
		if w.logger != nil {
			w.logger.WithError(err).Error(ctx, "failed to claim connector task")
		}

		return fmt.Errorf("claiming connector task: %w", err)
	}

	if result == nil {
		return nil
	}

	if w.logger != nil {
		w.logger.WithFields(map[string]interface{}{"task_id": result.TaskID.String(), "task_type": result.TaskType, "image": result.Image, "config_len": len(result.Config)}).Info(ctx, "claimed connector task")
	}

	opts := ConnectorTaskOptions{
		TaskID:         result.TaskID.String(),
		TaskType:       result.TaskType,
		Image:          result.Image,
		Config:         result.Config,
		InternalAPIURL: w.internalAPIURL,
	}

	if w.logger != nil {
		w.logger.WithFields(map[string]interface{}{"task_id": result.TaskID.String(), "internal_api_url": w.internalAPIURL}).Info(ctx, "launching k8s job")
	}

	k8sJobName, err := w.k8sRunner.LaunchConnectorTask(ctx, opts)
	if err != nil {
		if w.logger != nil {
			w.logger.WithError(err).WithField("task_id", result.TaskID.String()).Error(ctx, "failed to launch k8s job")
		}
		// Report failure via backend so the task is not stuck.
		w.failTask(ctx, result, fmt.Errorf("creating k8s job: %w", err))

		return nil
	}

	if w.logger != nil {
		w.logger.WithFields(map[string]interface{}{"task_id": result.TaskID.String(), "task_type": result.TaskType, "k8s_job_name": k8sJobName}).Info(ctx, "launched k8s connector task")
	}

	return nil
}

// failTask reports a connector task failure via ExecutorBackend.
func (w *K8sConnectorTaskWorker) failTask(ctx context.Context, task *ClaimConnectorTaskResult, reason error) {
	if w.logger != nil {
		w.logger.WithError(reason).WithField("task_id", task.TaskID.String()).Error(ctx, "k8s connector task worker: task failed")
	}

	if err := w.backend.ReportConnectorTaskResult(ctx, ReportConnectorTaskResultParams{
		TaskID:       task.TaskID,
		Success:      false,
		ErrorMessage: reason.Error(),
	}); err != nil {
		if w.logger != nil {
			w.logger.WithError(err).WithField("task_id", task.TaskID.String()).Error(ctx, "failed to report connector task result")
		}
	}
}
