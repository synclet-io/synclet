package pipelineexeck8s

import (
	"context"
)

// K8sTaskLauncherAdapter adapts SyncRunner to the K8sTaskLauncher interface.
// It translates TaskOptions to ConnectorTaskOptions and calls LaunchConnectorTask.
type K8sTaskLauncherAdapter struct {
	runner *SyncRunner
}

// NewK8sTaskLauncherAdapter creates a new adapter.
func NewK8sTaskLauncherAdapter(runner *SyncRunner) *K8sTaskLauncherAdapter {
	return &K8sTaskLauncherAdapter{runner: runner}
}

// LaunchTask delegates to the SyncRunner.
func (a *K8sTaskLauncherAdapter) LaunchTask(ctx context.Context, opts TaskOptions) (string, error) {
	return a.runner.LaunchConnectorTask(ctx, ConnectorTaskOptions{
		TaskID:   opts.TaskID,
		TaskType: opts.TaskType.String(),
		Image:    opts.Image,
		Config:   opts.Config,
	})
}
