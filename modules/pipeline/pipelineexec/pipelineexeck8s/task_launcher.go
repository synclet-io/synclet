package pipelineexeck8s

import (
	"context"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// K8sTaskLauncherAdapter adapts SyncRunner to the K8sTaskLauncher interface.
// It translates TaskOptions to ConnectorTaskOptions and calls LaunchConnectorTask.
type K8sTaskLauncherAdapter struct {
	runner          *SyncRunner
	runtimeDefaults pipelineservice.RuntimeDefaults
}

// NewK8sTaskLauncherAdapter creates a new adapter.
func NewK8sTaskLauncherAdapter(runner *SyncRunner, runtimeDefaults pipelineservice.RuntimeDefaults) *K8sTaskLauncherAdapter {
	return &K8sTaskLauncherAdapter{runner: runner, runtimeDefaults: runtimeDefaults}
}

// LaunchTask delegates to the SyncRunner. The pod runs as the task's entity
// account when it names one, else as the global default.
func (a *K8sTaskLauncherAdapter) LaunchTask(ctx context.Context, opts TaskOptions) (string, error) {
	serviceAccount := opts.ServiceAccountName
	if serviceAccount == "" {
		serviceAccount = a.runtimeDefaults.ServiceAccountName
	}

	return a.runner.LaunchConnectorTask(ctx, ConnectorTaskOptions{
		TaskID:   opts.TaskID,
		TaskType: opts.TaskType.String(),
		Image:    opts.Image,
		Config:   opts.Config,

		ServiceAccountName: serviceAccount,
	})
}
