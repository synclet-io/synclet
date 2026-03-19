package pipelineadapt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/pkg/k8s"
)

// K8sSyncLauncherAdapter adapts k8s.SyncRunner to the pipelineservice.K8sSyncLauncher interface.
// It resolves all configs via use cases and launches a K8s pod.
type K8sSyncLauncherAdapter struct {
	runner                *k8s.SyncRunner
	getLaunchBundle       *pipelinejobs.GetLaunchBundle
	countConnectionJobs   *pipelinejobs.CountConnectionJobs
	getConfiguredCatalog  *pipelinecatalog.GetConfiguredCatalog
	populateGenerationIDs *pipelinecatalog.PopulateGenerationIDs
	getSyncState          *pipelinestate.GetSyncState
	setK8sJobName         *pipelinejobs.SetK8sJobName
	runtimeDefaults       pipelineservice.RuntimeDefaults
	logger                *logging.Logger
}

// NewK8sSyncLauncherAdapter creates a new K8s sync launcher adapter.
func NewK8sSyncLauncherAdapter(
	runner *k8s.SyncRunner,
	getLaunchBundle *pipelinejobs.GetLaunchBundle,
	countConnectionJobs *pipelinejobs.CountConnectionJobs,
	getConfiguredCatalog *pipelinecatalog.GetConfiguredCatalog,
	populateGenerationIDs *pipelinecatalog.PopulateGenerationIDs,
	getSyncState *pipelinestate.GetSyncState,
	setK8sJobName *pipelinejobs.SetK8sJobName,
	runtimeDefaults pipelineservice.RuntimeDefaults,
	logger *logging.Logger,
) *K8sSyncLauncherAdapter {
	return &K8sSyncLauncherAdapter{
		runner:                runner,
		getLaunchBundle:       getLaunchBundle,
		countConnectionJobs:   countConnectionJobs,
		getConfiguredCatalog:  getConfiguredCatalog,
		populateGenerationIDs: populateGenerationIDs,
		getSyncState:          getSyncState,
		setK8sJobName:         setK8sJobName,
		runtimeDefaults:       runtimeDefaults,
		logger:                logger,
	}
}

