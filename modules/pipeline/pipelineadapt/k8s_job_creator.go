package pipelineadapt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"
	corev1 "k8s.io/api/core/v1"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesync"
	"github.com/synclet-io/synclet/pkg/k8s"
)

// K8sJobCreatorAdapter adapts k8s.SyncRunner to the pipelinesync.K8sJobCreator interface.
// It translates K8sSyncJobOptions into k8s.SyncOptions and calls LaunchSync.
type K8sJobCreatorAdapter struct {
	runner          *k8s.SyncRunner
	runtimeDefaults pipelineservice.RuntimeDefaults
	logger          *logging.Logger
}

// NewK8sJobCreatorAdapter creates a new K8sJobCreator adapter.
func NewK8sJobCreatorAdapter(runner *k8s.SyncRunner, runtimeDefaults pipelineservice.RuntimeDefaults, logger *logging.Logger) *K8sJobCreatorAdapter {
	return &K8sJobCreatorAdapter{runner: runner, runtimeDefaults: runtimeDefaults, logger: logger}
}

// CreateSyncJob translates K8sSyncJobOptions to k8s.SyncOptions and launches a K8s Job.
func (a *K8sJobCreatorAdapter) CreateSyncJob(ctx context.Context, opts pipelinesync.K8sSyncJobOptions) (string, error) {
	// Resolve runtime config for resource limits.
	var runtimeConfigPtr *string
	if opts.RuntimeConfig != "" {
		runtimeConfigPtr = &opts.RuntimeConfig
	}
	runtimeCfg := pipelineservice.ResolveRuntimeConfig(a.runtimeDefaults, pipelineservice.ParseRuntimeConfig(runtimeConfigPtr))
	srcMemLimit, srcCPULimit, srcMemReq, srcCPUReq := pipelineservice.ToContainerResources(runtimeCfg)

	// Parse K8s scheduling fields from runtime config.
	var tolerations []corev1.Toleration
	if len(runtimeCfg.Tolerations) > 0 {
		if err := json.Unmarshal(runtimeCfg.Tolerations, &tolerations); err != nil {
			a.logger.WithError(err).Warn(ctx, "failed to parse tolerations from runtime config")
		}
	}
	var nodeSelector map[string]string
	if len(runtimeCfg.NodeSelector) > 0 {
		if err := json.Unmarshal(runtimeCfg.NodeSelector, &nodeSelector); err != nil {
			a.logger.WithError(err).Warn(ctx, "failed to parse nodeSelector from runtime config")
		}
	}
	var affinity *corev1.Affinity
	if len(runtimeCfg.Affinity) > 0 {
		affinity = &corev1.Affinity{}
		if err := json.Unmarshal(runtimeCfg.Affinity, affinity); err != nil {
			a.logger.WithError(err).Warn(ctx, "failed to parse affinity from runtime config")
			affinity = nil
		}
	}

	k8sJobName, err := a.runner.LaunchSync(ctx, k8s.SyncOptions{
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

		// Use source runtime config for resource limits.
		SourceMemoryLimit:   srcMemLimit,
		SourceCPULimit:      srcCPULimit,
		SourceMemoryRequest: srcMemReq,
		SourceCPURequest:    srcCPUReq,

		// Pod-level scheduling.
		Tolerations:        tolerations,
		NodeSelector:       nodeSelector,
		Affinity:           affinity,
		ServiceAccountName: runtimeCfg.ServiceAccountName,
	})
	if err != nil {
		return "", fmt.Errorf("launching k8s sync: %w", err)
	}

	return k8sJobName, nil
}
