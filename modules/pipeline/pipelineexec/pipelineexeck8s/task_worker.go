package pipelineexeck8s

import (
	"context"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// K8sTaskLauncher abstracts the K8s runner for launching connector task jobs.
type K8sTaskLauncher interface {
	LaunchTask(ctx context.Context, opts TaskOptions) (string, error)
}

// TaskOptions contains parameters to launch a connector task K8s Job.
type TaskOptions struct {
	TaskID   string
	TaskType pipelineservice.ConnectorTaskType
	Image    string // Connector image
	Config   []byte // Decrypted config JSON (nil for spec)
}

// K8sTaskWorker is a fire-and-forget jobber that claims pending connector tasks
// and creates K8s Jobs with 2-container pods (coordinator + connector).
// Per D-17, it does NOT wait for pod completion -- it returns immediately
// after submitting the K8s Job. Uses ExecutorBackend for all server
// communication per D-14.
type K8sTaskWorker struct {
	backend   pipelineexec.ExecutorBackend
	k8sRunner K8sTaskLauncher
	workerID  string
	logger    *logging.Logger
}

// NewK8sTaskWorker creates a new K8sTaskWorker.
func NewK8sTaskWorker(
	backend pipelineexec.ExecutorBackend,
	k8sRunner K8sTaskLauncher,
	workerID string,
	logger *logging.Logger,
) *K8sTaskWorker {
	return &K8sTaskWorker{
		backend:   backend,
		k8sRunner: k8sRunner,
		workerID:  workerID,
		logger:    logger,
	}
}

// Execute claims the next pending connector task and creates a K8s Job with a 2-container pod.
// Fire-and-forget per D-09: returns immediately after K8s Job creation.
func (w *K8sTaskWorker) Execute(ctx context.Context) error {
	claimedTask, err := w.backend.ClaimConnectorTask(ctx, w.workerID)
	if err != nil {
		w.logger.WithError(err).Error(ctx, "failed to claim connector task")

		return fmt.Errorf("claiming connector task: %w", err)
	}

	if claimedTask == nil {
		return nil
	}

	w.logger.WithFields(map[string]interface{}{"task_id": claimedTask.TaskID.String(), "task_type": claimedTask.TaskType, "image": claimedTask.Image, "config_len": len(claimedTask.Config)}).Info(ctx, "claimed connector task")

	opts := TaskOptions{
		TaskID:   claimedTask.TaskID.String(),
		TaskType: claimedTask.TaskType,
		Image:    claimedTask.Image,
		Config:   claimedTask.Config,
	}

	w.logger.WithField("task_id", claimedTask.TaskID.String()).Info(ctx, "launching k8s job")

	k8sJobName, err := w.k8sRunner.LaunchTask(ctx, opts)
	if err != nil {
		w.logger.WithError(err).WithField("task_id", claimedTask.TaskID.String()).Error(ctx, "failed to launch k8s job")
		// Report failure via backend so the task is not stuck.
		w.failTask(ctx, claimedTask, fmt.Errorf("creating k8s job: %w", err))

		return nil
	}

	w.logger.WithFields(map[string]interface{}{"task_id": claimedTask.TaskID.String(), "task_type": claimedTask.TaskType, "k8s_job_name": k8sJobName}).Info(ctx, "launched k8s connector task")

	return nil
}

// failTask reports a connector task failure via ExecutorBackend.
func (w *K8sTaskWorker) failTask(ctx context.Context, task *pipelineexec.ClaimConnectorTaskResult, reason error) {
	w.logger.WithError(reason).WithField("task_id", task.TaskID.String()).Error(ctx, "k8s connector task worker: task failed")

	if err := w.backend.ReportConnectorTaskResult(ctx, pipelineexec.ReportConnectorTaskResultParams{
		TaskID:       task.TaskID,
		Success:      false,
		ErrorMessage: reason.Error(),
	}); err != nil {
		w.logger.WithError(err).WithField("task_id", task.TaskID.String()).Error(ctx, "failed to report connector task result")
	}
}
