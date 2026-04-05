package pipelineexeck8s

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"
	corev1 "k8s.io/api/core/v1"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// K8sSyncLauncherAdapter adapts SyncRunner to the K8sSyncLauncher interface.
// It translates K8sSyncJobOptions into SyncOptions and calls LaunchSync.
type K8sSyncLauncherAdapter struct {
	runner          *SyncRunner
	runtimeDefaults pipelineservice.RuntimeDefaults
	logger          *logging.Logger
}

// NewK8sSyncLauncherAdapter creates a new K8sSyncLauncher adapter.
func NewK8sSyncLauncherAdapter(runner *SyncRunner, runtimeDefaults pipelineservice.RuntimeDefaults, logger *logging.Logger) *K8sSyncLauncherAdapter {
	return &K8sSyncLauncherAdapter{runner: runner, runtimeDefaults: runtimeDefaults, logger: logger}
}

// LaunchSyncJob translates K8sSyncJobOptions to SyncOptions and launches a K8s Job.
func (a *K8sSyncLauncherAdapter) LaunchSyncJob(ctx context.Context, opts K8sSyncJobOptions) (string, error) {
	// Resolve runtime config for source resource limits.
	var srcRuntimeConfigPtr *string
	if opts.RuntimeConfig != "" {
		srcRuntimeConfigPtr = &opts.RuntimeConfig
	}

	srcRuntimeCfg := pipelineservice.ResolveRuntimeConfig(a.runtimeDefaults, pipelineservice.ParseRuntimeConfig(srcRuntimeConfigPtr))
	srcMemLimit, srcCPULimit, srcMemReq, srcCPUReq := pipelineservice.ToContainerResources(srcRuntimeCfg)

	// Resolve runtime config for destination resource limits.
	var destRuntimeConfigPtr *string
	if opts.DestRuntimeConfig != "" {
		destRuntimeConfigPtr = &opts.DestRuntimeConfig
	}

	destRuntimeCfg := pipelineservice.ResolveRuntimeConfig(a.runtimeDefaults, pipelineservice.ParseRuntimeConfig(destRuntimeConfigPtr))
	destMemLimit, destCPULimit, destMemReq, destCPUReq := pipelineservice.ToContainerResources(destRuntimeCfg)

	// Parse K8s scheduling fields from source runtime config (pod-level scheduling).
	var tolerations []corev1.Toleration
	if len(srcRuntimeCfg.Tolerations) > 0 {
		if err := json.Unmarshal(srcRuntimeCfg.Tolerations, &tolerations); err != nil {
			a.logger.WithError(err).Warn(ctx, "failed to parse tolerations from runtime config")
		}
	}

	var nodeSelector map[string]string
	if len(srcRuntimeCfg.NodeSelector) > 0 {
		if err := json.Unmarshal(srcRuntimeCfg.NodeSelector, &nodeSelector); err != nil {
			a.logger.WithError(err).Warn(ctx, "failed to parse nodeSelector from runtime config")
		}
	}

	var affinity *corev1.Affinity
	if len(srcRuntimeCfg.Affinity) > 0 {
		affinity = &corev1.Affinity{}
		if err := json.Unmarshal(srcRuntimeCfg.Affinity, affinity); err != nil {
			a.logger.WithError(err).Warn(ctx, "failed to parse affinity from runtime config")

			affinity = nil
		}
	}

	k8sJobName, err := a.runner.LaunchSync(ctx, SyncOptions{
		JobID:         opts.JobID.String(),
		ConnectionID:  opts.ConnectionID.String(),
		SourceID:      opts.SourceID.String(),
		DestinationID: opts.DestinationID.String(),
		SourceImage:   opts.SourceImage,
		SourceConfig:  opts.SourceConfig,
		DestImage:     opts.DestImage,
		DestConfig:    opts.DestConfig,
		SourceCatalog: opts.SourceCatalog,
		DestCatalog:   opts.DestCatalog,
		State:         opts.State,

		NamespaceDefinition:   opts.NamespaceDefinition,
		CustomNamespaceFormat: opts.CustomNamespaceFormat,
		StreamPrefix:          opts.StreamPrefix,

		// Per-container resource limits.
		SourceMemoryLimit:   srcMemLimit,
		SourceCPULimit:      srcCPULimit,
		SourceMemoryRequest: srcMemReq,
		SourceCPURequest:    srcCPUReq,
		DestMemoryLimit:     destMemLimit,
		DestCPULimit:        destCPULimit,
		DestMemoryRequest:   destMemReq,
		DestCPURequest:      destCPUReq,

		// Pod-level scheduling.
		Tolerations:        tolerations,
		NodeSelector:       nodeSelector,
		Affinity:           affinity,
		ServiceAccountName: srcRuntimeCfg.ServiceAccountName,
	})
	if err != nil {
		return "", fmt.Errorf("launching k8s sync: %w", err)
	}

	return k8sJobName, nil
}
