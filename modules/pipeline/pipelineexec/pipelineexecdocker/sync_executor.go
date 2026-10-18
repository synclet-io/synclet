package pipelineexecdocker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-pnp/go-pnp/logging"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	pkgcontainer "github.com/synclet-io/synclet/pkg/container"
)

// dockerVolumesToMounts converts user-defined docker volumes into bind mounts.
func dockerVolumesToMounts(volumes []pipelineservice.DockerVolume) []pkgcontainer.BindMount {
	if len(volumes) == 0 {
		return nil
	}

	mounts := make([]pkgcontainer.BindMount, 0, len(volumes))
	for _, v := range volumes {
		mounts = append(mounts, pkgcontainer.BindMount{
			HostPath:      v.HostPath,
			ContainerPath: v.ContainerPath,
			ReadOnly:      v.ReadOnly,
		})
	}

	return mounts
}

// DockerSyncExecutor runs a sync job from the pre-loaded data of a SyncBundle
// and reports state, config updates and logs through the SyncReporter.
type DockerSyncExecutor struct {
	client          *ConnectorClient
	reporter        pipelineexec.SyncReporter
	runtimeDefaults pipelineservice.RuntimeDefaults
	idleTimeout     time.Duration
	logger          *logging.Logger
}

// DockerSyncExecutorParams holds all dependencies for DockerSyncExecutor.
type DockerSyncExecutorParams struct {
	Client          *ConnectorClient
	Reporter        pipelineexec.SyncReporter
	RuntimeDefaults pipelineservice.RuntimeDefaults
	IdleTimeout     time.Duration
	Logger          *logging.Logger
}

// NewDockerSyncExecutor creates a new sync executor.
func NewDockerSyncExecutor(params DockerSyncExecutorParams) *DockerSyncExecutor {
	return &DockerSyncExecutor{
		client:          params.Client,
		reporter:        params.Reporter,
		runtimeDefaults: params.RuntimeDefaults,
		idleTimeout:     params.IdleTimeout,
		logger:          params.Logger.Named("sync-executor"),
	}
}

// Execute runs a sync for the given bundle of pre-loaded data. onStarted is
// called once both the source and destination containers have started.
func (e *DockerSyncExecutor) Execute(ctx context.Context, bundle *pipelineexec.SyncBundle, onStarted func()) (*pipelineservice.SyncStats, error) {
	catalog := bundle.ConfiguredCatalog

	// Labels for container tracking and orphan cleanup.
	labels := map[string]string{
		"synclet.io/managed":       "true",
		"synclet.io/job-id":        bundle.Job.ID.String(),
		"synclet.io/connection-id": bundle.ConnectionID.String(),
	}

	// Resolve runtime config for source and set resource limits.
	srcRuntimeCfg := pipelineservice.ResolveRuntimeConfig(e.runtimeDefaults, pipelineservice.ParseRuntimeConfig(&bundle.SourceRuntimeConfig))

	srcMemLimit, srcCPULimit, _, _ := pipelineservice.ToContainerResources(srcRuntimeCfg)

	// Start source read.
	sourceStdout, sourceCleanup, err := e.client.Read(ctx, bundle.SourceImage, bundle.SourceConfig, catalog, bundle.StateBlob, labels, ResourceLimits{
		MemoryLimit: srcMemLimit,
		CPULimit:    srcCPULimit,
		NetworkMode: srcRuntimeCfg.DockerNetwork,
		ExtraMounts: dockerVolumesToMounts(srcRuntimeCfg.DockerVolumes),
	})
	if err != nil {
		return nil, fmt.Errorf("starting source read: %w", err)
	}
	defer sourceCleanup()

	// Build destination catalog with filtered schemas for selected fields.
	destCatalog, err := pipelineservice.BuildDestinationCatalog(catalog)
	if err != nil {
		return nil, fmt.Errorf("building destination catalog: %w", err)
	}

	// Apply namespace rewriting and stream prefix to dest catalog.
	// Source catalog retains original namespaces for filtering and rewriter mapping.
	pipelineservice.ApplyNamespaceAndPrefix(destCatalog, bundle.NamespaceDefinition, bundle.CustomNamespaceFormat, bundle.StreamPrefix)

	// Create pipe for source-to-dest message routing.
	srcPipeReader, srcPipeWriter := io.Pipe()

	// Resolve runtime config for destination and set resource limits.
	destRuntimeCfg := pipelineservice.ResolveRuntimeConfig(e.runtimeDefaults, pipelineservice.ParseRuntimeConfig(&bundle.DestRuntimeConfig))

	destMemLimit, destCPULimit, _, _ := pipelineservice.ToContainerResources(destRuntimeCfg)

	// Start destination write with pipe reader as stdin.
	destStdout, destCleanup, err := e.client.Write(ctx, bundle.DestImage, bundle.DestConfig, destCatalog, srcPipeReader, labels, ResourceLimits{
		MemoryLimit: destMemLimit,
		CPULimit:    destCPULimit,
		NetworkMode: destRuntimeCfg.DockerNetwork,
		ExtraMounts: dockerVolumesToMounts(destRuntimeCfg.DockerVolumes),
	})
	if err != nil {
		_ = srcPipeWriter.Close()
		_ = srcPipeReader.Close()

		return nil, fmt.Errorf("starting destination write: %w", err)
	}
	defer destCleanup()

	onStarted()

	// Create handler for Docker mode side effects.
	handler := NewDockerHandler(DockerHandlerParams{
		ConnectionID:  bundle.ConnectionID,
		SourceID:      bundle.SourceID,
		DestinationID: bundle.DestinationID,
		JobID:         bundle.Job.ID,
		Reporter:      e.reporter,
		Logger:        e.logger,
	})

	stopLogFlush := handler.StartLogFlush(ctx)
	defer stopLogFlush()

	// Wrap source output to filter record data by selected fields.
	filteredSource := pipelineexec.NewFilteringReader(sourceStdout, catalog)

	// Create namespace rewriter from the ORIGINAL source catalog (before namespace rewriting).
	// This maps source namespace/stream names to destination namespace/stream names in RECORD
	// and STATE messages before they are forwarded to the destination connector.
	rewriter := pipelineexec.NewNamespaceRewriter(catalog, bundle.NamespaceDefinition, bundle.CustomNamespaceFormat, bundle.StreamPrefix)

	// Route messages using the shared router.
	redactor := pipelineexec.NewRedactor(bundle.SourceConfig, bundle.DestConfig)

	routeStats, routeErr := pipelineexec.Run(ctx, filteredSource, srcPipeWriter, destStdout, pipelineexec.RedactHandler(handler, redactor), pipelineexec.RunConfig{
		IdleTimeout: e.idleTimeout,
		Rewriter:    rewriter,
	}, e.logger.Named("message-router"))

	// Map router stats to SyncStats.
	stats := &pipelineservice.SyncStats{}
	if routeStats != nil {
		stats.RecordsRead = routeStats.RecordsRead
		stats.BytesSynced = routeStats.BytesSynced
		stats.Duration = routeStats.Duration
	}

	return stats, routeErr
}
