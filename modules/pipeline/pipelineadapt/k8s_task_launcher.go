package pipelineadapt

import (
	"context"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesync"
	"github.com/synclet-io/synclet/pkg/k8s"
)

// K8sConnectorTaskLauncherAdapter adapts k8s.SyncRunner to the pipelinesync.K8sConnectorTaskLauncher interface.
// It translates ConnectorTaskOptions to k8s.ConnectorTaskOptions and calls LaunchConnectorTask.
type K8sConnectorTaskLauncherAdapter struct {
	runner *k8s.SyncRunner
}

// NewK8sConnectorTaskLauncherAdapter creates a new adapter.
func NewK8sConnectorTaskLauncherAdapter(runner *k8s.SyncRunner) *K8sConnectorTaskLauncherAdapter {
	return &K8sConnectorTaskLauncherAdapter{runner: runner}
}

// LaunchConnectorTask delegates to the k8s.SyncRunner.
func (a *K8sConnectorTaskLauncherAdapter) LaunchConnectorTask(ctx context.Context, opts pipelinesync.ConnectorTaskOptions) (string, error) {
	return a.runner.LaunchConnectorTask(ctx, k8s.ConnectorTaskOptions{
		TaskID:         opts.TaskID,
		TaskType:       opts.TaskType.String(),
		Image:          opts.Image,
		Config:         opts.Config,
		InternalAPIURL: opts.InternalAPIURL,
	})
}