// Launch resolves all job configs via use cases and launches a K8s sync pod.
func (a *K8sSyncLauncherAdapter) Launch(ctx context.Context, jobID uuid.UUID) error {
	// Load the job and all related entities via use case.
	bundle, err := a.getLaunchBundle.Execute(ctx, pipelinejobs.GetLaunchBundleParams{JobID: jobID})
	if err != nil {
		return err
	}

	job := bundle.Job
	conn := bundle.Connection
	src := bundle.Source
	dest := bundle.Destination
	srcMC := bundle.SourceManagedConnector
	destMC := bundle.DestManagedConnector

	// Get configured catalog.
	catalog, err := a.getConfiguredCatalog.Execute(ctx, pipelinecatalog.GetConfiguredCatalogParams{
		ConnectionID: job.ConnectionID,
	})
	if err != nil {
		return fmt.Errorf("loading configured catalog: %w", err)
	}

	// Populate sync metadata (sync_id, generation_id, minimum_generation_id) so
	// Airbyte CDK connectors receive valid non-zero integers instead of defaults.
	jobCount, err := a.countConnectionJobs.Execute(ctx, pipelinejobs.CountConnectionJobsParams{
		ConnectionID: job.ConnectionID,
	})
	if err != nil {
		return fmt.Errorf("counting jobs for sync_id: %w", err)
	}
	if err := a.populateGenerationIDs.Execute(ctx, pipelinecatalog.PopulateGenerationIDsParams{
		ConnectionID: job.ConnectionID,
		Catalog:      catalog,
		SyncID:       jobCount,
	}); err != nil {
		return fmt.Errorf("populating generation IDs: %w", err)
	}

	// Build destination catalog with namespace/prefix rewriting.
	// Source catalog keeps original namespaces; destination catalog gets rewritten.
	destCatalog, err := pipelinecatalog.BuildDestinationCatalog(catalog)
	if err != nil {
		return fmt.Errorf("building destination catalog: %w", err)
	}
	pipelinecatalog.ApplyNamespaceAndPrefix(destCatalog, conn.NamespaceDefinition, conn.CustomNamespaceFormat, conn.StreamPrefix)

	sourceCatalogJSON, err := json.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("marshaling source catalog: %w", err)
	}
	destCatalogJSON, err := json.Marshal(destCatalog)
	if err != nil {
		return fmt.Errorf("marshaling catalog: %w", err)
	}

	// Get state.
	stateBlob, err := a.getSyncState.Execute(ctx, pipelinestate.GetSyncStateParams{ConnectionID: job.ConnectionID})
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	var stateJSON []byte
	if len(stateBlob) > 0 {
		stateJSON = []byte(stateBlob)
	}

	// Resolve runtime config for source and destination.
	srcCfg := pipelineservice.ResolveRuntimeConfig(a.runtimeDefaults, pipelineservice.ParseRuntimeConfig(src.RuntimeConfig))
	destCfg := pipelineservice.ResolveRuntimeConfig(a.runtimeDefaults, pipelineservice.ParseRuntimeConfig(dest.RuntimeConfig))

	srcMemLimit, srcCPULimit, srcMemReq, srcCPUReq := pipelineservice.ToContainerResources(srcCfg)
	destMemLimit, destCPULimit, destMemReq, destCPUReq := pipelineservice.ToContainerResources(destCfg)

	// Parse K8s scheduling fields from source config (pod-level, so we merge from both).
	// Use source config for scheduling since it typically drives the pod placement.
	var tolerations []corev1.Toleration
	if len(srcCfg.Tolerations) > 0 {
		if err := json.Unmarshal(srcCfg.Tolerations, &tolerations); err != nil {
			a.logger.WithError(err).Warn(ctx, "failed to parse tolerations from runtime config")
		}
	}
	var nodeSelector map[string]string
	if len(srcCfg.NodeSelector) > 0 {
		if err := json.Unmarshal(srcCfg.NodeSelector, &nodeSelector); err != nil {
			a.logger.WithError(err).Warn(ctx, "failed to parse nodeSelector from runtime config")
		}
	}
	var affinity *corev1.Affinity
	if len(srcCfg.Affinity) > 0 {
		affinity = &corev1.Affinity{}
		if err := json.Unmarshal(srcCfg.Affinity, affinity); err != nil {
			a.logger.WithError(err).Warn(ctx, "failed to parse affinity from runtime config")
			affinity = nil
		}
	}

	// Launch the K8s job.
	k8sJobName, err := a.runner.LaunchSync(ctx, k8s.SyncOptions{
		JobID:         jobID.String(),
		ConnectionID:  job.ConnectionID.String(),
		SourceID:      src.ID.String(),
		DestinationID: dest.ID.String(),
		SourceImage:   srcMC.DockerImage + ":" + srcMC.DockerTag,
		SourceConfig:  []byte(src.Config),
		DestImage:     destMC.DockerImage + ":" + destMC.DockerTag,
		DestConfig:    []byte(dest.Config),
		SourceCatalog: sourceCatalogJSON,
		DestCatalog:   destCatalogJSON,
		State:         stateJSON,

		NamespaceDefinition:   conn.NamespaceDefinition.String(),
		CustomNamespaceFormat: ptrToString(conn.CustomNamespaceFormat),
		StreamPrefix:          ptrToString(conn.StreamPrefix),

		SourceMemoryLimit:   srcMemLimit,
		SourceCPULimit:      srcCPULimit,
		SourceMemoryRequest: srcMemReq,
		SourceCPURequest:    srcCPUReq,
		DestMemoryLimit:     destMemLimit,
		DestCPULimit:        destCPULimit,
		DestMemoryRequest:   destMemReq,
		DestCPURequest:      destCPUReq,

		Tolerations:        tolerations,
		NodeSelector:       nodeSelector,
		Affinity:           affinity,
		ServiceAccountName: srcCfg.ServiceAccountName,
	})
	if err != nil {
		return fmt.Errorf("launching k8s sync: %w", err)
	}

	// Store the K8s job name on the job record.
	if err := a.setK8sJobName.Execute(ctx, pipelinejobs.SetK8sJobNameParams{
		ID:         jobID,
		K8sJobName: k8sJobName,
	}); err != nil {
		return fmt.Errorf("setting k8s job name: %w", err)
	}

	return nil
}

func ptrToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
